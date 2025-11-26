package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:TMPL-LIFECYCLE] Test scenario start endpoint
func TestHandleScenarioStart(t *testing.T) {
	t.Run("REQ:TMPL-LIFECYCLE - scenario not found", func(t *testing.T) {
		tmpRoot := t.TempDir()
		os.Setenv("VROOLI_ROOT", tmpRoot)
		defer os.Unsetenv("VROOLI_ROOT")

		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}
		srv.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", srv.handleScenarioStart).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/lifecycle/nonexistent/start", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

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
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}
		srv.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", srv.handleScenarioStart).Methods("POST")

		// Use empty vars to trigger scenario_id required error
		req := httptest.NewRequest("POST", "/api/v1/lifecycle/{scenario_id}/start", nil)
		w := httptest.NewRecorder()

		// Call handler directly without router to test empty scenario_id
		srv.handleScenarioStart(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario stop endpoint
func TestHandleScenarioStop(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}

		// Call handler directly without router to test empty scenario_id
		req := httptest.NewRequest("POST", "/api/v1/lifecycle//stop", nil)
		w := httptest.NewRecorder()

		srv.handleScenarioStop(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario restart endpoint
func TestHandleScenarioRestart(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}

		req := httptest.NewRequest("POST", "/api/v1/lifecycle//restart", nil)
		w := httptest.NewRecorder()

		srv.handleScenarioRestart(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario logs endpoint
func TestHandleScenarioLogs(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}

		req := httptest.NewRequest("GET", "/api/v1/lifecycle//logs", nil)
		w := httptest.NewRecorder()

		srv.handleScenarioLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-LIFECYCLE] Test scenario status endpoint
func TestHandleScenarioStatus(t *testing.T) {
	t.Run("missing scenario_id parameter", func(t *testing.T) {
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}

		req := httptest.NewRequest("GET", "/api/v1/lifecycle//status", nil)
		w := httptest.NewRecorder()

		srv.handleScenarioStatus(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test isScenarioNotFound helper function
func TestIsScenarioNotFound(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{"contains not found", "Error: Scenario not found", true},
		{"contains does not exist", "Scenario does not exist", true},
		{"contains No such scenario", "No such scenario", true},
		{"contains No lifecycle log found", "No lifecycle log found", true},
		{"normal output", "Scenario started successfully", false},
		{"empty output", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isScenarioNotFound(tt.output)
			if result != tt.expected {
				t.Errorf("isScenarioNotFound(%q) = %v, expected %v", tt.output, result, tt.expected)
			}
		})
	}
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
			handler := handleTemplateOnly(tt.feature)

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
func TestResolveIssueTrackerBase(t *testing.T) {
	// Save original execCommandContext and restore after tests
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()

	// Create server instance for method calls
	srv := &Server{
		router:          mux.NewRouter(),
		templateService: NewTemplateService(),
	}

	t.Run("explicit API base URL", func(t *testing.T) {
		os.Setenv("APP_ISSUE_TRACKER_API_BASE", "http://custom-host:8080/api/v1")
		defer os.Unsetenv("APP_ISSUE_TRACKER_API_BASE")

		base, err := srv.resolveIssueTrackerBase()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "http://custom-host:8080/api/v1"
		if base != expected {
			t.Errorf("Expected %s, got %s", expected, base)
		}
	})

	t.Run("explicit API port", func(t *testing.T) {
		os.Unsetenv("APP_ISSUE_TRACKER_API_BASE")
		os.Setenv("APP_ISSUE_TRACKER_API_PORT", "9999")
		defer os.Unsetenv("APP_ISSUE_TRACKER_API_PORT")

		base, err := srv.resolveIssueTrackerBase()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "http://localhost:9999/api/v1"
		if base != expected {
			t.Errorf("Expected %s, got %s", expected, base)
		}
	})

	t.Run("CLI discovery fallback", func(t *testing.T) {
		os.Unsetenv("APP_ISSUE_TRACKER_API_BASE")
		os.Unsetenv("APP_ISSUE_TRACKER_API_PORT")

		// Mock vrooli CLI command returning port
		execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("echo", "8765")
		}

		base, err := srv.resolveIssueTrackerBase()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "http://localhost:8765/api/v1"
		if base != expected {
			t.Errorf("Expected %s, got %s", expected, base)
		}
	})

	t.Run("CLI discovery fails", func(t *testing.T) {
		os.Unsetenv("APP_ISSUE_TRACKER_API_BASE")
		os.Unsetenv("APP_ISSUE_TRACKER_API_PORT")

		// Mock CLI command that fails
		execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "exit 1")
		}

		_, err := srv.resolveIssueTrackerBase()
		if err == nil {
			t.Error("Expected error when CLI discovery fails, got nil")
		}
	})

	t.Run("trailing slash trimming", func(t *testing.T) {
		os.Setenv("APP_ISSUE_TRACKER_API_BASE", "http://localhost:8080/api/v1/")
		defer os.Unsetenv("APP_ISSUE_TRACKER_API_BASE")

		base, err := srv.resolveIssueTrackerBase()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "http://localhost:8080/api/v1"
		if base != expected {
			t.Errorf("Expected trailing slash removed: %s, got %s", expected, base)
		}
	})
}
