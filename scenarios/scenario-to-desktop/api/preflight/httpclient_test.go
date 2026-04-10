package preflight

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPRuntimeClient(t *testing.T) {
	client := NewHTTPRuntimeClient("http://localhost:8080", "test-token", nil)
	if client == nil {
		t.Fatalf("expected client to be created")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL 'http://localhost:8080', got %q", client.baseURL)
	}
	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got %q", client.token)
	}
}

func TestNewHTTPRuntimeClientWithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 10 * time.Second}
	client := NewHTTPRuntimeClient("http://localhost:8080", "test-token", nil,
		WithHTTPClient(customHTTPClient),
	)
	if client.httpClient != customHTTPClient {
		t.Errorf("expected custom HTTP client to be used")
	}
}

func TestHTTPRuntimeClient_Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Errorf("expected path '/status', got %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Runtime{RuntimeVersion: "1.0.0", InstanceID: "test-instance"})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.RuntimeVersion != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", status.RuntimeVersion)
	}
}

func TestHTTPRuntimeClient_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Status()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_Secrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/secrets" {
			t.Errorf("expected path '/secrets', got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"secrets": []Secret{
				{ID: "API_KEY", Required: true, HasValue: false},
				{ID: "DB_PASSWORD", Required: true, HasValue: true},
			},
		})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	secrets, err := client.Secrets()
	if err != nil {
		t.Fatalf("Secrets() error: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}
}

func TestHTTPRuntimeClient_ApplySecrets(t *testing.T) {
	t.Run("posts secrets", func(t *testing.T) {
		var receivedPayload map[string]map[string]string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		err := client.ApplySecrets(map[string]string{"API_KEY": "secret123"})
		if err != nil {
			t.Fatalf("ApplySecrets() error: %v", err)
		}
		if receivedPayload["secrets"]["API_KEY"] != "secret123" {
			t.Errorf("expected API_KEY 'secret123', got %q", receivedPayload["secrets"]["API_KEY"])
		}
	})

	t.Run("skips empty secrets", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		err := client.ApplySecrets(map[string]string{"API_KEY": ""})
		if err != nil {
			t.Fatalf("ApplySecrets() error: %v", err)
		}
		if called {
			t.Errorf("expected no HTTP call for empty secrets")
		}
	})

	t.Run("skips nil secrets", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		err := client.ApplySecrets(nil)
		if err != nil {
			t.Fatalf("ApplySecrets() error: %v", err)
		}
		if called {
			t.Errorf("expected no HTTP call for nil secrets")
		}
	})
}

func TestHTTPRuntimeClient_Ports(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"services": map[string]map[string]int{
				"api": {"http": 8080},
				"web": {"http": 3000},
			},
		})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	ports, err := client.Ports()
	if err != nil {
		t.Fatalf("Ports() error: %v", err)
	}
	if ports["api"]["http"] != 8080 {
		t.Errorf("expected api http port 8080, got %d", ports["api"]["http"])
	}
}

func TestHTTPRuntimeClient_Telemetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Telemetry{Path: "/telemetry", UploadURL: "http://localhost:8080/telemetry"})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	telemetry, err := client.Telemetry()
	if err != nil {
		t.Fatalf("Telemetry() error: %v", err)
	}
	if telemetry.Path != "/telemetry" {
		t.Errorf("expected path '/telemetry', got %q", telemetry.Path)
	}
}

func TestHTTPRuntimeClient_LogTails(t *testing.T) {
	t.Run("returns nil for nil manifest", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost:8080", "", nil)
		tails := client.LogTails(Request{LogTailLines: 10})
		if tails != nil {
			t.Errorf("expected nil for nil manifest")
		}
	})
}

func TestHTTPRuntimeClient_AuthorizationHeader(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Runtime{})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "test-token", nil)
	_, _ = client.Status()

	if receivedAuth != "Bearer test-token" {
		t.Errorf("expected 'Bearer test-token', got %q", receivedAuth)
	}
}

func TestUpdatePreflightResult(t *testing.T) {
	t.Run("nil prev creates new response", func(t *testing.T) {
		result := updatePreflightResult(nil, func(next *Response) {
			next.Status = "updated"
		})
		if result.Status != "updated" {
			t.Errorf("expected status 'updated', got %q", result.Status)
		}
	})

	t.Run("existing prev is updated", func(t *testing.T) {
		prev := &Response{Status: "initial"}
		result := updatePreflightResult(prev, func(next *Response) {
			next.Status = "updated"
		})
		if result.Status != "updated" {
			t.Errorf("expected status 'updated', got %q", result.Status)
		}
	})
}

