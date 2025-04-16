package middlewares

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"main.go/internal/const/errs"
)


func ClerkAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// get clerk id from request
		// TODO: check for validity
		
		pkey, exists := os.LookupEnv("PublicSecretKey")
		if !exists {
			ctx.JSON(http.StatusInternalServerError, errs.Error{
				Type: errs.NotFound,
				Message: "Internal failure, resources not found.",
			})
			return 
		}

		clerkID := ctx.GetHeader("ClerkID")
		if clerkID == "" {
			ctx.JSON(http.StatusBadRequest, errs.Error{
				Type: errs.MissingRequiredField,
				Message: "Missing clerk ID in headers (ClerkID).",
				ToRespondWith: true,
			})
			return
		}

		skey := ctx.GetHeader("SecretKey")
		if skey == "" {
			ctx.JSON(http.StatusBadRequest, errs.Error{
				Type: errs.MissingRequiredField,
				Message: "Missing secret key in headers (SecretKey).",
				ToRespondWith: true,
			})
			return
		}

		if skey != pkey {
			ctx.JSON(http.StatusUnauthorized, errs.Error{
				Type: errs.Unauthorized,
				Message: "Unauthorized to access, Failed to match public access key.",
			})
			return
		}

		ctx.Set("clerkID", clerkID)
		ctx.Next()
	}
}