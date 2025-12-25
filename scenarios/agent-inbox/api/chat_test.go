package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:BUBBLE-001] Test create new chat
func TestCreateChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Test Chat", "model": "claude-3-5-sonnet-20241022"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var chat Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if chat.Name != "Test Chat" {
		t.Errorf("Expected name 'Test Chat', got %s", chat.Name)
	}

	if chat.ID == "" {
		t.Error("Expected chat ID to be set")
	}
}

// [REQ:BUBBLE-001] Test create chat with default values
func TestCreateChatWithDefaults(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var chat Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if chat.Name != "New Chat" {
		t.Errorf("Expected default name 'New Chat', got %s", chat.Name)
	}

	if chat.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected default model 'claude-3-5-sonnet-20241022', got %s", chat.Model)
	}

	if chat.ViewMode != "bubble" {
		t.Errorf("Expected default view_mode 'bubble', got %s", chat.ViewMode)
	}
}

// [REQ:BUBBLE-001] Test create chat with terminal view mode
func TestCreateChatWithTerminalView(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Terminal Chat", "view_mode": "terminal"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var chat Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if chat.ViewMode != "terminal" {
		t.Errorf("Expected view_mode 'terminal', got %s", chat.ViewMode)
	}
}

// [REQ:BUBBLE-001] Test create chat with invalid view mode
func TestCreateChatWithInvalidViewMode(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Invalid Chat", "view_mode": "invalid"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:BUBBLE-006] [REQ:NAME-003] Test update chat (rename, model change)
func TestUpdateChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Update name
	body := bytes.NewBuffer([]byte(`{"name": "Updated Name"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/chats/"+chat.ID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated Chat
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", updated.Name)
	}
}

// [REQ:BUBBLE-006] Test switch model mid-conversation
func TestUpdateChatModel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChatWithModel(t, ts, "Model Test", "openai/gpt-4o")

	if chat.Model != "openai/gpt-4o" {
		t.Errorf("Expected initial model 'openai/gpt-4o', got %s", chat.Model)
	}

	// Switch model
	body := bytes.NewBuffer([]byte(`{"model": "anthropic/claude-3.5-sonnet"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/chats/"+chat.ID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated Chat
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if updated.Model != "anthropic/claude-3.5-sonnet" {
		t.Errorf("Expected model 'anthropic/claude-3.5-sonnet', got %s", updated.Model)
	}
}

// Test update chat with no fields (should fail)
func TestUpdateChatNoFields(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	body := bytes.NewBuffer([]byte(`{}`))
	req := httptest.NewRequest("PATCH", "/api/v1/chats/"+chat.ID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test update nonexistent chat
func TestUpdateNonexistentChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Test"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/chats/00000000-0000-0000-0000-000000000000", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test update with invalid UUID
func TestUpdateChatInvalidID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Test"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/chats/invalid-id", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test get single chat
func TestGetChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result struct {
		Chat     Chat      `json:"chat"`
		Messages []Message `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Chat.ID != chat.ID {
		t.Errorf("Expected chat ID %s, got %s", chat.ID, result.Chat.ID)
	}
}

// Test get nonexistent chat
func TestGetNonexistentChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("GET", "/api/v1/chats/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
