package main

import (
	"agent-inbox/domain"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:PERSIST-003] Test messages ordered by created_at
func TestGetChatMessagesOrdered(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Add messages
	addTestMessage(t, ts, chat.ID, "user", "First")
	addTestMessage(t, ts, chat.ID, "assistant", "Second")
	addTestMessage(t, ts, chat.ID, "user", "Third")

	// Get chat with messages
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat     domain.Chat      `json:"chat"`
		Messages []domain.Message `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(result.Messages))
	}

	if result.Messages[0].Content != "First" {
		t.Errorf("Expected first message 'First', got '%s'", result.Messages[0].Content)
	}
	if result.Messages[1].Content != "Second" {
		t.Errorf("Expected second message 'Second', got '%s'", result.Messages[1].Content)
	}
	if result.Messages[2].Content != "Third" {
		t.Errorf("Expected third message 'Third', got '%s'", result.Messages[2].Content)
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

// [REQ:PERSIST-004] Test delete chat cascades to messages
func TestDeleteChatCascadesToMessages(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	addTestMessage(t, ts, chat.ID, "user", "Hello")
	addTestMessage(t, ts, chat.ID, "assistant", "Hi!")

	// Delete the chat
	req := httptest.NewRequest("DELETE", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify messages are deleted (by checking DB directly)
	var count int
	err := ts.repo.DB().QueryRow("SELECT COUNT(*) FROM messages WHERE chat_id = $1", chat.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query messages: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 messages after delete, got %d", count)
	}
}

// [REQ:PERSIST-004] Test delete nonexistent chat
func TestDeleteNonexistentChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("DELETE", "/api/v1/chats/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:PERSIST-004] Test delete with invalid UUID
func TestDeleteChatInvalidID(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("DELETE", "/api/v1/chats/invalid-id", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:PERSIST-001] Test chat preview updated when message added
func TestChatPreviewUpdatedOnMessage(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Initial preview should be empty
	if chat.Preview != "" {
		t.Errorf("Expected empty preview, got '%s'", chat.Preview)
	}

	// Add a message
	addTestMessage(t, ts, chat.ID, "user", "This is the new preview")

	// Get chat and verify preview updated
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat domain.Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Chat.Preview != "This is the new preview" {
		t.Errorf("Expected preview 'This is the new preview', got '%s'", result.Chat.Preview)
	}
}

// [REQ:PERSIST-001] Test chat preview truncated for long messages
func TestChatPreviewTruncated(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	// Add a long message
	longContent := "This is a very long message that should be truncated in the preview because it exceeds the maximum length allowed for preview snippets in the inbox list view."
	addTestMessage(t, ts, chat.ID, "user", longContent)

	// Get chat and verify preview is truncated
	req := httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat domain.Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Preview should be 100 chars + "..."
	if len(result.Chat.Preview) > 103 {
		t.Errorf("Expected preview to be truncated, got length %d", len(result.Chat.Preview))
	}
}
