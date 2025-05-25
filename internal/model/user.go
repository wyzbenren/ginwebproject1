package model

import "gorm.io/gorm"

// 预定义数据库模型
type User struct {
	gorm.Model
	Username string
	Password string
	Email    string
}
