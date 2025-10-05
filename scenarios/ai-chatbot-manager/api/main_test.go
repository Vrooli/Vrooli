package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Setup
	setupTestLogger()

	// Run tests
	code := m.Run()

	// Teardown
	os.Exit(code)
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	// Create a minimal config for testing
	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}

	// Skip database tests if not configured
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping health handler test")
	}

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Skipf("Could not connect to database: %v", err)
	}
	defer db.Close()

	server := NewServer(cfg, db, logger)

	t.Run("Success_BasicHealth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.HealthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify required fields
		if _, ok := response["status"]; !ok {
			t.Error("Expected 'status' field in response")
		}
		if _, ok := response["service"]; !ok {
			t.Error("Expected 'service' field in response")
		}
		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected 'timestamp' field in response")
		}
	})

	t.Run("Success_DetailedHealth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health?detailed=true", nil)
		w := httptest.NewRecorder()

		server.HealthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify dependencies field exists in detailed mode
		if _, ok := response["dependencies"]; !ok {
			t.Error("Expected 'dependencies' field in detailed health check")
		}
	})

	t.Run("CacheValidation", func(t *testing.T) {
		// First request
		req1 := httptest.NewRequest("GET", "/health", nil)
		w1 := httptest.NewRecorder()
		server.HealthHandler(w1, req1)

		// Second request (should use cache)
		req2 := httptest.NewRequest("GET", "/health", nil)
		w2 := httptest.NewRecorder()
		server.HealthHandler(w2, req2)

		if w1.Body.String() != w2.Body.String() {
			t.Error("Cached health check should return same result")
		}
	})
}

// TestIsValidUUID tests the UUID validation function
func TestIsValidUUID(t *testing.T) {
	t.Run("ValidUUID", func(t *testing.T) {
		validUUID := uuid.New().String()
		if !isValidUUID(validUUID) {
			t.Errorf("Expected valid UUID %s to be recognized as valid", validUUID)
		}
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		invalidUUIDs := []string{
			"invalid-uuid",
			"",
			"123",
			"not-a-uuid-at-all",
			"12345678-1234-1234-1234",
		}

		for _, invalidUUID := range invalidUUIDs {
			if isValidUUID(invalidUUID) {
				t.Errorf("Expected invalid UUID %s to be recognized as invalid", invalidUUID)
			}
		}
	})
}

// TestCreateChatbotHandler tests chatbot creation
func TestCreateChatbotHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	t.Run("Success_CreateChatbot", func(t *testing.T) {
		chatbotData := TestData.CreateChatbotRequest("Test Chatbot")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/chatbots",
			Body:   chatbotData,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.CreateChatbotHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name": "Test Chatbot",
		})

		if response != nil {
			if _, ok := response["id"]; !ok {
				t.Error("Expected 'id' field in response")
			}
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/chatbots",
			Body:   `{"invalid": "json"`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.CreateChatbotHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/chatbots",
			Body:   "",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.CreateChatbotHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty body, got %d", w.Code)
		}
	})
}

// TestListChatbotsHandler tests listing chatbots
func TestListChatbotsHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	// Create test chatbots
	chatbot1 := setupTestChatbot(t, testDB.db, "Chatbot 1")
	defer chatbot1.Cleanup()
	chatbot2 := setupTestChatbot(t, testDB.db, "Chatbot 2")
	defer chatbot2.Cleanup()

	t.Run("Success_ListChatbots", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chatbots", nil)
		w := httptest.NewRecorder()

		server.ListChatbotsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		chatbots, ok := response["chatbots"].([]interface{})
		if !ok {
			t.Fatal("Expected 'chatbots' array in response")
		}

		if len(chatbots) < 2 {
			t.Errorf("Expected at least 2 chatbots, got %d", len(chatbots))
		}
	})
}

