package main

import (
	"fmt"
	"ginwebproject1/internal/router/middleware"

	"github.com/golang-jwt/jwt"
)

func main2() {
	jwtinfo := middleware.NewJWT()
	claims := jwt.MapClaims{
		"sub":  123123, // 用户 ID
		"name": "zz",   // 用户名
	}
	mykey, _ := jwtinfo.GenerateJWT(claims)
	prasetoken, _ := jwtinfo.PraseToken(mykey)
	fmt.Println(mykey)
	fmt.Println(prasetoken)
}
