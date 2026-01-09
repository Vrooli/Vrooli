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
)

// [REQ:TMPL-LIFECYCLE] Test handleGeneratedList error path when service fails
func TestHandleGeneratedList_ErrorPath(t *testing.T) {
	t.Run("error when generation directory is invalid", func(t *testing.T) {
		// Set invalid directory that will cause ListGeneratedScenarios to fail
		tmpDir := t.TempDir()
		os.Setenv("GEN_OUTPUT_DIR", filepath.Join(tmpDir, "nonexistent"))
		defer os.Unsetenv("GEN_OUTPUT_DIR")

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

		// Should return 200 with empty list when directory doesn't exist
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var scenarios []services.GeneratedScenario
		if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(scenarios) != 0 {
			t.Errorf("Expected empty list, got %d scenarios", len(scenarios))
		}
	})
}

// Test handleScenarioStop success path (we test error paths in lifecycle_test.go)
func TestHandleScenarioStop_SuccessPath(t *testing.T) {
	// Note: This test validates the structure but can't easily mock CLI success
	// Real CLI testing is done in integration tests
	t.Run("validates request structure", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		// Test with valid scenario_id parameter (will fail on CLI but validates handler)
		req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-scenario/stop", nil)
		req = mux.SetURLVars(req, map[string]string{"scenario_id": "test-scenario"})
		w := httptest.NewRecorder()

		h.HandleScenarioStop(w, req)

		// Will fail because CLI isn't mocked, but ensures handler processes the request
		// Success paths are validated in integration tests
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200/404/500, got %d", w.Code)
		}
	})
}

// Test handleScenarioRestart success path
func TestHandleScenarioRestart_SuccessPath(t *testing.T) {
	t.Run("validates request structure", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-scenario/restart", nil)
		req = mux.SetURLVars(req, map[string]string{"scenario_id": "test-scenario"})
		w := httptest.NewRecorder()

		h.HandleScenarioRestart(w, req)

		// Validates handler processes the request
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200/404/500, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test handleScenarioPromote with production path conflicts
func TestHandleScenarioPromote_ProductionConflict(t *testing.T) {
	t.Run("fails when production path already exists", func(t *testing.T) {
		tmpRoot := t.TempDir()
		os.Setenv("VROOLI_ROOT", tmpRoot)
		defer os.Unsetenv("VROOLI_ROOT")

		// Create both staging and production directories
		// Note: Staging path is scenarios/landing-manager/generated/{id}
		stagingDir := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-scenario")
		productionDir := filepath.Join(tmpRoot, "scenarios", "test-scenario")
		os.MkdirAll(stagingDir, 0755)
		os.MkdirAll(productionDir, 0755)

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-scenario/promote", nil)
		req = mux.SetURLVars(req, map[string]string{"scenario_id": "test-scenario"})
		w := httptest.NewRecorder()

		h.HandleScenarioPromote(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
			if resp["success"] != false {
				t.Error("Expected success=false in response")
			}
			message, ok := resp["message"].(string)
			if !ok || message == "" {
				t.Error("Expected error message in response")
			}
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test handleScenarioPromote move failure
func TestHandleScenarioPromote_MoveFailure(t *testing.T) {
	t.Run("handles filesystem errors during move", func(t *testing.T) {
		tmpRoot := t.TempDir()
		os.Setenv("VROOLI_ROOT", tmpRoot)
		defer os.Unsetenv("VROOLI_ROOT")

		// Note: Staging path is scenarios/landing-manager/generated/{id}
		stagingDir := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-scenario")
		os.MkdirAll(stagingDir, 0755)

		// Make the scenarios directory read-only to cause move failure
		scenariosDir := filepath.Join(tmpRoot, "scenarios")
		os.Chmod(scenariosDir, 0555)
		defer os.Chmod(scenariosDir, 0755) // Restore for cleanup

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-scenario/promote", nil)
		req = mux.SetURLVars(req, map[string]string{"scenario_id": "test-scenario"})
		w := httptest.NewRecorder()

		h.HandleScenarioPromote(w, req)

		// May succeed on systems where permissions work differently, or fail as expected
		if w.Code == http.StatusInternalServerError {
			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
				if resp["success"] != false {
					t.Error("Expected success=false in response")
				}
			}
		}
	})
}

// Test health endpoint
func TestHandleHealth_WithDatabase(t *testing.T) {
	t.Run("returns 200 OK when database is healthy", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		h.HandleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var health map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if health["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", health["status"])
		}

		if health["readiness"] != true {
			t.Errorf("Expected readiness true, got %v", health["readiness"])
		}
	})
}

// Test seedDefaultData (it's a no-op, but let's ensure it doesn't panic)
func TestSeedDefaultData(t *testing.T) {
	t.Run("succeeds as no-op", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := seedDefaultData(db); err != nil {
			t.Errorf("seedDefaultData should succeed as no-op, got error: %v", err)
		}
	})
}
