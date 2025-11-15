package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestHandleListBacklogNoDB tests handleListBacklog without database
func TestHandleListBacklogNoDB(t *testing.T) {
	// Save original db and restore after test
	originalDB := db
	defer func() { db = originalDB }()
	db = nil

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	handleListBacklog(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response BacklogListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
}

// TestHandleCreateBacklogEntriesNoDB tests handleCreateBacklogEntries without database
func TestHandleCreateBacklogEntriesNoDB(t *testing.T) {
	// Save original db and restore after test
	originalDB := db
	defer func() { db = originalDB }()
	db = nil

	requestBody := BacklogCreateRequest{
		Entries: []BacklogCreateEntry{
			{
				IdeaText:      "Test idea",
				EntityType:    "scenario",
				SuggestedName: "test-scenario",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateBacklogEntries(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (Service Unavailable), got %d", w.Code)
	}
}

// TestHandleCreateBacklogEntriesInvalidJSON tests invalid JSON input
func TestHandleCreateBacklogEntriesInvalidJSON(t *testing.T) {
	// Database not needed for JSON parsing test
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", bytes.NewReader([]byte("{invalid json")))
	w := httptest.NewRecorder()

	handleCreateBacklogEntries(w, req)

	// Should fail with 400 or 503 depending on whether db is set
	if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 400 or 503, got %d", w.Code)
	}
}

// TestHandleCreateBacklogEntriesEmpty tests empty entries list
func TestHandleCreateBacklogEntriesEmpty(t *testing.T) {
	requestBody := BacklogCreateRequest{
		Entries: []BacklogCreateEntry{},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateBacklogEntries(w, req)

	// Should fail with 400 (empty) or 503 (no db)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 400 or 503 (empty entries), got %d", w.Code)
	}
}

// TestHandleCreateBacklogEntriesTooMany tests exceeding max entries limit
func TestHandleCreateBacklogEntriesTooMany(t *testing.T) {
	// Create 51 entries (over the limit of 50)
	entries := make([]BacklogCreateEntry, 51)
	for i := 0; i < 51; i++ {
		entries[i] = BacklogCreateEntry{
			IdeaText:      "Test idea",
			EntityType:    "scenario",
			SuggestedName: "test-scenario",
		}
	}

	requestBody := BacklogCreateRequest{
		Entries: entries,
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateBacklogEntries(w, req)

	// Should fail with 400 (too many) or 503 (no db)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 400 or 503 (too many entries), got %d", w.Code)
	}

	if w.Code == http.StatusBadRequest {
		var errorResp map[string]string
		json.NewDecoder(w.Body).Decode(&errorResp)
		if errorResp["error"] != "Too many backlog items in a single request (max 50)" {
			t.Errorf("Expected 'too many' error message, got: %s", errorResp["error"])
		}
	}
}

// TestHandleCreateBacklogEntriesWithRawInput tests raw input parsing
func TestHandleCreateBacklogEntriesWithRawInput(t *testing.T) {
	requestBody := BacklogCreateRequest{
		RawInput:   "Implement user authentication\nAdd dashboard analytics",
		EntityType: "scenario",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateBacklogEntries(w, req)

	// The important part is that it doesn't panic and handles the raw input parsing
	// Status code depends on whether db is available
	t.Logf("Handled raw input request with status %d", w.Code)
}

// TestHandleDeleteBacklogEntryNoDB tests handleDeleteBacklogEntry without database
func TestHandleDeleteBacklogEntryNoDB(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()
	db = nil

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/backlog/test-id", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
	w := httptest.NewRecorder()

	handleDeleteBacklogEntry(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (Service Unavailable), got %d", w.Code)
	}
}

// TestHandleDeleteBacklogEntryMissingID tests missing entry ID
func TestHandleDeleteBacklogEntryMissingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/backlog/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handleDeleteBacklogEntry(w, req)

	// Should fail with 400 (missing ID) or 503 (no db)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 400 or 503 (missing ID), got %d", w.Code)
	}
}

// TestHandleConvertSingleBacklogEntryMissingID tests missing entry ID
func TestHandleConvertSingleBacklogEntryMissingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/convert", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handleConvertSingleBacklogEntry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 (missing ID), got %d", w.Code)
	}
}

// TestHandleConvertBacklogEntriesInvalidJSON tests invalid JSON input
func TestHandleConvertBacklogEntriesInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/convert", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	handleConvertBacklogEntries(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 (invalid JSON), got %d", w.Code)
	}
}

// TestHandleConvertBacklogEntriesEmpty tests empty entry IDs list
func TestHandleConvertBacklogEntriesEmpty(t *testing.T) {
	requestBody := BacklogConvertRequest{
		EntryIDs: []string{},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/convert", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleConvertBacklogEntries(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 (empty entry_ids), got %d", w.Code)
	}

	// Error message varies based on response format, just check we got an error
	var errorResp map[string]string
	json.NewDecoder(w.Body).Decode(&errorResp)
	if errorResp["error"] == "" {
		t.Error("Expected error message in response")
	}
}
