package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"user-management-service/models"
	"user-management-service/utils"

	"google.golang.org/api/iterator"
)

func HandleUserProfileUpdate(user models.User) (int, string) {
	fmt.Printf("Updating user profile for UID: %s\n", user.UID)
	uid := user.UID

	// Get the existing user document
	docRef := utils.FirestoreClient.Collection("users").Doc(uid)
	doc, err := docRef.Get(context.Background())
	if err != nil {
		log.Println("error getting user document from database:", err)
		return http.StatusInternalServerError, "Error getting user document from database"
	}

	var existingUser models.User
	if err := doc.DataTo(&existingUser); err != nil {
		log.Println("Error unmarshalling user data from Firestore:", err)
		return http.StatusInternalServerError, "Error unmarshalling user data from Firestore"
	}

	// Update the user document
	existingUser.Username = user.Username
	existingUser.Age = user.Age
	existingUser.Gender = user.Gender
	existingUser.HighScore = user.HighScore

	if _, err := docRef.Set(context.Background(), existingUser); err != nil {
		log.Println("Error updating user document:", err)
		return http.StatusInternalServerError, "Error updating user document"
	}

	log.Println("User profile updated successfully")
	return http.StatusOK, "User profile updates successfully"
}

func CheckUserHasUsername(uid string) (bool, error) {
	ctx := context.Background()

	log.Printf("Checking username for UID: %s\n", uid)
	iter := utils.FirestoreClient.Collection("users").Where("UID", "==", uid).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		log.Printf("User not found for UID: %s\n", uid)
		return false, nil // User not found, return false without error
	}
	if err != nil {
		log.Printf("Error iterating documents for UID %s: %v\n", uid, err)
		return false, err
	}

	log.Printf("Document found for UID: %s\n", doc.Ref.ID)
	var user struct {
		Username string `json:"username"`
	}
	if err := doc.DataTo(&user); err != nil {
		log.Printf("Error unmarshalling document data for UID %s: %v\n", uid, err)
		return false, err
	}

	hasUsername := user.Username != ""
	log.Printf("User has username: %v for UID: %s\n", hasUsername, uid)
	return hasUsername, nil
}

func UpdateUserHighScore(uid string, highScore float64) error {
	ctx := context.Background()

	log.Printf("Updating high score for UID: %s\n", uid)
	docRef := utils.FirestoreClient.Collection("users").Doc(uid)
	doc, err := docRef.Get(ctx)
	if err != nil {
		log.Printf("Error getting user document for UID %s: %v\n", uid, err)
		return err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		log.Printf("Error unmarshalling user data for UID %s: %v\n", uid, err)
		return err
	}

	user.HighScore = highScore
	if _, err := docRef.Set(ctx, user); err != nil {
		log.Printf("Error updating high score for UID %s: %v\n", uid, err)
		return err
	}

	log.Printf("High score updated successfully for UID: %s\n", uid)
	return nil
}