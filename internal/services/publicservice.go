package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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


// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>


func (s *PublicService) SignupPost(ctx *gin.Context, signupData *dto.SignupData) (*errs.Error) {

	if signupData.Email == "" || signupData.Password == "" {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "The email and password cannot be empty. Try again!",
			ToRespondWith: true,
		}
	}

	passStr := signupData.Password

	passLen := utf8.RuneCountInString(passStr)
	if passLen <= 12 || passLen >= 45 {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "The password should be more than 12 and less than 45 characters. Try again!",
			ToRespondWith: true,
		}
	}

	upperCount := 0
	lowerCount := 0
	digitsCount := 0
	specCount := 0

	allowedSymbols := "!@#$%^&*"

	for _, c := range passStr {
		switch {
		case unicode.IsUpper(c):
			upperCount++
		case unicode.IsLower(c):
			lowerCount++
		case unicode.IsDigit(c):
			digitsCount++
		default:
			if strings.ContainsRune(allowedSymbols, c) {
				specCount++
			} else {
				return &errs.Error{
					Type: errs.PreconditionFailed,
					Message: "Invalid characters used in password. Please use only given valid characters.",
					ToRespondWith: true,
				}
			}
		}
	}

	if upperCount < 1 {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "No Upper characters used. Please use atleast one of all given ranges.",
			ToRespondWith: true,
		}
	}
	if lowerCount < 1 {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "No Lower characters used. Please use atleast one of all given ranges.",
			ToRespondWith: true,
		}
	}
	if digitsCount < 1 {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "No Digits used. Please use atleast one of all given ranges.",
			ToRespondWith: true,
		}
	}
	if specCount < 1 {
		return &errs.Error{
			Type: errs.PreconditionFailed,
			Message: "No Special characters used. Please use atleast one of all given ranges.",
			ToRespondWith: true,
		}
	}
	fmt.Println("error")

	hashed_pass, err := bcrypt.GenerateFromPassword([]byte(signupData.Password), 10)
	if err != nil {
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to generate hash from password : " + err.Error(),
		}
	}
	signupData.Password = string(hashed_pass)
	fmt.Println("error")

	err = s.queries.SignupUser(ctx, sqlc.SignupUserParams{
		Email: signupData.Email,
		Password: signupData.Password,
		Role: 1,
	})
	if err != nil {
		var pgerr *pgconn.PgError
		if (errors.As(err, &pgerr)) {
			if (pgerr.Code == errs.UniqueViolation) {
				return &errs.Error{
					Type: errs.UniqueViolation,
					Message: "User with Email-Id already exists. Try with different Id.",
					ToRespondWith: true,
				}		
			}
		}
		return &errs.Error{
			Type: errs.Internal,
			Message: "Failed to insert new user : " + err.Error(),
		}
	}
	fmt.Println("error")

	return nil
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

	// 1) check if service exists
	serviceData, err := s.queries.GetServiceData(ctx, data.Name)
	if err != nil {
		// TODO: process pgerr
		return nil, nil
	}

	// 2) check if user is owner of service
	if serviceData.UserID != userID {
		return nil, nil
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