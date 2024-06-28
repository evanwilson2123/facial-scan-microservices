package main

import (
	"auth-service/controllers"
	"auth-service/models"
	"auth-service/utils"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main(){
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	utils.InitFirebase()
	defer utils.CloseFirestore()

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:4000, https://your-frontend-domain.com",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-User-ID",
		AllowCredentials: true,
	}))

	app.Get("/health", controllers.HealthCheck)

	go startKafkaConsumer()

	log.Fatal(app.Listen(":8080"))


	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}


func startKafkaConsumer() {
	brokers := []string{"kafka:9092"}
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup(brokers, "auth-service-group", config)
	if err != nil {
		log.Fatalf("Error creating consumer group: %v", err)
	}

	handler := ConsumerGroupHandler{}

	for {
		err := consumer.Consume(context.Background(), []string{"user-registration"}, handler)
		if err != nil {
			log.Printf("Error from consumer: %v", err)
		}
	}
}

type ConsumerGroupHandler struct{}

func (ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (ConsumerGroupHandler)	Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h ConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var user models.User
		err := json.Unmarshal(msg.Value, &user)
		if err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		statusCode, responseMessage := controllers.HandleUserRegistration(user)
		response := map[string]interface{}{
			"message": responseMessage,
			"email": user.Email,
			"statusCode": statusCode,
		}
		produceResponseMessage(response)
		
		sess.MarkMessage(msg, "")
		}
	return nil
}

func produceResponseMessage(response map[string]interface{}) { // Add opening curly brace here
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
		Topic: "user-registration-response",
		Value: sarama.StringEncoder(msg),
		Key:  sarama.StringEncoder(response["email"].(string)),
	}

	_, _, err = producer.SendMessage(kafkaMsg)
	if err != nil {
		log.Printf("Error producing message: %v", err)
	}
}