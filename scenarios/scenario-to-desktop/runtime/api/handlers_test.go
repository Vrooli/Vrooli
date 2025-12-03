package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/secrets"
)

// mockRuntime implements Runtime for testing.
type mockRuntime struct {
	statuses       map[string]health.Status
	ports          map[string]map[string]int
	telemetryPath  string
	uploadURL      string
	manifest       *manifest.Manifest
	appDataDir     string
	fs             infra.FileSystem
	secretStore    *secrets.Manager
	shutdownCalled bool
	startCalled    bool
	telemetryLogs  []string
	gpuStatus      gpu.Status
}

func (m *mockRuntime) Shutdown(ctx context.Context) error {
	m.shutdownCalled = true
	return nil
}

func (m *mockRuntime) ServiceStatuses() map[string]health.Status {
	return m.statuses
}

func (m *mockRuntime) PortMap() map[string]map[string]int {
	return m.ports
}

func (m *mockRuntime) TelemetryPath() string {
	return m.telemetryPath
}

func (m *mockRuntime) TelemetryUploadURL() string {
	return m.uploadURL
}

func (m *mockRuntime) Manifest() *manifest.Manifest {
	return m.manifest
}

func (m *mockRuntime) AppDataDir() string {
	return m.appDataDir
}

func (m *mockRuntime) FileSystem() infra.FileSystem {
	return m.fs
}

func (m *mockRuntime) SecretStore() SecretStore {
	return m.secretStore
}

func (m *mockRuntime) StartServicesIfReady() {
	m.startCalled = true
}

func (m *mockRuntime) RecordTelemetry(event string, details map[string]interface{}) error {
	m.telemetryLogs = append(m.telemetryLogs, event)
	return nil
}

func (m *mockRuntime) GPUStatus() gpu.Status {
	return m.gpuStatus
}

// testRuntime creates a mock runtime configured for testing.
func testRuntime(t *testing.T, m *manifest.Manifest) *mockRuntime {
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
	fs := infra.RealFileSystem{}

	return &mockRuntime{
		statuses:      map[string]health.Status{"api": {Ready: true, Message: "ready"}},
		ports:         map[string]map[string]int{"api": {"http": 47001}},
		telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		uploadURL:     "",
		manifest:      m,
		appDataDir:    appData,
		fs:            fs,
		secretStore:   secrets.NewManager(m, fs, filepath.Join(appData, "secrets.json")),
		gpuStatus: gpu.Status{
			Available: true,
			Method:    "mock",
			Reason:    "test",
		},
	}
}

