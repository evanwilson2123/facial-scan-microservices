package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image-upload-service/utils"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

type ImageRequest struct {
	ImageUrl string `json:"image_url"`
	UserId  string `json:"user_id"`
}

type ImageDataStore struct {
	ImageURL 			  string  `json:"image_url"`
	UserId				  string  `json:"user_id"`
	TotalScore			  float32 `json:"total_score"`	
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

type ImageResponse struct {
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

	consumer, err := sarama.NewConsumerGroup(brokers, "image-upload-group", config)
	if err != nil {
		log.Fatalf("Error creating consumer group: %v", err)
	}

	handler := ConsumerGroupHandler{}

	for {
		err := consumer.Consume(context.Background(), []string{"image-upload"}, handler)
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
		case "image-upload":
			processImageUpload(msg)
		case "image-processing-response":
			processImageProcessingResponse(msg)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func processImageUpload(msg *sarama.ConsumerMessage) {
	bucketName := os.Getenv("BUCKET_NAME")
	bucket := utils.StorageClient.Bucket(bucketName)

	// Extract metadata from headers
	var filename, contentType, userID string
	for _, header := range msg.Headers {
		switch string(header.Key) {
		case "filename":
			filename = string(header.Value)
		case "contentType":
			contentType = string(header.Value)
		case "userID":
			userID = string(header.Value)
		}
	}
	// Create a new object in the bucket
	fileName := fmt.Sprintf("images/%d_%s", time.Now().Unix(), filename)
	object := bucket.Object(fileName)
	writer := object.NewWriter(context.Background())
	writer.ContentType = contentType

	// Write the file data to the object
	if _, err := writer.Write(msg.Value); err != nil {
		log.Printf("Error writing file to GCS: %v", err)
		return
	}
	if err := writer.Close(); err != nil {
		log.Printf("Error closing writer: %v", err)	
		return
	}

	if err := object.ACL().Set(context.Background(), storage.AllUsers, storage.RoleReader); err != nil {
		log.Printf("Error setting object ACL: %v", err)
		return
	}

	publicUrl := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, fileName)
	log.Printf("Image uploaded successfully: %s", publicUrl)


	// Create a message for the Image Processing Service
	imageRequest := ImageRequest{
		ImageUrl: publicUrl,
		UserId: userID,
	}

	jsonData, err := json.Marshal(imageRequest)
	if err != nil {
		log.Printf("Error marshalling image request: %v", err)
		return
	}

	kafkaMessage := &sarama.ProducerMessage{
		Topic: "image-processing",
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

func processImageProcessingResponse(msg *sarama.ConsumerMessage) {
	var imageResponse ImageResponse
	if err := json.Unmarshal(msg.Value, &imageResponse); err != nil {
		log.Printf("Error unmarshalling image response: %v", err)
		return
	}

	// Save the complete ImageDataStore to Firestore
	imageData := ImageDataStore{
		ImageURL:              imageResponse.ImageURL,
		UserId:                imageResponse.UserId,
		TotalScore:            imageResponse.TotalScore,
		Symmetry:              imageResponse.Symmetry,
		FacialDefinition:      imageResponse.FacialDefinition,
		Jawline:               imageResponse.Jawline,
		Cheekbones:            imageResponse.Cheekbones,
		JawlineToCheekbones:   imageResponse.JawlineToCheekbones,
		CanthalTilt:           imageResponse.CanthalTilt,
		ProportionAndRatios:   imageResponse.ProportionAndRatios,
		SkinQuality:           imageResponse.SkinQuality,
		LipFullness:           imageResponse.LipFullness,
		FacialFat:             imageResponse.FacialFat,
		CompleteFacialHarmony: imageResponse.CompleteFacialHarmony,
	}

	ctx := context.Background()
	_, _, err := utils.FirestoreClient.Collection("images").Add(ctx, imageData)
	if err != nil {
		log.Printf("Error adding image data to Firestore: %v", err)
	}
}