func TestRuntimeHandleFromSession(t *testing.T) {
	t.Run("nil session returns empty handle", func(t *testing.T) {
		handle := runtimeHandleFromSession(nil)
		if handle == nil {
			t.Fatalf("expected non-nil handle")
		}
		if handle.Client != nil {
			t.Errorf("expected nil client for nil session")
		}
		if handle.SessionID != "" {
			t.Errorf("expected empty session ID")
		}
	})

	t.Run("valid session creates handle", func(t *testing.T) {
		session := &Session{
			ID:        "test-session",
			BaseURL:   "http://localhost:8080",
			Token:     "test-token",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		handle := runtimeHandleFromSession(session)
		if handle.SessionID != "test-session" {
			t.Errorf("expected session ID 'test-session', got %q", handle.SessionID)
		}
		if handle.Client == nil {
			t.Errorf("expected non-nil client")
		}
		if handle.ExpiresAt.IsZero() {
			t.Errorf("expected ExpiresAt to be set")
		}
	})
}

func TestCollectServiceFingerprints(t *testing.T) {
	t.Run("nil manifest returns nil", func(t *testing.T) {
		result := collectServiceFingerprints(nil, "/tmp")
		if result != nil {
			t.Errorf("expected nil for nil manifest")
		}
	})
}

func TestSha256File(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := sha256File("/nonexistent/path/to/file")
		if err == nil {
			t.Errorf("expected error for nonexistent file")
		}
	})
}

func TestReadFileWithRetry(t *testing.T) {
	t.Run("nonexistent file returns error after timeout", func(t *testing.T) {
		_, err := readFileWithRetry("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Errorf("expected error for nonexistent file")
		}
	})
}

func TestReadPortFileWithRetry(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := readPortFileWithRetry("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Errorf("expected error for nonexistent file")
		}
	})
}

func TestWaitForRuntimeHealth(t *testing.T) {
	t.Run("returns error when server not responding", func(t *testing.T) {
		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := waitForRuntimeHealth(client, "http://localhost:59999", 200*time.Millisecond)
		if err == nil {
			t.Errorf("expected error for non-responding server")
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
		err := waitForRuntimeHealth(client, server.URL, 1*time.Second)
		if err != nil {
			t.Errorf("expected nil error for responding server, got %v", err)
		}
	})
}

func TestBuildPreflightChecks(t *testing.T) {
	t.Run("nil inputs returns empty checks", func(t *testing.T) {
		checks := buildPreflightChecks(nil, nil, nil, nil, nil, nil, nil, Request{})
		if len(checks) != 0 {
			t.Errorf("expected 0 checks, got %d", len(checks))
		}
	})

	t.Run("adds checks for ready", func(t *testing.T) {
		ready := &Ready{Ready: true}
		checks := buildPreflightChecks(nil, nil, ready, nil, nil, nil, nil, Request{})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].ID != "runtime-ready" {
			t.Errorf("expected ID 'runtime-ready', got %q", checks[0].ID)
		}
		if checks[0].Status != "pass" {
			t.Errorf("expected status 'pass', got %q", checks[0].Status)
		}
	})

	t.Run("adds checks for not ready", func(t *testing.T) {
		ready := &Ready{Ready: false}
		checks := buildPreflightChecks(nil, nil, ready, nil, nil, nil, nil, Request{})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].Status != "fail" {
			t.Errorf("expected status 'fail', got %q", checks[0].Status)
		}
	})

	t.Run("adds checks for secrets", func(t *testing.T) {
		secrets := []Secret{{ID: "API_KEY", Required: true, HasValue: true}}
		checks := buildPreflightChecks(nil, nil, nil, secrets, nil, nil, nil, Request{})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].ID != "secrets" {
			t.Errorf("expected ID 'secrets', got %q", checks[0].ID)
		}
	})

	t.Run("adds diagnostics check for start_services", func(t *testing.T) {
		ports := map[string]map[string]int{"api": {"http": 8080}}
		checks := buildPreflightChecks(nil, nil, nil, nil, ports, nil, nil, Request{StartServices: true})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].ID != "diagnostics" {
			t.Errorf("expected ID 'diagnostics', got %q", checks[0].ID)
		}
	})
}

func TestServiceStartJanitor(t *testing.T) {
	t.Run("does not panic with nil stores", func(t *testing.T) {
		service := &DefaultService{
			sessions: nil,
			jobs:     nil,
		}
		// Should not panic
		service.StartJanitor()
	})
}
