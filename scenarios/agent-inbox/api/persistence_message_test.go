package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-inbox/domain"
)

// [REQ:PERSIST-001] [REQ:PERSIST-002] Test storing chat and messages
func TestAddMessage(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Add a message
	body := bytes.NewBuffer([]byte(`{"role": "user", "content": "Hello, world!"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var msg domain.Message
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", msg.Content)
	}

	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got %s", msg.Role)
	}
}

// [REQ:PERSIST-002] Test message role validation
func TestAddMessageRoleValidation(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Test invalid role
	body := bytes.NewBuffer([]byte(`{"role": "invalid", "content": "Hello"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:PERSIST-002] Test valid roles
func TestAddMessageValidRoles(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	validRoles := []string{"user", "assistant", "system"}

	for _, role := range validRoles {
		body := bytes.NewBuffer([]byte(`{"role": "` + role + `", "content": "Test message"}`))
		req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201 for role '%s', got %d", role, w.Code)
		}
	}
}

// [REQ:PERSIST-002] Test message content required
func TestAddMessageContentRequired(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Missing content
	body := bytes.NewBuffer([]byte(`{"role": "user"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:PERSIST-002] Test message role required
func TestAddMessageRoleRequired(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Missing role
	body := bytes.NewBuffer([]byte(`{"content": "Hello"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:PERSIST-002] Test adding message to nonexistent chat
func TestAddMessageToNonexistentChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"role": "user", "content": "Hello"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/00000000-0000-0000-0000-000000000000/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:PERSIST-002] Test message with token count
func TestAddMessageWithTokenCount(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	body := bytes.NewBuffer([]byte(`{"role": "assistant", "content": "Hello!", "token_count": 5}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var msg domain.Message
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if msg.TokenCount != 5 {
		t.Errorf("Expected token_count 5, got %d", msg.TokenCount)
	}
}

// [REQ:PERSIST-003] Test load chat history
func TestGetChatWithMessages(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Add messages
	addTestMessage(t, ts, chat.ID, "user", "Hello")
	addTestMessage(t, ts, chat.ID, "assistant", "Hi there!")

	// Get chat with messages
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result struct {
		Chat     domain.Chat      `json:"chat"`
		Messages []domain.Message `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result.Messages))
	}
}
