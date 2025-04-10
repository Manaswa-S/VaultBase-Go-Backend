package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"main.go/internal/config"
	"main.go/internal/const/errs"
	"main.go/internal/dto"
	sqlc "main.go/internal/sqlc/generate"
	apikeys "main.go/internal/utils/apiKeys"
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

type CreateClerk struct {
	Email string
	Username string
	Password string
}

type CreatedClerk struct {
	ID string
	Created_At int64
}

// clerk ops

func (s *PublicService) ClerkCreate(ctx *gin.Context, createData *CreateClerk) (*CreatedClerk, error) {

	userData, err := s.ClerkUserClient.Create(ctx, &user.CreateParams{
		EmailAddresses: &[]string{createData.Email},
		Username: &createData.Username,
		Password: &createData.Password,
	})
	if err != nil {
		return nil, err
	}	

	return &CreatedClerk{
		ID: userData.ID,
		Created_At: userData.CreatedAt,
	}, nil
}


func (s *PublicService) GetUserIDFromClerkID(ctx *gin.Context, clerkId string) (int64, error) {

	userID, err := s.queries.GetUserIDFromClerkID(ctx, clerkId)
	if err != nil {
		// TODO: add other checks for pgerr too.
		return 0, err
	}

	return userID, nil
}


func (s *PublicService) ClerkVerify(ctx *gin.Context, clerkId string) (string, error) {

	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("clerk : failed to get session claims from contest")
	}

	userData, err := s.ClerkUserClient.Get(ctx, claims.Subject)
	if err != nil {
		return "", err
	}

	return userData.ID, nil
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

	if !userData.Confirmed {
		return nil, &errs.Error{
			Type: errs.IncompleteAction,
			Message: "Please confirm your email to proceed.",
			ToRespondWith: true,
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

	fmt.Println(data.Name)
	fmt.Println(data.Cache)
	fmt.Println(data.Storage)


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



func (s *PublicService) StorageData(ctx *gin.Context, userID int64, servicename string, scope, interval int64) (*dto.StorageData, *errs.Error) {

	serviceData, errf := s.userIsServiceOwner(ctx, userID, servicename)
	if errf != nil {
		return nil, errf
	}

	data, err := s.queries.GetAllStorageData(ctx, serviceData.Sid)
	if err != nil {
		return nil, &errs.Error{}
	}

	if scope <= 0 || interval <= 0 {
		scope = DefaultStorageAnalyticsTimeScope
		interval = DefaultStorageAnalyticsInterval
	}

	xparts := (scope / interval) + 1
	yFreq := make([]int64, xparts)	
	xTime := make([]int64, xparts)
	xTimeStr := make([]string, xparts)
	XTimeTime := make([]time.Time, xparts)

	for i := range xparts {
		secondsInPast := ((xparts - i - 1) * DefaultStorageAnalyticsInterval)
		xTime[i] = secondsInPast
		xTimeStr[i] = time.Unix(time.Now().Unix() - secondsInPast, 0).Format("2006-01-02 15:04:05 MST")
		XTimeTime[i] = time.Unix(time.Now().Unix() - secondsInPast, 0)
	}

	for _, d := range data {
		tSince := time.Now().Unix() - d.CreatedAt.Time.Unix()

		if tSince <= (scope) {
			index := ((xparts - 1) - (tSince / DefaultStorageAnalyticsInterval))
			// fmt.Println(index)
			yFreq[index]++			
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