package preflight

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

func TestNewHandler(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	if h == nil {
		t.Fatal("expected handler to be created")
	}
}

func TestBundleHandler(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/preflight", strings.NewReader("{invalid}"))
		rr := httptest.NewRecorder()

		h.BundleHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("missing manifest path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/preflight", strings.NewReader(`{}`))
		rr := httptest.NewRecorder()

		h.BundleHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("manifest file not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/preflight", strings.NewReader(`{"bundle_manifest_path": "/nonexistent/manifest.json"}`))
		rr := httptest.NewRecorder()

		h.BundleHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestStartHandler(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/preflight/start", strings.NewReader("{invalid}"))
		rr := httptest.NewRecorder()

		h.StartHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/preflight/start", strings.NewReader(`{"bundle_manifest_path": "/tmp/test"}`))
		rr := httptest.NewRecorder()

		h.StartHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp JobStartResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.JobID == "" {
			t.Error("expected job_id to be set")
		}
	})
}

func TestStatusHandler(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	t.Run("missing job_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/status", nil)
		rr := httptest.NewRecorder()

		h.StatusHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("job not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/status?job_id=nonexistent", nil)
		rr := httptest.NewRecorder()

		h.StatusHandler(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		job := service.CreateJob()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/status?job_id="+job.ID, nil)
		rr := httptest.NewRecorder()

		h.StatusHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp JobStatusResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.JobID != job.ID {
			t.Errorf("expected job_id %q, got %q", job.ID, resp.JobID)
		}
	})

	t.Run("job with steps", func(t *testing.T) {
		jobStore := NewInMemoryJobStore()
		service := NewService(WithJobStore(jobStore))
		h := NewHandler(service)

		job := jobStore.Create()
		// Create() adds 5 default steps: validation, runtime, secrets, services, diagnostics
		// SetStep updates existing steps or adds new ones
		jobStore.SetStep(job.ID, "validation", "pass", "manifest valid")
		jobStore.SetStep(job.ID, "runtime", "running", "starting")
		jobStore.SetStep(job.ID, "custom_step", "pending", "waiting")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/status?job_id="+job.ID, nil)
		rr := httptest.NewRecorder()

		h.StatusHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp JobStatusResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		// 5 defaults + 1 custom = 6 total (validation and runtime are overwritten)
		if len(resp.Steps) != 6 {
			t.Errorf("expected 6 steps, got %d", len(resp.Steps))
		}
	})
}

func TestHealthHandler(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	t.Run("missing both params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/health", nil)
		rr := httptest.NewRecorder()

		h.HealthHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("missing session_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/health?service_id=api", nil)
		rr := httptest.NewRecorder()

		h.HealthHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("missing service_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/health?session_id=test", nil)
		rr := httptest.NewRecorder()

		h.HealthHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("session not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/preflight/health?session_id=nonexistent&service_id=api", nil)
		rr := httptest.NewRecorder()

		h.HealthHandler(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})
}

func TestManifestHandler(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/bundle-manifest", strings.NewReader("{invalid}"))
		rr := httptest.NewRecorder()

		h.ManifestHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/bundle-manifest", strings.NewReader(`{"bundle_manifest_path": ""}`))
		rr := httptest.NewRecorder()

		h.ManifestHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("whitespace only path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/bundle-manifest", strings.NewReader(`{"bundle_manifest_path": "   "}`))
		rr := httptest.NewRecorder()

		h.ManifestHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/bundle-manifest", strings.NewReader(`{"bundle_manifest_path": "/nonexistent/manifest.json"}`))
		rr := httptest.NewRecorder()

		h.ManifestHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("invalid manifest JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "manifest.json")
		if err := os.WriteFile(manifestPath, []byte("{invalid json"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		body := `{"bundle_manifest_path": "` + manifestPath + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/bundle-manifest", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.ManifestHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "manifest.json")
		manifestContent := `{"name": "test-app", "version": "1.0.0"}`
		if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		body := `{"bundle_manifest_path": "` + manifestPath + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/bundle-manifest", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.ManifestHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp ManifestResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Path == "" {
			t.Error("expected path to be set")
		}
	})
}

func TestRegisterRoutes(t *testing.T) {
	service := NewService()
	h := NewHandler(service)

	router := mux.NewRouter()
	h.RegisterRoutes(router)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/desktop/preflight"},
		{http.MethodPost, "/api/v1/desktop/preflight/start"},
		{http.MethodGet, "/api/v1/desktop/preflight/status"},
		{http.MethodGet, "/api/v1/desktop/preflight/health"},
		{http.MethodPost, "/api/v1/desktop/bundle-manifest"},
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

func TestFindManifestService(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		_, ok := findManifestService(nil, "test")
		if ok {
			t.Error("expected false for nil manifest")
		}
	})

	t.Run("service not found", func(t *testing.T) {
		manifest := &bundlemanifest.Manifest{
			Services: []bundlemanifest.Service{
				{ID: "api"},
				{ID: "web"},
			},
		}
		_, ok := findManifestService(manifest, "nonexistent")
		if ok {
			t.Error("expected false for nonexistent service")
		}
	})

	t.Run("service found", func(t *testing.T) {
		manifest := &bundlemanifest.Manifest{
			Services: []bundlemanifest.Service{
				{ID: "api"},
				{ID: "web"},
			},
		}
		svc, ok := findManifestService(manifest, "api")
		if !ok {
			t.Error("expected true for existing service")
		}
		if svc.ID != "api" {
			t.Errorf("expected service ID 'api', got %q", svc.ID)
		}
	})
}

func TestFetchJSON(t *testing.T) {
	t.Run("success with payload", func(t *testing.T) {
		var receivedPayload map[string]string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("expected auth header")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("expected content-type header")
			}
			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := &http.Client{}
		var out map[string]string
		status, err := fetchJSON(client, server.URL, "test-token", "/test", http.MethodPost, map[string]string{"key": "value"}, &out, nil)
		if err != nil {
			t.Fatalf("fetchJSON() error: %v", err)
		}
		if status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}
		if receivedPayload["key"] != "value" {
			t.Errorf("expected key=value in payload")
		}
	})

	t.Run("error status returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		client := &http.Client{}
		_, err := fetchJSON(client, server.URL, "", "/test", http.MethodGet, nil, nil, nil)
		if err == nil {
			t.Fatal("expected error for 500 status")
		}
	})

	t.Run("allowed status with output", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"error": "validation"})
		}))
		defer server.Close()

		client := &http.Client{}
		var out map[string]string
		allow := map[int]bool{http.StatusUnprocessableEntity: true}
		status, err := fetchJSON(client, server.URL, "", "/test", http.MethodGet, nil, &out, allow)
		if err != nil {
			t.Fatalf("expected no error for allowed status, got %v", err)
		}
		if status != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", status)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := &http.Client{}
		var out map[string]string
		_, err := fetchJSON(client, server.URL, "", "/test", http.MethodGet, nil, &out, nil)
		if err == nil {
			t.Fatal("expected decode error")
		}
	})

	t.Run("allowed status decode error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := &http.Client{}
		var out map[string]string
		allow := map[int]bool{http.StatusUnprocessableEntity: true}
		_, err := fetchJSON(client, server.URL, "", "/test", http.MethodGet, nil, &out, allow)
		if err == nil {
			t.Fatal("expected decode error for allowed status")
		}
	})

	t.Run("no output parameter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		status, err := fetchJSON(client, server.URL, "", "/test", http.MethodGet, nil, nil, nil)
		if err != nil {
			t.Fatalf("fetchJSON() error: %v", err)
		}
		if status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}
	})
}
