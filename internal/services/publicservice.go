package services

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"main.go/internal/config"
	"main.go/internal/const/errs"
	"main.go/internal/dto"
	sqlc "main.go/internal/sqlc/generate"
	apikeys "main.go/internal/utils/apikeys"
)

type PublicService struct {
	queries *sqlc.Queries
	DB *pgxpool.Pool
	ClerkUserClient *user.Client

}

func NewPublicService(queries *sqlc.Queries, db *pgxpool.Pool, clerkUserClient *user.Client) *PublicService {
	return &PublicService{
		queries: queries,
		DB: db,
		ClerkUserClient: clerkUserClient,
	}
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>


func (s *PublicService) GetUserIDFromClerkID(ctx *gin.Context, clerkId string) (int64, error) {

	userID, err := s.queries.GetUserIDFromClerkID(ctx, clerkId)
	if err != nil {
		// TODO: add other checks for pgerr too.
		return 0, err
	}

	return userID, nil
}

func (s *PublicService) userIsServiceOwner(ctx *gin.Context, userID int64, servicename string) (*sqlc.GetServiceDataRow, *errs.Error) {
	
	serviceData, err := s.queries.GetServiceData(ctx, servicename)
	if err != nil {
		// TODO: process pgerr
		return nil, &errs.Error{}
	}	

	// 2) check if user is owner of service
	if serviceData.UserID != userID {
		return nil, &errs.Error{}	
	}
	return &serviceData, nil
}

func (s *PublicService) parseScopeInterval(scopeStr, intervalStr string) (int64, int64, *errs.Error) {

	scope, err := strconv.ParseInt(scopeStr, 10, 64)
	if err != nil {
		return 0, 0, &errs.Error{
			Type: errs.InvalidFormat,
			Message: "Failed to parse given scope to int64.",
			ToRespondWith: true,
		}
	}

	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		return 0, 0, &errs.Error{
			Type: errs.InvalidFormat,
			Message: "Failed to parse given interval to int64.",
			ToRespondWith: true,
		}
	}

	return scope, interval, nil
}


// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>


func (s *PublicService) NewUser(ctx *gin.Context, signupData *dto.SignupData) (bool, *errs.Error) {

	cnt, err := s.queries.CheckUserExistence(ctx, signupData.ClerkID)
	if err != nil {
		return false, nil // err
	}

	if cnt > 0 {
		return false, nil
	}

	err = s.queries.SignupUser(ctx, sqlc.SignupUserParams{
		Email: signupData.Email,
		Role: 1,
		ClerkID: signupData.ClerkID,
	})
	if err != nil {
		var pgerr *pgconn.PgError
		if (errors.As(err, &pgerr)) {
			if (pgerr.Code == errs.UniqueViolation) {
				return false, &errs.Error{
					Type: errs.UniqueViolation,
					Message: "User with Email-Id already exists. Try with different Id.",
					ToRespondWith: true,
				}		
			}
		}
		return false, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to insert new user : " + err.Error(),
		}
	}
	fmt.Println("error")

	return false, nil
}

