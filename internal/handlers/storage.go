package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"main.go/internal/const/errs"
	"main.go/internal/services"
)

type StorageHandler struct {
	StorageService *services.StorageService
}

func NewStorageHandler(service *services.StorageService) *StorageHandler {
	return &StorageHandler{
		StorageService: service,
	}
}

func (h *StorageHandler) RegisterRoute(storageRoute *gin.RouterGroup) {
	storageRoute.POST("/upload", h.UploadNewFile)
	storageRoute.GET("/download/:filekey", h.DownloadFile)
}


func (h *StorageHandler) UploadNewFile(ctx *gin.Context) {

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

	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "File to upload is invalid or missing.",
			ToRespondWith: true,
		})
		return
	}

	errf := h.StorageService.UploadNewFile(ctx, apiKey, file)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			ctx.Set("error", errf.Message)
			fmt.Println(errf.Message)
			ctx.Status(http.StatusInternalServerError)
		} 
		return
	}
}

func (h *StorageHandler) DownloadFile(ctx *gin.Context) {

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

	fileKey := ctx.Param("filekey")
	if fileKey == "" {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "File key is invalid or missing.",
			ToRespondWith: true,
		})
		return
	}

	errf := h.StorageService.DownloadFile(ctx, apiKey, fileKey)
	if errf != nil {
		fmt.Println(errf.Message)
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			ctx.Set("error", errf.Message)
			ctx.Status(http.StatusInternalServerError)
		} 
		return
	}

}