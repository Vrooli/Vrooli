package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestConfigLoading tests configuration loading
func TestConfigLoading(t *testing.T) {
	t.Run("LoadConfigFromEnv", func(t *testing.T) {
		// Set required env vars
		t.Setenv("API_PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
		t.Setenv("OLLAMA_URL", "http://localhost:11434")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if cfg.APIPort != "8080" {
			t.Errorf("Expected API port 8080, got %s", cfg.APIPort)
		}

		if cfg.DatabaseURL == "" {
			t.Error("Expected DATABASE_URL to be set")
		}

		if cfg.OllamaURL == "" {
			t.Error("Expected OLLAMA_URL to be set")
		}
	})
}

// TestLoggerCreation tests logger functionality
func TestLoggerCreation(t *testing.T) {
	t.Run("CreateLogger", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatal("Expected logger to be created")
		}
	})
}

// TestServerCreation tests server creation
func TestServerCreation(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   "http://localhost:11434",
	}

	t.Run("CreateServer", func(t *testing.T) {
		server := NewServer(cfg, testDB.db, logger)
		if server == nil {
			t.Fatal("Expected server to be created")
		}

		if server.router == nil {
			t.Error("Expected router to be initialized")
		}

		if server.connectionManager == nil {
			t.Error("Expected connection manager to be initialized")
		}

		if server.eventPublisher == nil {
			t.Error("Expected event publisher to be initialized")
		}
	})
}

// TestMiddleware tests middleware functionality
func TestMiddleware(t *testing.T) {
	t.Run("CORSMiddleware", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := CORSMiddleware(handler)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		// Check CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS allow origin header")
		}
	})

	t.Run("RecoveryMiddleware", func(t *testing.T) {
		testLogger := NewLogger()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		wrapped := RecoveryMiddleware(testLogger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		// Should not panic
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 after panic, got %d", w.Code)
		}
	})

	t.Run("ContentTypeMiddleware", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := ContentTypeMiddleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}

// TestRateLimiter tests rate limiting middleware
func TestRateLimiter(t *testing.T) {
	t.Run("CreateRateLimiter", func(t *testing.T) {
		limiter := NewRateLimiter(10, time.Minute)
		if limiter == nil {
			t.Fatal("Expected rate limiter to be created")
		}
	})

	t.Run("RateLimitEnforcement", func(t *testing.T) {
		limiter := NewRateLimiter(2, time.Second)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := limiter.Middleware(handler)

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "127.0.0.1:1234"
		w1 := httptest.NewRecorder()
		wrapped.ServeHTTP(w1, req1)

		if w1.Code != http.StatusOK {
			t.Errorf("First request should succeed, got %d", w1.Code)
		}

		// Second request should succeed
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "127.0.0.1:1234"
		w2 := httptest.NewRecorder()
		wrapped.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Second request should succeed, got %d", w2.Code)
		}

		// Third request should be rate limited
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.RemoteAddr = "127.0.0.1:1234"
		w3 := httptest.NewRecorder()
		wrapped.ServeHTTP(w3, req3)

		if w3.Code != http.StatusTooManyRequests {
			t.Errorf("Third request should be rate limited, got %d", w3.Code)
		}
	})
}

// TestConnectionManager tests WebSocket connection management
func TestConnectionManager(t *testing.T) {
	testLogger := NewLogger()

	t.Run("CreateConnectionManager", func(t *testing.T) {
		cm := NewConnectionManager(testLogger)
		if cm == nil {
			t.Fatal("Expected connection manager to be created")
		}
	})

	t.Run("StartConnectionManager", func(t *testing.T) {
		cm := NewConnectionManager(testLogger)

		// Start manager in goroutine
		go cm.Start()

		// Give it a moment to initialize
		time.Sleep(10 * time.Millisecond)

		// Manager should be running - we can't easily test WebSocket connections
		// without a full server setup, but we verified it initializes
	})
}

// TestEventPublisher tests event publishing
func TestEventPublisher(t *testing.T) {
	testLogger := NewLogger()

	t.Run("CreateEventPublisher", func(t *testing.T) {
		ep := NewEventPublisher(testLogger)
		if ep == nil {
			t.Fatal("Expected event publisher to be created")
		}
	})

	t.Run("PublishEvent", func(t *testing.T) {
		ep := NewEventPublisher(testLogger)

		payload := map[string]interface{}{
			"chatbot_id": uuid.New().String(),
			"test_data":  "value",
		}

		// Should not panic
		ep.PublishEvent("test.event", payload)
	})
}

