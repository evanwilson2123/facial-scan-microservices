package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"user-management-service/controllers"
	"user-management-service/models"
	"user-management-service/utils"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }
    fmt.Println("Starting server...")

    utils.InitFirebase()
    defer utils.CloseFirestore()

    app := fiber.New()

    app.Use(cors.New(cors.Config{
        AllowOrigins:     "http://localhost:8081, https://your-frontend-domain.com",
        AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
        AllowCredentials: true,
    }))

    go startKafkaConsumer()

    log.Fatal(app.Listen(":8081"))
}

func startKafkaConsumer() {
    brokers := []string{"kafka:9092"}
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetNewest

    consumer, err := sarama.NewConsumerGroup(brokers, "user-management-group", config)
    if err != nil {
        log.Fatalf("Error creating consumer group: %v", err)
    }

    handler := ConsumerGroupHandler{}

    for {
        err := consumer.Consume(context.Background(), []string{"user-profile-update", "username-check"}, handler)
        if err != nil {
            log.Printf("Error from consumer: %v", err)
        }
    }
}

type ConsumerGroupHandler struct{}

func (ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h ConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
		switch msg.Topic {
		case "user-profile-update":
			var user models.User
			err := json.Unmarshal(msg.Value, &user)
			if err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}
			statusCode, responseMessage := controllers.HandleUserProfileUpdate(user)
			if statusCode != http.StatusOK {
				log.Printf("Error updating user profile: %v", responseMessage)
				continue
			}
			response := map[string]interface{}{
				"message":    responseMessage,
				"email":      user.Email,
				"statusCode": statusCode,
			}
			produceResponseMessage(response, "user-profile-update-response", user.Email)
			sess.MarkMessage(msg, "")
		case "username-check":
			var check struct {
				UID string `json:"uid"`
			}
			err := json.Unmarshal(msg.Value, &check)
			if err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}
			hasUsername, err := controllers.CheckUserHasUsername(check.UID)
			if err != nil {
				log.Printf("Error checking username: %v", err)
				continue
			}
			response := map[string]interface{}{
				"hasUsername": hasUsername,
				"uid":         check.UID,
				"statusCode":  http.StatusOK,
			}
			produceResponseMessage(response, "username-check-response", check.UID)
			sess.MarkMessage(msg, "")
		}
	}
	return nil
}


func produceResponseMessage(response map[string]interface{}, topic, key string) {
	producer, err := sarama.NewSyncProducer([]string{"kafka:9092"}, nil)
	if err != nil {
		log.Printf("Error creating producer: %v", err)
		return
	}
	defer producer.Close()

	msg, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
		Key:   sarama.StringEncoder(key),
	}

	_, _, err = producer.SendMessage(kafkaMsg)
	if err != nil {
		log.Printf("Error producing message: %v", err)
	}
}
