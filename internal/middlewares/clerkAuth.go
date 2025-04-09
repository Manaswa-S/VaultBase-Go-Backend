package middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
)


func ClerkAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// get clerk id from request
		// check for validity

		clerkID := ctx.GetHeader("clerkID")
		if clerkID == "" {
			fmt.Println("empty clerk id header") 
		}

		skey := ctx.GetHeader("secret_key")
		if skey == "" {
			fmt.Println("empty clerk id header") 
		}
		fmt.Println(skey)



		ctx.Set("clerkID", clerkID)

		ctx.Next()
	}
}