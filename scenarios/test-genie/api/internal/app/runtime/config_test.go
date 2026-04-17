package runtime

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesLifecycleEnvs(t *testing.T) {
	dataRoot := t.TempDir()
	scenarioRoot := t.TempDir()
	t.Setenv("TEST_GENIE_SQLITE_PATH", filepath.Join(dataRoot, "custom.db"))
	t.Setenv("SCENARIOS_ROOT", scenarioRoot)
	t.Setenv("API_PORT", "4789")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.Port != "4789" {
		t.Fatalf("expected port to be 4789, got %s", cfg.Port)
	}
	expectedPath := filepath.Join(dataRoot, "custom.db")
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

func TestResolveDatabaseConfigPrefersExplicitValue(t *testing.T) {
	t.Setenv("TEST_GENIE_SQLITE_PATH", filepath.Join(t.TempDir(), "explicit.db"))
	t.Setenv("SCENARIO_DATA_DIR", filepath.Join(t.TempDir(), "ignored"))

	cfg, err := resolveDatabaseConfig()
	if err != nil {
		t.Fatalf("resolveDatabaseConfig() error: %v", err)
	}
	if !strings.HasSuffix(cfg.Path, "explicit.db") {
		t.Fatalf("expected explicit sqlite path, got %s", cfg.Path)
	}
}

func TestResolveDatabaseConfigFallsBackToScenarioDataDir(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("TEST_GENIE_SQLITE_PATH", "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")
	t.Setenv("SCENARIO_DATA_DIR", dataRoot)

	cfg, err := resolveDatabaseConfig()
	if err != nil {
		t.Fatalf("resolveDatabaseConfig() error: %v", err)
	}
	expectedPath := filepath.Join(dataRoot, "test-genie.db")
	if cfg.Path != expectedPath {
		t.Fatalf("expected %s, got %s", expectedPath, cfg.Path)
	}
}

func TestResolveDatabaseConfigFallsBackToStorageResolver(t *testing.T) {
	t.Setenv("TEST_GENIE_SQLITE_PATH", "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")
	t.Setenv("SCENARIO_DATA_DIR", "")
	t.Setenv("SQLITE_DATABASE_PATH", "")
	t.Setenv("VROOLI_DATA", "")

	cfg, err := resolveDatabaseConfig()
	if err != nil {
		t.Fatalf("resolveDatabaseConfig() error: %v", err)
	}

	if !strings.HasSuffix(cfg.Path, "test-genie.db") {
		t.Fatalf("expected fallback path to end with test-genie.db, got %s", cfg.Path)
	}
	if !filepath.IsAbs(cfg.Path) {
		t.Fatalf("expected fallback path to be absolute, got %s", cfg.Path)
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
