package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestServer wraps Server for testing
type TestServer struct {
	*Server
}

// setupTestServer creates a test server with an in-memory approach
// For integration tests, use a real test database
func setupTestServer(t *testing.T) *TestServer {
	// Skip if DATABASE_URL is not set (unit tests without DB)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	srv := &Server{
		config: &Config{Port: "0", DatabaseURL: dbURL},
		db:     db,
		router: mux.NewRouter(),
	}

	if err := srv.initSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	srv.setupRoutes()

	return &TestServer{srv}
}

func (ts *TestServer) cleanup(t *testing.T) {
	// Clean up test data
	_, _ = ts.db.Exec("DELETE FROM messages")
	_, _ = ts.db.Exec("DELETE FROM chat_labels")
	_, _ = ts.db.Exec("DELETE FROM chats")
	_, _ = ts.db.Exec("DELETE FROM labels")
	ts.db.Close()
}

// [REQ:INBOX-LIST-001] Test display chat list
func TestListChats(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create a test chat
	chat := createTestChat(t, ts)

	// Test list endpoint
	req := httptest.NewRequest("GET", "/api/v1/chats", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var chats []Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(chats) != 1 {
		t.Errorf("Expected 1 chat, got %d", len(chats))
	}

	if chats[0].ID != chat.ID {
		t.Errorf("Expected chat ID %s, got %s", chat.ID, chats[0].ID)
	}
}

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

// [REQ:INBOX-LIST-002] [REQ:INBOX-LIST-003] Test read/unread indicators and toggle
func TestToggleRead(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Initially chat should be unread (is_read = false)
	if chat.IsRead {
		t.Error("New chat should be unread")
	}

	// Toggle read status
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/read", bytes.NewBuffer([]byte(`{"value": true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !result["is_read"] {
		t.Error("Expected chat to be marked as read")
	}
}

// [REQ:INBOX-LIST-004] Test archive chats
func TestToggleArchive(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Archive the chat
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/archive", bytes.NewBuffer([]byte(`{"value": true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify chat is archived
	req = httptest.NewRequest("GET", "/api/v1/chats?archived=true", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var archivedChats []Chat
	if err := json.Unmarshal(w.Body.Bytes(), &archivedChats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	found := false
	for _, c := range archivedChats {
		if c.ID == chat.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find archived chat")
	}
}

// [REQ:INBOX-LIST-006] Test starred chats
func TestToggleStar(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Star the chat
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/star", bytes.NewBuffer([]byte(`{"value": true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify starred filter works
	req = httptest.NewRequest("GET", "/api/v1/chats?starred=true", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var starredChats []Chat
	if err := json.Unmarshal(w.Body.Bytes(), &starredChats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	found := false
	for _, c := range starredChats {
		if c.ID == chat.ID && c.IsStarred {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find starred chat")
	}
}

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

	var msg Message
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
		Chat     Chat      `json:"chat"`
		Messages []Message `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result.Messages))
	}
}

// [REQ:PERSIST-004] Test delete chat
func TestDeleteChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Delete the chat
	req := httptest.NewRequest("DELETE", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify chat is deleted
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:LABEL-001] [REQ:LABEL-002] Test create and list labels
func TestLabels(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create a label
	body := bytes.NewBuffer([]byte(`{"name": "Important", "color": "#ef4444"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var label Label
	if err := json.Unmarshal(w.Body.Bytes(), &label); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if label.Name != "Important" {
		t.Errorf("Expected name 'Important', got %s", label.Name)
	}

	// List labels
	req = httptest.NewRequest("GET", "/api/v1/labels", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var labels []Label
	if err := json.Unmarshal(w.Body.Bytes(), &labels); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}
}

// [REQ:LABEL-003] Test assign label to chat
func TestAssignLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	label := createTestLabel(t, ts, "Work", "#3b82f6")

	// Assign label
	req := httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify label is assigned
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	found := false
	for _, labelID := range result.Chat.LabelIDs {
		if labelID == label.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected label to be assigned to chat")
	}
}

// Test health endpoint
func TestHealth(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}
}

// Helper functions

func createTestChat(t *testing.T, ts *TestServer) Chat {
	body := bytes.NewBuffer([]byte(`{"name": "Test Chat"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test chat: %s", w.Body.String())
	}

	var chat Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse chat: %v", err)
	}

	return chat
}

func addTestMessage(t *testing.T, ts *TestServer, chatID, role, content string) Message {
	body := bytes.NewBuffer([]byte(`{"role": "` + role + `", "content": "` + content + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chatID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add test message: %s", w.Body.String())
	}

	var msg Message
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	return msg
}

func createTestLabel(t *testing.T, ts *TestServer, name, color string) Label {
	body := bytes.NewBuffer([]byte(`{"name": "` + name + `", "color": "` + color + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test label: %s", w.Body.String())
	}

	var label Label
	if err := json.Unmarshal(w.Body.Bytes(), &label); err != nil {
		t.Fatalf("Failed to parse label: %v", err)
	}

	return label
}