// TestWidgetGeneration tests widget embed code generation
func TestWidgetGeneration(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   "http://localhost:11434",
	}

	server := NewServer(cfg, testDB.db, logger)

	t.Run("GenerateWidgetEmbedCode", func(t *testing.T) {
		chatbotID := uuid.New().String()
		widgetConfig := map[string]interface{}{
			"theme":    "light",
			"position": "bottom-right",
		}

		embedCode := server.GenerateWidgetEmbedCode(chatbotID, widgetConfig)

		if embedCode == "" {
			t.Error("Expected embed code to be generated")
		}

		// Verify embed code contains chatbot ID
		if !strings.Contains(embedCode, chatbotID) {
			t.Error("Expected embed code to contain chatbot ID")
		}

		// Verify embed code contains script tag
		if !strings.Contains(embedCode, "<script") {
			t.Error("Expected embed code to contain script tag")
		}
	})
}

// TestModels tests data model structures
func TestModels(t *testing.T) {
	t.Run("CreateChatbot", func(t *testing.T) {
		chatbot := &Chatbot{
			ID:          uuid.New().String(),
			Name:        "Test Bot",
			Description: "Test description",
			IsActive:    true,
			ModelConfig: map[string]interface{}{
				"model": "llama2",
			},
		}

		if chatbot.ID == "" {
			t.Error("Expected chatbot ID to be set")
		}

		if chatbot.Name != "Test Bot" {
			t.Error("Expected chatbot name to match")
		}
	})

	t.Run("CreateTenant", func(t *testing.T) {
		tenant := &Tenant{
			ID:          uuid.New().String(),
			Name:        "Test Tenant",
			Slug:        "test-tenant",
			Plan:        "pro",
			MaxChatbots: 10,
			IsActive:    true,
		}

		if tenant.ID == "" {
			t.Error("Expected tenant ID to be set")
		}

		if tenant.Slug != "test-tenant" {
			t.Error("Expected tenant slug to match")
		}
	})

	t.Run("CreateConversation", func(t *testing.T) {
		conversation := &Conversation{
			ID:           uuid.New().String(),
			ChatbotID:    uuid.New().String(),
			SessionID:    uuid.New().String(),
			LeadCaptured: false,
		}

		if conversation.ID == "" {
			t.Error("Expected conversation ID to be set")
		}
	})

	t.Run("CreateMessage", func(t *testing.T) {
		message := &Message{
			ID:             uuid.New().String(),
			ConversationID: uuid.New().String(),
			Role:           "user",
			Content:        "Hello",
		}

		if message.Role != "user" {
			t.Error("Expected message role to be user")
		}

		if message.Content != "Hello" {
			t.Error("Expected message content to match")
		}
	})
}

// TestErrorHandling tests error handling patterns
func TestErrorHandling(t *testing.T) {
	t.Run("InvalidUUIDHandling", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		// Test with invalid UUID
		_, err := testDB.db.GetChatbot("invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID")
		}
	})
}

// TestJSONSerialization tests JSON encoding/decoding
func TestJSONSerialization(t *testing.T) {
	t.Run("SerializeChatbot", func(t *testing.T) {
		chatbot := &Chatbot{
			ID:          uuid.New().String(),
			Name:        "Test Bot",
			Description: "Test description",
			IsActive:    true,
		}

		data, err := json.Marshal(chatbot)
		if err != nil {
			t.Fatalf("Failed to marshal chatbot: %v", err)
		}

		var decoded Chatbot
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal chatbot: %v", err)
		}

		if decoded.Name != chatbot.Name {
			t.Error("Expected chatbot name to match after serialization")
		}
	})

	t.Run("SerializeChatRequest", func(t *testing.T) {
		req := ChatRequest{
			Message:   "Hello",
			SessionID: uuid.New().String(),
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal chat request: %v", err)
		}

		var decoded ChatRequest
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal chat request: %v", err)
		}

		if decoded.Message != req.Message {
			t.Error("Expected message to match after serialization")
		}
	})
}