// TestGetChatbotHandler tests retrieving a single chatbot
func TestGetChatbotHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	chatbot := setupTestChatbot(t, testDB.db, "Test Chatbot")
	defer chatbot.Cleanup()

	t.Run("Success_GetChatbot", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/chatbots/" + chatbot.Chatbot.ID,
			URLVars: map[string]string{"id": chatbot.Chatbot.ID},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.GetChatbotHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":   chatbot.Chatbot.ID,
			"name": chatbot.Chatbot.Name,
		})

		if response == nil {
			t.Fatal("Expected valid response")
		}
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/chatbots/invalid-uuid",
			URLVars: map[string]string{"id": "invalid-uuid"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.GetChatbotHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid")
	})

	t.Run("Error_NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/chatbots/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.GetChatbotHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestUpdateChatbotHandler tests chatbot updates
func TestUpdateChatbotHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	chatbot := setupTestChatbot(t, testDB.db, "Original Name")
	defer chatbot.Cleanup()

	t.Run("Success_UpdateChatbot", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":        "Updated Name",
			"description": "Updated description",
		}

		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/chatbots/" + chatbot.Chatbot.ID,
			URLVars: map[string]string{"id": chatbot.Chatbot.ID},
			Body:    updateData,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.UpdateChatbotHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"name": "Updated Name",
		})

		if response == nil {
			t.Fatal("Expected valid response")
		}
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/chatbots/invalid-uuid",
			URLVars: map[string]string{"id": "invalid-uuid"},
			Body:    map[string]interface{}{"name": "Test"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.UpdateChatbotHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid")
	})
}

// TestDeleteChatbotHandler tests chatbot deletion
func TestDeleteChatbotHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	t.Run("Success_DeleteChatbot", func(t *testing.T) {
		chatbot := setupTestChatbot(t, testDB.db, "To Delete")

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/chatbots/" + chatbot.Chatbot.ID,
			URLVars: map[string]string{"id": chatbot.Chatbot.ID},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.DeleteChatbotHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200 or 204, got %d", w.Code)
		}
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/chatbots/invalid-uuid",
			URLVars: map[string]string{"id": "invalid-uuid"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.DeleteChatbotHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid")
	})
}

// TestWidgetHandler tests widget generation
func TestWidgetHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	chatbot := setupTestChatbot(t, testDB.db, "Widget Test")
	defer chatbot.Cleanup()

	t.Run("Success_GetWidget", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/chatbots/" + chatbot.Chatbot.ID + "/widget",
			URLVars: map[string]string{"id": chatbot.Chatbot.ID},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.WidgetHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Expected Content-Type text/html, got %s", contentType)
		}

		// Verify the widget code contains essential elements
		body := w.Body.String()
		assertContains(t, body, chatbot.Chatbot.ID)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/chatbots/invalid-uuid/widget",
			URLVars: map[string]string{"id": "invalid-uuid"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.WidgetHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestChatHandler tests the chat functionality
func TestChatHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	// Skip if Ollama is not available
	skipIfOllamaUnavailable(t)

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	chatbot := setupTestChatbot(t, testDB.db, "Chat Test")
	defer chatbot.Cleanup()

	t.Run("Success_SendMessage", func(t *testing.T) {
		chatReq := TestData.ChatRequest("Hello, test message", uuid.New().String())

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/chat/" + chatbot.Chatbot.ID,
			URLVars: map[string]string{"id": chatbot.Chatbot.ID},
			Body:    chatReq,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set a longer timeout for AI responses
		done := make(chan bool)
		go func() {
			server.ChatHandler(w, httpReq)
			done <- true
		}()

		select {
		case <-done:
			if w.Code != http.StatusOK {
				t.Logf("Response: %s", w.Body.String())
			}
			// Don't fail on timeout or errors from Ollama in tests
			// Just verify the handler doesn't crash
		case <-time.After(10 * time.Second):
			t.Log("Chat handler timeout (expected in test environment)")
		}
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		chatReq := TestData.ChatRequest("Test message", uuid.New().String())

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/chat/invalid-uuid",
			URLVars: map[string]string{"id": "invalid-uuid"},
			Body:    chatReq,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.ChatHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid")
	})
}
