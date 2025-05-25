package api

type RegisterRequest struct {
	// binding:"required"：字段不能为空。
	// binding:"required,email"：字段不能为空，且必须符合邮箱格式。
	UserName string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" bingding:"required"`
}

type UpdateRequest struct {
	Username string `json:"username" binding:"required"`
}
