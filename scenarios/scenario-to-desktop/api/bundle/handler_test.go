package bundle

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// mockPackager implements Packager for testing
type mockPackager struct {
	result *PackageResult
	err    error
}

func (m *mockPackager) Package(appPath, manifestPath string, requestedPlatforms []string) (*PackageResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestNewHandler(t *testing.T) {
	packager := &mockPackager{}
	h := NewHandler(packager)

	if h == nil {
		t.Fatal("expected handler to be created")
	}
	if h.packager != packager {
		t.Error("expected packager to be set")
	}
}

func TestRegisterRoutes(t *testing.T) {
	h := NewHandler(&mockPackager{})
	router := mux.NewRouter()

	h.RegisterRoutes(router)

	// Verify routes are registered
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/bundle/package"},
		{http.MethodPost, "/api/v1/desktop/package"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		match := &mux.RouteMatch{}
		if !router.Match(req, match) {
			t.Errorf("expected route %s %s to be registered", tt.method, tt.path)
		}
	}
}

func TestPackageHandler(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		h := NewHandler(&mockPackager{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/package", bytes.NewBufferString("not json"))
		rr := httptest.NewRecorder()

		h.PackageHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("packager error", func(t *testing.T) {
		packager := &mockPackager{err: errors.New("manifest not found")}
		h := NewHandler(packager)

		request := PackageRequest{
			AppPath:            "/tmp/app",
			BundleManifestPath: "/tmp/manifest.json",
			Platforms:          []string{"linux"},
		}
		body, _ := json.Marshal(request)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/package", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.PackageHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("successful packaging", func(t *testing.T) {
		packager := &mockPackager{
			result: &PackageResult{
				BundleDir:       "/tmp/bundle",
				ManifestPath:    "/tmp/manifest.json",
				RuntimeBinaries: map[string]string{"linux": "/tmp/runtime"},
				CopiedArtifacts: []string{"/tmp/artifact1"},
				TotalSizeBytes:  1024,
				TotalSizeHuman:  "1 KB",
			},
		}
		h := NewHandler(packager)

		request := PackageRequest{
			AppPath:            "/tmp/app",
			BundleManifestPath: "/tmp/manifest.json",
			Platforms:          []string{"linux"},
		}
		body, _ := json.Marshal(request)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/package", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.PackageHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp PackageResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", resp.Status)
		}
		if resp.BundleDir != "/tmp/bundle" {
			t.Errorf("expected bundle_dir '/tmp/bundle', got %q", resp.BundleDir)
		}
	})

	t.Run("successful packaging with size warning", func(t *testing.T) {
		packager := &mockPackager{
			result: &PackageResult{
				BundleDir:      "/tmp/bundle",
				ManifestPath:   "/tmp/manifest.json",
				TotalSizeBytes: 500 * 1024 * 1024, // 500 MB
				TotalSizeHuman: "500 MB",
				SizeWarning: &SizeWarning{
					Level:   "warning",
					Message: "Bundle size exceeds recommended limit",
				},
			},
		}
		h := NewHandler(packager)

		request := PackageRequest{
			AppPath: "/tmp/app",
		}
		body, _ := json.Marshal(request)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/package", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.PackageHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp PackageResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.SizeWarning == nil {
			t.Error("expected size warning to be included")
		}
	})
}
