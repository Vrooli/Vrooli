package system

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// mockBuildStore implements BuildStore for testing.
type mockBuildStore struct {
	statuses map[string]*BuildStatus
}

func (m *mockBuildStore) Snapshot() map[string]*BuildStatus {
	if m.statuses == nil {
		return map[string]*BuildStatus{}
	}
	return m.statuses
}

func TestNewHandler(t *testing.T) {
	wineService := NewWineService(slog.Default())
	buildStore := &mockBuildStore{}
	h := NewHandler(wineService, buildStore, "/tmp/templates")

	if h == nil {
		t.Fatal("expected handler to be created")
	}
	if h.wineService != wineService {
		t.Error("wineService not set correctly")
	}
	if h.builds != buildStore {
		t.Error("builds not set correctly")
	}
	if h.templateDir != "/tmp/templates" {
		t.Errorf("templateDir = %q, want %q", h.templateDir, "/tmp/templates")
	}
}

func TestStatusHandler(t *testing.T) {
	tests := []struct {
		name           string
		builds         map[string]*BuildStatus
		expectBuilding int
		expectComplete int
		expectFailed   int
	}{
		{
			name:           "no builds",
			builds:         nil,
			expectBuilding: 0,
			expectComplete: 0,
			expectFailed:   0,
		},
		{
			name: "mixed statuses",
			builds: map[string]*BuildStatus{
				"build1": {Status: "building"},
				"build2": {Status: "ready"},
				"build3": {Status: "failed"},
				"build4": {Status: "ready"},
			},
			expectBuilding: 1,
			expectComplete: 2,
			expectFailed:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wineService := NewWineService(slog.Default())
			buildStore := &mockBuildStore{statuses: tc.builds}
			h := NewHandler(wineService, buildStore, "/tmp/templates")

			req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
			rr := httptest.NewRecorder()

			h.StatusHandler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}

			var resp StatusResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Service["status"] != "running" {
				t.Errorf("expected service status 'running', got %v", resp.Service["status"])
			}

			stats := resp.Statistics
			if stats["active_builds"].(float64) != float64(tc.expectBuilding) {
				t.Errorf("expected %d active builds, got %v", tc.expectBuilding, stats["active_builds"])
			}
			if stats["completed_builds"].(float64) != float64(tc.expectComplete) {
				t.Errorf("expected %d completed builds, got %v", tc.expectComplete, stats["completed_builds"])
			}
			if stats["failed_builds"].(float64) != float64(tc.expectFailed) {
				t.Errorf("expected %d failed builds, got %v", tc.expectFailed, stats["failed_builds"])
			}
		})
	}
}

func TestStatusHandlerNilBuildStore(t *testing.T) {
	wineService := NewWineService(slog.Default())
	h := NewHandler(wineService, nil, "/tmp/templates")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rr := httptest.NewRecorder()

	h.StatusHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp StatusResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	stats := resp.Statistics
	if stats["total_builds"].(float64) != 0 {
		t.Errorf("expected 0 total builds, got %v", stats["total_builds"])
	}
}

func TestListTemplatesHandler(t *testing.T) {
	wineService := NewWineService(slog.Default())
	h := NewHandler(wineService, nil, "/tmp/templates")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	rr := httptest.NewRecorder()

	h.ListTemplatesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp TemplatesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Count != len(resp.Templates) {
		t.Errorf("count mismatch: count=%d, len(templates)=%d", resp.Count, len(resp.Templates))
	}

	// Verify expected templates exist
	expectedTypes := []string{"universal", "advanced", "multi_window", "kiosk"}
	for _, expectedType := range expectedTypes {
		found := false
		for _, tmpl := range resp.Templates {
			if tmpl.Type == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template type %q not found", expectedType)
		}
	}
}

func TestGetTemplateHandler(t *testing.T) {
	// Create a temp template directory with a test template
	tmpDir := t.TempDir()
	advancedDir := filepath.Join(tmpDir, "advanced")
	if err := os.MkdirAll(advancedDir, 0o755); err != nil {
		t.Fatalf("failed to create advanced dir: %v", err)
	}

	templateContent := `{"name": "Universal App", "type": "universal"}`
	if err := os.WriteFile(filepath.Join(advancedDir, "universal-app.json"), []byte(templateContent), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	wineService := NewWineService(slog.Default())
	h := NewHandler(wineService, nil, tmpDir)

	tests := []struct {
		name         string
		templateType string
		expectStatus int
	}{
		{"valid universal", "universal", http.StatusOK},
		{"valid basic alias", "basic", http.StatusOK},
		{"invalid type", "invalid", http.StatusBadRequest},
		{"missing template", "advanced", http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use actual gorilla/mux router to properly set path variables
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/templates/{type}", h.GetTemplateHandler).Methods("GET")

			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+tc.templateType, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.expectStatus {
				t.Errorf("expected status %d, got %d", tc.expectStatus, rr.Code)
			}
		})
	}
}

func TestInstallWineHandler(t *testing.T) {
	wineService := NewWineService(slog.Default())
	h := NewHandler(wineService, nil, "/tmp/templates")

	tests := []struct {
		name         string
		body         string
		expectStatus int
	}{
		{
			name:         "valid flatpak method",
			body:         `{"method": "flatpak"}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "valid skip method",
			body:         `{"method": "skip"}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "invalid method",
			body:         `{"method": "invalid"}`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid json",
			body:         `{invalid}`,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/system/wine/install", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.InstallWineHandler(rr, req)

			if rr.Code != tc.expectStatus {
				t.Errorf("expected status %d, got %d: %s", tc.expectStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestGetWineInstallStatusHandler(t *testing.T) {
	wineService := NewWineService(slog.Default())
	h := NewHandler(wineService, nil, "/tmp/templates")

	// First, start an installation
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/wine/install", strings.NewReader(`{"method": "skip"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.InstallWineHandler(rr, req)

	var installResp WineInstallResponse
	if err := json.NewDecoder(rr.Body).Decode(&installResp); err != nil {
		t.Fatalf("failed to decode install response: %v", err)
	}

	// Use actual gorilla/mux router to properly set path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/system/wine/install/status/{install_id}", h.GetWineInstallStatusHandler).Methods("GET")

	t.Run("existing install", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/wine/install/status/"+installResp.InstallID, nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("non-existent install", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/wine/install/status/nonexistent", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})
}

func TestCheckWineHandler(t *testing.T) {
	wineService := NewWineService(slog.Default())
	h := NewHandler(wineService, nil, "/tmp/templates")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/wine/check", nil)
	rr := httptest.NewRecorder()

	h.CheckWineHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp WineCheckResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// The response should have platform information
	if resp.Platform == "" {
		t.Error("expected platform to be set")
	}
}
