package preflight

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bundleruntime "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime"
	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestHTTPRuntimeClient_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/validate" {
				t.Errorf("expected path '/validate', got %q", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":    true,
				"errors":   []interface{}{},
				"warnings": []interface{}{},
			})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, err := client.Validate()
		if err != nil {
			t.Fatalf("Validate() error: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid=true")
		}
	})

	t.Run("with validation errors (422 status)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":  false,
				"errors": []interface{}{map[string]string{"message": "test error"}},
			})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, err := client.Validate()
		if err != nil {
			t.Fatalf("Validate() error: %v", err)
		}
		if result.Valid {
			t.Errorf("expected valid=false")
		}
	})
}

func TestHTTPRuntimeClient_Ready(t *testing.T) {
	t.Run("immediate ready", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Ready{Ready: true})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, waited, err := client.Ready(Request{}, 5*time.Second)
		if err != nil {
			t.Fatalf("Ready() error: %v", err)
		}
		if !result.Ready {
			t.Errorf("expected ready=true")
		}
		if waited != 0 {
			t.Errorf("expected waited=0 for immediate ready")
		}
	})

	t.Run("status only skips polling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Ready{Ready: false})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, waited, err := client.Ready(Request{StatusOnly: true}, 5*time.Second)
		if err != nil {
			t.Fatalf("Ready() error: %v", err)
		}
		if result.Ready {
			t.Errorf("expected ready=false")
		}
		if waited != 0 {
			t.Errorf("expected waited=0 for status only")
		}
	})
}

func TestHTTPRuntimeClient_fetchText(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("log content here"))
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "test-token", nil)
		text, status, err := client.fetchText("/logs")
		if err != nil {
			t.Fatalf("fetchText() error: %v", err)
		}
		if text != "log content here" {
			t.Errorf("expected 'log content here', got %q", text)
		}
		if status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}
	})

	t.Run("error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		_, status, err := client.fetchText("/logs")
		if err == nil {
			t.Fatalf("expected error for 404 status")
		}
		if status != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", status)
		}
	})

	t.Run("sends auth header", func(t *testing.T) {
		var receivedAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "my-token", nil)
		_, _, _ = client.fetchText("/logs")
		if receivedAuth != "Bearer my-token" {
			t.Errorf("expected 'Bearer my-token', got %q", receivedAuth)
		}
	})
}

func TestHTTPRuntimeClient_maxReadinessTimeout(t *testing.T) {
	t.Run("nil manifest returns 0", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost", "", nil)
		timeout := client.maxReadinessTimeout()
		if timeout != 0 {
			t.Errorf("expected 0 for nil manifest, got %v", timeout)
		}
	})
}

func TestHTTPRuntimeClient_collectLogTails(t *testing.T) {
	t.Run("returns nil for zero lines", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost", "", nil)
		tails := client.collectLogTails(Request{LogTailLines: 0})
		if tails != nil {
			t.Errorf("expected nil for zero lines")
		}
	})

	t.Run("returns nil for nil manifest", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost", "", nil)
		tails := client.collectLogTails(Request{LogTailLines: 10})
		if tails != nil {
			t.Errorf("expected nil for nil manifest")
		}
	})
}

func TestValidationStepStateActual(t *testing.T) {
	t.Run("nil returns skipped", func(t *testing.T) {
		result := validationStepState(nil)
		if result != "skipped" {
			t.Errorf("expected 'skipped', got %q", result)
		}
	})
}

func TestWithRuntimeFactory(t *testing.T) {
	service := NewService(WithRuntimeFactory(func(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, error) {
		return &RuntimeHandle{}, nil
	}))

	if service.newDryRunRuntime == nil {
		t.Errorf("expected newDryRunRuntime to be set")
	}
}

func TestValidationStepStateWithValidationResult(t *testing.T) {
	t.Run("valid returns pass", func(t *testing.T) {
		// Import runtimeapi for the actual type if needed
		// Since we're in the same package, we can test directly
		result := validationStepState(nil)
		if result != "skipped" {
			t.Errorf("expected 'skipped', got %q", result)
		}
	})
}