// NewService registers a new service instance for a valid user, 
// generates an associated API key with selected features (cache, storage),
// and returns the service metadata along with the generated key.
// Ensures uniqueness of service names per user and maintains data consistency using a transaction.
func (s *PublicService) NewProject(ctx *gin.Context, userID int64, data *dto.NewProject) (*dto.NewProjectResp, *errs.Error) {

	// 0) validate user data
	if len(data.Name) < 8 {
		return nil, &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "The service name should atleast be 8 characters long.",
			ToRespondWith: true,
		}
	}

	// 1) check if user actually exists and if is confirmed and valid
	userData, err := s.queries.GetUserData(ctx, userID)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == errs.NoRowsMatch {
				return nil, &errs.Error{
					Type: errs.NotFound,
					Message: "No such user found. Please register first.",
					ToRespondWith: true,
				}
			}
		}
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to retrieve user info : " + err.Error(),
		}
	}

	if userData.Deleted {
		return nil, &errs.Error{
			Type: errs.NotFound,
			Message: "No such user found. Try again!",
			ToRespondWith: true,
		}
	}

	// 2)	check for service name uniqueness

	sameCount, err := s.queries.GetServiceCountForUserID(ctx, sqlc.GetServiceCountForUserIDParams{
		UserID: userID,
		Name: data.Name,
	})
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to get service count for userid : " + err.Error(),
		}
	}
	if sameCount > 0 {
		return nil, &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "A user cannot have 2 services with the same name.",
			ToRespondWith: true,
		}
	}

	// 3.0)	acquire a transaction to avoid orphaned api keys

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to acquire a transaction : " + err.Error(),
		}
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			fmt.Println(err)
		}
	}()
	txQueries := s.queries.WithTx(tx)

	// 3) generate api key and insert into table

	apiCreds, err := apikeys.CreateWithOutSeed()
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to generate api key : " + err.Error(),
		}
	}
	expiresAt := time.Now().Unix() + config.DefaultAPIKeyTTL

	keyData, err := txQueries.InsertKey(ctx, sqlc.InsertKeyParams{
		Key: apiCreds.Key,
		Cache: data.Cache,
		Storage: data.Storage,
		ExpiresAt: expiresAt,
		ID: apiCreds.ID,
	})
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to insert a new api key : " + err.Error(),
		}
	}

	// 4) create a new appropriate service, select confirms 
	serviceData, err := txQueries.InsertNewService(ctx, sqlc.InsertNewServiceParams{
		UserID: userID,
		KeyID: keyData.KeyID,
		Name: data.Name,
	})
	if err != nil {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to insert new service : " + err.Error(),
		}
	}

	err = tx.Commit(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return nil, &errs.Error{
			Type: errs.Internal,
			Message: "Failed to commit new service db transaction : " + err.Error(),
		}
	}

	// _, _ = apikeys.CreateWithGiven(serviceData.String())

	// 5) respond with api key and confirmation and some other details
	return &dto.NewProjectResp{
		ServiceUUID: serviceData.String(),
		KeyInfo: &dto.APIKeyResponse{
			ID: apiCreds.ID,
			Key: apiCreds.Key,
			CreatedAt: keyData.CreatedAt.Time.Unix(),
			ExpiresAt: expiresAt,
			Cache: data.Cache,
			Storage: data.Storage,
		},
	}, nil
}


// EnableService changes the confirmation services allowed on a certain key/project.
func (s *PublicService) ToggleService(ctx *gin.Context, userID int64, data *dto.NewProject) (*dto.APIKeyResponse, *errs.Error) {

	// 1) check if service exists
	serviceData, errf := s.userIsServiceOwner(ctx, userID, data.Name)
	if errf != nil {
		return nil, errf
	}

	// 3) update requested attribute 
	updatedKeyData, err := s.queries.UpdateKeyServicesConfirmation(ctx, sqlc.UpdateKeyServicesConfirmationParams{
		KeyID: serviceData.KeyID,
		Cache: data.Cache,
		Storage: data.Storage,
	})
	if err != nil {
		return nil, nil
	}

	return &dto.APIKeyResponse{
		Cache: updatedKeyData.Cache,
		Storage: updatedKeyData.Storage,
	}, nil
}

// DeleteService deletes a service and also deletes the key associated with it.
func (s *PublicService) DeleteService(ctx *gin.Context, userID int64, data *dto.NewProject) (*errs.Error) {

	// 1) check if service exists
	serviceData, err := s.queries.GetServiceData(ctx, data.Name)
	if err != nil {
		// TODO: process pgerr
		return nil
	}

	// 2) check if user is owner of service
	if serviceData.UserID != userID {
		return nil
	}

	// 3) Delete the service

	err = s.queries.DeleteKey(ctx, serviceData.KeyID)
	if err != nil {
		return nil
	}

	// this isnt really neccessary as the key deletion cascades to service deletion too
	err = s.queries.DeleteService(ctx, serviceData.Sid)
	if err != nil {
		return nil
	}

	return nil
}

