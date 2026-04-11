package ports

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func TestBuildEnvironmentHonorsRealTestGenieContract(t *testing.T) {
	root := repoRoot(t)
	home := t.TempDir()

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	item, err := scenario.Load(root, "test-genie", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load test-genie: %v", err)
	}

	records := []process.Record{
		{
			PID:      os.Getpid(),
			Scenario: item.Slug,
			Step:     "start-ui",
			Port:     36223,
		},
	}

	env, err := manager.BuildEnvironment(item, records)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	if env.AllocatedPorts["ui"] != 36223 {
		t.Fatalf("fixed UI port = %d, want 36223", env.AllocatedPorts["ui"])
	}
	if env.AllocatedPorts["api"] < 15000 || env.AllocatedPorts["api"] > 19999 {
		t.Fatalf("API port = %d outside expected range", env.AllocatedPorts["api"])
	}
	if env.AllocatedPorts["websocket"] < 25000 || env.AllocatedPorts["websocket"] > 29999 {
		t.Fatalf("WS port = %d outside expected range", env.AllocatedPorts["websocket"])
	}
	wantSQLite := filepath.Join(item.Path, "data", "test-genie.db")
	if env.EnvVars["TEST_GENIE_SQLITE_PATH"] != wantSQLite {
		t.Fatalf("TEST_GENIE_SQLITE_PATH = %q, want %q", env.EnvVars["TEST_GENIE_SQLITE_PATH"], wantSQLite)
	}
}

func TestBuildEnvironmentHonorsRealSimpleTestPostgresContract(t *testing.T) {
	root := repoRoot(t)
	home := t.TempDir()

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	item, err := scenario.Load(root, "simple-test", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load simple-test: %v", err)
	}

	env, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	if env.EnvVars["POSTGRES_DB"] != "vrooli_simple_test" {
		t.Fatalf("POSTGRES_DB = %q, want vrooli_simple_test", env.EnvVars["POSTGRES_DB"])
	}
}

func TestBuildEnvironmentAllocatesPortsAndExpandsScenarioEnv(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, `declare -g -A RESOURCE_PORTS=(["postgres"]="5433")`)
	writeResourceExports(t, root, "postgres", "#!/usr/bin/env bash\nexport POSTGRES_USER=tester\nexport POSTGRES_PASSWORD=secret\nexport POSTGRES_HOST=localhost\nexport POSTGRES_SSLMODE=disable\n")

	item := scenario.Scenario{
		Slug: "alpha",
		Path: filepath.Join(root, "scenarios", "alpha"),
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT", Range: "21000-21010"},
				"ui":  {EnvVar: "UI_PORT", Port: intPtr(21100)},
			},
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{
					"postgres": {Enabled: true, Database: "alpha_db"},
				},
			},
			Environment: map[string]string{
				"SQLITE_PATH": "${SCENARIO_DATA_DIR}/alpha.db",
			},
		},
	}
	if err := os.MkdirAll(item.Path, 0o755); err != nil {
		t.Fatalf("mkdir scenario path: %v", err)
	}

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	first, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	second, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment second call: %v", err)
	}

	if first.AllocatedPorts["ui"] != 21100 {
		t.Fatalf("fixed UI port = %d, want 21100", first.AllocatedPorts["ui"])
	}
	if first.AllocatedPorts["api"] < 21000 || first.AllocatedPorts["api"] > 21010 {
		t.Fatalf("API_PORT = %d outside expected range", first.AllocatedPorts["api"])
	}
	if first.AllocatedPorts["api"] != second.AllocatedPorts["api"] {
		t.Fatalf("expected same-scenario API port reuse, got %d then %d", first.AllocatedPorts["api"], second.AllocatedPorts["api"])
	}
	if first.EnvVars["POSTGRES_PORT"] != "5433" {
		t.Fatalf("POSTGRES_PORT = %q", first.EnvVars["POSTGRES_PORT"])
	}
	if first.EnvVars["POSTGRES_DB"] != "alpha_db" {
		t.Fatalf("POSTGRES_DB = %q", first.EnvVars["POSTGRES_DB"])
	}
	if !strings.Contains(first.EnvVars["DATABASE_URL"], "/alpha_db?") {
		t.Fatalf("DATABASE_URL = %q", first.EnvVars["DATABASE_URL"])
	}
	if want := filepath.Join(item.Path, "data", "alpha.db"); first.EnvVars["SQLITE_PATH"] != want {
		t.Fatalf("SQLITE_PATH = %q, want %q", first.EnvVars["SQLITE_PATH"], want)
	}
}

func TestCleanStaleLocksRemovesDeadOwners(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.WriteLock(21234, "alpha", 999999); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}

	if err := manager.CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock removal, stat err=%v", err)
	}
}

func writePortRegistry(t *testing.T, root, contents string) {
	t.Helper()
	path := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := "#!/usr/bin/env bash\n" + contents + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeResourceExports(t *testing.T, root, resource, contents string) {
	t.Helper()
	path := filepath.Join(root, "resources", resource, "config", "exports.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func intPtr(value int) *int {
	return &value
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
