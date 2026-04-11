package ports

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-10

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

func TestCleanStaleLocksPreservesLiveOwners(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.WriteLock(21234, "alpha", os.Getpid()); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}

	if err := manager.CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")); err != nil {
		t.Fatalf("expected live-owner lock to remain: %v", err)
	}
}

func TestRemoveScenarioLocksOnlyRemovesMatchingScenario(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.WriteLock(21234, "alpha", os.Getpid()); err != nil {
		t.Fatalf("WriteLock(alpha): %v", err)
	}
	if err := manager.WriteLock(21235, "beta", os.Getpid()); err != nil {
		t.Fatalf("WriteLock(beta): %v", err)
	}
	stateFile := filepath.Join(home, ".vrooli", "state", "scenarios", "alpha.json")
	if err := os.WriteFile(stateFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", stateFile, err)
	}

	locks, err := manager.LocksForScenario("alpha")
	if err != nil {
		t.Fatalf("LocksForScenario(alpha): %v", err)
	}
	if len(locks) != 1 || locks[0].Port != 21234 {
		t.Fatalf("alpha locks = %#v", locks)
	}

	if err := manager.RemoveScenarioLocks("alpha"); err != nil {
		t.Fatalf("RemoveScenarioLocks(alpha): %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")); !os.IsNotExist(err) {
		t.Fatalf("expected alpha lock removal, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21235.lock")); err != nil {
		t.Fatalf("expected beta lock to remain: %v", err)
	}
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Fatalf("expected scenario state file removal, stat err=%v", err)
	}
}

func TestEnsurePortClaimedRejectsRecentForeignStaleLock(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	manager.Now = func() time.Time { return now }

	if err := manager.WriteLock(21234, "beta", 999999); err != nil {
		t.Fatalf("WriteLock(beta): %v", err)
	}

	if _, err := manager.ensurePortClaimed(21234, "alpha", nil); err == nil {
		t.Fatalf("expected recent foreign stale lock to block port claim")
	} else if !strings.Contains(err.Error(), "recent stale lock held by scenario") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsurePortClaimedRejectsLiveForeignLock(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.WriteLock(21236, "beta", os.Getpid()); err != nil {
		t.Fatalf("WriteLock(beta): %v", err)
	}

	if _, err := manager.ensurePortClaimed(21236, "alpha", nil); err == nil {
		t.Fatalf("expected live foreign owner to block port claim")
	} else if !strings.Contains(err.Error(), `locked by scenario "beta"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildEnvironmentFallsBackToLegacyDefaultsExports(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, `declare -g -A RESOURCE_PORTS=(["postgres"]="5433")`)

	defaultsPath := filepath.Join(root, "resources", "postgres", "config", "defaults.sh")
	if err := os.MkdirAll(filepath.Dir(defaultsPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(defaultsPath), err)
	}
	if err := os.WriteFile(defaultsPath, []byte("#!/usr/bin/env bash\nexport POSTGRES_USER=legacy\nexport POSTGRES_PASSWORD=secret\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", defaultsPath, err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: filepath.Join(root, "scenarios", "alpha"),
		Manifest: scenario.ServiceManifest{
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{
					"postgres": {Enabled: true},
				},
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
	env, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}

	if env.EnvVars["POSTGRES_USER"] != "legacy" {
		t.Fatalf("POSTGRES_USER = %q, want legacy", env.EnvVars["POSTGRES_USER"])
	}
	if env.EnvVars["POSTGRES_PASSWORD"] != "secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want secret", env.EnvVars["POSTGRES_PASSWORD"])
	}
}

func TestClaimLockAllowsSameScenarioAndRejectsForeignOwner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	if err := manager.claimLock(21234, "alpha", 111); err != nil {
		t.Fatalf("claimLock(alpha first): %v", err)
	}
	if err := manager.claimLock(21234, "alpha", 222); err != nil {
		t.Fatalf("claimLock(alpha rewrite): %v", err)
	}

	lock, exists, err := manager.ReadLock(21234)
	if err != nil {
		t.Fatalf("ReadLock: %v", err)
	}
	if !exists || lock.Scenario != "alpha" || lock.PID != 222 {
		t.Fatalf("lock = %#v", lock)
	}

	if err := manager.claimLock(21234, "beta", 333); err == nil {
		t.Fatalf("expected foreign scenario to be rejected")
	} else if !strings.Contains(err.Error(), `locked by scenario "alpha"`) {
		t.Fatalf("unexpected foreign lock error: %v", err)
	}

	emptyLock := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21235.lock")
	if err := os.WriteFile(emptyLock, []byte("\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", emptyLock, err)
	}
	if err := manager.claimLock(21235, "alpha", 444); err != nil {
		t.Fatalf("claimLock(empty existing file): %v", err)
	}
}

func TestEnsurePortClaimedPrefersRuntimeOwnerAndRejectsReservedPorts(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, `declare -g -A RESOURCE_PORTS=(["postgres"]="5433")`)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	pid, err := manager.ensurePortClaimed(21236, "alpha", []process.Record{{
		PID:  os.Getpid(),
		Port: 21236,
	}})
	if err != nil {
		t.Fatalf("ensurePortClaimed(runtime owner): %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("pid = %d, want %d", pid, os.Getpid())
	}

	lock, exists, err := manager.ReadLock(21236)
	if err != nil {
		t.Fatalf("ReadLock: %v", err)
	}
	if !exists || lock.Scenario != "alpha" {
		t.Fatalf("lock = %#v", lock)
	}

	if _, err := manager.ensurePortClaimed(5433, "alpha", nil); err == nil {
		t.Fatalf("expected reserved resource port to fail")
	} else if !strings.Contains(err.Error(), "reserved for resource") {
		t.Fatalf("unexpected reserved-port error: %v", err)
	}
}

func TestParseRangeAndIsTCPPortInUse(t *testing.T) {
	start, end, err := parseRange("21000-21010")
	if err != nil {
		t.Fatalf("parseRange valid: %v", err)
	}
	if start != 21000 || end != 21010 {
		t.Fatalf("range = %d-%d", start, end)
	}

	if _, _, err := parseRange("invalid"); err == nil {
		t.Fatalf("expected invalid range to fail")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	inUse, err := isTCPPortInUse(port)
	if err != nil {
		t.Fatalf("isTCPPortInUse(occupied): %v", err)
	}
	if !inUse {
		t.Fatalf("expected live listener to mark port in use")
	}

	_ = listener.Close()
	inUse, err = isTCPPortInUse(port)
	if err != nil {
		t.Fatalf("isTCPPortInUse(released): %v", err)
	}
	if inUse {
		t.Fatalf("expected released listener to free port")
	}
}

func TestLoadResourceExportsReturnsEmptyWhenConfigMissing(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, "declare -g -A RESOURCE_PORTS=()")

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	exports, err := manager.loadResourceExports("redis")
	if err != nil {
		t.Fatalf("loadResourceExports(redis): %v", err)
	}
	if len(exports) != 0 {
		t.Fatalf("exports = %#v, want empty", exports)
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
