package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"main.go/internal/const/errs"
	"main.go/internal/dto"
	"main.go/internal/services"
)

type PublicHandler struct {
	PublicService *services.PublicService
}

func NewPublicHandler(service *services.PublicService) *PublicHandler {
	return &PublicHandler{
		PublicService: service,
	}
}

func (h *PublicHandler) RegisterRoute(publicRoute *gin.RouterGroup) {

	publicRoute.POST("/signupdata", h.SignupData)
	// publicRoute.POST("loginpost", h.LoginPost)

	// to create a new project for an existing user
	publicRoute.POST("/newproject", h.NewProject)
	// toggle confirmed services for a project
	publicRoute.POST("/toggleservice", h.ToggleService)
	// delete a project completely
	publicRoute.POST("/deleteproject", h.DeleteService)

}


// extractUserID extracts the clerk ID and other required parameters from the context with explicit type assertion.
// any returned error is directly included in the response as returned
func (h *PublicHandler) extractClerkID(ctx *gin.Context) (string, *errs.Error) {

	clerkID, exists := ctx.Get("clerkID")
	if !exists {
		return "", &errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing user ID in request.",
			ToRespondWith: true,
		}
	}

	clerkIDstr := fmt.Sprintf("%s", clerkID)

	return clerkIDstr, nil 
}


func (h *PublicHandler) SignupData(ctx *gin.Context) {

	signupData := new(dto.SignupData)
	err := ctx.Bind(signupData)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.IncompleteForm,
			Message: "Signup form is incomplete or invalid.",
			ToRespondWith: true,
		})
		return
	}

	errf := h.PublicService.SignupPost(ctx, signupData)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
			ctx.Set("error", errf.Message)
		}
		return
	}


}

func (h *PublicHandler) NewProject(ctx *gin.Context) {

	// 1) get user details
	data := new(dto.NewProject)
	err := ctx.Bind(data)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.IncompleteForm,
			Message: "Incomplete or invalid new service form.",
			ToRespondWith: true,
		})
		return
	}

	// TODO: extract userid

	clerkID, errf := h.extractClerkID(ctx)
	if errf != nil {
		ctx.JSON(http.StatusBadRequest, errf)
		return
	}

	userID, err := h.PublicService.GetUserIDFromClerkID(ctx, clerkID)
	if err != nil {
		return
	}

	// 2) delegate to service
	keyResp, errf := h.PublicService.NewProject(ctx, userID, data)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
		}
		return
	}

	// 3) respond appropriately
	ctx.JSON(http.StatusCreated, keyResp)
}

func (h *PublicHandler) ToggleService(ctx *gin.Context) {
	
	// 1) get user details
	data := new(dto.NewProject)
	err := ctx.Bind(data)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.IncompleteForm,
			Message: "Incomplete or invalid new service form.",
			ToRespondWith: true,
		})
		return
	}

	// TODO: extract userid

	clerkID, errf := h.extractClerkID(ctx)
	if errf != nil {
		ctx.JSON(http.StatusBadRequest, errf)
		return
	}

	userID, err := h.PublicService.GetUserIDFromClerkID(ctx, clerkID)
	if err != nil {
		return
	}

	// 2) delegate to service
	keyResp, errf := h.PublicService.ToggleService(ctx, userID, data)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
		}
		return
	}

	// 3) respond appropriately
	ctx.JSON(http.StatusCreated, keyResp)
}

func (h *PublicHandler) DeleteService(ctx *gin.Context) {

	// 1) get user details
	data := new(dto.NewProject)
	err := ctx.Bind(data)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.IncompleteForm,
			Message: "Incomplete or invalid new service form.",
			ToRespondWith: true,
		})
		return
	}

	// TODO: extract userid

	clerkID, errf := h.extractClerkID(ctx)
	if errf != nil {
		ctx.JSON(http.StatusBadRequest, errf)
		return
	}

	userID, err := h.PublicService.GetUserIDFromClerkID(ctx, clerkID)
	if err != nil {
		return
	}

	// 2) delegate to service
	errf = h.PublicService.DeleteService(ctx, userID, data)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
		}
		return
	}

	// 3) respond appropriately
	ctx.JSON(http.StatusOK, gin.H{
		"Status": "Service deleted successfully",
	})
}