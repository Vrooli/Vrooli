package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// TestHandleListDraftsNoDB tests draft listing when database is not available
func TestHandleListDraftsNoDB(t *testing.T) {
	// Ensure db is nil
	originalDB := db
	db = nil
	defer func() { db = originalDB }()

	req := httptest.NewRequest("GET", "/api/v1/drafts", nil)
	w := httptest.NewRecorder()

	handleListDrafts(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleListDrafts() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleCreateDraftInvalidJSON tests draft creation with invalid JSON
func TestHandleCreateDraftInvalidJSON(t *testing.T) {
	// Ensure db is nil to test early return
	originalDB := db
	db = nil
	defer func() { db = originalDB }()

	invalidJSON := []byte(`{"entity_type": "scenario", "entity_name": `)
	req := httptest.NewRequest("POST", "/api/v1/drafts", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDraft(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleCreateDraft() status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleAIGenerateSectionNoDB tests AI generation when database is not available
func TestHandleAIGenerateSectionNoDB(t *testing.T) {
	// Ensure db is nil
	originalDB := db
	db = nil
	defer func() { db = originalDB }()

	reqBody := map[string]string{
		"section": "features",
		"context": "test context",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/drafts/test-id/ai/generate-section", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}/ai/generate-section", handleAIGenerateSection)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleAIGenerateSection() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleGetDraftNoDB tests the get draft endpoint without database
func TestHandleGetDraftNoDB(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/drafts/test-id", nil)
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}", handleGetDraft)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleGetDraft() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleUpdateDraftNoDB tests the update draft endpoint without database
func TestHandleUpdateDraftNoDB(t *testing.T) {
	reqBody := map[string]any{
		"content": "# Updated PRD Content\n\nTest update",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/v1/drafts/test-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}", handleUpdateDraft)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleUpdateDraft() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleUpdateDraftInvalidJSON tests the update draft endpoint with invalid JSON
func TestHandleUpdateDraftInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/v1/drafts/test-id", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}", handleUpdateDraft)
	router.ServeHTTP(w, req)

	// Without DB, should return 503 before reaching JSON parsing
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleUpdateDraft() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleDeleteDraftNoDB tests the delete draft endpoint without database
func TestHandleDeleteDraftNoDB(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/v1/drafts/test-id", nil)
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}", handleDeleteDraft)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleDeleteDraft() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandlePublishDraftNoDB tests the publish draft endpoint without database
func TestHandlePublishDraftNoDB(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/drafts/test-id/publish", nil)
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}/publish", handlePublishDraft)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handlePublishDraft() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleValidateDraftNoDB tests the validate draft endpoint without database
func TestHandleValidateDraftNoDB(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/drafts/test-id/validate", nil)
	w := httptest.NewRecorder()

	// Set up router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/drafts/{id}/validate", handleValidateDraft)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleValidateDraft() without DB status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestSaveDraftToFile tests the saveDraftToFile utility function
func TestSaveDraftToFile(t *testing.T) {
	// Change to parent directory to test with relative paths
	originalDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	os.Chdir(apiDir)
	defer os.Chdir(originalDir)

	tests := []struct {
		name        string
		entityType  string
		entityName  string
		content     string
		expectError bool
	}{
		{
			name:        "save scenario draft",
			entityType:  "scenario",
			entityName:  "test-scenario",
			content:     "# Test PRD\n\nTest content",
			expectError: false,
		},
		{
			name:        "save resource draft",
			entityType:  "resource",
			entityName:  "test-resource",
			content:     "# Resource PRD\n\nResource content",
			expectError: false,
		},
		{
			name:        "save with empty content",
			entityType:  "scenario",
			entityName:  "empty-test",
			content:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := saveDraftToFile(tt.entityType, tt.entityName, tt.content)
			if tt.expectError && err == nil {
				t.Errorf("saveDraftToFile() expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("saveDraftToFile() unexpected error: %v", err)
			}

			if !tt.expectError {
				// Verify file was created at relative path
				expectedPath := filepath.Join("../data/prd-drafts", tt.entityType, tt.entityName+".md")

				if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
					t.Errorf("saveDraftToFile() did not create file at %s", expectedPath)
				}

				// Verify content
				content, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Errorf("Failed to read saved file: %v", err)
				}
				if string(content) != tt.content {
					t.Errorf("saveDraftToFile() content = %q, want %q", string(content), tt.content)
				}
			}
		})
	}
}
