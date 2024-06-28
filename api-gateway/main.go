package main

import (
	"log"

	"api-gateway/routes"
	"api-gateway/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Initialize Firebase
	utils.InitFirebase()
	defer utils.CloseFirestore()

	utils.InitKafkaConsumer()

	// Create a new Fiber instance
	app := fiber.New()

	// Set up CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173, https://9b7aa3157677.ngrok.app",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Set up routes
	routes.SetupRoutes(app)

	// Start the server
	log.Fatal(app.Listen(":4001"))
}
