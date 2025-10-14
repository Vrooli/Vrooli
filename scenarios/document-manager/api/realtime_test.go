package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestInitRedis(t *testing.T) {
	tests := []struct {
		name      string
		redisURL  string
		expectErr bool
	}{
		{
			name:      "EmptyRedisURL",
			redisURL:  "",
			expectErr: false, // Should not error, just skip initialization
		},
		{
			name:      "InvalidRedisURL",
			redisURL:  "invalid://url",
			expectErr: true,
		},
		{
			name:      "ValidRedisURLButNotConnected",
			redisURL:  "redis://localhost:9999", // Non-existent port
			expectErr: false,                     // Should not error, just log warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global redisClient
			redisClient = nil

			err := initRedis(tt.redisURL)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify redisClient state
			if tt.redisURL == "" && redisClient != nil {
				t.Error("Expected redisClient to be nil for empty URL")
			}
		})
	}
}

func TestPublishEvent(t *testing.T) {
	tests := []struct {
		name          string
		setupRedis    bool
		eventType     string
		data          interface{}
		expectPublish bool
	}{
		{
			name:          "PublishWithNoRedisClient",
			setupRedis:    false,
			eventType:     EventApplicationCreated,
			data:          map[string]string{"test": "data"},
			expectPublish: false,
		},
		{
			name:       "PublishWithValidData",
			setupRedis: false, // Still no actual Redis, just test code path
			eventType:  EventAgentUpdated,
			data: map[string]interface{}{
				"id":   "test-id",
				"name": "Test Agent",
			},
			expectPublish: false,
		},
		{
			name:          "PublishWithComplexData",
			setupRedis:    false,
			eventType:     EventQueueItemCreated,
			data:          struct{ Field string }{"value"},
			expectPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset redisClient
			redisClient = nil

			// This should not panic even without Redis
			publishEvent(tt.eventType, tt.data)
		})
	}
}

func TestEventTypes(t *testing.T) {
	// Verify all event type constants are defined
	eventTypes := []string{
		EventApplicationCreated,
		EventApplicationUpdated,
		EventApplicationDeleted,
		EventAgentCreated,
		EventAgentUpdated,
		EventAgentDeleted,
		EventQueueItemCreated,
		EventQueueItemUpdated,
		EventQueueItemDeleted,
	}

	for _, eventType := range eventTypes {
		if eventType == "" {
			t.Error("Event type constant is empty")
		}
	}

	// Verify they have expected prefixes
	prefixTests := []struct {
		eventType string
		prefix    string
	}{
		{EventApplicationCreated, "application:"},
		{EventAgentCreated, "agent:"},
		{EventQueueItemCreated, "queue:"},
	}

	for _, tt := range prefixTests {
		if len(tt.eventType) < len(tt.prefix) {
			t.Errorf("Event type %s is too short", tt.eventType)
		}
		if tt.eventType[:len(tt.prefix)] != tt.prefix {
			t.Errorf("Event type %s does not start with %s", tt.eventType, tt.prefix)
		}
	}
}

func TestEventStructure(t *testing.T) {
	t.Run("EventMarshaling", func(t *testing.T) {
		event := Event{
			Type:      EventApplicationCreated,
			Timestamp: time.Now(),
			Data: map[string]string{
				"id":   "test-id",
				"name": "Test App",
			},
		}

		// Verify it can be marshaled to JSON
		eventJSON, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		// Verify it can be unmarshaled
		var unmarshaledEvent Event
		err = json.Unmarshal(eventJSON, &unmarshaledEvent)
		if err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}

		// Verify fields
		if unmarshaledEvent.Type != event.Type {
			t.Errorf("Expected type %s, got %s", event.Type, unmarshaledEvent.Type)
		}
	})
}

func TestCloseRedis(t *testing.T) {
	t.Run("CloseWithNilClient", func(t *testing.T) {
		redisClient = nil
		// Should not panic
		closeRedis()
	})

	t.Run("CloseWithMockClient", func(t *testing.T) {
		// Create a mock Redis client that won't actually connect
		redisClient = redis.NewClient(&redis.Options{
			Addr: "localhost:9999",
		})
		// Should not panic
		closeRedis()
		redisClient = nil
	})
}

func TestPublishEventWithMarshalError(t *testing.T) {
	t.Run("UnmarshalableData", func(t *testing.T) {
		// Reset redisClient
		redisClient = nil

		// Create a mock Redis client (won't actually connect)
		redisClient = redis.NewClient(&redis.Options{
			Addr: "localhost:9999",
		})
		defer func() {
			redisClient = nil
		}()

		// Create data that can't be marshaled (channels can't be marshaled)
		ch := make(chan int)
		defer close(ch)

		// This should not panic, just log error
		publishEvent(EventApplicationCreated, ch)
	})
}

func TestContextUsage(t *testing.T) {
	t.Run("ContextIsBackground", func(t *testing.T) {
		if ctx != context.Background() {
			t.Error("Expected context to be Background context")
		}
	})
}

func TestRedisClientGracefulDegradation(t *testing.T) {
	t.Run("OperationsWorkWithoutRedis", func(t *testing.T) {
		// Simulate scenario with no Redis
		redisClient = nil

		// All these should work without panicking
		publishEvent(EventApplicationCreated, map[string]string{"test": "data"})
		publishEvent(EventAgentUpdated, nil)
		publishEvent(EventQueueItemDeleted, struct{}{})

		// Verify no panic occurred
	})
}
