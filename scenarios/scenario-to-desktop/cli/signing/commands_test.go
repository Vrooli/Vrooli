package signing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func newTestClient(handler http.Handler) *cliutil.APIClient {
	server := httptest.NewServer(handler)
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
			BaseOptions: cliutil.APIBaseOptions{DefaultBase: server.URL},
		}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{DefaultBase: server.URL}
		},
		func() string { return "" },
	)
}

// --- Get ---

func TestGet_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Get([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestGet_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/signing/my-scenario") {
			t.Errorf("path = %q, want to contain '/signing/my-scenario'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema_version":"1.0","windows":{}}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Get([]string{"my-scenario"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Set ---

func TestSet_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Set([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
}

func TestSet_MissingConfig(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Set([]string{"my-scenario"})
	if err == nil {
		t.Fatal("expected error for missing --config")
	}
}

func TestSet_InvalidJSON(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Set([]string{"my-scenario", "--config", "not-json"})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("error = %q, want 'invalid JSON'", err.Error())
	}
}

func TestSet_InlineJSON(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Set([]string{"my-scenario", "--config", `{"windows":{"enabled":true}}`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	win, ok := receivedBody["windows"].(map[string]interface{})
	if !ok {
		t.Fatal("expected windows key in body")
	}
	if win["enabled"] != true {
		t.Errorf("windows.enabled = %v, want true", win["enabled"])
	}
}

func TestSet_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"linux":{"enabled":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Set([]string{"my-scenario", "--config", "@" + configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := receivedBody["linux"]; !ok {
		t.Error("expected linux key in body from file")
	}
}

func TestSet_FromFile_NotFound(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Set([]string{"my-scenario", "--config", "@/nonexistent/file.json"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("error = %q, want 'failed to read config file'", err.Error())
	}
}

// --- Delete ---

func TestDelete_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Delete([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
}

func TestDelete_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"deleted"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Delete([]string{"my-scenario"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Validate ---

func TestValidate_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Validate([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/validate") {
			t.Errorf("path = %q, want to end with '/validate'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"errors":[]}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Validate([]string{"my-scenario"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":false,"errors":[{"message":"cert expired"}]}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Validate([]string{"my-scenario"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Ready ---

func TestReady_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Ready([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
}

func TestReady_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":true,"platforms":{"windows":{"ready":true},"linux":{"ready":false,"reason":"no GPG key"}}}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Ready([]string{"my-scenario"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Discover ---

func TestDiscover_MissingPlatform(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Discover([]string{})
	if err == nil {
		t.Fatal("expected error for missing platform")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestDiscover_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/discover/windows") {
			t.Errorf("path = %q, want to contain '/discover/windows'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"subject":"CN=Test","issuer":"CN=CA"}]`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Discover([]string{"windows"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GenerateKey ---

func TestGenerateKey_MissingRequiredArgs(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))

	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"scenario only", []string{"my-scenario"}},
		{"missing email", []string{"my-scenario", "--name", "Test"}},
		{"missing name", []string{"my-scenario", "--email", "test@test.com"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmds.GenerateKey(tc.args)
			if err == nil {
				t.Fatal("expected error for missing required args")
			}
		})
	}
}

func TestGenerateKey_Success(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"created","key_id":"ABC123","fingerprint":"DEADBEEF"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.GenerateKey([]string{"my-scenario", "--name", "Test User", "--email", "test@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["name"] != "Test User" {
		t.Errorf("name = %v, want 'Test User'", receivedBody["name"])
	}
	if receivedBody["email"] != "test@example.com" {
		t.Errorf("email = %v, want 'test@example.com'", receivedBody["email"])
	}
}

func TestGenerateKey_WithOptionalFlags(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"created","key_id":"ABC123","fingerprint":"DEADBEEF"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.GenerateKey([]string{
		"my-scenario",
		"--name", "Test User",
		"--email", "test@example.com",
		"--passphrase", "secret123",
		"--force",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["passphrase"] != "secret123" {
		t.Errorf("passphrase = %v, want 'secret123'", receivedBody["passphrase"])
	}
	if receivedBody["force"] != true {
		t.Errorf("force = %v, want true", receivedBody["force"])
	}
}

// --- Prerequisites ---

func TestPrerequisites_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"signtool","available":true}]`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Prerequisites([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- API Error Handling ---

func TestAPIError_PropagatesHTTPErrors(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal failure"}`))
	})

	cmds := New(newTestClient(handler))

	// All commands that make API calls should propagate errors
	tests := []struct {
		name string
		fn   func() error
	}{
		{"Get", func() error { return cmds.Get([]string{"s"}) }},
		{"Delete", func() error { return cmds.Delete([]string{"s"}) }},
		{"Validate", func() error { return cmds.Validate([]string{"s"}) }},
		{"Ready", func() error { return cmds.Ready([]string{"s"}) }},
		{"Discover", func() error { return cmds.Discover([]string{"windows"}) }},
		{"Prerequisites", func() error { return cmds.Prerequisites([]string{}) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Error("expected error from 500 response")
			}
		})
	}
}
