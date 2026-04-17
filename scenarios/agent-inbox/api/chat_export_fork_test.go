package main

import (
	"agent-inbox/domain"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:PERSIST-005] Test export chat as markdown
func TestExportChatMarkdown(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat
	chat := createTestChatWithModel(t, ts, "Export Test Chat", "claude-3-5-sonnet-20241022")

	// Add a message
	body := bytes.NewBuffer([]byte(`{"role": "user", "content": "Hello, world!"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add message: %d", w.Code)
	}

	// Export as markdown
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID+"/export?format=markdown", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/markdown; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/markdown; charset=utf-8', got %s", contentType)
	}

	contentDisp := w.Header().Get("Content-Disposition")
	if contentDisp == "" {
		t.Error("Expected Content-Disposition header")
	}

	body_content := w.Body.String()
	if !bytes.Contains(w.Body.Bytes(), []byte("# Export Test Chat")) {
		t.Errorf("Expected markdown to contain chat name header, got: %s", body_content)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Hello, world!")) {
		t.Errorf("Expected markdown to contain message content, got: %s", body_content)
	}
}

// [REQ:PERSIST-005] Test export chat as JSON
func TestExportChatJSON(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat
	chat := createTestChatWithModel(t, ts, "JSON Export Test", "claude-3-5-sonnet-20241022")

	// Export as JSON
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID+"/export?format=json", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type 'application/json; charset=utf-8', got %s", contentType)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	if _, ok := result["chat"]; !ok {
		t.Error("Expected JSON to contain 'chat' field")
	}
	if _, ok := result["messages"]; !ok {
		t.Error("Expected JSON to contain 'messages' field")
	}
}

// [REQ:PERSIST-005] Test export chat as plain text
func TestExportChatTxt(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat
	chat := createTestChatWithModel(t, ts, "TXT Export Test", "claude-3-5-sonnet-20241022")

	// Export as plain text
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID+"/export?format=txt", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/plain; charset=utf-8', got %s", contentType)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("TXT Export Test")) {
		t.Errorf("Expected plain text to contain chat name, got: %s", w.Body.String())
	}
}

// [REQ:PERSIST-005] Test export with invalid format returns error
func TestExportChatInvalidFormat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat
	chat := createTestChat(t, ts)

	// Try invalid format
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID+"/export?format=invalid", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:PERSIST-005] Test export defaults to markdown
func TestExportChatDefaultFormat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat
	chat := createTestChat(t, ts)

	// Export without format param
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID+"/export", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/markdown; charset=utf-8" {
		t.Errorf("Expected default format to be markdown, got Content-Type %s", contentType)
	}
}

// [REQ:BUBBLE-007] Test fork chat creates new chat with message ancestry
func TestForkChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create chat with multiple messages
	chat := createTestChatWithModel(t, ts, "Fork Test Chat", "claude-3-5-sonnet-20241022")

	// Add 3 messages
	msg1 := addTestMessage(t, ts, chat.ID, "user", "First message")
	msg2 := addTestMessage(t, ts, chat.ID, "user", "Second message - fork point")
	_ = addTestMessage(t, ts, chat.ID, "user", "Third message after fork")

	// Fork from msg2
	body := bytes.NewBuffer([]byte(`{"message_id": "` + msg2.ID + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/fork", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var forkedChat domain.Chat
	if err := json.Unmarshal(w.Body.Bytes(), &forkedChat); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify forked chat properties
	if forkedChat.Name != "Fork Test Chat (fork)" {
		t.Errorf("Expected name 'Fork Test Chat (fork)', got %s", forkedChat.Name)
	}

	if forkedChat.ID == chat.ID {
		t.Error("Forked chat should have new ID")
	}

	// Verify forked chat has only 2 messages (msg1 and msg2)
	req = httptest.NewRequest("GET", "/api/v1/chats/"+forkedChat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to get forked chat: %d", w.Code)
	}

	var result struct {
		Chat     domain.Chat      `json:"chat"`
		Messages []domain.Message `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result.Messages) != 2 {
		t.Errorf("Expected 2 messages in forked chat, got %d", len(result.Messages))
	}

	// Verify message content
	if result.Messages[0].Content != msg1.Content {
		t.Errorf("Expected first message '%s', got '%s'", msg1.Content, result.Messages[0].Content)
	}
	if result.Messages[1].Content != msg2.Content {
		t.Errorf("Expected second message '%s', got '%s'", msg2.Content, result.Messages[1].Content)
	}
}

// [REQ:BUBBLE-007] Test fork requires message_id
func TestForkChatRequiresMessageID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	body := bytes.NewBuffer([]byte(`{}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/fork", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:BUBBLE-007] Test fork nonexistent chat
func TestForkNonexistentChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"message_id": "00000000-0000-0000-0000-000000000000"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/00000000-0000-0000-0000-000000000000/fork", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