func (s *PublicService) AllProjects(ctx *gin.Context, userID int64) ([]*dto.NewProjectResp, *errs.Error) {

	projsData, err := s.queries.GetAllProjects(ctx, userID)
	if err != nil {
		return nil, nil
	}

	resp := make([]*dto.NewProjectResp, 0)

	for _, proj := range projsData {
		resp = append(resp, &dto.NewProjectResp{
			ServiceUUID: proj.ServiceUuid.String(),
			ServiceName: proj.Name,
			ServiceCreatedAt: proj.CreatedAt.Time.Unix(),

			KeyInfo: &dto.APIKeyResponse{
				ID: proj.ID.String,
				Key: proj.Key.String,
				CreatedAt: proj.CreatedAt.Time.Unix(),
				ExpiresAt: proj.ExpiresAt.Int64,
				Cache: proj.Cache.Bool,
				Storage: proj.Storage.Bool,
			},
		})
	}

	return resp, nil
}



const (
	DefaultStorageAnalyticsTimeScope int64 = 21600 // in seconds // 7 days
	DefaultStorageAnalyticsInterval int64 = 3600 // in seconds // 24 hours		
)
const (
	DefaultCacheAnalyticsTimeScope int64 = 21600 // in seconds // 7 days
	DefaultCacheAnalyticsInterval int64 = 3600 // in seconds // 24 hours		
)


func (s *PublicService) StorageData(ctx *gin.Context, userID int64, stream string, servicename string, scopeStr, intervalStr string) (*dto.StorageData, *errs.Error) {

	scope, interval, errf := s.parseScopeInterval(scopeStr, intervalStr)
	if errf != nil {
		return nil, errf
	}

	serviceData, errf := s.userIsServiceOwner(ctx, userID, servicename)
	if errf != nil {
		return nil, errf
	}

	var data []sqlc.GetAllStorageDataRow
	var err error

	// TODO: add direct range on sql using scope, WHERE AFTER scope

	switch stream {
	case "download":
		data, err = s.queries.GetAllStorageData(ctx, sqlc.GetAllStorageDataParams{
			ServiceID: serviceData.Sid,
			Upload: false,
			Download: true,
		})
	case "upload":
		data, err = s.queries.GetAllStorageData(ctx, sqlc.GetAllStorageDataParams{
			ServiceID: serviceData.Sid,
			Upload: true,
			Download: false,
		})
	case "all":
		data, err = s.queries.GetAllStorageData(ctx, sqlc.GetAllStorageDataParams{
			ServiceID: serviceData.Sid,
			Upload: true,
			Download: true,
		})
	default:
		return nil, &errs.Error{
			Type: errs.NotFound,
			Message: "Invalid storage stream choice.",
			ToRespondWith: true,
		}
	}
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code != errs.NoRowsMatch {
				return nil, &errs.Error{
					Type: errs.Internal,
					Message: "Failed to get storage analytics data : " + err.Error(),
				}	
			}
		}
	}

	if scope <= 0 || interval <= 0 {
		scope = DefaultStorageAnalyticsTimeScope
		interval = DefaultStorageAnalyticsInterval
	}
	
	if scope < interval {
		return nil, &errs.Error{
			Type: errs.InvalidFormat,
			Message: "The scope cannot be lesser than the interval.",
			ToRespondWith: true,
		}
	}

	xparts := (scope / interval) + 1
	yFreq := make([]int64, xparts)	
	xTime := make([]int64, xparts)
	xTimeStr := make([]string, xparts)
	XTimeTime := make([]time.Time, xparts)

	usrTimeZone := ctx.GetHeader("X-Timezone")
	if usrTimeZone == "" {
		return nil, &errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing user timezone header, 'X-Timezone'.",
			ToRespondWith: true,
		}
	}

	usrLocation, err := time.LoadLocation(usrTimeZone)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.InvalidState,
			Message: "Invalid user timezone in headers. Check IANA Time Zone database for valid choices.",
			ToRespondWith: true,
		}
	}
	usrLocalTime := time.Now().In(usrLocation)

	for i := range xparts {
		secondsInPast := ((xparts - i - 1) * interval)
		xTime[i] = secondsInPast
		t := usrLocalTime.Add(-time.Duration(secondsInPast) * time.Second)
		xTimeStr[i] = t.Format("2006-01-02 15:04:05 MST")
		XTimeTime[i] = t
	}

	for _, d := range data {
		tSince := usrLocalTime.Unix() - d.CreatedAt.Time.Unix()
		if tSince <= (scope) && tSince >= 0 {
			index := ((xparts - 1) - (tSince / interval))
			if index >= 0 && index < xparts {
				yFreq[index]++
			}
		}
	}

	return &dto.StorageData{
		Scope: scope,
		Interval: interval,
		XParts: xparts,
		XTime: xTime,
		YFreq: yFreq,
		XTimeStr: xTimeStr,
		XTimeTime: XTimeTime,
	}, nil
}

