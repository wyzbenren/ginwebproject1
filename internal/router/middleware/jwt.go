package middleware

import (
	"fmt"
	"ginwebproject1/pkg"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
)

const Secret = "zhouzhan"

type JWTINFO struct {
	privateKey []byte
	publicKey  []byte
}

func NewJWT() *JWTINFO {
	// 加载 JWT 公钥和私钥 的函数，目的是为了让你的服务能创建和验证 基于 RSA 非对称加密的 JWT Token
	var err error
	var publicKey []byte
	var privateKey []byte

	privateKey, err = os.ReadFile("./internal/router/middleware/private.key")
	if err != nil {
		log.Fatalf("私钥加载失败 %v", err)
	}
	publicKey, err = os.ReadFile("./internal/router/middleware/public.key")
	if err != nil {
		log.Fatalf("公钥加载失败 %v", err)
	}
	return &JWTINFO{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

// 生成签名后的jwt
func (j *JWTINFO) GenerateJWT(claims jwt.MapClaims) (string, error) {
	// 用 RSA 私钥生成 JWT JSON Web Token
	// 接收一组 claims（声明/负载），类型是 jwt.MapClaims（底层是 map[string]interface{}）
	// 返回签名后的 JWT 字符串 和可能的 error

	// 先解析私钥
	rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(j.privateKey)
	if err != nil {
		return "", fmt.Errorf("解析RSA加密私钥失败:%v", err)
	}
	// 构建带声明的 JWT Token
	// claims  要放到JWT 负载里的数据
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// 用刚才解析得到的 rsaPrivateKey 对 token 进行签名
	// 返回的是一个完整的 JWT 字符串（格式：header.payload.signature），可以在网络中传输
	tokenString, err := token.SignedString(rsaPrivateKey)
	if err != nil {
		return "", fmt.Errorf("签名失败 err %v", err)
	}
	return tokenString, nil
}

// 解析签名后的jwt xxxxx.yyyyy.zzzzz  Header（含 alg）Payload（含 claims）Signature（签名）
func (j *JWTINFO) PraseToken(tokenString string) (jwt.MapClaims, error) {
	// 解析公钥
	rsaPublicKey, err := jwt.ParseRSAPublicKeyFromPEM(j.publicKey)
	if err != nil {
		return nil, fmt.Errorf("解析rsa公钥失败 err:%v", err)
	}
	// 解析并验证 token
	// func 回调函数
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			// 手动验证签名方式 防止伪造一个 token，把 alg 改成 none
			return nil, fmt.Errorf("签名方法错误 err:%v", t.Header["alg"])
		}
		return rsaPublicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("JWT 验证失败 err:%v", err)
	}
	// 检查token有效性
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("invalid token")
	}
}

func VerifyJWT() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.GetHeader("token")
		if tokenString == "" {
			ctx.JSON(http.StatusUnauthorized, map[string]string{
				"msg": "请先登录",
			})
			ctx.Abort()
			// 阻止后续 handler 但不会阻止当前函数执行
			return
		}
		j := NewJWT()
		claims, err := j.PraseToken(tokenString)
		if err != nil {
			zap.S().Errorf("JWT解析错误 err:%v", err)
			ctx.JSON(http.StatusUnauthorized, pkg.Fail(pkg.UserTokenErrCode))
			ctx.Abort()
			return
		}
		ctx.Set("claims", claims)
		ctx.Next()
	}
}
