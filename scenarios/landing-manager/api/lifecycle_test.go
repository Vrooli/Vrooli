package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"landing-manager/handlers"
	"landing-manager/services"
)

// [REQ:TMPL-LIFECYCLE] Test scenario start endpoint
func TestHandleScenarioStart(t *testing.T) {
	t.Run("REQ:TMPL-LIFECYCLE - scenario not found", func(t *testing.T) {
		tmpRoot := t.TempDir()
		os.Setenv("VROOLI_ROOT", tmpRoot)
		defer os.Unsetenv("VROOLI_ROOT")

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", h.HandleScenarioStart).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/lifecycle/nonexistent/start", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Errorf("Expected success=false for not found, got %v", resp["success"])
		}
		if !strings.Contains(resp["message"].(string), "not found") {
			t.Errorf("Expected 'not found' in message, got %v", resp["message"])
		}
	})

	t.Run("missing scenario_id parameter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		// Use empty vars to trigger scenario_id required error
		req := httptest.NewRequest("POST", "/api/v1/lifecycle/{scenario_id}/start", nil)
		w := httptest.NewRecorder()

		// Call handler directly without router to test empty scenario_id
		h.HandleScenarioStart(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario stop endpoint
func TestHandleScenarioStop(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		// Call handler directly without router to test empty scenario_id
		req := httptest.NewRequest("POST", "/api/v1/lifecycle//stop", nil)
		w := httptest.NewRecorder()

		h.HandleScenarioStop(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario restart endpoint
func TestHandleScenarioRestart(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("POST", "/api/v1/lifecycle//restart", nil)
		w := httptest.NewRecorder()

		h.HandleScenarioRestart(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario logs endpoint
func TestHandleScenarioLogs(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("GET", "/api/v1/lifecycle//logs", nil)
		w := httptest.NewRecorder()

		h.HandleScenarioLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario status endpoint
func TestHandleScenarioStatus(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		req := httptest.NewRequest("GET", "/api/v1/lifecycle//status", nil)
		w := httptest.NewRecorder()

		h.HandleScenarioStatus(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test handleTemplateOnly for unimplemented endpoints
func TestHandleTemplateOnly(t *testing.T) {
	tests := []struct {
		name       string
		feature    string
		method     string
		path       string
		statusCode int
	}{
		{
			name:       "admin login",
			feature:    "admin authentication",
			method:     "POST",
			path:       "/api/v1/admin/login",
			statusCode: http.StatusNotImplemented,
		},
		{
			name:       "variant selection",
			feature:    "variant selection",
			method:     "GET",
			path:       "/api/v1/variants/select",
			statusCode: http.StatusNotImplemented,
		},
		{
			name:       "metrics tracking",
			feature:    "metrics tracking",
			method:     "POST",
			path:       "/api/v1/metrics/track",
			statusCode: http.StatusNotImplemented,
		},
		{
			name:       "checkout creation",
			feature:    "checkout",
			method:     "POST",
			path:       "/api/v1/checkout/create",
			statusCode: http.StatusNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.HandleTemplateOnly(tt.feature)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}

			var resp map[string]string
			json.Unmarshal(w.Body.Bytes(), &resp)

			if resp["status"] != "template_only" {
				t.Errorf("Expected status=template_only, got %v", resp["status"])
			}

			if resp["feature"] != tt.feature {
				t.Errorf("Expected feature=%s, got %v", tt.feature, resp["feature"])
			}

			if !strings.Contains(resp["message"], "generated landing scenario") {
				t.Errorf("Expected message to contain 'generated landing scenario', got %v", resp["message"])
			}
		})
	}
}

// Test resolveIssueTrackerBase with various configurations
// NOTE: resolveIssueTrackerBase is unexported in handlers package, so these tests are skipped
func TestResolveIssueTrackerBase(t *testing.T) {
	t.Run("explicit API base URL", func(t *testing.T) {
		t.Skip("resolveIssueTrackerBase is unexported - tested via integration tests")
	})

	t.Run("explicit API port", func(t *testing.T) {
		t.Skip("resolveIssueTrackerBase is unexported - tested via integration tests")
	})

	t.Run("CLI discovery fallback", func(t *testing.T) {
		t.Skip("resolveIssueTrackerBase is unexported - tested via integration tests")
	})

	t.Run("CLI discovery fails", func(t *testing.T) {
		t.Skip("resolveIssueTrackerBase is unexported - tested via integration tests")
	})

	t.Run("trailing slash trimming", func(t *testing.T) {
		t.Skip("resolveIssueTrackerBase is unexported - tested via integration tests")
	})
}
