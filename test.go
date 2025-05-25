package main

import (
	"fmt"
	"ginwebproject1/internal"
	"ginwebproject1/internal/config"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 调用自定义初始化工作 返回路由
	router := internal.Exec()
	// 创建一个 http.Server 实例
	s := &http.Server{
		Addr:           "0.0.0.0:" + strconv.Itoa(config.Config.Port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 二进制左移 2^10 1,048,576 1024 * 1024 字节 = 1MB= 1mb 表示允许的最大 HTTP 请求头大小为 1MB
	}
	fmt.Printf("服务启动在端口：%d\n", config.Config.Port)
	err := s.ListenAndServe()
	if err != nil {
		zap.S().Panicf("监听失败,err:%v", err.Error())
	}
}
