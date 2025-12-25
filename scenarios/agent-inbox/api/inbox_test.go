package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

// [REQ:INBOX-LIST-001] Test chat list shows preview snippet
func TestListChatsWithPreview(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	addTestMessage(t, ts, chat.ID, "user", "Hello, how are you today?")

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
		t.Fatalf("Expected 1 chat, got %d", len(chats))
	}

	// Preview should contain the message content
	if chats[0].Preview != "Hello, how are you today?" {
		t.Errorf("Expected preview 'Hello, how are you today?', got '%s'", chats[0].Preview)
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

// [REQ:INBOX-LIST-003] Test toggle read without explicit value (toggle behavior)
func TestToggleReadWithoutValue(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// First toggle (false -> true)
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/read", bytes.NewBuffer([]byte(`{}`)))
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
		t.Error("Expected chat to be marked as read after toggle")
	}

	// Second toggle (true -> false)
	req = httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/read", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result["is_read"] {
		t.Error("Expected chat to be marked as unread after second toggle")
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

// [REQ:INBOX-LIST-004] Test archived chats are excluded from main list
func TestArchivedChatsExcludedFromMainList(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Archive the chat
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat.ID+"/archive", bytes.NewBuffer([]byte(`{"value": true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Verify chat is NOT in main list (archived=false is default)
	req = httptest.NewRequest("GET", "/api/v1/chats", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var chats []Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, c := range chats {
		if c.ID == chat.ID {
			t.Error("Archived chat should not appear in main list")
		}
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

// [REQ:INBOX-LIST-006] Test starred chats appear at top of list
func TestStarredChatsAppearFirst(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create two chats
	chat1 := createTestChat(t, ts)
	chat2 := createTestChat(t, ts)

	// Star chat1
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chat1.ID+"/star", bytes.NewBuffer([]byte(`{"value": true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Get all chats - starred should be first
	req = httptest.NewRequest("GET", "/api/v1/chats", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var chats []Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(chats) < 2 {
		t.Fatalf("Expected at least 2 chats, got %d", len(chats))
	}

	// The starred chat should be first
	if chats[0].ID != chat1.ID {
		t.Errorf("Expected starred chat %s to be first, got %s", chat1.ID, chats[0].ID)
	}

	_ = chat2 // used to create a second chat
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