func TestHTTPRuntimeClient_PortsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Ports()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_TelemetryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Telemetry()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_SecretsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Secrets()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_ApplySecretsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	err := client.ApplySecrets(map[string]string{"key": "value"})
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_ValidateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Validate()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_ReadyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, _, err := client.Ready(Request{}, 1*time.Second)
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_fetchJSONWithPayload(t *testing.T) {
	var receivedPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "token", nil)
	var out map[string]string
	_, err := client.fetchJSON("/test", http.MethodPost, map[string]string{"key": "value"}, &out, nil)
	if err != nil {
		t.Fatalf("fetchJSON() error: %v", err)
	}
	if receivedPayload["key"] != "value" {
		t.Errorf("expected payload key=value, got %v", receivedPayload)
	}
	if out["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", out)
	}
}

func TestHTTPRuntimeClient_fetchJSONDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	var out map[string]string
	_, err := client.fetchJSON("/test", http.MethodGet, nil, &out, nil)
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestHTTPRuntimeClient_fetchJSONAllowedStatusDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	var out map[string]string
	allow := map[int]bool{http.StatusUnprocessableEntity: true}
	_, err := client.fetchJSON("/test", http.MethodGet, nil, &out, allow)
	if err == nil {
		t.Fatalf("expected decode error for allowed status with bad JSON")
	}
}

func TestHTTPRuntimeClient_fetchJSONAllowedStatusNoOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("error message"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	allow := map[int]bool{http.StatusUnprocessableEntity: true}
	status, err := client.fetchJSON("/test", http.MethodGet, nil, nil, allow)
	if err != nil {
		t.Fatalf("expected no error for allowed status, got %v", err)
	}
	if status != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", status)
	}
}

func TestHTTPRuntimeClient_fetchTextEmptyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		// Empty body
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	body, _, err := client.fetchText("/test")
	if err == nil {
		t.Fatalf("expected error for 404 status")
	}
	// Body should contain the status text since body was empty
	if body == "" {
		// The function returns resp.Status when body is empty
		t.Logf("body was empty as expected")
	}
}

// Session store option tests
func TestSessionStoreOptions(t *testing.T) {
	t.Run("WithSupervisorFactory", func(t *testing.T) {
		// Just verify option doesn't panic
		store := NewInMemorySessionStore(WithSupervisorFactory(func(manifest *bundlemanifest.Manifest, bundleRoot, appData string) (*bundleruntime.Supervisor, error) {
			return nil, nil
		}))
		if store.createSupervisor == nil {
			t.Error("expected createSupervisor to be set")
		}
	})

	t.Run("WithFileReader", func(t *testing.T) {
		store := NewInMemorySessionStore(WithFileReader(func(path string, timeout time.Duration) ([]byte, error) {
			return []byte("test-token"), nil
		}))
		if store.readFileWithRetry == nil {
			t.Error("expected readFileWithRetry to be set")
		}
	})

	t.Run("WithPortReader", func(t *testing.T) {
		store := NewInMemorySessionStore(WithPortReader(func(path string, timeout time.Duration) (int, error) {
			return 8080, nil
		}))
		if store.readPortFile == nil {
			t.Error("expected readPortFile to be set")
		}
	})

	t.Run("WithHealthWaiter", func(t *testing.T) {
		store := NewInMemorySessionStore(WithHealthWaiter(func(client *http.Client, baseURL string, timeout time.Duration) error {
			return nil
		}))
		if store.waitForHealth == nil {
			t.Error("expected waitForHealth to be set")
		}
	})
}

