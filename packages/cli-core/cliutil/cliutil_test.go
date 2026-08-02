package cliutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDetermineAPIBasePrecedence(t *testing.T) {
	t.Setenv("TEST_ENV_BASE", "http://from-env")
	t.Setenv("TEST_PORT_ENV", "9999")

	opts := APIBaseOptions{
		Override:    "http://override",
		EnvVars:     []string{"TEST_ENV_BASE"},
		ConfigBase:  "http://config",
		PortEnvVars: []string{"TEST_PORT_ENV"},
		PortDetector: func() string {
			return "1111"
		},
		DefaultBase: "http://default",
	}

	base := DetermineAPIBase(opts)
	if base != "http://override" {
		t.Fatalf("expected override to win, got %s", base)
	}

	opts.Override = ""
	base = DetermineAPIBase(opts)
	if base != "http://from-env" {
		t.Fatalf("expected env to win, got %s", base)
	}

	t.Setenv("TEST_ENV_BASE", "")
	base = DetermineAPIBase(opts)
	if base != "http://config" {
		t.Fatalf("expected config to win, got %s", base)
	}

	opts.ConfigBase = ""
	base = DetermineAPIBase(opts)
	if base != "http://localhost:9999" {
		t.Fatalf("expected port env to win, got %s", base)
	}

	t.Setenv("TEST_PORT_ENV", "")
	base = DetermineAPIBase(opts)
	if base != "http://localhost:1111" {
		t.Fatalf("expected port detector to win, got %s", base)
	}

	opts.PortDetector = nil
	base = DetermineAPIBase(opts)
	if base != "http://default" {
		t.Fatalf("expected default base, got %s", base)
	}
}

func TestDetermineAPIBaseIgnoresGenericBaseEnvInAgentContext(t *testing.T) {
	t.Setenv(EnvIdentityToken, "tok")
	t.Setenv("API_BASE_URL", "http://wrong.example")
	t.Setenv("VITE_API_BASE_URL", "http://also-wrong.example")
	t.Setenv("DEMO_API_BASE", "http://scenario.example")

	base := DetermineAPIBase(APIBaseOptions{
		EnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL", "DEMO_API_BASE"},
	})
	if base != "http://scenario.example" {
		t.Fatalf("expected scenario-specific env to win, got %s", base)
	}
}

func TestDetermineAPIBaseKeepsGenericBaseEnvOutsideAgentContext(t *testing.T) {
	t.Setenv("API_BASE_URL", "http://generic.example")

	base := DetermineAPIBase(APIBaseOptions{
		EnvVars: []string{"API_BASE_URL"},
	})
	if base != "http://generic.example" {
		t.Fatalf("expected generic env outside agent context, got %s", base)
	}
}

func TestDetermineAPIBaseIgnoresGenericAPIPortLeakage(t *testing.T) {
	t.Setenv(EnvIdentityToken, "tok")
	t.Setenv("API_PORT", "18800")

	base := DetermineAPIBase(APIBaseOptions{
		PortEnvVars:  []string{"SWARM_MANAGER_API_PORT"},
		PortDetector: func() string { return "15000" },
	})
	if base != "http://localhost:15000" {
		t.Fatalf("expected lifecycle detector to win over generic API_PORT, got %s", base)
	}
}

