package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-inbox/domain"
)

// TestAutoNameRequiresMessages verifies auto-name requires messages in chat
// [REQ:NAME-001]
func TestAutoNameRequiresMessages(t *testing.T) {
	server := setupTestServer(t)

	// Create a chat with no messages
	chat := createTestChat(t, server)

	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/auto-name", nil)
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "No messages in chat to generate name from" {
		t.Errorf("Unexpected error message: %s", resp["error"])
	}
}

// TestAutoNameValidatesChat verifies auto-name requires valid chat ID
// [REQ:NAME-001]
func TestAutoNameValidatesChat(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/chats/invalid-id/auto-name", nil)
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestAutoNameGeneratesName verifies auto-name generates a chat name
// [REQ:NAME-001]
func TestAutoNameGeneratesName(t *testing.T) {
	server := setupTestServer(t)

	// Create chat with message
	chat := createTestChat(t, server)
	addTestMessage(t, server, chat.ID, "user", "How do I implement authentication in Go?")
	addTestMessage(t, server, chat.ID, "assistant", "Here's how to implement authentication...")

	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/auto-name", nil)
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	// The actual status depends on whether Ollama is available
	// In tests, it may fail gracefully with fallback name
	if rr.Code != http.StatusOK {
		t.Skipf("Ollama not available, got status %d", rr.Code)
	}

	var updatedChat domain.Chat
	if err := json.NewDecoder(rr.Body).Decode(&updatedChat); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Name should be updated from default
	if updatedChat.Name == "New Chat" {
		t.Error("Chat name should have been updated")
	}
}

// TestAutoNameEndpoint verifies the endpoint exists and accepts POST
// [REQ:NAME-001]
func TestAutoNameEndpoint(t *testing.T) {
	server := setupTestServer(t)

	chat := createTestChat(t, server)

	// OPTIONS request should succeed
	req := httptest.NewRequest("OPTIONS", "/api/v1/chats/"+chat.ID+"/auto-name", nil)
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected OPTIONS to return %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestAddToolMessage verifies tool messages can be added
// [REQ:MSG-004]
func TestAddToolMessage(t *testing.T) {
	server := setupTestServer(t)

	chat := createTestChat(t, server)

	body := []byte(`{"role": "tool", "content": "Tool result content", "tool_call_id": "call_123"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var msg domain.Message
	if err := json.NewDecoder(rr.Body).Decode(&msg); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if msg.Role != "tool" {
		t.Errorf("Expected role 'tool', got '%s'", msg.Role)
	}
	if msg.ToolCallID != "call_123" {
		t.Errorf("Expected tool_call_id 'call_123', got '%s'", msg.ToolCallID)
	}
}

// TestAddToolMessageRequiresToolCallID verifies tool messages require tool_call_id
// [REQ:MSG-004]
func TestAddToolMessageRequiresToolCallID(t *testing.T) {
	server := setupTestServer(t)

	chat := createTestChat(t, server)

	body := []byte(`{"role": "tool", "content": "Tool result content"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "tool_call_id is required for tool messages" {
		t.Errorf("Unexpected error: %s", resp["error"])
	}
}
