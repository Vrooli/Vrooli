package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// TestHandleScenarioStart_MissingScenarioID tests error handling for empty scenario_id
func TestHandleScenarioStart_MissingScenarioID(t *testing.T) {
	server := &Server{}
	// Use empty string as scenario_id to simulate missing/empty path parameter
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lifecycle/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": ""})
	w := httptest.NewRecorder()

	server.handleScenarioStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty scenario_id, got %d", w.Code)
	}
}

// TestHandleScenarioStop_MissingScenarioID tests error handling for empty scenario_id
func TestHandleScenarioStop_MissingScenarioID(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lifecycle/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": ""})
	w := httptest.NewRecorder()

	server.handleScenarioStop(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty scenario_id, got %d", w.Code)
	}
}

// TestHandleScenarioRestart_MissingScenarioID tests error handling for empty scenario_id
func TestHandleScenarioRestart_MissingScenarioID(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lifecycle/restart", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": ""})
	w := httptest.NewRecorder()

	server.handleScenarioRestart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty scenario_id, got %d", w.Code)
	}
}

// TestHandleScenarioStatus_MissingScenarioID tests error handling for empty scenario_id
func TestHandleScenarioStatus_MissingScenarioID(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lifecycle/status", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": ""})
	w := httptest.NewRecorder()

	server.handleScenarioStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty scenario_id, got %d", w.Code)
	}
}

// TestHandleScenarioLogs_MissingScenarioID tests error handling for empty scenario_id
func TestHandleScenarioLogs_MissingScenarioID(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lifecycle/logs", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": ""})
	w := httptest.NewRecorder()

	server.handleScenarioLogs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty scenario_id, got %d", w.Code)
	}
}

// TestHandleScenarioStart_NonexistentScenario tests behavior when scenario doesn't exist
func TestHandleScenarioStart_NonexistentScenario(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lifecycle/nonexistent-scenario-999/start", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", server.handleScenarioStart).Methods(http.MethodPost)
	router.ServeHTTP(w, req)

	// Should return 404 when scenario not found in staging or production
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent scenario, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}

	message, ok := response["message"].(string)
	if !ok || message == "" {
		t.Error("Expected error message in response")
	}
}

// TestHandleScenarioStop_NonexistentScenario tests behavior when scenario doesn't exist
// Note: vrooli scenario stop is idempotent - it succeeds even if scenario doesn't exist or isn't running
func TestHandleScenarioStop_NonexistentScenario(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lifecycle/nonexistent-scenario-999/stop", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/stop", server.handleScenarioStop).Methods(http.MethodPost)
	router.ServeHTTP(w, req)

	// Stop is idempotent - should succeed even if scenario doesn't exist
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for stop (idempotent), got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Errorf("Expected success=true for idempotent stop, got %v", response["success"])
	}
}

// TestHandleScenarioRestart_NonexistentScenario tests behavior when scenario doesn't exist
func TestHandleScenarioRestart_NonexistentScenario(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lifecycle/nonexistent-scenario-999/restart", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/restart", server.handleScenarioRestart).Methods(http.MethodPost)
	router.ServeHTTP(w, req)

	// Should return 404 when scenario not found
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent scenario, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}
}