func TestResolveSourceRoot(t *testing.T) {
	temp := t.TempDir()
	child := filepath.Join(temp, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("SOURCE_ROOT_ENV", child)

	root := ResolveSourceRoot("unknown", "SOURCE_ROOT_ENV")
	if root != child {
		t.Fatalf("expected env root %s, got %s", child, root)
	}

	t.Setenv("SOURCE_ROOT_ENV", "")
	root = ResolveSourceRoot("unknown", "SOURCE_ROOT_ENV")
	if root != "" {
		t.Fatalf("expected empty when unresolved, got %s", root)
	}
}

func TestResolveConfigDirPrefersEnv(t *testing.T) {
	temp := t.TempDir()
	override := filepath.Join(temp, "custom")
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", override)

	dir, err := ResolveConfigDir("demo", "CLI_CONFIG_DIR_OVERRIDE")
	if err != nil {
		t.Fatalf("ResolveConfigDir: %v", err)
	}
	if dir != override {
		t.Fatalf("expected override dir %s, got %s", override, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("expected directory to exist at %s", dir)
	}
}

func TestLoadAPIConfigRoundTrip(t *testing.T) {
	temp := t.TempDir()
	override := filepath.Join(temp, "cfg")
	t.Setenv("APP_CONFIG_DIR", override)

	file, cfg, err := LoadAPIConfig("demo", "APP_CONFIG_DIR")
	if err != nil {
		t.Fatalf("LoadAPIConfig: %v", err)
	}
	if cfg.APIBase != "" || cfg.Token != "" {
		t.Fatalf("expected empty config by default")
	}

	updated := APIConfig{APIBase: "http://example.com", Token: "secret"}
	if err := file.Save(updated); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var reloaded APIConfig
	if err := file.Load(&reloaded); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded != updated {
		t.Fatalf("expected %+v, got %+v", updated, reloaded)
	}
}

func TestConfigFileLoadSave(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "nested", "config.json")
	cfg, err := NewConfigFile(path)
	if err != nil {
		t.Fatalf("NewConfigFile: %v", err)
	}

	type sample struct {
		Name string `json:"name"`
	}
	expected := sample{Name: "test"}
	if err := cfg.Save(expected); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var loaded sample
	if err := cfg.Load(&loaded); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != expected {
		t.Fatalf("loaded mismatch: %+v", loaded)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(cfg.Path)
		if err != nil {
			t.Fatalf("stat config: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("expected config file permissions 600, got %v", info.Mode().Perm())
		}
		dirInfo, err := os.Stat(filepath.Dir(cfg.Path))
		if err != nil {
			t.Fatalf("stat config dir: %v", err)
		}
		if dirInfo.Mode().Perm() != 0o700 {
			t.Fatalf("expected config dir permissions 700, got %v", dirInfo.Mode().Perm())
		}
	}
}

func TestResolveConfigDirUsesNamespacedDefault(t *testing.T) {
	temp := t.TempDir()
	cfgRoot := filepath.Join(temp, "cfg")
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", cfgRoot)
	} else {
		t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	}

	dir, err := ResolveConfigDir("demo-app")
	if err != nil {
		t.Fatalf("ResolveConfigDir: %v", err)
	}

	expected := filepath.Join(cfgRoot, "vrooli", "demo-app")
	if dir != expected {
		t.Fatalf("expected namespaced config dir %s, got %s", expected, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("expected directory to exist: %v", err)
	}
}

func TestResolveConfigDirFallsBackToLegacyWhenPresent(t *testing.T) {
	temp := t.TempDir()
	cfgRoot := filepath.Join(temp, "cfg")
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", cfgRoot)
	} else {
		t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	}
	legacyDir := filepath.Join(cfgRoot, "demo-legacy")
	if err := os.MkdirAll(legacyDir, 0o700); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte(`{"api_base":"http://legacy"}`), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	dir, err := ResolveConfigDir("demo-legacy")
	if err != nil {
		t.Fatalf("ResolveConfigDir: %v", err)
	}
	if dir != legacyDir {
		t.Fatalf("expected to reuse legacy dir %s, got %s", legacyDir, dir)
	}
}

func TestResolveTimeout(t *testing.T) {
	fallback := 10 * time.Second
	if got := ResolveTimeout([]string{"MISSING"}, fallback); got != fallback {
		t.Fatalf("expected fallback timeout, got %v", got)
	}

	t.Setenv("TIMEOUT_SECS", "45")
	if got := ResolveTimeout([]string{"TIMEOUT_SECS"}, fallback); got != 45*time.Second {
		t.Fatalf("expected 45s, got %v", got)
	}

	t.Setenv("TIMEOUT_DURATION", "2m")
	if got := ResolveTimeout([]string{"TIMEOUT_DURATION", "TIMEOUT_SECS"}, fallback); got != 2*time.Minute {
		t.Fatalf("expected 2m, got %v", got)
	}

	t.Setenv("TIMEOUT_BAD", "not-a-duration")
	if got := ResolveTimeout([]string{"TIMEOUT_BAD", "TIMEOUT_DURATION"}, fallback); got != 2*time.Minute {
		t.Fatalf("expected to skip bad value and use next, got %v", got)
	}
}

func TestHTTPClientBaseValidation(t *testing.T) {
	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: ""},
	})
	if _, err := client.Do(http.MethodGet, "/health", nil, nil); err == nil || !strings.Contains(err.Error(), "api base URL is empty") {
		t.Fatalf("expected empty base error, got %v", err)
	}

	client = NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: "::::"},
	})
	if _, err := client.Do(http.MethodGet, "/health", nil, nil); err == nil || !strings.Contains(err.Error(), "invalid api base URL") {
		t.Fatalf("expected invalid base error, got %v", err)
	}

	client = NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: "http://"},
	})
	if _, err := client.Do(http.MethodGet, "/health", nil, nil); err == nil || !strings.Contains(err.Error(), "invalid api base URL") {
		t.Fatalf("expected invalid host error, got %v", err)
	}
}