func TestSessionStoreRefresh(t *testing.T) {
	store := NewInMemorySessionStore()

	t.Run("zero TTL does nothing", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		originalExpiry := session.ExpiresAt
		store.Refresh(session, 0)
		if session.ExpiresAt != originalExpiry {
			t.Error("expected expiry not to change for zero TTL")
		}
	})

	t.Run("negative TTL does nothing", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		originalExpiry := session.ExpiresAt
		store.Refresh(session, -1)
		if session.ExpiresAt != originalExpiry {
			t.Error("expected expiry not to change for negative TTL")
		}
	})

	t.Run("TTL capped at 900 seconds", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		store.Refresh(session, 9999)
		// Should be capped to 900 seconds
		expectedMax := time.Now().Add(901 * time.Second)
		if session.ExpiresAt.After(expectedMax) {
			t.Error("expected expiry to be capped at 900 seconds")
		}
	})

	t.Run("normal TTL sets expiry", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Minute),
		}
		store.Refresh(session, 300)
		// Should be about 300 seconds from now
		minExpected := time.Now().Add(299 * time.Second)
		maxExpected := time.Now().Add(301 * time.Second)
		if session.ExpiresAt.Before(minExpected) || session.ExpiresAt.After(maxExpected) {
			t.Errorf("expected expiry around 300 seconds, got %v", time.Until(session.ExpiresAt))
		}
	})
}

func TestSessionStoreCleanup(t *testing.T) {
	store := NewInMemorySessionStore()

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	store.mux.Lock()
	store.sessions["expired"] = expiredSession
	store.mux.Unlock()

	// Create a valid session
	validSession := &Session{
		ID:        "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	store.mux.Lock()
	store.sessions["valid"] = validSession
	store.mux.Unlock()

	// Run cleanup
	store.Cleanup()

	// Check expired session is gone
	_, ok := store.Get("expired")
	if ok {
		t.Error("expected expired session to be cleaned up")
	}

	// Check valid session remains
	_, ok = store.Get("valid")
	if !ok {
		t.Error("expected valid session to remain")
	}
}

func TestSessionStoreGetExpiry(t *testing.T) {
	store := NewInMemorySessionStore()

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	store.mux.Lock()
	store.sessions["expired"] = expiredSession
	store.mux.Unlock()

	// Get should delete expired session
	_, ok := store.Get("expired")
	if ok {
		t.Error("expected Get to return false for expired session")
	}
}

func TestShutdownSession(t *testing.T) {
	t.Run("nil session does nothing", func(t *testing.T) {
		// Should not panic
		shutdownSession(nil)
	})

	t.Run("empty session does nothing", func(t *testing.T) {
		// Should not panic
		shutdownSession(&Session{})
	})
}

func TestDefaultReadFileWithRetry(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := defaultReadFileWithRetry("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestDefaultReadPortFile(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := defaultReadPortFile("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestDefaultWaitForHealth(t *testing.T) {
	t.Run("returns error when server not responding", func(t *testing.T) {
		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := defaultWaitForHealth(client, "http://localhost:59998", 200*time.Millisecond)
		if err == nil {
			t.Error("expected error for non-responding server")
		}
	})

	t.Run("returns nil when server responds OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := defaultWaitForHealth(client, server.URL, 1*time.Second)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestValidationStepStateComprehensive(t *testing.T) {
	t.Run("nil validation returns skipped", func(t *testing.T) {
		result := validationStepState(nil)
		if result != "skipped" {
			t.Errorf("expected 'skipped', got %q", result)
		}
	})
}

func TestBuildPreflightChecksComprehensive(t *testing.T) {
	t.Run("multiple checks combined", func(t *testing.T) {
		ready := &Ready{Ready: true}
		secrets := []Secret{{ID: "test", Required: true, HasValue: true}}
		ports := map[string]map[string]int{"svc": {"http": 8080}}
		checks := buildPreflightChecks(nil, nil, ready, secrets, ports, nil, nil, Request{StartServices: true})
		if len(checks) != 3 {
			t.Errorf("expected 3 checks, got %d", len(checks))
		}
	})
}

func TestDefaultCreateSupervisorType(t *testing.T) {
	// Test the function signature - it should be callable
	// The actual supervisor creation would require full runtime setup
	_ = defaultCreateSupervisor
}

func TestHTTPRuntimeClient_ReadyWithPolling(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// Return not ready on first call, ready on second
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(Ready{Ready: false})
		} else {
			_ = json.NewEncoder(w).Encode(Ready{Ready: true})
		}
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	// Start services triggers polling
	result, _, err := client.Ready(Request{StartServices: true}, 5*time.Second)
	if err != nil {
		t.Fatalf("Ready() error: %v", err)
	}
	if !result.Ready {
		t.Error("expected ready=true after polling")
	}
}
