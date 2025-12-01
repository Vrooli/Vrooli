package cliutil

import (
	"net/http"
	"os"
	"path/filepath"
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
}
