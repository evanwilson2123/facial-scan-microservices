package utils

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/rs/xid"
	"google.golang.org/api/option"
)

var FirebaseApp *firebase.App
var AuthClient *auth.Client
var FirestoreClient *firestore.Client
var StorageClient *storage.Client

func InitFirebase() {
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	file, err := os.Open(credentialsFile)
	if err != nil {
		log.Fatalf("Error opening credentials file: %v", err)
	}
	file.Close()
	fmt.Println("Credentials file opened successfully")

	opt := option.WithCredentialsFile(credentialsFile)
	FirebaseApp, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing firebase app: %v\n", err)
	}
	AuthClient, err = FirebaseApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing firebase auth client: %v\n", err)
	}
	appId := os.Getenv("APP_ID")
	FirestoreClient, err = firestore.NewClient(context.Background(), appId, opt)
	if err != nil {
		log.Fatalf("error initializing firestore client: %v\n", err)
	}
	StorageClient, err = storage.NewClient(context.Background(), opt)
	if err != nil {
		log.Fatalf("error initializing storage client: %v\n", err)
	}
}

func ValidateToken(idToken string) (*auth.Token, error) {
	token, err := AuthClient.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		return nil, err
	}
	return token, nil
} 

func CloseFirestore() {
	if FirestoreClient != nil {
		err := FirestoreClient.Close()
		if err != nil {
			log.Fatalf("error closing firestore client: %v\n", err)
		}
	}
}

func GenerateUID() string {
	return xid.New().String()
}