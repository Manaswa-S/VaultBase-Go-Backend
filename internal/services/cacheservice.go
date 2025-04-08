package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"main.go/internal/config"
	"main.go/internal/const/errs"
	"main.go/internal/dto"
	sqlc "main.go/internal/sqlc/generate"
)

type CacheSourceURL struct {
	PutCacheURL string
	GetCacheURL string
}

type CacheService struct {
	queries *sqlc.Queries
	httpClient *http.Client

	urls *CacheSourceURL
}

func NewCacheService(queries *sqlc.Queries, client *http.Client, sourceURLs *CacheSourceURL) *CacheService {
	return &CacheService{
		queries: queries,
		httpClient: client,
		urls: sourceURLs,
	}
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *CacheService) validateAPIKey(ctx *gin.Context, apiKey string) (*sqlc.GetUserDataFromAPIKeyRow,*errs.Error) {

	// TODO: need better api key validation like ttl and all

	userData, err := s.queries.GetUserDataFromAPIKey(ctx, apiKey)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.NotFound,
			Message: "API key not found.",
			ToRespondWith: true,
		}
	}

	if !userData.Cache {
		return nil, &errs.Error{
			Type: errs.Unauthorized,
			Message: "API key is found but user is not authorized to use the Cache service.",
			ToRespondWith: true,
		}
	}
	
	return &userData, nil
}

func (s *CacheService) hitSourceURL(ctx *gin.Context, method string, url string, body io.Reader) (*http.Response, *errs.Error) {

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to create a new request for cache : " + err.Error(),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", "vaultbase1234")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to get response from cache source : " + err.Error(),
		}
	}	

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return nil, &errs.Error{
				Type: errs.NotFound,
				Message: "Cache key not found.",
				ToRespondWith: true,
			}
		case http.StatusPreconditionFailed:
			return nil, &errs.Error{
				Type: errs.PreconditionFailed,
				Message: "Cache key already exists.",
				ToRespondWith: true,
			}
		case http.StatusConflict:
			return nil, &errs.Error{
				Type: errs.PreconditionFailed,
				Message: "Cache key already exists.",
				ToRespondWith: true,
			}
		default:
			return nil, &errs.Error{
				Type: errs.Internal,
				Message: "Failed to get response from cache source : Source responded with code other than 200-OK : " + resp.Status,
			}
		}
	}

	return resp, nil
}


// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *CacheService) PutNewCache(ctx *gin.Context, data *dto.SetCacheKeyIncoming, apiKey string) (*errs.Error) {

	userData, errf := s.validateAPIKey(ctx, apiKey)
	if errf != nil {
		return errf
	}

	outGoing := dto.SetCacheKeyOutgoing{
		UID: userData.UserUiid.String(),
		Key: data.CacheKey,
		Value: data.CacheValue,
		TTL: data.CacheTTL,
	}

	outGoingBytes, err := json.Marshal(outGoing)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to marshal outgoing put cache json struct : " + err.Error(),
		}
	}

	url := fmt.Sprintf("%s%s", config.SourceBaseDomain, config.CacheSetURL)

	resp, errf := s.hitSourceURL(ctx, "POST", url, bytes.NewBuffer(outGoingBytes))
	if errf != nil {
		fmt.Println(errf.Message)
		return errf
	}
	defer resp.Body.Close()

	return nil
}	

func (s *CacheService) GetCache(ctx *gin.Context, apiKey string, cacheKey string) (*errs.Error) {

	userData, errf := s.validateAPIKey(ctx, apiKey)
	if errf != nil {
		return errf
	}

	getCacheURL := fmt.Sprintf("%s%s/%s/%s", config.SourceBaseDomain, config.CacheGetURL, userData.UserUiid.String(), cacheKey)

	resp, errf := s.hitSourceURL(ctx, "GET", getCacheURL, nil)
	if errf != nil {
		return errf
	}
	defer resp.Body.Close()

	_, err := io.Copy(ctx.Writer, resp.Body)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to copy source response body : " + err.Error(),
		}
	}

	return nil
}