// TestListGeneratedScenarios_ErrorPaths tests error handling in ListGeneratedScenarios
func TestListGeneratedScenarios_ErrorPaths(t *testing.T) {
	t.Run("nonexistent_generated_directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		ts := &TemplateService{
			templatesDir: tmpDir, // valid templates dir
			// generatedDir will be derived from executable path but won't exist
		}

		// Should return empty list if generated directory doesn't exist
		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("Expected no error for missing generated dir, got: %v", err)
		}
		if scenarios == nil {
			t.Error("Expected empty list, got nil")
		}
	})

	t.Run("scenario_without_service_json", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create template directory with test template
		templatesDir := tmpDir + "/templates"
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates dir: %v", err)
		}

		// Override GEN_OUTPUT_DIR to use tmpDir
		t.Setenv("GEN_OUTPUT_DIR", tmpDir+"/generated")

		ts := NewTemplateService()
		ts.templatesDir = templatesDir

		// Create a scenario directory WITHOUT .vrooli/service.json but with a dummy file
		scenarioDir := tmpDir + "/generated/incomplete-scenario"
		if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}
		// Create a dummy file so the directory isn't empty
		dummyFile := scenarioDir + "/dummy.txt"
		if err := os.WriteFile(dummyFile, []byte("test"), 0o644); err != nil {
			t.Fatalf("Failed to write dummy file: %v", err)
		}

		// ListGeneratedScenarios returns all directories, even without service.json (defensive coding)
		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// The incomplete scenario should be included (defensive behavior)
		found := false
		for _, s := range scenarios {
			if s.ScenarioID == "incomplete-scenario" {
				found = true
				// Should have default values since service.json is missing
				if s.Name != "incomplete-scenario" {
					t.Errorf("Expected Name to default to slug, got: %s", s.Name)
				}
				if s.Status != "present" {
					t.Errorf("Expected Status 'present', got: %s", s.Status)
				}
			}
		}
		if !found {
			t.Error("Expected incomplete scenario to be included in list")
		}
	})

	t.Run("scenario_with_invalid_service_json", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatesDir := tmpDir + "/templates"
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates dir: %v", err)
		}

		t.Setenv("GEN_OUTPUT_DIR", tmpDir+"/generated")

		ts := NewTemplateService()
		ts.templatesDir = templatesDir

		// Create scenario with invalid JSON in service.json
		scenarioDir := tmpDir + "/generated/invalid-json"
		vrooliDir := scenarioDir + "/.vrooli"
		if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
			t.Fatalf("Failed to create .vrooli dir: %v", err)
		}

		// Write invalid JSON
		invalidJSON := []byte(`{"service": {"displayName": "Test", invalid json}`)
		if err := os.WriteFile(vrooliDir+"/service.json", invalidJSON, 0o644); err != nil {
			t.Fatalf("Failed to write invalid service.json: %v", err)
		}

		// Should handle gracefully - use scenario ID as name
		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("Expected no error with invalid JSON, got: %v", err)
		}

		found := false
		for _, s := range scenarios {
			if s.ScenarioID == "invalid-json" {
				found = true
				// Should fall back to slug since JSON is invalid
				if s.Name != "invalid-json" {
					t.Errorf("Expected Name to fall back to slug, got: %s", s.Name)
				}
			}
		}
		if !found {
			t.Error("Expected scenario with invalid JSON to be included")
		}
	})

	t.Run("non_directory_entries_ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatesDir := tmpDir + "/templates"
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates dir: %v", err)
		}

		t.Setenv("GEN_OUTPUT_DIR", tmpDir+"/generated")

		ts := NewTemplateService()
		ts.templatesDir = templatesDir

		// Create generated directory with files and directories
		genDir := tmpDir + "/generated"
		if err := os.MkdirAll(genDir, 0o755); err != nil {
			t.Fatalf("Failed to create generated dir: %v", err)
		}

		// Create a file (should be ignored)
		if err := os.WriteFile(genDir+"/file.txt", []byte("test"), 0o644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Create a valid directory
		validDir := genDir + "/valid-scenario"
		if err := os.MkdirAll(validDir, 0o755); err != nil {
			t.Fatalf("Failed to create valid scenario dir: %v", err)
		}

		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should only include the directory, not the file
		if len(scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(scenarios))
		}

		if len(scenarios) > 0 && scenarios[0].ScenarioID != "valid-scenario" {
			t.Errorf("Expected 'valid-scenario', got '%s'", scenarios[0].ScenarioID)
		}
	})

	t.Run("scenario_with_invalid_template_json", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatesDir := tmpDir + "/templates"
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates dir: %v", err)
		}

		t.Setenv("GEN_OUTPUT_DIR", tmpDir+"/generated")

		ts := NewTemplateService()
		ts.templatesDir = templatesDir

		// Create scenario with invalid template.json
		scenarioDir := tmpDir + "/generated/invalid-template"
		vrooliDir := scenarioDir + "/.vrooli"
		if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
			t.Fatalf("Failed to create .vrooli dir: %v", err)
		}

		// Write invalid JSON for template provenance
		invalidJSON := []byte(`{"template_id": 123, invalid}`)
		if err := os.WriteFile(vrooliDir+"/template.json", invalidJSON, 0o644); err != nil {
			t.Fatalf("Failed to write invalid template.json: %v", err)
		}

		// Should handle gracefully - scenario without template info
		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("Expected no error with invalid template.json, got: %v", err)
		}

		found := false
		for _, s := range scenarios {
			if s.ScenarioID == "invalid-template" {
				found = true
				// Template fields should be empty since JSON is invalid
				if s.TemplateID != "" {
					t.Errorf("Expected empty TemplateID, got: %s", s.TemplateID)
				}
			}
		}
		if !found {
			t.Error("Expected scenario with invalid template.json to be included")
		}
	})
}

// TestGenerateScenario_EdgeCases tests edge cases in generation
func TestGenerateScenario_EdgeCases(t *testing.T) {
	t.Run("empty_options_map", func(t *testing.T) {
		ts, _ := setupGenerationTest(t)

		// Passing nil options should work
		result, err := ts.GenerateScenario("test-template", "Test", "test", nil)
		if err != nil {
			t.Errorf("Expected no error with nil options, got: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
	})

	t.Run("options_with_unexpected_types", func(t *testing.T) {
		ts, _ := setupGenerationTest(t)

		// Options with unexpected values should be handled gracefully
		opts := map[string]interface{}{
			"unexpected_key": "unexpected_value",
			"numeric_value":  12345,
		}

		result, err := ts.GenerateScenario("test-template", "Test", "test", opts)
		if err != nil {
			t.Errorf("Expected no error with unexpected options, got: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
	})
}
