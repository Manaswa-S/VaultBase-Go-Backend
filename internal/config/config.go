package config


const (
	JWTAccessExpiration = 3600 // seconds //  
	JWTRefreshExpiration = 604800 // seconds // 7 days // 604800 seconds
)

const (
	StorageUploadFileSizeLimit int64 = 75000000 // bytes
)

const (
	DefaultAPIKeyTTL int64 = 604800 // seconds // 7 days // 604800 seconds
)


const (
	// SourceBaseDomain = "https://gcc3fbf0-3000.inc1.devtunnels.ms"
	// SourceBaseDomain = "https://service-api-q77p.onrender.com"

	SourceBaseDomain = "https://frank-warm-prawn.ngrok-free.app"

	CacheSetURL = "/api/caching/set"
	CacheGetURL = "/api/caching/get"


	StorageUploadURL = "/api/storage/upload-file"
	StorageDownloadURL = "/api/storage/get-file"

)