func TestHandleHealth(t *testing.T) {
	rt := testRuntime(t, nil)
	server := NewServer(rt, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	mux.ServeHTTP(w, req)

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
		rt := testRuntime(t, nil)
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if body["ready"] != true {
			t.Errorf("handleReady() ready = %v, want true", body["ready"])
		}

		gpuInfo, ok := body["gpu"].(map[string]interface{})
		if !ok {
			t.Fatalf("handleReady() gpu missing or wrong type")
		}
		if gpuInfo["available"] != true {
			t.Errorf("handleReady() gpu.available = %v, want true", gpuInfo["available"])
		}
		reqs, ok := gpuInfo["requirements"].(map[string]interface{})
		if !ok {
			t.Fatalf("handleReady() gpu.requirements missing or wrong type")
		}
		if reqs["api"] != nil {
			t.Errorf("handleReady() expected no gpu requirement for api, got %v", reqs["api"])
		}
	})

	t.Run("service not ready", func(t *testing.T) {
		rt := testRuntime(t, nil)
		rt.statuses["api"] = health.Status{Ready: false, Message: "starting"}
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

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
	rt := testRuntime(t, nil)
	server := NewServer(rt, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/ports", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	mux.ServeHTTP(w, req)

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
		rt := testRuntime(t, nil)
		server := NewServer(rt, "test-token")

		logPath := filepath.Join(rt.appDataDir, "logs", "api.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			t.Fatal(err)
		}
		logContent := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=api&lines=3", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

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
		rt := testRuntime(t, nil)
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=unknown", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("handleLogs() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("rejects log dir that is a directory", func(t *testing.T) {
		rt := testRuntime(t, nil)
		server := NewServer(rt, "test-token")

		logDir := filepath.Join(rt.appDataDir, "logs", "api.log")
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			t.Fatalf("failed creating directory for log dir test: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=api&lines=5", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("handleLogs() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "directory") {
			t.Fatalf("handleLogs() unexpected body: %q", string(body))
		}
	})

	t.Run("rejects service without log file configured", func(t *testing.T) {
		m := &manifest.Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           manifest.App{Name: "test-app", Version: "1.0.0"},
			IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
			Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
			Services: []manifest.Service{
				{
					ID:        "api",
					Type:      "api",
					Binaries:  map[string]manifest.Binary{"linux-x64": {Path: "bin/api"}},
					Health:    manifest.HealthCheck{Type: "http", Path: "/health", PortName: "http"},
					Readiness: manifest.ReadinessCheck{Type: "port_open", PortName: "http"},
					LogDir:    "",
				},
			},
		}
		rt := testRuntime(t, m)
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=api", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("handleLogs() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "no log_dir") {
			t.Fatalf("handleLogs() expected missing log_dir message, got %q", string(body))
		}
	})

	t.Run("rejects when log file missing", func(t *testing.T) {
		rt := testRuntime(t, nil)
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=api&lines=5", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("handleLogs() status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "log path unavailable") {
			t.Fatalf("handleLogs() expected log path unavailable error, got %q", string(body))
		}
	})

	t.Run("defaults to 200 lines when not provided", func(t *testing.T) {
		rt := testRuntime(t, nil)
		server := NewServer(rt, "test-token")

		lines := make([]string, 0, 250)
		for i := 1; i <= 250; i++ {
			lines = append(lines, fmt.Sprintf("line%d", i))
		}
		logContent := strings.Join(lines, "\n")

		logPath := filepath.Join(rt.appDataDir, "logs", "api.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(logPath, []byte(logContent), 0o644); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/logs/tail?serviceId=api", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("handleLogs() status = %d, body = %s", resp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if !strings.HasPrefix(bodyStr, "line51\n") {
			t.Fatalf("handleLogs() should start from line51 when defaulting to 200 lines, got prefix %q", bodyStr[:12])
		}
		if !strings.Contains(bodyStr, "line250") {
			t.Fatalf("handleLogs() missing last line, got %q", bodyStr)
		}
	})
}

func TestHandleShutdown(t *testing.T) {
	rt := testRuntime(t, nil)
	server := NewServer(rt, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/shutdown", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	mux.ServeHTTP(w, req)

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

	time.Sleep(150 * time.Millisecond)
	if !rt.shutdownCalled {
		t.Error("handleShutdown() should have called Shutdown()")
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
	rt := testRuntime(t, m)
	rt.secretStore.Set(map[string]string{"api_key": "secret-value"})
	server := NewServer(rt, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/secrets", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleSecrets GET status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	secretsList, ok := body["secrets"].([]interface{})
	if !ok || len(secretsList) != 1 {
		t.Fatalf("handleSecrets GET secrets = %v, want 1 secret", body["secrets"])
	}

	secret := secretsList[0].(map[string]interface{})
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
		rt := testRuntime(t, m)
		server := NewServer(rt, "test-token")

		payload := `{"secrets": {"api_key": "my-secret"}}`
		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(payload))
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("handleSecrets POST status = %d, body = %s", resp.StatusCode, string(body))
		}

		if !rt.startCalled {
			t.Error("handleSecrets POST should have called StartServicesIfReady()")
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
		rt := testRuntime(t, m)
		server := NewServer(rt, "test-token")

		payload := `{"secrets": {"api_key": "my-secret"}}`
		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(payload))
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("handleSecrets POST status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("records telemetry and blocks start when required secrets missing", func(t *testing.T) {
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
					Secrets:   []string{"api_key", "db_pass"},
				},
			},
		}
		rt := testRuntime(t, m)
		server := NewServer(rt, "test-token")

		payload := `{"secrets": {"api_key": "my-secret"}}`
		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(payload))
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("handleSecrets POST status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
		if rt.startCalled {
			t.Fatalf("handleSecrets POST should not start services when required secrets missing")
		}
		if len(rt.telemetryLogs) != 1 || rt.telemetryLogs[0] != "secrets_missing" {
			t.Fatalf("telemetry logs = %v, want [secrets_missing]", rt.telemetryLogs)
		}
	})

	t.Run("persists merged secrets", func(t *testing.T) {
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
					Secrets:   []string{"api_key", "db_pass"},
				},
			},
		}
		rt := testRuntime(t, m)
		rt.secretStore.Set(map[string]string{"api_key": "existing"})
		server := NewServer(rt, "test-token")

		payload := `{"secrets": {"db_pass": "new-pass"}}`
		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(payload))
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("handleSecrets POST status = %d, body = %s", resp.StatusCode, string(body))
		}

		loaded, err := rt.secretStore.Load()
		if err != nil {
			t.Fatalf("Load() after POST: %v", err)
		}

		want := map[string]string{"api_key": "existing", "db_pass": "new-pass"}
		if !reflect.DeepEqual(want, loaded) {
			t.Fatalf("persisted secrets mismatch: got %v, want %v", loaded, want)
		}
	})

	t.Run("rejects invalid JSON payload", func(t *testing.T) {
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
		rt := testRuntime(t, m)
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader("{invalid-json}"))
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("handleSecrets POST status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
		if rt.startCalled {
			t.Fatalf("handleSecrets POST should not start services on invalid JSON")
		}

		loaded, err := rt.secretStore.Load()
		if err != nil {
			t.Fatalf("Load() after invalid JSON: %v", err)
		}
		if len(loaded) != 0 {
			t.Fatalf("expected no secrets persisted on invalid JSON, got %v", loaded)
		}
	})

	t.Run("accepts empty secrets map when all secrets are optional", func(t *testing.T) {
		required := false
		m := &manifest.Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           manifest.App{Name: "test-app", Version: "1.0.0"},
			IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
			Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
			Secrets: []manifest.Secret{
				{ID: "optional_key", Class: "api_key", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "OPTIONAL_KEY"}},
			},
			Services: []manifest.Service{
				{
					ID:        "api",
					Type:      "api",
					Binaries:  map[string]manifest.Binary{"linux-x64": {Path: "bin/api"}},
					Health:    manifest.HealthCheck{Type: "http"},
					Readiness: manifest.ReadinessCheck{Type: "port_open"},
					Secrets:   []string{"optional_key"},
				},
			},
		}
		rt := testRuntime(t, m)
		server := NewServer(rt, "test-token")

		req := httptest.NewRequest(http.MethodPost, "/secrets", strings.NewReader(`{}`))
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		server.RegisterHandlers(mux)
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("handleSecrets POST status = %d, body = %s", resp.StatusCode, string(body))
		}
		if !rt.startCalled {
			t.Fatalf("handleSecrets POST should start services when only optional secrets are declared")
		}

		loaded, err := rt.secretStore.Load()
		if err != nil {
			t.Fatalf("Load() after optional secrets POST: %v", err)
		}
		if len(loaded) != 0 {
			t.Fatalf("expected empty secrets persisted, got %v", loaded)
		}
	})
}

func TestHandleTelemetry(t *testing.T) {
	rt := testRuntime(t, nil)
	rt.uploadURL = "https://example.com/upload"
	server := NewServer(rt, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/telemetry", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	mux.ServeHTTP(w, req)

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
	rt := testRuntime(t, nil)
	server := NewServer(rt, "test-token")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/test", handler)
	server.RegisterHandlers(mux)

	wrapped := server.AuthMiddleware(mux)

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

func TestRegisterHandlers(t *testing.T) {
	rt := testRuntime(t, nil)
	server := NewServer(rt, "test-token")

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)

	routes := []string{"/healthz", "/readyz", "/ports", "/shutdown", "/secrets", "/telemetry"}

	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code == http.StatusNotFound {
			t.Errorf("RegisterHandlers() missing route %s", route)
		}
	}
}
