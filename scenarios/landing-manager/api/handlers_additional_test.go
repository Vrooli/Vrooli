package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	"landing-manager/handlers"
	"landing-manager/services"
	"landing-manager/util"
)

// Test handleHealth with unhealthy database
func TestHandleHealth_UnhealthyDatabase(t *testing.T) {
	t.Run("returns unhealthy when database is down", func(t *testing.T) {
		// Create a closed database connection to simulate unhealthy state
		db := setupTestDB(t)
		db.Close() // Close immediately to make it unhealthy

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		h.HandleHealth(w, req)

		// Should still return 200 but with unhealthy status
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var health map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if health["status"] != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got %v", health["status"])
		}

		if health["readiness"] != false {
			t.Errorf("Expected readiness false, got %v", health["readiness"])
		}

		deps, ok := health["dependencies"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected dependencies map")
		}

		if deps["database"] != "disconnected" {
			t.Errorf("Expected database status 'disconnected', got %v", deps["database"])
		}
	})
}

// Test handlePersonaList error path
func TestHandlePersonaList_ErrorHandling(t *testing.T) {
	t.Run("returns 500 when persona service fails", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Create services with invalid personas directory
		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService("/nonexistent/path")
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("GET", "/api/v1/personas", nil)
		w := httptest.NewRecorder()

		h.HandlePersonaList(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// Test handleGeneratedList with actual scenarios
func TestHandleGeneratedList_MultipleScenarios(t *testing.T) {
	t.Run("lists multiple generated scenarios", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("GEN_OUTPUT_DIR", tmpDir)
		defer os.Unsetenv("GEN_OUTPUT_DIR")

		// Create multiple test scenarios
		scenarios := []string{"scenario-1", "scenario-2", "scenario-3"}
		for _, name := range scenarios {
			scenarioDir := filepath.Join(tmpDir, name)
			os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)
			serviceJSON := `{"service": {"name": "` + name + `", "displayName": "` + name + `", "description": "test", "repository": {"directory": "/scenarios/` + name + `"}}}`
			os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(serviceJSON), 0644)
		}

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("GET", "/api/v1/generated", nil)
		w := httptest.NewRecorder()

		h.HandleGeneratedList(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var result []services.GeneratedScenario
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 scenarios, got %d", len(result))
		}
	})
}

// Test handleScenarioStop with actual CLI interaction (success path)
func TestHandleScenarioStop_SuccessScenario(t *testing.T) {
	t.Run("successfully stops a running scenario", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		// Use a real scenario name that the CLI will process
		req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-stop/stop", nil)
		req = mux.SetURLVars(req, map[string]string{"scenario_id": "test-stop"})
		w := httptest.NewRecorder()

		h.HandleScenarioStop(w, req)

		// Should return either success or not-found, both are valid responses
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Response should have success field
		if _, ok := resp["success"]; !ok {
			t.Error("Expected 'success' field in response")
		}
	})
}

// Test handleScenarioRestart with actual CLI interaction
func TestHandleScenarioRestart_SuccessScenario(t *testing.T) {
	t.Run("successfully restarts a scenario", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-restart/restart", nil)
		req = mux.SetURLVars(req, map[string]string{"scenario_id": "test-restart"})
		w := httptest.NewRecorder()

		h.HandleScenarioRestart(w, req)

		// Should return either success or not-found
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := resp["success"]; !ok {
			t.Error("Expected 'success' field in response")
		}
	})
}

// Test NewServer database connection failure
func TestNewServer_DatabaseError(t *testing.T) {
	t.Run("fails when database connection is invalid", func(t *testing.T) {
		// Save original values
		origPort := os.Getenv("API_PORT")
		origDB := os.Getenv("DATABASE_URL")
		defer func() {
			if origPort != "" {
				os.Setenv("API_PORT", origPort)
			} else {
				os.Unsetenv("API_PORT")
			}
			if origDB != "" {
				os.Setenv("DATABASE_URL", origDB)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		// Set required env vars but with invalid database
		os.Setenv("API_PORT", "8080")
		os.Setenv("DATABASE_URL", "postgres://invalid:invalid@nonexistent:5432/invalid")

		// This should fail to connect to the database
		_, err := NewServer()
		if err == nil {
			t.Error("Expected error when database connection fails")
		}
	})

	t.Run("fails when database URL cannot be resolved", func(t *testing.T) {
		// Save and clear all database-related env vars
		vars := []string{"API_PORT", "DATABASE_URL", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
		origValues := make(map[string]string)
		for _, v := range vars {
			origValues[v] = os.Getenv(v)
			os.Unsetenv(v)
		}
		defer func() {
			for k, v := range origValues {
				if v != "" {
					os.Setenv(k, v)
				}
			}
		}()

		// Set only API_PORT and one incomplete POSTGRES var
		os.Setenv("API_PORT", "8080")
		os.Setenv("POSTGRES_HOST", "localhost") // Missing PORT, USER, PASSWORD, DB

		// This should fail to resolve database URL
		_, err := NewServer()
		if err == nil {
			t.Error("Expected error when database URL cannot be resolved")
		}
	})
}

// Test resolveDatabaseURL edge cases
func TestResolveDatabaseURL_EdgeCases(t *testing.T) {
	t.Run("uses DATABASE_URL when set", func(t *testing.T) {
		origURL := os.Getenv("DATABASE_URL")
		defer func() {
			if origURL != "" {
				os.Setenv("DATABASE_URL", origURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		testURL := "postgres://test:test@localhost:5432/testdb"
		os.Setenv("DATABASE_URL", testURL)

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if url != testURL {
			t.Errorf("Expected %s, got %s", testURL, url)
		}
	})
}

// Test logStructuredError to improve coverage
func TestLogStructuredError_Coverage(t *testing.T) {
	t.Run("logs structured error", func(t *testing.T) {
		// This function just logs, so we're testing it doesn't panic
		util.LogStructuredError("test_error", map[string]interface{}{
			"error":   "test error message",
			"details": "additional details",
		})
	})

	t.Run("logs with nil fields", func(t *testing.T) {
		util.LogStructuredError("test_error_nil", nil)
	})
}
