package runtime

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesLifecycleEnvs(t *testing.T) {
	dataRoot := t.TempDir()
	scenarioRoot := t.TempDir()
	t.Setenv("SCENARIO_NAME", "test-genie")
	t.Setenv("VROOLI_SCENARIO", "test-genie")
	t.Setenv("SCENARIO_DATA_DIR", dataRoot)
	t.Setenv("VROOLI_VARIANT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	inheritedPath := filepath.Join(t.TempDir(), "autoheal.sqlite")
	t.Setenv("TEST_GENIE_SQLITE_PATH", inheritedPath)
	t.Setenv("SQLITE_PATH", inheritedPath)
	t.Setenv("SQLITE_DB", inheritedPath)
	t.Setenv("SCENARIOS_ROOT", scenarioRoot)
	t.Setenv("API_PORT", "4789")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.Port != "4789" {
		t.Fatalf("expected port to be 4789, got %s", cfg.Port)
	}
	expectedPath := filepath.Join(dataRoot, "test-genie.db")
	if cfg.DatabasePath != expectedPath {
		t.Fatalf("unexpected database path: %s", cfg.DatabasePath)
	}
	if !strings.HasPrefix(cfg.DatabaseDSN, "file:"+expectedPath) {
		t.Fatalf("unexpected database DSN: %s", cfg.DatabaseDSN)
	}
	expectedRoot, _ := filepath.Abs(scenarioRoot)
	if cfg.ScenariosRoot != expectedRoot {
		t.Fatalf("expected scenarios root %s, got %s", expectedRoot, cfg.ScenariosRoot)
	}
}

func TestLoadConfigRequiresPort(t *testing.T) {
	t.Setenv("SCENARIO_DATA_DIR", t.TempDir())
	t.Setenv("API_PORT", "")

	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "API_PORT") {
		t.Fatalf("expected API_PORT validation error, got %v", err)
	}
}

func TestResolveScenariosRootPrefersEnv(t *testing.T) {
	root := filepath.Join(t.TempDir(), "scenarios-root")
	t.Setenv("SCENARIOS_ROOT", root)

	got, err := resolveScenariosRoot()
	if err != nil {
		t.Fatalf("resolveScenariosRoot() error: %v", err)
	}
	expected, _ := filepath.Abs(root)
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

func TestRequireEnv(t *testing.T) {
	t.Setenv("REQUIRED_KEY", "  value  ")
	val, err := requireEnv("REQUIRED_KEY")
	if err != nil {
		t.Fatalf("requireEnv returned error: %v", err)
	}
	if val != "value" {
		t.Fatalf("expected trimmed value, got %q", val)
	}

	t.Setenv("MISSING_KEY", "")
	if _, err := requireEnv("MISSING_KEY"); err == nil || !strings.Contains(err.Error(), "MISSING_KEY") {
		t.Fatalf("expected missing env error, got %v", err)
	}
}
