package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client
var ctx = context.Background()

// Event types for pub/sub
const (
	EventApplicationCreated = "application:created"
	EventApplicationUpdated = "application:updated"
	EventApplicationDeleted = "application:deleted"
	EventAgentCreated       = "agent:created"
	EventAgentUpdated       = "agent:updated"
	EventAgentDeleted       = "agent:deleted"
	EventQueueItemCreated   = "queue:created"
	EventQueueItemUpdated   = "queue:updated"
	EventQueueItemDeleted   = "queue:deleted"
)

// Event represents a real-time event to be published
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// initRedis initializes Redis connection
func initRedis(redisURL string) error {
	if redisURL == "" {
		log.Println("Redis URL not configured, real-time updates disabled")
		return nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("invalid Redis URL: %w", err)
	}

	redisClient = redis.NewClient(opt)

	// Test connection
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v. Real-time updates will be disabled.", err)
		redisClient = nil
		return nil
	}

	log.Println("Redis connected successfully for real-time updates")
	return nil
}

// publishEvent publishes an event to Redis pub/sub
func publishEvent(eventType string, data interface{}) {
	// Skip if Redis not configured
	if redisClient == nil {
		return
	}

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	// Publish to a general channel
	err = redisClient.Publish(ctx, "document-manager:events", string(eventJSON)).Err()
	if err != nil {
		log.Printf("Error publishing event: %v", err)
		return
	}

	log.Printf("Published event: %s", eventType)
}

// closeRedis closes the Redis connection
func closeRedis() {
	if redisClient != nil {
		redisClient.Close()
	}
}
