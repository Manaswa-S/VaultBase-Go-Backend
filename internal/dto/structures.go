package dto

import (
	"mime/multipart"
)

type Token struct {
	Issuer string
	Subject string
	ExpiresAt int64
	IssuedAt int64
	Role int64
	ID int64
	Email string	

	Version string
}

type JWTTokens struct {
	JWTAccess string
	JWTRefresh string
}

type SignupData struct {
	Email string
	Password string
}

type LoginData struct {
	Email string
	Password string
}

type LoginResponse struct {
	Tokens *JWTTokens
	Role int64
}




type NewProject struct {
	Name string
	// TODO: add other things too
	Cache bool
	Storage bool
}

type NewProjectResp struct {
	ServiceUUID string
	KeyInfo *APIKeyResponse
}

type APIKeyResponse struct {
	ID string
	Key string
	CreatedAt int64
	ExpiresAt int64
	Cache bool
	Storage bool
}	





// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
// CORE SERVICE STRUCTS

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
// CACHE SERVICE

// new cache put incoming to proxy
type SetCacheKeyIncoming struct {
	APIKey string // personal API key

	CacheKey string // as a string without spaces, see documentation for more details
	CacheValue string // currently string, can later be upgraded to any
	CacheTTL int64 // as interger in milliseconds

	UpdateIfExists bool // whether to update if the key already exists
}
type SetCacheKeyOutgoing struct {
	UID string `json:"uid"`
	Key string `json:"key"`
	Value string `json:"value"`
	TTL int64 `json:"ttl"`

}

// get cache incoming to proxy
// currently not in use
type GetCacheKeyIncoming struct {
	APIKey string // personal API Key
 
	CacheKey string // the cache key	
}



// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
// STORAGE SERVICE

type UploadNewFileIncoming struct {
	
}

type UploadNewFileOutgoing struct {
	File *multipart.FileHeader `json:"file"`
	UID string	`json:"uid"`
}

type DownloadFileOutgoing struct {
	UUID string `json:"uid"`
	Key string `json:"key"`
}