package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ginwebproject1/internal/config"
	"ginwebproject1/internal/model"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// redis Redis key 规范 s:gin-demo:xxx  s代表key的类型为string gin-demo为服务名 xxx为自定义值
// s代表string
// hs代表hashmap
// se代表set
// zs代表zset
// l代表list
// bf代表布隆过滤器
// hy代表hyperloglog
// b代表bitmap

func userInfoKey(id string) string {
	return fmt.Sprintf("s:ginwebproject1:%v", id)
}

func GetUserInfo(ctx context.Context, userId string) (*model.User, error) {
	// 从redis中查询用户信息
	result, err := config.RedisClient.Get(ctx, userInfoKey(userId)).Result()
	// 查询错误
	if err != nil {
		return nil, err
	}
	// 查询为空
	if result == "" {
		return nil, redis.Nil
	}
	// 将查询到的信息反序列化到结构体中
	var user model.User
	err = json.Unmarshal([]byte(result), &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func SetUserInfo(ctx context.Context, user model.User) error {
	// 序列化为json
	marshal, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// 0：代表 不过期（永久缓存）
	_, err = config.RedisClient.Set(ctx, userInfoKey(strconv.Itoa(int(user.ID))), marshal, 0).Result()
	if err != nil {
		return err
	}
	return nil
}

func RefreshUserInfo(ctx context.Context, userId string) (*model.User, error) {
	user := model.User{}
	tx := config.DB.Where("id=?", userId).First(&user)
	// 处理查询错误
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return &user, tx.Error
	}
	// 查询为空
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return &user, nil
	}
	// 写入到 redis
	err := SetUserInfo(ctx, user)
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func DeleteUserInfo(ctx context.Context, userId string) error {
	_, err := config.RedisClient.Del(ctx, userInfoKey(userId)).Result()
	if err != nil {
		return err
	}
	return err
}
