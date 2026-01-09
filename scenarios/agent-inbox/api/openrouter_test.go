package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"agent-inbox/integrations"
)

// [REQ:BUBBLE-005] Test list available models
func TestListModels(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var models []integrations.ModelInfo
	if err := json.Unmarshal(w.Body.Bytes(), &models); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model")
	}
}

// [REQ:BUBBLE-005] Test models have required fields
func TestListModelsFields(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var models []integrations.ModelInfo
	if err := json.Unmarshal(w.Body.Bytes(), &models); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, model := range models {
		if model.ID == "" {
			t.Error("Model ID should not be empty")
		}
		if model.Name == "" {
			t.Error("Model Name should not be empty")
		}
	}
}

// [REQ:BUBBLE-005] Test models include Claude
func TestListModelsIncludesClaude(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var models []integrations.ModelInfo
	if err := json.Unmarshal(w.Body.Bytes(), &models); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	foundClaude := false
	for _, model := range models {
		if model.ID == "anthropic/claude-3.5-sonnet" || model.ID == "anthropic/claude-3-haiku" {
			foundClaude = true
			break
		}
	}

	if !foundClaude {
		t.Error("Expected to find at least one Claude model")
	}
}

// [REQ:BUBBLE-004] Test chat completion requires API key
func TestChatCompleteRequiresAPIKey(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Ensure no API key is set
	originalKey := os.Getenv("OPENROUTER_API_KEY")
	os.Unsetenv("OPENROUTER_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENROUTER_API_KEY", originalKey)
		}
	}()

	chat := createTestChat(t, ts)
	addTestMessage(t, ts, chat.ID, "user", "Hello")

	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/complete", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result["error"] != "OpenRouter API key not configured" {
		t.Errorf("Expected error about API key, got: %s", result["error"])
	}
}

// [REQ:BUBBLE-004] Test chat completion requires messages
func TestChatCompleteRequiresMessages(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Set a dummy API key for this test
	os.Setenv("OPENROUTER_API_KEY", "test-key")
	defer os.Unsetenv("OPENROUTER_API_KEY")

	chat := createTestChat(t, ts)
	// Don't add any messages

	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/complete", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result["error"] != "No messages in chat" {
		t.Errorf("Expected error about no messages, got: %s", result["error"])
	}
}

// [REQ:BUBBLE-004] Test chat completion validates chat exists
func TestChatCompleteValidatesChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	os.Setenv("OPENROUTER_API_KEY", "test-key")
	defer os.Unsetenv("OPENROUTER_API_KEY")

	req := httptest.NewRequest("POST", "/api/v1/chats/00000000-0000-0000-0000-000000000000/complete", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:BUBBLE-003] Test streaming parameter accepted
func TestChatCompleteAcceptsStreamParam(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// This test just verifies the endpoint accepts the stream parameter
	// Actual streaming requires a real API key and external service
	os.Setenv("OPENROUTER_API_KEY", "test-key")
	defer os.Unsetenv("OPENROUTER_API_KEY")

	chat := createTestChat(t, ts)
	addTestMessage(t, ts, chat.ID, "user", "Hello")

	// The request should be accepted (stream=true)
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/complete?stream=true", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Will fail to connect to OpenRouter but won't be a client error
	if w.Code == http.StatusBadRequest {
		t.Error("Should accept stream parameter without error")
	}
}

// [REQ:BUBBLE-004] Test uses chat model for completion
func TestChatCompleteUsesCorrectModel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat with specific model
	chat := createTestChatWithModel(t, ts, "GPT Test", "openai/gpt-4o")

	if chat.Model != "openai/gpt-4o" {
		t.Errorf("Expected model 'openai/gpt-4o', got %s", chat.Model)
	}

	// The model is stored and should be used for completions
	// (Full integration test would verify this against real API)
}

// Test OPTIONS for models endpoint (CORS)
func TestModelsOptionsRequest(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("OPTIONS", "/api/v1/models", nil)
	req.Header.Set("Origin", "http://localhost:35000")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test OPTIONS for complete endpoint (CORS)
func TestCompleteOptionsRequest(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	req := httptest.NewRequest("OPTIONS", "/api/v1/chats/"+chat.ID+"/complete", nil)
	req.Header.Set("Origin", "http://localhost:35000")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