func TestValidateAPIBase(t *testing.T) {
	_, err := ValidateAPIBase(APIBaseOptions{EnvVars: []string{"MISSING"}})
	if err == nil || !strings.Contains(err.Error(), "api base URL is empty") {
		t.Fatalf("expected empty base error, got %v", err)
	}

	_, err = ValidateAPIBase(APIBaseOptions{DefaultBase: "::::"})
	if err == nil || !strings.Contains(err.Error(), "invalid api base URL") {
		t.Fatalf("expected invalid base error, got %v", err)
	}

	base, err := ValidateAPIBase(APIBaseOptions{DefaultBase: "http://localhost:1234"})
	if err != nil {
		t.Fatalf("validate base: %v", err)
	}
	if base != "http://localhost:1234" {
		t.Fatalf("unexpected base: %s", base)
	}
}

func TestHTTPClientTimeoutOverride(t *testing.T) {
	client := NewHTTPClient(HTTPClientOptions{Timeout: 5 * time.Second})
	if client.client.Timeout != 5*time.Second {
		t.Fatalf("expected timeout override, got %v", client.client.Timeout)
	}
}

func TestHTTPClientRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: server.URL},
		Timeout:     0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.DoWithContext(ctx, http.MethodGet, "/slow", nil, nil)
	if err == nil {
		t.Fatalf("expected context cancellation error")
	}
}

func TestHTTPClientHeaderSourceInjectsHeaders(t *testing.T) {
	gotHeaders := make(http.Header)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			gotHeaders[k] = v
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: server.URL},
	})

	calls := 0
	client.SetHeaderSource(func() map[string]string {
		calls++
		return map[string]string{
			"X-Vrooli-Attribution": "test-value",
			"X-Skip-Empty":         "",
			"X-Custom":             "another",
		}
	})

	if _, err := client.Do(http.MethodGet, "/test", nil, nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected header source called once, got %d", calls)
	}
	if got := gotHeaders.Get("X-Vrooli-Attribution"); got != "test-value" {
		t.Errorf("X-Vrooli-Attribution = %q, want test-value", got)
	}
	if got := gotHeaders.Get("X-Custom"); got != "another" {
		t.Errorf("X-Custom = %q, want another", got)
	}
	if _, present := gotHeaders["X-Skip-Empty"]; present {
		t.Errorf("empty-value header should be skipped, got %v", gotHeaders["X-Skip-Empty"])
	}

	// Second request: source must be re-invoked.
	if _, err := client.Do(http.MethodGet, "/test", nil, nil); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected header source called per-request, got %d", calls)
	}
}

