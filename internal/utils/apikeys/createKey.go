package apikeys

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type APICreds struct {
    ID   string `json:"id"`
    Key  string `json:"key"`
}

type DecodedKey struct {

}

type KeyMetadata struct {
	CreatedAt int64
}

// generateRandomStr generates a cryptographically safe string of given length.
func generateRandomStr(length int32) (randStr string, err error) {
	bytes := make([]byte, length)
	_, err = rand.Read(bytes)
	if err != nil {
        return "", err
    }
	return base64.URLEncoding.EncodeToString(bytes), nil
}


func CreateWithOutSeed() (*APICreds, error) {

	secretPassStr, exists := os.LookupEnv("APIKeySecretPassword")
	if !exists {
		return nil, fmt.Errorf("no api key signing password found in env")
	}
	secretPass := []byte(secretPassStr)

	keyVersion, exists := os.LookupEnv("APIKeyGenerationVersion")
	if !exists {
		return nil, fmt.Errorf("no api key signing password found in env")
	}

	apiID, err := generateRandomStr(64)
	if err != nil {
		return nil, err
	}

	metaBytes, err := json.Marshal(KeyMetadata{
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	metaEncoded := base64.URLEncoding.EncodeToString(metaBytes)

	h := hmac.New(sha256.New, secretPass)
	
	_, err = h.Write([]byte(apiID + "." + metaEncoded))
	if err != nil {
		return nil, err
	}

	apiKey := fmt.Sprintf("%s.%s.%s.%s", keyVersion, apiID, metaEncoded, base64.URLEncoding.EncodeToString(h.Sum(nil)))

	return &APICreds{
		ID: apiID,
		Key: apiKey,
	}, nil
}

func CreateWithGiven(serviceID string) (*APICreds, error) {

	secretPassStr, exists := os.LookupEnv("APIKeySecretPassword")
	if !exists {
		return nil, fmt.Errorf("no api key signing password found in env")
	}
	secretPass := []byte(secretPassStr)

	keyVersion, exists := os.LookupEnv("APIKeyGenerationVersion")
	if !exists {
		return nil, fmt.Errorf("no api key signing password found in env")
	}

	apiID, err := generateRandomStr(64)
	if err != nil {
		return nil, err
	}

	metaBytes, err := json.Marshal(KeyMetadata{
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	metaEncoded := base64.URLEncoding.EncodeToString(metaBytes)

	// shaHash := sha256.Sum256([]byte(serviceID))
	// fmt.Println(hex.EncodeToString(shaHash[:]))

	h := hmac.New(sha256.New, secretPass)
	
	_, err = h.Write([]byte(apiID + "." + metaEncoded))
	if err != nil {
		return nil, err
	}

	apiKey := fmt.Sprintf("%s.%s.%s.%s", keyVersion, apiID, metaEncoded, base64.URLEncoding.EncodeToString(h.Sum(nil)))

	return &APICreds{
		ID: apiID,
		Key: apiKey,
	}, nil
}


// TODO: there should be a recreate from old key function for ttl based renewation
// TODO: we should be able to validate keys


func DecodeKey(key string) (*DecodedKey, error) {

	



	return nil, nil
}