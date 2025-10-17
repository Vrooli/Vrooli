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

// TestHandleHealth tests the health check endpoint
func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
		return
	}

	if response.Service != "prd-control-tower-api" {
		t.Errorf("handleHealth() service = %q, want %q", response.Service, "prd-control-tower-api")
	}

	// Status may be "degraded" when db is nil, which is expected in unit tests
	if response.Status != "healthy" && response.Status != "degraded" {
		t.Errorf("handleHealth() status = %q, want %q or %q", response.Status, "healthy", "degraded")
	}

	// Verify response structure contains dependencies
	if response.Dependencies == nil {
		t.Errorf("handleHealth() dependencies should not be nil")
	}
}

// TestHandleGetCatalog tests the catalog endpoint
func TestHandleGetCatalog(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()
	t.Setenv("VROOLI_ROOT", tmpDir)

	// Create test scenarios directory structure
	scenariosDir := filepath.Join(tmpDir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0755); err != nil {
		t.Fatalf("Failed to create scenarios dir: %v", err)
	}

	// Create a test scenario with PRD
	testScenarioDir := filepath.Join(scenariosDir, "test-scenario")
	if err := os.MkdirAll(testScenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create test scenario dir: %v", err)
	}
	prdPath := filepath.Join(testScenarioDir, "PRD.md")
	prdContent := `# Test Scenario

This is a test scenario for unit testing.

## Features
- Feature 1
- Feature 2`
	if err := os.WriteFile(prdPath, []byte(prdContent), 0644); err != nil {
		t.Fatalf("Failed to create test PRD: %v", err)
	}

	// Create resources directory
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("Failed to create resources dir: %v", err)
	}

	// Create a test resource without PRD
	testResourceDir := filepath.Join(resourcesDir, "test-resource")
	if err := os.MkdirAll(testResourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test resource dir: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
	w := httptest.NewRecorder()

	handleGetCatalog(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleGetCatalog() status = %d, want %d", w.Code, http.StatusOK)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	var response CatalogResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode catalog response: %v", err)
		return
	}

	// Should find 1 scenario and 1 resource
	if response.Total != 2 {
		t.Errorf("handleGetCatalog() total = %d, want 2", response.Total)
	}

	// Verify scenario entry
	foundScenario := false
	foundResource := false
	for _, entry := range response.Entries {
		if entry.Type == "scenario" && entry.Name == "test-scenario" {
			foundScenario = true
			if !entry.HasPRD {
				t.Errorf("Scenario entry should have HasPRD = true")
			}
			if entry.Description != "This is a test scenario for unit testing." {
				t.Errorf("Description = %q, want %q", entry.Description, "This is a test scenario for unit testing.")
			}
		}
		if entry.Type == "resource" && entry.Name == "test-resource" {
			foundResource = true
			if entry.HasPRD {
				t.Errorf("Resource entry should have HasPRD = false")
			}
		}
	}

	if !foundScenario {
		t.Errorf("Did not find test-scenario in catalog")
	}
	if !foundResource {
		t.Errorf("Did not find test-resource in catalog")
	}
}

// TestHandleGetPublishedPRD tests retrieving a published PRD
func TestHandleGetPublishedPRD(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()
	t.Setenv("VROOLI_ROOT", tmpDir)

	// Create test scenario with PRD
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario")
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	prdContent := "# Test PRD\n\nThis is test content."
	prdPath := filepath.Join(scenarioDir, "PRD.md")
	if err := os.WriteFile(prdPath, []byte(prdContent), 0644); err != nil {
		t.Fatalf("Failed to create PRD: %v", err)
	}

	tests := []struct {
		name       string
		entityType string
		entityName string
		wantStatus int
	}{
		{
			name:       "existing PRD",
			entityType: "scenario",
			entityName: "test-scenario",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nonexistent PRD",
			entityType: "scenario",
			entityName: "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/catalog/"+tt.entityType+"/"+tt.entityName, nil)
			w := httptest.NewRecorder()

			// Set up router to extract path variables
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/catalog/{type}/{name}", handleGetPublishedPRD)
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("handleGetPublishedPRD() status = %d, want %d", w.Code, tt.wantStatus)
				return
			}

			if tt.wantStatus == http.StatusOK {
				var response PublishedPRDResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode PRD response: %v", err)
					return
				}

				if response.Content != prdContent {
					t.Errorf("handleGetPublishedPRD() content = %q, want %q", response.Content, prdContent)
				}

				if response.Name != tt.entityName {
					t.Errorf("handleGetPublishedPRD() name = %q, want %q", response.Name, tt.entityName)
				}
			}
		})
	}
}

// TestHandleGetCatalogWithInvalidEnvironment tests error handling
func TestHandleGetCatalogWithInvalidEnvironment(t *testing.T) {
	// Unset both VROOLI_ROOT and HOME to trigger error
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("HOME", "")

	req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
	w := httptest.NewRecorder()

	handleGetCatalog(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("handleGetCatalog() with invalid env status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	// Create a test handler that CORS middleware will wrap
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with CORS middleware
	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.HandleFunc("/test", testHandler)

	tests := []struct {
		name          string
		origin        string
		method        string
		expectAllowed bool
	}{
		{
			name:          "localhost:36300 allowed",
			origin:        "http://localhost:36300",
			method:        "GET",
			expectAllowed: true,
		},
		{
			name:          "127.0.0.1:36300 allowed",
			origin:        "http://127.0.0.1:36300",
			method:        "GET",
			expectAllowed: true,
		},
		{
			name:          "preflight request",
			origin:        "http://localhost:36300",
			method:        "OPTIONS",
			expectAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Check CORS headers
			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllowed {
				if allowOrigin != tt.origin && allowOrigin == "" {
					t.Errorf("Expected Access-Control-Allow-Origin to be set for %s, got %q", tt.origin, allowOrigin)
				}
			}

			// Preflight requests should return 200
			if tt.method == "OPTIONS" && w.Code != http.StatusOK {
				t.Errorf("Preflight request status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}

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
	reqBody := map[string]interface{}{
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
