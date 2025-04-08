package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"main.go/internal/config"
	"main.go/internal/const/errs"
	sqlc "main.go/internal/sqlc/generate"
)

type StorageSourceURL struct {
	UploadURL string
	DownloadURL string
}

type StorageService struct {
	queries *sqlc.Queries
	httpClient *http.Client

	urls *StorageSourceURL
}

func NewStorageService(queries *sqlc.Queries, client *http.Client, sourceURLs *StorageSourceURL) *StorageService {
	return &StorageService{
		queries: queries,
		httpClient: client,
		urls: sourceURLs,
	}
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>


func (s *StorageService) validateAPIKey(ctx *gin.Context, apiKey string) (*sqlc.GetUserDataFromAPIKeyRow,*errs.Error) {

	// TODO: need better api key validation like ttl and all

	userData, err := s.queries.GetUserDataFromAPIKey(ctx, apiKey)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.NotFound,
			Message: "API key not found.",
			ToRespondWith: true,
		}
	}

	if !userData.Storage {
		return nil, &errs.Error{
			Type: errs.Unauthorized,
			Message: "API key is found but user is not authorized to use the Storage service.",
			ToRespondWith: true,
		}
	}
	
	return &userData, nil
}

func (s *StorageService) hitSourceURL(ctx *gin.Context, method string, url string, body *bytes.Buffer, reqHeader string) (*http.Response, *errs.Error) {

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to create a new request for storage : " + err.Error(),
		}
	}

	req.Header.Set("Content-Type", reqHeader)
	req.Header.Set("authorization", "vaultbase1234")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to get response from storage source : " + err.Error(),
		}
	}

	if resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600 {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to get response from storage source : Source responded with code other than 201-OK : " + resp.Status,
		}
	}

	return resp, nil
}


func (s *StorageService) hitSourceURL2(ctx *gin.Context, method string, url string, body io.Reader) (*http.Response, *errs.Error) {

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to create a new request for storage : " + err.Error(),
		}
	}

	req.Header.Set("authorization", "vaultbase1234")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to get response from storage source : " + err.Error(),
		}
	}

	if resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600 {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to get response from storage source : Source responded with code other than 201-OK : " + resp.Status,
		}
	}

	return resp, nil
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>


func (s *StorageService) UploadNewFile(ctx *gin.Context, apiKey string, file *multipart.FileHeader) *errs.Error {

	userData, errf := s.validateAPIKey(ctx, apiKey)
	if errf != nil {
		return errf
	}	

	sizeLim := config.StorageUploadFileSizeLimit
	if file.Size > sizeLim {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: fmt.Sprintf("File size exceeds upload limit. Current upload limit: %d bytes.", sizeLim),
			ToRespondWith: true,
		}
	}

	srcFile, err := file.Open()
	if err != nil {
		return &errs.Error{
            Type: errs.Internal,
            Message: "Failed to open the uploaded file : " + err.Error(),
			ToRespondWith: true,
        }
    }
	defer srcFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err = writer.WriteField("uid", userData.UserUiid.String())
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to write the 'uid' field : " + err.Error(),
		}
	}
	
	part, err := writer.CreateFormFile("file", file.Filename)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to create the form file : " + err.Error(),
		}
	}

	srcFile.Seek(0, io.SeekStart) // Reset file position before copying
	_, err = io.Copy(part, srcFile)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to copy source file to form : " + err.Error(),
		}
	}

	err = writer.Close()
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to close writer.",
		}
	}

	url := fmt.Sprintf("%s%s", config.SourceBaseDomain, config.StorageUploadURL)

	resp, errf := s.hitSourceURL(ctx, "POST", url, body, writer.FormDataContentType())
	if errf != nil {
		errf.Message = "Failed to upload file : " + errf.Message
		return errf
	}
	defer resp.Body.Close()

	ctx.Status(resp.StatusCode)

	for key, values := range resp.Header {
		for _, value := range values {
			ctx.Writer.Header().Add(key, value)
		}
	}

	_, err = io.Copy(ctx.Writer, resp.Body)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to copy response into ctx writer : " + err.Error(),
		}
	}
		

	return nil
}





// {
//     "message": "File uploaded successfully",
//     "key": "d6f67dcb631d",
//     "fileUrl": "http://service-api-q77p.onrender.com/uploads/f00cd208-fed4-4573-ae4d-b48dbf9e956b/1743828885661-movie_list.txt"
// }


func (s *StorageService) DownloadFile(ctx *gin.Context, apiKey string, fileKey string) *errs.Error {

	userData, errf := s.validateAPIKey(ctx, apiKey)
	if errf != nil {
		return errf
	}

	fmt.Println(fileKey)

	// outgoing := dto.DownloadFileOutgoing{
	// 	UUID: userData.UserUiid.String(),
	// 	Key: fileKey,
	// }

	// outgoingBytes, err := json.Marshal(outgoing)
	// if err != nil {
	// 	return &errs.Error{
	// 		Type: errs.Internal,
	// 		Message: "Failed to marshal outgoing download struct.",
	// 	}
	// }

	url := fmt.Sprintf("%s%s/?uid=%s&key=%s", config.SourceBaseDomain, config.StorageDownloadURL, userData.UserUiid.String(), fileKey)
	resp, errf := s.hitSourceURL2(ctx, "GET", url, nil)
	if errf != nil {
		return errf
	}
	defer resp.Body.Close()

	ctx.Status(resp.StatusCode)

	for key, values := range resp.Header {
		for _, value := range values {

			if strings.ToLower(key) == "content-length" || strings.ToLower(key) == "transfer-encoding" {
				continue
			}
			ctx.Writer.Header().Add(key, value)
		}
	}

	_, err := io.Copy(ctx.Writer, resp.Body)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to stream file content to client : " + err.Error(),
		}
	}

	return nil
}
