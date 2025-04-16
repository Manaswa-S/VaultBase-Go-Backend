package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"main.go/internal/const/errs"
	"main.go/internal/dto"
	"main.go/internal/services"
)

type CacheHandler struct {
	CacheService *services.CacheService
}

func NewCacheHandler(service *services.CacheService) *CacheHandler {
	return &CacheHandler{
		CacheService: service,
	}
}

func (h *CacheHandler) RegisterRoute(cacheRoute *gin.RouterGroup) {

	// TODO: add a route to create a cache db first, a single user can have multiple dbs'

	cacheRoute.POST("/put", h.PutNewCache)
	cacheRoute.GET("/get/:cacheKey", h.GetCache)
}

func (h *CacheHandler) PutNewCache(ctx *gin.Context) {

	data := new(dto.SetCacheKeyIncoming)
	err := ctx.Bind(data)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Invalid new cache post form, missing or invalid fields.",
			ToRespondWith: true,
		})
		return
	}

	// get api key
	apiKey := ctx.GetHeader("API-Key")
	if apiKey == "" {
		ctx.JSON(http.StatusUnauthorized, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing API key in request headers.",
			ToRespondWith: true,
		})
		return
	}

	errf := h.CacheService.PutNewCache(ctx, data, apiKey)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
			ctx.Set("error", errf.Message)
		}
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"status": "Cache key-value has been created or updated.",
	})
}

func (h *CacheHandler) GetCache(ctx *gin.Context) {

	cacheKey := ctx.Param("cacheKey")
	if cacheKey == "" {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing cache key in param (cacheKey).",
			ToRespondWith: true,
		})
		return
	}

	// get api key
	apiKey := ctx.GetHeader("API-Key")
	if apiKey == "" {
		ctx.JSON(http.StatusUnauthorized, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing API key in request headers.",
			ToRespondWith: true,
		})
		return
	}	

	errf := h.CacheService.GetCache(ctx, apiKey, cacheKey)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
			ctx.Set("error", errf.Message)
		}
		return
	}

	ctx.Status(http.StatusOK)
}