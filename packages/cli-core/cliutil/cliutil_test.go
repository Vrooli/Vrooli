package cliutil

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
