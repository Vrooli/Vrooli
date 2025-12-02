package bundleruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// mockPortAllocator is a test double for PortAllocator.
type mockPortAllocator struct {
	ports map[string]map[string]int
}

func (m *mockPortAllocator) Allocate() error { return nil }
func (m *mockPortAllocator) Resolve(serviceID, portName string) (int, error) {
	if ports, ok := m.ports[serviceID]; ok {
		if port, ok := ports[portName]; ok {
			return port, nil
		}
	}
	return 0, nil
}
func (m *mockPortAllocator) Map() map[string]map[string]int { return m.ports }

// testSupervisor creates a supervisor configured for testing.
func testSupervisor(t *testing.T, m *manifest.Manifest) *Supervisor {
	t.Helper()

	if m == nil {
		m = &manifest.Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           manifest.App{Name: "test-app", Version: "1.0.0"},
			IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
			Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
			Services: []manifest.Service{
				{
					ID:   "api",
					Type: "api",
					Binaries: map[string]manifest.Binary{
						"linux-x64": {Path: "bin/api"},
					},
					Health:    manifest.HealthCheck{Type: "http", Path: "/health", PortName: "http"},
					Readiness: manifest.ReadinessCheck{Type: "port_open", PortName: "http"},
					Ports: &manifest.ServicePorts{
						Requested: []manifest.PortRequest{{Name: "http", Range: manifest.PortRange{Min: 47000, Max: 48000}}},
					},
					LogDir: "logs/api.log",
				},
			},
		}
	}

	appData := t.TempDir()

	// Create mock port allocator with preset ports.
	mockPorts := &mockPortAllocator{
		ports: map[string]map[string]int{"api": {"http": 47001}},
	}

	s, err := NewSupervisor(Options{
		AppDataDir:    appData,
		Manifest:      m,
		BundlePath:    t.TempDir(),
		DryRun:        true,
		Clock:         &fakeClock{now: time.Now()},
		FileSystem:    RealFileSystem{},
		PortAllocator: mockPorts,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	s.authToken = "test-token"
	s.telemetryPath = filepath.Join(appData, "telemetry.jsonl")
	s.migrationsPath = filepath.Join(appData, "migrations.json")
	s.serviceStatus = map[string]ServiceStatus{"api": {Ready: true, Message: "ready"}}
	s.started = true

	return s
}

func TestHandleHealth(t *testing.T) {
	s := testSupervisor(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("handleHealth() status = %v, want 'ok'", body["status"])
	}

	if body["services"].(float64) != 1 {
		t.Errorf("handleHealth() services = %v, want 1", body["services"])
	}
}

func TestHandleReady(t *testing.T) {
	t.Run("all services ready", func(t *testing.T) {
		s := testSupervisor(t, nil)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		s.handleReady(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if body["ready"] != true {
			t.Errorf("handleReady() ready = %v, want true", body["ready"])
		}
	})

	t.Run("service not ready", func(t *testing.T) {
		s := testSupervisor(t, nil)
		s.serviceStatus["api"] = ServiceStatus{Ready: false, Message: "starting"}

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		s.handleReady(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if body["ready"] != false {
			t.Errorf("handleReady() ready = %v, want false", body["ready"])
		}
	})
}

func TestHandlePorts(t *testing.T) {
	s := testSupervisor(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/ports", nil)
	w := httptest.NewRecorder()

	s.handlePorts(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	services, ok := body["services"].(map[string]interface{})
	if !ok {
		t.Fatalf("handlePorts() services not a map")
	}

	apiPorts, ok := services["api"].(map[string]interface{})
	if !ok {
		t.Fatalf("handlePorts() api ports not a map")
	}

	if apiPorts["http"].(float64) != 47001 {
		t.Errorf("handlePorts() api.http = %v, want 47001", apiPorts["http"])
	}
}

func TestHandleLogs(t *testing.T) {
	t.Run("returns log content", func(t *testing.T) {
		s := testSupervisor(t, nil)

		// Create log file
		logPath := filepath.Join(s.appData, "logs", "api.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			t.Fatal(err)
		}
		logContent := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=api&lines=3", nil)
		w := httptest.NewRecorder()

		s.handleLogs(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("handleLogs() status = %d, body = %s", resp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "line5") {
			t.Errorf("handleLogs() should contain last lines, got %q", string(body))
		}
	})

	t.Run("rejects unknown service", func(t *testing.T) {
		s := testSupervisor(t, nil)

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=unknown", nil)
		w := httptest.NewRecorder()

		s.handleLogs(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("handleLogs() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestHandleShutdown(t *testing.T) {
	s := testSupervisor(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/shutdown", nil)
	w := httptest.NewRecorder()

	s.handleShutdown(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleShutdown() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["status"] != "stopping" {
		t.Errorf("handleShutdown() status = %v, want 'stopping'", body["status"])
	}
}

func TestHandleSecretsGet(t *testing.T) {
	m := &manifest.Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
		Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
		Secrets: []manifest.Secret{
			{ID: "api_key", Class: "api_key", Description: "API Key", Target: manifest.SecretTarget{Type: "env", Name: "API_KEY"}},
		},
		Services: []manifest.Service{
			{
				ID:        "api",
				Type:      "api",
				Binaries:  map[string]manifest.Binary{"linux-x64": {Path: "bin/api"}},
				Health:    manifest.HealthCheck{Type: "http"},
				Readiness: manifest.ReadinessCheck{Type: "port_open"},
			},
		},
	}
	s := testSupervisor(t, m)
	s.secretStore.Set(map[string]string{"api_key": "secret-value"})

	req := httptest.NewRequest(http.MethodGet, "/secrets", nil)
	w := httptest.NewRecorder()

	s.handleSecrets(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleSecrets GET status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	secrets, ok := body["secrets"].([]interface{})
	if !ok || len(secrets) != 1 {
		t.Fatalf("handleSecrets GET secrets = %v, want 1 secret", body["secrets"])
	}

	secret := secrets[0].(map[string]interface{})
	if secret["id"] != "api_key" {
		t.Errorf("handleSecrets GET secret.id = %v, want 'api_key'", secret["id"])
	}
	if secret["has_value"] != true {
		t.Errorf("handleSecrets GET secret.has_value = %v, want true", secret["has_value"])
	}
}

func TestHandleSecretsPost(t *testing.T) {
	t.Run("accepts valid secrets", func(t *testing.T) {
		required := true
		m := &manifest.Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           manifest.App{Name: "test-app", Version: "1.0.0"},
			IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
			Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
			Secrets: []manifest.Secret{
				{ID: "api_key", Class: "api_key", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "API_KEY"}},
			},
			Services: []manifest.Service{
				{
					ID:        "api",
					Type:      "api",
					Binaries:  map[string]manifest.Binary{"linux-x64": {Path: "bin/api"}},
					Health:    manifest.HealthCheck{Type: "http"},
					Readiness: manifest.ReadinessCheck{Type: "port_open"},
				},
			},
		}
		s := testSupervisor(t, m)

		payload := `{"secrets": {"api_key": "my-secret"}}`
		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(payload))
		w := httptest.NewRecorder()

		s.handleSecrets(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("handleSecrets POST status = %d, body = %s", resp.StatusCode, string(body))
		}
	})

	t.Run("rejects missing required secrets", func(t *testing.T) {
		required := true
		m := &manifest.Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           manifest.App{Name: "test-app", Version: "1.0.0"},
			IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
			Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
			Secrets: []manifest.Secret{
				{ID: "api_key", Class: "api_key", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "API_KEY"}},
				{ID: "db_pass", Class: "password", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "DB_PASS"}},
			},
			Services: []manifest.Service{
				{
					ID:        "api",
					Type:      "api",
					Binaries:  map[string]manifest.Binary{"linux-x64": {Path: "bin/api"}},
					Health:    manifest.HealthCheck{Type: "http"},
					Readiness: manifest.ReadinessCheck{Type: "port_open"},
				},
			},
		}
		s := testSupervisor(t, m)

		payload := `{"secrets": {"api_key": "my-secret"}}`
		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(payload))
		w := httptest.NewRecorder()

		s.handleSecrets(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("handleSecrets POST status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestHandleTelemetry(t *testing.T) {
	s := testSupervisor(t, nil)
	s.opts.Manifest.Telemetry.UploadTo = "https://example.com/upload"

	req := httptest.NewRequest(http.MethodGet, "/telemetry", nil)
	w := httptest.NewRecorder()

	s.handleTelemetry(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleTelemetry() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["upload_url"] != "https://example.com/upload" {
		t.Errorf("handleTelemetry() upload_url = %v, want 'https://example.com/upload'", body["upload_url"])
	}
}

func TestAuthMiddleware(t *testing.T) {
	s := testSupervisor(t, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/test", handler)
	mux.HandleFunc("/healthz", s.handleHealth)

	wrapped := s.authMiddleware(mux)

	t.Run("allows unauthenticated /healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("authMiddleware(/healthz) status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("rejects missing auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("authMiddleware() without auth status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("authMiddleware() with wrong token status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("allows valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("authMiddleware() with valid token status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

// fakeClock implements Clock for testing.
type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time                         { return c.now }
func (c *fakeClock) Sleep(d time.Duration)                  {}
func (c *fakeClock) After(d time.Duration) <-chan time.Time { return time.After(0) }
func (c *fakeClock) NewTicker(d time.Duration) Ticker       { return &fakeTicker{} }

type fakeTicker struct{}

func (t *fakeTicker) C() <-chan time.Time { return make(chan time.Time) }
func (t *fakeTicker) Stop()               {}

// Integration test for full handler registration
func TestRegisterHandlers(t *testing.T) {
	s := testSupervisor(t, nil)

	mux := http.NewServeMux()
	s.registerHandlers(mux)

	// Verify all expected routes are registered by making requests
	routes := []string{"/healthz", "/readyz", "/ports", "/shutdown", "/secrets", "/telemetry"}

	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Any response other than 404 indicates the route exists
		if w.Code == http.StatusNotFound {
			t.Errorf("registerHandlers() missing route %s", route)
		}
	}
}

// Test that runtime context is used for shutdown
func TestHandleShutdownUsesContext(t *testing.T) {
	s := testSupervisor(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.runtimeCtx = ctx
	s.cancel = cancel

	req := httptest.NewRequest(http.MethodGet, "/shutdown", nil)
	w := httptest.NewRecorder()

	s.handleShutdown(w, req)

	// Just verify it doesn't panic and returns proper response
	if w.Code != http.StatusOK {
		t.Errorf("handleShutdown() status = %d, want %d", w.Code, http.StatusOK)
	}
}