func (s *PublicService) CacheData(ctx *gin.Context, userID int64, stream string, servicename string, scopeStr, intervalStr string) (*dto.StorageData, *errs.Error) {

	scope, interval, errf := s.parseScopeInterval(scopeStr, intervalStr)
	if errf != nil {
		return nil, errf 
	}

	serviceData, errf := s.userIsServiceOwner(ctx, userID, servicename)
	if errf != nil {
		return nil, errf
	}

	var data []sqlc.GetAllCacheDataRow
	var err error

	// TODO: add direct range on sql using scope, WHERE AFTER scope

	switch stream {
	case "get":
		data, err = s.queries.GetAllCacheData(ctx, sqlc.GetAllCacheDataParams{
			ServiceID: serviceData.Sid,
			Get: true,
			Put: false,
		})
	case "upload":
		data, err = s.queries.GetAllCacheData(ctx, sqlc.GetAllCacheDataParams{
			ServiceID: serviceData.Sid,
			Get: false,
			Put: true,
		})
	case "all":
		data, err = s.queries.GetAllCacheData(ctx, sqlc.GetAllCacheDataParams{
			ServiceID: serviceData.Sid,
			Get: true,
			Put: true,
		})
	default:
		return nil, &errs.Error{
			Type: errs.NotFound,
			Message: "Invalid cache stream choice.",
			ToRespondWith: true,
		}
	}
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code != errs.NoRowsMatch {
				return nil, &errs.Error{
					Type: errs.Internal,
					Message: "Failed to get cache analytics data : " + err.Error(),
				}	
			}
		}
	}

	if scope <= 0 || interval <= 0 {
		scope = DefaultCacheAnalyticsTimeScope
		interval = DefaultCacheAnalyticsInterval
	}

	if scope < interval {
		return nil, &errs.Error{
			Type: errs.InvalidFormat,
			Message: "The scope cannot be lesser than the interval.",
			ToRespondWith: true,
		}
	}

	xparts := (scope / interval) + 1
	yFreq := make([]int64, xparts)	
	xTime := make([]int64, xparts)
	xTimeStr := make([]string, xparts)
	XTimeTime := make([]time.Time, xparts)

	usrTimeZone := ctx.GetHeader("X-Timezone")
	if usrTimeZone == "" {
		return nil, &errs.Error{
			Type: errs.MissingRequiredField,
			Message: "Missing user timezone header, 'X-Timezone'.",
			ToRespondWith: true,
		}
	}

	usrLocation, err := time.LoadLocation(usrTimeZone)
	if err != nil {
		return nil, &errs.Error{
			Type: errs.InvalidState,
			Message: "Invalid user timezone in headers. Check IANA Time Zone database for valid choices.",
			ToRespondWith: true,
		}
	}
	usrLocalTime := time.Now().In(usrLocation)

	for i := range xparts {
		secondsInPast := ((xparts - i - 1) * interval)
		xTime[i] = secondsInPast
		t := usrLocalTime.Add(-time.Duration(secondsInPast) * time.Second)
		xTimeStr[i] = t.Format("2006-01-02 15:04:05 MST")
		XTimeTime[i] = t
	}

	for _, d := range data {
		tSince := usrLocalTime.Unix() - d.CreatedAt.Time.Unix()
		if tSince <= (scope) && tSince >= 0 {
			index := ((xparts - 1) - (tSince / interval))
			if index >= 0 && index < xparts {
				yFreq[index]++
			}
		}
	}

	return &dto.StorageData{
		Scope: scope,
		Interval: interval,
		XParts: xparts,
		XTime: xTime,
		YFreq: yFreq,
		XTimeStr: xTimeStr,
		XTimeTime: XTimeTime,
	}, nil
}