func TestHTTPClientHeaderSourceClearsWhenNil(t *testing.T) {
	gotHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: server.URL},
	})

	client.SetHeaderSource(func() map[string]string {
		return map[string]string{"X-Custom": "set"}
	})
	if _, err := client.Do(http.MethodGet, "/test", nil, nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if gotHeader != "set" {
		t.Fatalf("expected X-Custom=set, got %q", gotHeader)
	}

	client.SetHeaderSource(nil)
	gotHeader = ""
	if _, err := client.Do(http.MethodGet, "/test", nil, nil); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if gotHeader != "" {
		t.Fatalf("expected X-Custom cleared, got %q", gotHeader)
	}
}

func TestHTTPClientApplicationAndInvocationHeaderSourcesCompose(t *testing.T) {
	gotHeaders := make(http.Header)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{BaseOptions: APIBaseOptions{DefaultBase: server.URL}})
	client.SetHeaderSource(func() map[string]string { return map[string]string{"X-Vrooli-Attribution": "required"} })
	client.SetInvocationHeaderSource(func() map[string]string { return map[string]string{"X-Vrooli-Invocation-ID": "invocation"} })

	if _, err := client.Do(http.MethodPost, "/capture", nil, map[string]string{"ok": "true"}); err != nil {
		t.Fatalf("request: %v", err)
	}
	if got := gotHeaders.Get("X-Vrooli-Attribution"); got != "required" {
		t.Fatalf("attribution header = %q, want required", got)
	}
	if got := gotHeaders.Get("X-Vrooli-Invocation-ID"); got != "invocation" {
		t.Fatalf("invocation header = %q, want invocation", got)
	}
}

func TestIdentityForwardingTransportAddsProcessIdentityForRawClient(t *testing.T) {
	t.Setenv(EnvIdentityToken, "opaque-agent-token")
	gotIdentity := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIdentity = r.Header.Get(HeaderAgentIdentityToken)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &http.Client{Transport: identityForwardingTransport{next: http.DefaultTransport}}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("raw client request: %v", err)
	}
	resp.Body.Close()
	if gotIdentity != "opaque-agent-token" {
		t.Fatalf("identity header = %q, want process token", gotIdentity)
	}
}

func TestIdentityForwardingTransportPreservesExplicitIdentity(t *testing.T) {
	t.Setenv(EnvIdentityToken, "environment-token")
	gotIdentity := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIdentity = r.Header.Get(HeaderAgentIdentityToken)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(HeaderAgentIdentityToken, "explicit-token")
	resp, err := identityForwardingTransport{next: http.DefaultTransport}.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	resp.Body.Close()
	if gotIdentity != "explicit-token" {
		t.Fatalf("identity header = %q, want explicit token", gotIdentity)
	}
}

func TestStringListFlagCollectsValues(t *testing.T) {
	var list StringList
	list.Set("a")
	list.Set("b")
	values := list.Values()
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("unexpected values: %+v", values)
	}
	values[0] = "mutated"
	// Ensure Values() returns a copy.
	second := list.Values()
	if second[0] != "a" {
		t.Fatalf("expected copy to remain unchanged, got %+v", second)
	}
}

func TestParseCSVAndMergeArgs(t *testing.T) {
	parsed := ParseCSV("a, b, ,c")
	if len(parsed) != 3 || parsed[1] != "b" {
		t.Fatalf("unexpected parsed csv: %+v", parsed)
	}

	merged := MergeArgs([]string{"one"}, []string{"", "two", " three "})
	if len(merged) != 3 || merged[2] != "three" {
		t.Fatalf("unexpected merged args: %+v", merged)
	}
}

func TestReadFileString(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "file.txt")
	content := "hello\nworld"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	got, err := ReadFileString(path)
	if err != nil {
		t.Fatalf("ReadFileString: %v", err)
	}
	if got != content {
		t.Fatalf("expected %q, got %q", content, got)
	}
}
