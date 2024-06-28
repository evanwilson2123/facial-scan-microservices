package controllers

import (
	"auth-service/models"
	"auth-service/utils"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/api/iterator"
)

func HandleUserRegistration(user models.User) (int, string) {

	exists, err := emailExist(user.Email)
	if err != nil {
		return http.StatusInternalServerError, "Error checking if email exists"
	}
	if exists {
		log.Printf("Email %s already exists", user.Email)
		return http.StatusConflict, "Email already exists"
	}
	// exists, err = usernameExist(user.Username)
	// if err != nil {
	// 	return http.StatusInternalServerError, "Error checking if username exists"
	// }
	// if exists {
	// 	log.Printf("Username %s already exists", user.Username)
	// 	return http.StatusConflict, "Username already exists"
	// }
	user.CreatedAt = time.Now().Unix()
	user.Password = utils.HashPassword(user.Password)
	_, err = utils.FirestoreClient.Collection("users").Doc(user.UID).Set(context.Background(), user)
	if err != nil {
		log.Printf("Error saving user to database: %v", err)
		return http.StatusInternalServerError, "Error saving user to database"
	}

	log.Printf("User created successfully: %v", user.Email)
	return http.StatusOK, "User created successfully"
}

func emailExist(email string) (bool, error) {
	iter := utils.FirestoreClient.Collection("users").Where("Email", "==", email).Limit(1).Documents(context.Background())

	_, err := iter.Next()
	if err == iterator.Done {
		// No documents found
		return false, nil
	}
	if err != nil {
		// Error occurred during iteration
		return false, err
	}
	return true, nil
}

// func usernameExist(username string) (bool, error) {
// 	iter := utils.FirestoreClient.Collection("users").Where("Username", "==", username).Limit(1).Documents(context.Background())
// 	_, err := iter.Next()
// 	if err == iterator.Done {
// 		// No documents found
// 		return false, nil
// 	}
// 	if err != nil {
// 		return false, err
// 	}
// 	return true, nil
// }


func HealthCheck(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":"healthy",
	})
}
