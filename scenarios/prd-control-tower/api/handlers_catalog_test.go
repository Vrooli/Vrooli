package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

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

				if response.ContentHTML == "" {
					t.Errorf("handleGetPublishedPRD() content_html should not be empty")
				} else if !strings.Contains(response.ContentHTML, "<p>This is test content.</p>") {
					t.Errorf("handleGetPublishedPRD() content_html missing expected paragraph, got %q", response.ContentHTML)
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
