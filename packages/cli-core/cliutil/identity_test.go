package cliutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectIdentity(t *testing.T) {
	t.Run("returns zero value when env var not set", func(t *testing.T) {
		t.Setenv(EnvIdentityToken, "")

		env := DetectIdentity()
		if env.Token != "" {
			t.Fatalf("expected empty Token, got %q", env.Token)
		}
	})

	t.Run("returns populated struct when env var set", func(t *testing.T) {
		t.Setenv(EnvIdentityToken, "test-token-abc")

		env := DetectIdentity()
		if env.Token != "test-token-abc" {
			t.Errorf("Token = %q, want %q", env.Token, "test-token-abc")
		}
	})
}

func TestIsIdentityPresent(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"token set", "some-token", true},
		{"token empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := IdentityEnv{Token: tt.token}
			if got := env.IsIdentityPresent(); got != tt.want {
				t.Errorf("IsIdentityPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyIdentity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/identity/verify" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["token"] != "valid-token" {
			t.Errorf("token = %q, want %q", body["token"], "valid-token")
		}

		resp := VerifyResult{
			Valid: true,
			Claims: &VerifiedClaims{
				RunID:      "run-uuid-123",
				TaskID:     "task-uuid-456",
				ProfileKey: "default",
				ScopePath:  "scenarios/my-scenario",
				IssuedAt:   1700000000,
				ExpiresAt:  1700003600,
				Meta:       map[string]string{"key": "value"},
			},
			RunStatus: "running",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)

	env := IdentityEnv{Token: "valid-token"}
	result, err := env.VerifyIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected Valid=true")
	}
	if result.Claims == nil {
		t.Fatal("expected non-nil Claims")
	}
	if result.Claims.RunID != "run-uuid-123" {
		t.Errorf("RunID = %q, want %q", result.Claims.RunID, "run-uuid-123")
	}
	if result.Claims.TaskID != "task-uuid-456" {
		t.Errorf("TaskID = %q, want %q", result.Claims.TaskID, "task-uuid-456")
	}
	if result.Claims.ProfileKey != "default" {
		t.Errorf("ProfileKey = %q, want %q", result.Claims.ProfileKey, "default")
	}
	if result.Claims.ScopePath != "scenarios/my-scenario" {
		t.Errorf("ScopePath = %q, want %q", result.Claims.ScopePath, "scenarios/my-scenario")
	}
	if result.RunStatus != "running" {
		t.Errorf("RunStatus = %q, want %q", result.RunStatus, "running")
	}
	if result.Claims.Meta["key"] != "value" {
		t.Errorf("Meta[key] = %q, want %q", result.Claims.Meta["key"], "value")
	}
}

func TestVerifyIdentity_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "token signature invalid",
		})
	}))
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)

	env := IdentityEnv{Token: "bad-token"}
	result, err := env.VerifyIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected Valid=false")
	}
	if result.Error != "token signature invalid" {
		t.Errorf("Error = %q, want %q", result.Error, "token signature invalid")
	}
}

func TestVerifyIdentity_NetworkError(t *testing.T) {
	// Start and immediately close the server to get a connection error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)

	env := IdentityEnv{Token: "some-token"}
	_, err := env.VerifyIdentity()
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}

func TestVerifyIdentity_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json{{{"))
	}))
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)

	env := IdentityEnv{Token: "some-token"}
	_, err := env.VerifyIdentity()
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestVerifyIdentity_EmptyToken(t *testing.T) {
	env := IdentityEnv{Token: ""}
	_, err := env.VerifyIdentity()
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestVerifyIdentity_NoBaseURL(t *testing.T) {
	t.Setenv("AGENT_MANAGER_API_BASE", "")
	t.Setenv("AGENT_MANAGER_API_URL", "")
	t.Setenv("AGENT_MANAGER_API_PORT", "")

	originalDetector := detectAgentManagerPort
	t.Cleanup(func() { detectAgentManagerPort = originalDetector })
	detectAgentManagerPort = func() string { return "" }

	env := IdentityEnv{Token: "some-token"}
	_, err := env.VerifyIdentity()
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
	if !strings.Contains(err.Error(), "not discoverable") {
		t.Fatalf("expected discovery guidance, got %v", err)
	}
}

func TestVerifyIdentity_UsesLifecyclePortDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/identity/verify" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(VerifyResult{Valid: true, RunStatus: "running"})
	}))
	defer server.Close()

	originalDetector := detectAgentManagerPort
	t.Cleanup(func() { detectAgentManagerPort = originalDetector })
	detectAgentManagerPort = func() string {
		return strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	}

	env := IdentityEnv{Token: "valid-token"}
	result, err := env.VerifyIdentity()
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected Valid=true")
	}
}
