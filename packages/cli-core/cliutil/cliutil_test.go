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
