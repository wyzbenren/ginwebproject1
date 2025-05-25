package router

import (
	"ginwebproject1/internal/logic"
	"ginwebproject1/internal/router/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	// 注册功能添加路由
	router := gin.Default() // 默认路由 具有logger和recovery
	// //cors.Default() 中间件来启用CORS支持。这将允许来自任何源的GET，POST和OPTIONS请求，并允许特定的标头和方法
	router.Use(cors.Default())
	// 注册post请求路径  logic.Register用于处理请求
	// 配置路由后，可以用POST方式访问地址127.0.0.1:9091/register触发logic.Register函数的代码逻辑
	router.POST("register", logic.Register)
	router.POST("login", logic.Login)
	{
		g1 := router.Group("user").Use(middleware.VerifyJWT())
		g1.POST("info", logic.Info)
		g1.POST("update", logic.Update)
		g1.POST("Delete", logic.Delete)
	}
	return router
}
