package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"main.go/cmd"
	"main.go/internal/handlers"
	"main.go/internal/middlewares"
	"main.go/internal/services"
	apikeys "main.go/internal/utils/apiKeys"
)

 func main() {

	apikeys.CreateWithOutSeed()

	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Error loading environment variables: %v", err)
		return
	}

	err = cmd.InitDB()
	if err != nil {
        fmt.Printf("Error initializing DB: %v", err)
        return
    }

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "clerkID", "secret_key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// router.MaxMultipartMemory = 50 << 20 
	routes(router)

	err = router.Run(os.Getenv("PORT"))
	if err != nil {
		fmt.Printf("Error running router : %v", err)
		return
	}
 }

 func routes(router *gin.Engine) error {


	// TODO: give proper names for groups
	womid := router.Group("")
	womid.Use()
	wmid := router.Group("")
	// wmid.Use(middlewares.Authenticate(), middlewares.Authorize())

	
	queries := cmd.Queries
	db := cmd.PostgresPool
	// TODO: separate and strengthen this later on
	httpClient := &http.Client{}

	clerkClient := NewClerkClient()
	if clerkClient == nil {
		return fmt.Errorf("failed to get clerk client")
	}

	publicService := services.NewPublicService(queries, db, clerkClient)
	publicHandler := handlers.NewPublicHandler(publicService)
	publicGroup := womid.Group("/public")
	publicGroup.Use(middlewares.ClerkAuth())
	publicHandler.RegisterRoute(publicGroup)


	cacheService := services.NewCacheService(queries, httpClient, &services.CacheSourceURL{
		PutCacheURL: "/api/caching/set",
		GetCacheURL: "/api/caching/get",
	})
	cacheHandler := handlers.NewCacheHandler(cacheService)
	cacheGroup := wmid.Group("/cache")
	cacheHandler.RegisterRoute(cacheGroup)

	storageService := services.NewStorageService(queries, httpClient, &services.StorageSourceURL{
		UploadURL: "/api/storage/upload-file",
		DownloadURL: "/api/storage/get-file",
	})
	storageHandler := handlers.NewStorageHandler(storageService)
	storageGroup := wmid.Group("/storage")
	storageHandler.RegisterRoute(storageGroup)



	return nil
 }


 func NewClerkClient() *user.Client {
	clerkKey, exists := os.LookupEnv("ClerkSecretKey")
	if !exists {
		return nil
	}

	clerkConf := clerk.ClientConfig{}
	clerkConf.Key = &clerkKey

	return user.NewClient(&clerkConf)

 }