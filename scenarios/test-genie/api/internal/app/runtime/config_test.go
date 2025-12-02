package runtime

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesLifecycleEnvs(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/test?sslmode=disable")
	scenarioRoot := t.TempDir()
	t.Setenv("SCENARIOS_ROOT", scenarioRoot)
	t.Setenv("API_PORT", "4789")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.Port != "4789" {
		t.Fatalf("expected port to be 4789, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/test?sslmode=disable" {
		t.Fatalf("unexpected database URL: %s", cfg.DatabaseURL)
	}
	expectedRoot, _ := filepath.Abs(scenarioRoot)
	if cfg.ScenariosRoot != expectedRoot {
		t.Fatalf("expected scenarios root %s, got %s", expectedRoot, cfg.ScenariosRoot)
	}
}

func TestLoadConfigRequiresPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/test?sslmode=disable")
	t.Setenv("SCENARIOS_ROOT", t.TempDir())
	t.Setenv("API_PORT", "")

	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "API_PORT") {
		t.Fatalf("expected API_PORT validation error, got %v", err)
	}
}

func TestResolveDatabaseURLPrefersExplicitValue(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://direct/url")
	t.Setenv("POSTGRES_HOST", "ignored")
	url, err := resolveDatabaseURL()
	if err != nil {
		t.Fatalf("resolveDatabaseURL() error: %v", err)
	}
	if url != "postgres://direct/url" {
		t.Fatalf("expected direct database URL, got %s", url)
	}
}

func TestResolveDatabaseURLBuildsFromParts(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "pguser")
	t.Setenv("POSTGRES_PASSWORD", "pgpass")
	t.Setenv("POSTGRES_DB", "pgdb")

	got, err := resolveDatabaseURL()
	if err != nil {
		t.Fatalf("resolveDatabaseURL() error: %v", err)
	}
	expected := "postgres://pguser:pgpass@localhost:5432/pgdb?sslmode=disable"
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

func TestResolveDatabaseURLErrorsWhenIncomplete(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "")

	if _, err := resolveDatabaseURL(); err == nil {
		t.Fatal("expected error for incomplete postgres env settings")
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
