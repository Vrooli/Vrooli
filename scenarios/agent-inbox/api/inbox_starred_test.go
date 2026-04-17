package main

import (
	"agent-inbox/domain"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	var starredChats []domain.Chat
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

	var chats []domain.Chat
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
