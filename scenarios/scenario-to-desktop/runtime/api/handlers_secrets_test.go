package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

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
		if len(rt.telemetryLogs) != 1 || rt.telemetryLogs[0] != "secrets_validation_failed" {
			t.Fatalf("telemetry logs = %v, want [secrets_validation_failed]", rt.telemetryLogs)
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
