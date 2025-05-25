package pkg

import "golang.org/x/crypto/bcrypt"

// 加盐字符串（salt），用于增加加密后的复杂度。
const salt = "wyzbenren"

func HashPassword(password string) string {
	// hash 加密密码
	saltedPassword := password + salt
	// Cost 决定了加密操作的复杂度
	hassedPassword, _ := bcrypt.GenerateFromPassword([]byte(saltedPassword), bcrypt.DefaultCost)
	return string(hassedPassword)
}

func CheckPassWord(hassedpassword, password string) error {
	saltedpassword := password + salt
	return bcrypt.CompareHashAndPassword([]byte(hassedpassword), []byte(saltedpassword))
}
