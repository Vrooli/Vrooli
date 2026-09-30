package build

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

// mockService implements Service for testing
type mockService struct {
	performDesktopBuildCalls  int
	performScenarioBuildCalls int
}

func (m *mockService) PerformDesktopBuild(buildID string, req *BuildRequest) {
	m.performDesktopBuildCalls++
}

func (m *mockService) PerformScenarioDesktopBuild(buildID, scenarioName, desktopPath string, platforms []string, clean bool) {
	m.performScenarioBuildCalls++
}

func (m *mockService) BuildPlatform(buildID, scenarioName, desktopPath, distPath, platform string) {
}

func TestNewHandler(t *testing.T) {
	service := &mockService{}
	store := NewStore()
	h := NewHandler(service, store)

	if h == nil {
		t.Fatal("expected handler to be created")
	}
	if h.service != service {
		t.Error("service not set correctly")
	}
	if h.store != store {
		t.Error("store not set correctly")
	}
}

func TestNewHandlerWithOptions(t *testing.T) {
	service := &mockService{}
	store := NewStore()
	logger := slog.Default()

	h := NewHandler(service, store,
		WithScenarioRoot("/tmp/scenarios"),
		WithHandlerLogger(logger),
	)

	if h.scenarioRoot != "/tmp/scenarios" {
		t.Errorf("expected scenarioRoot '/tmp/scenarios', got %q", h.scenarioRoot)
	}
	if h.logger != logger {
		t.Error("expected custom logger to be set")
	}
}

func TestRegisterRoutes(t *testing.T) {
	service := &mockService{}
	store := NewStore()
	h := NewHandler(service, store)

	router := mux.NewRouter()
	h.RegisterRoutes(router)

	// Only test routes that still exist after pipeline migration
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/desktop/download/test-scenario/linux"},
		{http.MethodPost, "/api/v1/desktop/webhook/build-complete"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Route exists if we don't get a 404 "page not found" from the router
			if strings.Contains(rr.Body.String(), "404 page not found") {
				t.Errorf("route not registered: %s %s", r.method, r.path)
			}
		})
	}
}

func TestHandleDownload(t *testing.T) {
	service := &mockService{}
	store := NewStore()
	tmpDir := t.TempDir()
	h := NewHandler(service, store,
		WithScenarioRoot(tmpDir),
		WithHandlerLogger(slog.Default()),
	)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/desktop/download/{scenario_name}/{platform}", h.HandleDownload).Methods("GET")

	t.Run("invalid platform", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/download/test-scenario/invalid", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("package not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/download/test-scenario/linux", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("successful download", func(t *testing.T) {
		// Create a test scenario with dist-electron directory
		distPath := filepath.Join(tmpDir, "test-scenario", "platforms", "electron", "dist-electron")
		if err := os.MkdirAll(distPath, 0o755); err != nil {
			t.Fatalf("failed to create dist path: %v", err)
		}
		appImageFile := filepath.Join(distPath, "TestApp.AppImage")
		if err := os.WriteFile(appImageFile, []byte("test content"), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/download/test-scenario/linux", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleBuildCompleteWebhook(t *testing.T) {
	service := &mockService{}
	store := NewStore()
	h := NewHandler(service, store, WithHandlerLogger(slog.Default()))

	t.Run("missing build_id header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/webhook/build-complete", strings.NewReader(`{}`))
		rr := httptest.NewRecorder()

		h.HandleBuildCompleteWebhook(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/webhook/build-complete", strings.NewReader("{invalid}"))
		req.Header.Set("X-Build-ID", "build-123")
		rr := httptest.NewRecorder()

		h.HandleBuildCompleteWebhook(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		store.Save(&Status{BuildID: "build-123", Status: "building"})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/webhook/build-complete", strings.NewReader(`{"status": "completed"}`))
		req.Header.Set("X-Build-ID", "build-123")
		rr := httptest.NewRecorder()

		h.HandleBuildCompleteWebhook(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		// Verify status was updated
		status, _ := store.Get("build-123")
		if status.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", status.Status)
		}
	})
}

func TestDetectPackageContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"app.msi", "application/x-msi"},
		{"app.pkg", "application/vnd.apple.installer+xml"},
		{"app.exe", "application/x-msdownload"},
		{"app.dmg", "application/x-apple-diskimage"},
		{"app.AppImage", "application/x-executable"},
		{"app.deb", "application/vnd.debian.binary-package"},
		{"app.zip", "application/zip"},
		{"app.unknown", "application/octet-stream"},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			result := detectPackageContentType(tc.filename)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// Ensure json package is used (for response decoding in tests)
var _ = json.Unmarshal
