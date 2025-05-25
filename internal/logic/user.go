package logic

import (
	"errors"
	"ginwebproject1/internal/api"
	"ginwebproject1/internal/cache"
	"ginwebproject1/internal/config"
	"ginwebproject1/internal/model"
	"ginwebproject1/internal/router/middleware"
	"ginwebproject1/pkg"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Register(c *gin.Context) {
	var r api.RegisterRequest
	// 从请求体中解析 JSON 参数，绑定到 RegisterRequest 结构体上
	err := c.ShouldBindJSON(&r)

	if err != nil {
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		// 通过 *gin.Context 返回结构体中的内容
		return
	}
	// 逻辑请求
	u := model.User{
		Username: r.UserName,
		Password: pkg.HashPassword(r.Password),
		Email:    r.Email,
	}

	// 1. 处理重复名字情况
	var user model.User
	tx := config.DB.Where("username = ?", u.Username).First(&user)
	// 处理未找到以外的错误
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		zap.S().Errorf("Register queru user email:%v err:%v", u.Username, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return // 提前退出
	}
	// 处理用户名已存在
	if user.ID != 0 {
		// zap.S().Errorf("注册用户已存在：%s", user.Username)
		c.JSON(http.StatusOK, pkg.Fail(pkg.UserExistsErrCode))
		return
	}

	// 2. 处理相同邮箱情况
	// 重置 如果第一次查用户名时 user.Id != 0 user.Id != 0 永远为 true
	user = model.User{}
	tx = config.DB.Where("email = ?", u.Email).First(&user)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		zap.S().Errorf("Register queru user email:%v err:%v", u.Email, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return
	}

	if user.ID != 0 {
		// zap.S().Errorf("邮箱已存在：%s", user.Email)
		c.JSON(http.StatusOK, pkg.Fail(pkg.UserEmailExistsErrCode))
		return
	}

	// 无错误则注册
	tx = config.DB.Create(&u)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		zap.S().Errorf("Register Create user:%+v err:%v", u, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return
	}
	c.JSON(http.StatusOK, pkg.Success())
}

func Login(c *gin.Context) {
	var r api.LoginRequest
	// 从请求体中解析 JSON 参数， 绑定到结构体中
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
	}
	// 进入逻辑处理 查询用于书否在数据库内
	user := model.User{
		Username: r.Username,
	}
	tx := config.DB.Where("username = ?", user.Username).First(&user)
	// 出现未查询到用户以外 的错误
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		zap.S().Errorf("从数据库中查找用户 <%s>失败 err:%v", user.Username, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return
	}
	// 用户不存在的情况
	if user.ID == 0 {
		c.JSON(http.StatusOK, pkg.Fail(pkg.RecordNotFoundErrCode))
		return
	}
	// 存在 密码匹配
	if pkg.CheckPassWord(user.Password, r.Password) != nil {
		c.JSON(http.StatusOK, pkg.Fail(pkg.UserPasswordErrCode))
		return
	}
	// 成功 创建jwt
	j := middleware.NewJWT()
	// 每次登录创建新的token 防止token永久有效
	claims := jwt.MapClaims{
		"sub":      user.ID,                               // 用户ID
		"username": user.Username,                         //用户名
		"exp":      time.Now().Add(time.Hour * 24).Unix(), //期限
	}
	tokenString, err := j.GenerateJWT(claims)
	if err != nil {
		zap.S().Infof("[CreateToken] 生成token失败")
		// gin.H  map[string]interface{}简写
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "生成token失败",
		})
		return
	}
	c.JSON(http.StatusOK, pkg.SuccessWithData(tokenString))
}

func Info(c *gin.Context) {
	// 根据jwt取出用户信息
	claims, ok := c.Get("claims")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "未登录"})
		c.Abort()
		return
	}
	currentUser, ok := claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "claims 类型断言失败"})
		return
	}

	// 先获取用户数据库中的id
	userId := currentUser["sub"].(float64)

	if userId == 0 {
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		return
	}
	// 查询缓存
	u, err := cache.GetUserInfo(c.Request.Context(), strconv.Itoa(int(userId)))
	// 处理查询为空的情况
	if err != nil && !errors.Is(err, redis.Nil) {
		zap.S().Errorf("Info.cache.GetUserInfo  userId:%+v err:%v", userId, err)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return
	}
	// 查询存在的情况
	if u != nil {
		c.JSON(http.StatusOK, pkg.SuccessWithData(u))
		return
	}
	// 缓存中没有命中，就刷新缓存（查数据库）
	// 内部会调用数据库查询，并把查询结果写入 Redis 缓存
	u, err = cache.RefreshUserInfo(c.Request.Context(), strconv.Itoa(int(userId)))
	if err != nil {
		zap.S().Errorf("Info.refreshUserInfoCache  user:%+v err:%v", userId, err)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
	}

	c.JSON(http.StatusOK, pkg.SuccessWithData(u))

}

func Update(c *gin.Context) {
	// 用于更新用户名
	var r api.UpdateRequest
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		return
	}
	// 或取claims中的id 根据id修改username
	claims, _ := c.Get("claims")
	currentUser := claims.(jwt.MapClaims)
	userId := currentUser["sub"].(float64)
	if userId == 0 {
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		return
	}
	user := model.User{}
	config.DB.Where("id=?", userId).First(&user)
	// 更新
	tx := config.DB.Model(&user).Update("username", r.Username)
	if tx.Error != nil {
		zap.S().Errorf("Update  user:%+v err:%v", user, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		return
	}
	// 刷新redis缓存
	_, err = cache.RefreshUserInfo(c.Request.Context(), strconv.Itoa(int(userId)))
	if err != nil {
		zap.S().Errorf("Update.refreshUserInfoCache  user:%+v err:%v", user, err)
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		return
	}
	c.JSON(http.StatusOK, pkg.SuccessWithData(user))
}

func Delete(c *gin.Context) {
	// 根据claims中的id
	claims, _ := c.Get("claims")
	curentUser := claims.(jwt.MapClaims)
	userID := curentUser["sub"].(float64)

	if userID == 0 {
		c.JSON(http.StatusOK, pkg.Fail(pkg.ParamsErrCode))
		return
	}
	// 删除redis 和 数据库中的用户
	// 依旧先获取用户信息 再执行操作
	u := model.User{}
	config.DB.First(&u, userID)

	// 删除数据库中的用户
	tx := config.DB.Delete(&u)
	if tx.Error != nil {
		zap.S().Errorf("Delete  userId:%v err:%v", u.ID, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return
	}
	// 删除redis中的用户
	err := cache.DeleteUserInfo(c.Request.Context(), strconv.Itoa(int(userID)))
	if err != nil {
		zap.S().Errorf("Delete.DelUserInfo  userId:%v err:%v", u.ID, tx.Error)
		c.JSON(http.StatusOK, pkg.Fail(pkg.InternalErrCode))
		return
	}
	c.JSON(http.StatusOK, pkg.Success())
}
