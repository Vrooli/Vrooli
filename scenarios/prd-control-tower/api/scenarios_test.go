package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:PCT-CATALOG-ENUMERATE] Catalog enumerates all scenarios and resources with PRD status
func TestHandleScenarioExistence(t *testing.T) {
	// Save original VROOLI_ROOT and restore after test
	origRoot := os.Getenv("VROOLI_ROOT")
	defer func() {
		if origRoot != "" {
			os.Setenv("VROOLI_ROOT", origRoot)
		} else {
			os.Unsetenv("VROOLI_ROOT")
		}
	}()

	// Create temporary vrooli root
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)

	// Create a test scenario directory
	scenariosDir := filepath.Join(tmpRoot, "scenarios")
	testScenarioDir := filepath.Join(scenariosDir, "test-scenario")
	if err := os.MkdirAll(testScenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create test scenario dir: %v", err)
	}

	// Create a file (not directory) to test the IsDir check
	fileScenarioPath := filepath.Join(scenariosDir, "file-not-dir")
	if err := os.WriteFile(fileScenarioPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file scenario: %v", err)
	}

	tests := []struct {
		name           string
		scenarioName   string
		expectedStatus int
		expectedExists bool
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "existing scenario",
			scenarioName:   "test-scenario",
			expectedStatus: http.StatusOK,
			expectedExists: true,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ScenarioExistenceResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if !response.Exists {
					t.Error("Expected scenario to exist")
				}
				if response.Path == "" {
					t.Error("Expected path to be set")
				}
				if response.LastModified == "" {
					t.Error("Expected last_modified to be set")
				}
			},
		},
		{
			name:           "nonexistent scenario",
			scenarioName:   "nonexistent",
			expectedStatus: http.StatusOK,
			expectedExists: false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ScenarioExistenceResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Exists {
					t.Error("Expected scenario to not exist")
				}
			},
		},
		{
			name:           "file instead of directory",
			scenarioName:   "file-not-dir",
			expectedStatus: http.StatusOK,
			expectedExists: false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ScenarioExistenceResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Exists {
					t.Error("Expected file to not be treated as scenario directory")
				}
			},
		},
		{
			name:           "empty scenario name",
			scenarioName:   "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if !contains(w.Body.String(), "required") {
					t.Error("Expected error message about required scenario name")
				}
			},
		},
		{
			name:           "scenario name with path traversal",
			scenarioName:   "../test",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if !contains(w.Body.String(), "invalid") {
					t.Error("Expected error message about invalid scenario name")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/"+tt.scenarioName+"/exists", nil)
			req = mux.SetURLVars(req, map[string]string{"name": tt.scenarioName})
			w := httptest.NewRecorder()

			handleScenarioExistence(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleScenarioExistence() status = %d, want %d, body: %s",
					w.Code, tt.expectedStatus, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
