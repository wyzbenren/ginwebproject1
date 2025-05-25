package internal

import (
	"context"
	"fmt"
	"ginwebproject1/internal/config"
	"ginwebproject1/internal/model"
	"ginwebproject1/internal/router"
	"io"
	"os"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func Exec() *gin.Engine {
	InitConfig()
	InitLogger()
	InitMysql()
	InitRedis()
	// InitLocalCache()
	return router.InitRouter()
}

func InitConfig() {
	// 初始化配置文件路径
	configFileName := "./etc/config.yaml"

	// 设置viper
	v := viper.New()
	v.SetConfigFile(configFileName)

	// 错误检查
	if err := v.ReadInConfig(); err != nil {
		zap.S().Panicf("config 加载失败 err:%v", err)
	}
	// Unmarshal 反序列化 填充到config
	if err := v.Unmarshal(&config.Config); err != nil {
		zap.S().Panicf("解析配置文件失败 err:%v", err)
	}
	fmt.Printf("配置文件加载成功：%+v\n", config.Config)
}

func InitMysql() {
	// 读取config
	c := config.Config.MysqlConf
	// 构造 MySQL 的 DSN 链接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.DB)

	// 提前声明 err 防止变量遮蔽(在外层声明)
	var err error
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 表名不加s
		},
	})
	config.DB = db
	// 创建表
	config.DB.AutoMigrate(&model.User{})

	//错误处理
	if err != nil {
		zap.S().Panicf("数据库初始化失败 err%v", err)
		// 获取全局 SugaredLogger，方便格式化日志输出
	}

}

func InitRedis() {
	//  空的、根级别的上下文对象（context），可以传递取消信号的管道
	ctx := context.Background()
	// 读取config
	c := config.Config
	// 初始化redis链接
	//  //创建一个支持单机 / 主从 / 哨兵 / 集群的 Redis 客户端对象，用于后续操作 Redis
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    strings.Split(c.RedisConf.Host, ","), //strings.Split(c.Host, ",") strings.Split(...) 会把它变成字符串切片（[]string），供客户端选择连接 支持多个地址
		Password: c.RedisConf.Password,
	})

	err := redisClient.Ping(ctx).Err() // 测试redis能否连通
	if err != nil {
		zap.S().Panicf("redis加载失败 err%v", err)
	}
	config.RedisClient = redisClient
}

func InitLocalCache() {
	// 初始化本地缓存
	// 适用于热数据、短期使用的数据
	// 设置缓存中每个条目的生命周期 有效时间设置为永久
	interval := 8760 * 100 * time.Hour
	//  创建默认配置，bigcache.Config 包含缓存的大小、清理间隔、并发分片数等配置
	c := bigcache.DefaultConfig(interval)
	ctx := context.Background()
	localCache, err := bigcache.New(ctx, c)
	if err != nil {
		zap.S().Panicf("初始化本地缓存失败 err:%+v", err)
	}
	config.LocalCache = localCache
}

func InitLogger() {
	encoder := getEncoder()
	loggerInfo := getWriterInfo()
	logLevel := zapcore.InfoLevel
	switch config.Config.LogConf.Level {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	}
	// 生成了一个 zapcore.Core，表示你日志系统的核心逻辑
	coreInfo := zapcore.NewCore(encoder, loggerInfo, logLevel)
	// 创建日志器
	logger := zap.New(coreInfo)
	// 替换全局默认日志器
	zap.ReplaceGlobals(logger)
}

func getEncoder() zapcore.Encoder {
	// 用于决定日志的输出格式
	// 创建默认生产环境配置
	productionEncoderConfig := zap.NewProductionEncoderConfig()
	// 设置时间格式
	productionEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	//  设置日志级别格式为大写（INFO、ERROR）
	productionEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(productionEncoderConfig)
}

func getWriterInfo() zapcore.WriteSyncer {
	// 决定日志的输出目标
	logPath := config.Config.LogConf.Path + "/" + config.Config.Name + ".log"
	// 用于控制log大小
	l := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    config.Config.LogConf.MaxSize,    //最大MB
		MaxBackups: config.Config.LogConf.MaxBackups, //最大备份数
		MaxAge:     config.Config.LogConf.MaxAge,     // 最大保留天数
		Compress:   true,
	}

	var ws io.Writer
	// 根据运行环境选择输出目标
	if config.Config.Mode == "release" {
		ws = io.MultiWriter(l)
	} else {
		// 如果不是开发环境，那么会打印日志到日志文件和标准输出，也就是控制台
		ws = io.MultiWriter(l, os.Stdout)
	}
	return zapcore.AddSync(ws) // 将io.writer 包装成zapcore.WriteSyncer
}
