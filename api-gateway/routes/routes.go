package routes

import (
	// "api-gateway/middleware"
	"api-gateway/middleware"
	"api-gateway/models"
	"api-gateway/utils"
	"log"
	"net/http"
	"time"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	
	api := app.Group("/api", middleware.AuthRequired())

	api.Post("/register", func(c *fiber.Ctx) error {
		var user models.User
		if err := c.BodyParser(&user); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error":"Invalid request, bad user object",
			})
		}

		user.UID = c.Locals("user_id").(string)

		err := utils.ProduceKafkaMessage("user-registration", user)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error producing message to Kafka",
			})
		}

		response, statusCode, err := utils.ConsumeKafkaMessage("user-registration-response", user.Email, 2 * time.Second)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error consuming message from Kafka",
			})
		}

		return c.Status(statusCode).JSON(response)
	})

	
	
	api.Post("/create-account", func(c *fiber.Ctx) error {
		log.Println("Create account")
		uid := c.Locals("user_id").(string)
		var user models.User
		if err := c.BodyParser(&user); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error":"Invalid request, bad user object",
			})
		}
		user.UID = uid
		log.Println("Sending kafka message to create account")
		err := utils.ProduceKafkaMessage("user-profile-update", user)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error producing message to Kafka",
			})
		}
		log.Println("Consuming kafka message to create account")
		response, statusCode, err := utils.ConsumeKafkaMessage("user-profile-update-response", user.Email, 5 * time.Second)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error consuming message from Kafka",
			})
		}
		log.Printf("Response: %v", response)
		return c.Status(statusCode).JSON(response)
	})

	
	
	api.Get("/has-username", func(c *fiber.Ctx) error {
		var hasUsername struct {
			UID		string	 `json:"uid"`
		}
		hasUsername.UID = c.Locals("user_id").(string)
			err := utils.ProduceKafkaMessage("username-check", hasUsername)
			if err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error":"Error producing message to Kafka",
				})
			}

			response, statusCode, err := utils.ConsumeKafkaMessage("username-check-response", hasUsername.UID, 5 * time.Second)
			if err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error":"Error consuming message from Kafka",
				})
			}
			return c.Status(statusCode).JSON(response)
		})

	
	
	api.Post("/image-upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error":"No file uploaded",
			})
		}
		fileHeader, err := file.Open()
		if err != nil {
			log.Printf("Error opening file: %v", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error opening file",
			})
		}
		defer fileHeader.Close()

		// Create a buffer to read the file info
		buf := make([]byte, file.Size)
		_, err = fileHeader.Read(buf)
		if err != nil {
			log.Println("Error reading file")
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error reading file",
			})
		}
		uid := c.Locals("user_id").(string)
		kafkaMessage := &sarama.ProducerMessage{
			Topic: "image-upload",
			Value: sarama.ByteEncoder(buf),
			Headers: []sarama.RecordHeader{
				{Key: []byte("filename"), Value: []byte(file.Filename)},
				{Key: []byte("contentType"), Value: []byte(file.Header.Get("Content-Type"))},
				{Key: []byte("userID"), Value: []byte(uid)},
			},
		}

		// Produce the message to kafka
		err = utils.ProduceKafkaMessageWithHeaders(kafkaMessage)
		if err != nil {
			log.Println("Error producing message to kafka")
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"error creating kafka message",
			})
		}

		// Wait for response from image processing service
		response, statusCode, err := utils.ConsumeKafkaMessage("image-processing-response", uid, 30 * time.Second)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":"Error consuming message from Kafka",
			})
		}

		return c.Status(statusCode).JSON(response)
	})
}
