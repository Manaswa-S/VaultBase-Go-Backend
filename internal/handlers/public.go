package handlers

import (
	"fmt"
	"net/http"
	"strconv"

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

	publicRoute.POST("/newuser", h.NewUser)

	// get all user dashboard data
	// publicRoute.GET("/userdata", h.GetUserData)

	// get all projects for a user
	publicRoute.GET("/allprojects/:clerkID", h.AllProjects)
	// to create a new project for an existing user
	publicRoute.POST("/newproject", h.NewProject)
	// toggle confirmed services for a project
	publicRoute.POST("/toggleservice", h.ToggleService)
	// delete a project completely
	publicRoute.POST("/deleteproject", h.DeleteService)


	publicRoute.GET("/analytics/storage/:servicename/:scope/:interval", h.StorageAnalytics)
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


func (h *PublicHandler) NewUser(ctx *gin.Context) {

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

	exists, errf := h.PublicService.NewUser(ctx, signupData)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
			ctx.Set("error", errf.Message)
		}
		return
	}
	
	if exists {
		ctx.Status(http.StatusAlreadyReported)
		return
	}

	ctx.Status(http.StatusCreated)
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

	fmt.Println(userID)

	// 2) delegate to service
	newproj, errf := h.PublicService.NewProject(ctx, userID, data)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
		}
		return
	}

	// 3) respond appropriately
	ctx.JSON(http.StatusCreated, gin.H{
		"newproj": newproj,
	})
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
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing headers (clerkID or secret_key).",
			ToRespondWith: true,
		})
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

func (h *PublicHandler) AllProjects(ctx *gin.Context) {

	clerkID, errf := h.extractClerkID(ctx)
	if errf != nil {
		return
	}

	fmt.Println(clerkID)

	userID, err := h.PublicService.GetUserIDFromClerkID(ctx, clerkID)
	if err != nil {
		return
	}

	fmt.Println(userID)
	
	allProjs, errf := h.PublicService.AllProjects(ctx, userID)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
		}
		return
	}

	fmt.Println(allProjs[0])

	// 3) respond appropriately
	ctx.JSON(http.StatusOK, gin.H{
		"allprojs": allProjs,
	})

}



func (h *PublicHandler) StorageAnalytics(ctx *gin.Context) {

	serviceName := ctx.Param("servicename")
	scope := ctx.Param("scope")
	interval := ctx.Param("interval")
	if serviceName == "" || scope == "" || interval == "" {
		ctx.JSON(http.StatusBadRequest, errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing query params 'servicename' or 'scope' or 'interval'.",
			ToRespondWith: true,
		})
		return
	}

	clerkID, errf := h.extractClerkID(ctx)
	if errf != nil {
		return
	}

	userID, err := h.PublicService.GetUserIDFromClerkID(ctx, clerkID)
	if err != nil {
		return
	}

	scopeInt, err := strconv.ParseInt(scope, 10, 64)
	if err != nil {
		return
	}

	intervalInt, err := strconv.ParseInt(interval, 10, 64)
	if err != nil {
		return
	}

	resp, errf := h.PublicService.StorageData(ctx, userID, serviceName, scopeInt, intervalInt)
	if errf != nil {
		if errf.ToRespondWith {
			ctx.JSON(http.StatusBadRequest, errf)
		} else {
			fmt.Println(errf.Message)
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"storage": resp,
	})

}
