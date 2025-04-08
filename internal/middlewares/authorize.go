package middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func Authorize() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		fmt.Println("Authorizing...")



		
		ctx.Next()
	}
}