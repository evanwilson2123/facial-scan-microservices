package main

import (
	"context"
	"encoding/json"
	"fmt"
	"leaderboard-service/models"
	"leaderboard-service/utils"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
)

type LeaderBoard struct {
	Username      string         `json:"username"`
	ImageResponse ImageDataStore `json:"image_response"`
}

type ImageDataStore struct {
	ImageURL              string  `json:"image_url"`
	UserId                string  `json:"user_id"`
	TotalScore            float32 `json:"total_score"`
	Symmetry              float64 `json:"symmetry"`
	FacialDefinition      float64 `json:"facial_definition"`
	Jawline               float64 `json:"jawline"`
	Cheekbones            float64 `json:"cheekbones"`
	JawlineToCheekbones   float64 `json:"jawline_to_cheekbones"`
	CanthalTilt           float64 `json:"canthal_tilt"`
	ProportionAndRatios   float64 `json:"proportion_and_ratios"`
	SkinQuality           float64 `json:"skin_quality"`
	LipFullness           float64 `json:"lip_fullness"`
	FacialFat             float64 `json:"facial_fat"`
	CompleteFacialHarmony float64 `json:"complete_facial_harmony"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	fmt.Println("Starting server...")

	app := fiber.New()

	utils.InitFirebase()
	defer utils.CloseFirestore()

	go startKafkaConsumer()

	log.Fatal(app.Listen(":8082"))
}

func startKafkaConsumer() {
	brokers := []string{"kafka:9092"}
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup(brokers, "leaderboard-group", config)
	if err != nil {
		log.Fatalf("Error creating consumer group: %v", err)
	}

	handler := ConsumerGroupHandler{}

	for {
		err := consumer.Consume(context.Background(), []string{"leaderboard-male", "leaderboard-female"}, handler)
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
		case "leaderboard-male":
			response, statusCode, err := getLeaderboardMale()
			produceResponseMessage(response, "leaderboard-male-response", "leaderboard-male", statusCode, err)
		case "leaderboard-female":
			response, statusCode, err := getLeaderboardFemale()
			produceResponseMessage(response, "leaderboard-female-response", "leaderboard-female", statusCode, err)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func getLeaderboardMale() (map[string]interface{}, int, error) {
	ctx := context.Background()
	query := utils.FirestoreClient.Collection("users").Where("Gender", "==", "Male").OrderBy("HighScore", firestore.Desc).Limit(50)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var users []models.User
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating documents: %v\n", err)
			return nil, 500, err
		}

		var user models.User
		if err := doc.DataTo(&user); err != nil {
			log.Printf("Error unmarshalling document data: %v\n", err)
			return nil, 500, err
		}
		users = append(users, user)
		log.Printf("User: %v\n", user)
	}

	if len(users) == 0 {
		log.Println("No users found")
		return nil, 404, nil
	}

	var leaderboard []LeaderBoard
	for _, user := range users {
		processUserImage(ctx, &leaderboard, user)
	}

	response := map[string]interface{}{
		"leaderboard": leaderboard,
	}
	log.Printf("Leaderboard: %v\n", leaderboard)
	return response, 200, nil
}

func getLeaderboardFemale() (map[string]interface{}, int, error) {
	ctx := context.Background()
	query := utils.FirestoreClient.Collection("users").Where("Gender", "==", "Female").OrderBy("HighScore", firestore.Desc).Limit(50)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var users []models.User
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating documents: %v\n", err)
			return nil, 500, err
		}

		var user models.User
		if err := doc.DataTo(&user); err != nil {
			log.Printf("Error unmarshalling document data: %v\n", err)
			return nil, 500, err
		}
		users = append(users, user)
		log.Printf("User: %v\n", user)
	}

	if len(users) == 0 {
		log.Println("No users found")
		return nil, 404, nil
	}

	var leaderboard []LeaderBoard
	for _, user := range users {
		processUserImage(ctx, &leaderboard, user)
	}

	response := map[string]interface{}{
		"leaderboard": leaderboard,
	}
	log.Printf("Leaderboard: %v\n", leaderboard)
	return response, 200, nil
}

func processUserImage(ctx context.Context, leaderboard *[]LeaderBoard, user models.User) {
	imageQuery := utils.FirestoreClient.Collection("images").Where("UserId", "==", user.UID).OrderBy("TotalScore", firestore.Desc).Limit(1)
	imageIter := imageQuery.Documents(ctx)
	defer imageIter.Stop()

	for {
		imageDoc, err := imageIter.Next()
		if err == iterator.Done {
			log.Printf("No image found for user ID: %s", user.UID)
			break
		}
		if err != nil {
			log.Printf("Error iterating image documents: %v", err)
			return
		}

		var imageDataStore ImageDataStore
		if err := imageDoc.DataTo(&imageDataStore); err != nil {
			log.Printf("Error unmarshalling image data: %v", err)
			return
		}

		log.Printf("Successfully fetched image data for user ID: %s, Total Score: %f", user.UID, imageDataStore.TotalScore)
		log.Printf("Total score for user %s: %f", user.Username, imageDataStore.TotalScore)

		*leaderboard = append(*leaderboard, LeaderBoard{
			Username:      user.Username,
			ImageResponse: imageDataStore,
		})
	}
}

func produceResponseMessage(response map[string]interface{}, topic, key string, statusCode int, err error) {
	response["statusCode"] = statusCode
	if err != nil {
		response["error"] = err.Error()
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return
	}

	kafkaMessage := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(jsonData),
	}

	producer, err := sarama.NewSyncProducer([]string{"kafka:9092"}, nil)
	if err != nil {
		log.Printf("Error creating producer: %v", err)
		return
	}

	defer producer.Close()

	_, _, err = producer.SendMessage(kafkaMessage)
	if err != nil {
		log.Printf("Error producing message: %v", err)
	}
}
