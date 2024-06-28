package utils

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
)

var consumer sarama.ConsumerGroup

func ProduceKafkaMessage(topic string, message interface{}) error {
	brokers := []string{"kafka:9092"}
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return err
	}
	defer producer.Close()

	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	}

	partition, offset, err := producer.SendMessage(kafkaMsg)
	if err != nil {
		log.Printf("Error producing message to kafka: %v\n", err)
		return err
	}

	log.Printf("Message is stored in topic(%s)/partition(%d)/offset(%d)", topic, partition, offset)
	return nil
}

func InitKafkaConsumer() {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	var err error
	consumer, err = sarama.NewConsumerGroup([]string{"kafka:9092"}, "api-gateway-group", config)
	if err != nil {
		log.Fatalf("Error creating consumer group: %v", err)
	}
}

func ConsumeKafkaMessage(topic, key string, timeout time.Duration) (*fiber.Map, int, error) {
	handler := &ConsumerGroupHandler{
		ready:   make(chan bool),
		key:     key,
		message: make(chan *sarama.ConsumerMessage),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		for {
			if err := consumer.Consume(ctx, []string{topic}, handler); err != nil {
				log.Printf("Error consuming messages: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
			handler.ready = make(chan bool)
		}
	}()

	<-handler.ready
	select {
	case msg := <-handler.message:
		log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(msg.Value), msg.Timestamp, msg.Topic)

		var response fiber.Map
		if err := json.Unmarshal(msg.Value, &response); err != nil {
			return nil, 0, err
		}

		statusCode := http.StatusOK
		if code, exists := response["statusCode"].(float64); exists {
			statusCode = int(code)
			delete(response, "statusCode")
		}

		return &response, statusCode, nil
	case <-ctx.Done():
		return nil, 0, errors.New("timeout waiting for message")
	}
}

type ConsumerGroupHandler struct {
	ready   chan bool
	message chan *sarama.ConsumerMessage
	key     string
}

func (h *ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if string(message.Key) == h.key {
			h.message <- message
		}
		sess.MarkMessage(message, "")
	}
	return nil
}

func ProduceKafkaMessageWithHeaders(msg *sarama.ProducerMessage) error {
	brokers := []string{"kafka:9092"}
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return err
	}
	defer producer.Close()

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		log.Printf("Error producing message to kafka: %v\n", err)
		return err
	}

	log.Printf("Message is stored in topic(%s)/partition(%d)/offset(%d)", msg.Topic, partition, offset)
	return nil
}
