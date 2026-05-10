package ports

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/secrets"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-04-13

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
			Port:     21223,
		},
	}

	env, err := manager.BuildEnvironment(item, records)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	if env.AllocatedPorts["ui"] != 21223 {
		t.Fatalf("fixed UI port = %d, want 21223", env.AllocatedPorts["ui"])
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
	home := root
	writePortRegistry(t, root, map[string]int{"postgres": 5433})
	writeSecrets(t, home, root, map[string]string{
		"POSTGRES_USER":     "tester",
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_HOST":     "localhost",
		"POSTGRES_SSLMODE":  "disable",
	})

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
	writePortRegistry(t, root, nil)

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
	writePortRegistry(t, root, nil)

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
	writePortRegistry(t, root, nil)

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
	writePortRegistry(t, root, nil)

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
	writePortRegistry(t, root, nil)

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

func TestEnsurePortClaimedReportsVrooliScenarioOwner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	previousInUse := isTCPPortInUseFn
	previousInspect := inspectPortListenersFn
	previousReadEnv := readProcessEnvironmentPortFn
	isTCPPortInUseFn = func(port int) (bool, error) { return true, nil }
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
			Listeners:  []network.PortListener{{PID: 4242}},
		}, nil
	}
	readProcessEnvironmentPortFn = func(pid int) (map[string]string, error) {
		return map[string]string{
			"VROOLI_LIFECYCLE_MANAGED": "true",
			"VROOLI_SCENARIO":          "beta",
		}, nil
	}
	t.Cleanup(func() {
		isTCPPortInUseFn = previousInUse
		inspectPortListenersFn = previousInspect
		readProcessEnvironmentPortFn = previousReadEnv
	})

	if _, err := manager.ensurePortClaimed(21236, "alpha", nil); err == nil {
		t.Fatal("expected Vrooli listener conflict")
	} else if !strings.Contains(err.Error(), `Vrooli scenario "beta"`) || !strings.Contains(err.Error(), "4242") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsurePortClaimedReportsSameScenarioVrooliListener(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	previousInUse := isTCPPortInUseFn
	previousInspect := inspectPortListenersFn
	previousReadEnv := readProcessEnvironmentPortFn
	isTCPPortInUseFn = func(port int) (bool, error) { return true, nil }
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
			Listeners:  []network.PortListener{{PID: 5151}},
		}, nil
	}
	readProcessEnvironmentPortFn = func(pid int) (map[string]string, error) {
		return map[string]string{
			"VROOLI_LIFECYCLE_MANAGED": "true",
			"VROOLI_SCENARIO":          "alpha",
		}, nil
	}
	t.Cleanup(func() {
		isTCPPortInUseFn = previousInUse
		inspectPortListenersFn = previousInspect
		readProcessEnvironmentPortFn = previousReadEnv
	})

	if _, err := manager.ensurePortClaimed(21236, "alpha", nil); err == nil {
		t.Fatal("expected same-scenario listener conflict")
	} else if !strings.Contains(err.Error(), `existing Vrooli listener for scenario "alpha"`) || !strings.Contains(err.Error(), "5151") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsurePortClaimedFallsBackToGenericConflictForNonVrooliListeners(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	previousInUse := isTCPPortInUseFn
	previousInspect := inspectPortListenersFn
	previousReadEnv := readProcessEnvironmentPortFn
	isTCPPortInUseFn = func(port int) (bool, error) { return true, nil }
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
			Listeners:  []network.PortListener{{PID: 6161}},
		}, nil
	}
	readProcessEnvironmentPortFn = func(pid int) (map[string]string, error) {
		return map[string]string{}, nil
	}
	t.Cleanup(func() {
		isTCPPortInUseFn = previousInUse
		inspectPortListenersFn = previousInspect
		readProcessEnvironmentPortFn = previousReadEnv
	})

	if _, err := manager.ensurePortClaimed(21236, "alpha", nil); err == nil {
		t.Fatal("expected generic listener conflict")
	} else if !strings.Contains(err.Error(), "port already in use") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildEnvironmentUsesTypedResourceMetadataAndSecrets(t *testing.T) {
	root := t.TempDir()
	home := root
	writePortRegistry(t, root, map[string]int{"postgres": 5433})
	writeSecrets(t, home, root, map[string]string{
		"POSTGRES_USER":     "legacy",
		"POSTGRES_PASSWORD": "secret",
	})

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
	writePortRegistry(t, root, nil)

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

func TestRemoveLockIfMatchesPreservesReplacedOwner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	manager.Now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

	if err := manager.WriteLock(21234, "alpha", 111); err != nil {
		t.Fatalf("WriteLock(alpha): %v", err)
	}
	original, exists, err := manager.ReadLock(21234)
	if err != nil {
		t.Fatalf("ReadLock(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha lock to exist")
	}

	manager.Now = func() time.Time { return time.Unix(1_700_000_100, 0).UTC() }
	if err := manager.WriteLock(21234, "beta", 222); err != nil {
		t.Fatalf("WriteLock(beta): %v", err)
	}

	if err := manager.removeLockIfMatches(original); err != nil {
		t.Fatalf("removeLockIfMatches: %v", err)
	}

	lock, exists, err := manager.ReadLock(21234)
	if err != nil {
		t.Fatalf("ReadLock(beta): %v", err)
	}
	if !exists || lock.Scenario != "beta" || lock.PID != 222 {
		t.Fatalf("lock = %#v", lock)
	}
}

func TestEnsurePortClaimedPrefersRuntimeOwnerAndRejectsReservedPorts(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, map[string]int{"postgres": 5433})

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

func TestBuildEnvironmentWithRuntimeClaimsPreventsConcurrentPortClaims(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	first := createRuntimeInstanceForPortTests(t, store, "alpha")
	second := createRuntimeInstanceForPortTests(t, store, "beta")
	port := freeLocalPort(t)
	item := fixedPortScenario(root, "service", port)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, instance := range []scenarioruntime.Instance{first, second} {
		wg.Add(1)
		go func(instance scenarioruntime.Instance) {
			defer wg.Done()
			manager, err := NewManager(root, home)
			if err != nil {
				errs <- err
				return
			}
			_, err = manager.BuildEnvironmentWithRuntimeClaims(item, nil, RuntimeClaimOptions{
				Enabled:    true,
				Context:    ctx,
				Store:      store,
				InstanceID: instance.InstanceID,
			})
			errs <- err
		}(instance)
	}
	wg.Wait()
	close(errs)

	successes := 0
	failures := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		failures++
		if !strings.Contains(err.Error(), "active registry claim already owns port") &&
			!strings.Contains(err.Error(), "locked by scenario") {
			t.Fatalf("unexpected concurrent allocation error: %v", err)
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("successes=%d failures=%d, want one success and one failure", successes, failures)
	}
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		Statuses: []string{scenarioruntime.ClaimStatusReserved, scenarioruntime.ClaimStatusBound},
	})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].Port != port {
		t.Fatalf("active claims = %#v, want one claim for %d", claims, port)
	}
}

func TestBuildEnvironmentWithRuntimeClaimsExpiresAbandonedReservedClaim(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	abandoned := createRuntimeInstanceForPortTests(t, store, "alpha")
	replacement := createRuntimeInstanceForPortTests(t, store, "beta")
	port := freeLocalPort(t)
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Minute)
	oldClaim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID: abandoned.InstanceID,
		Scenario:   abandoned.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       port,
		ExpiresAt:  &expiresAt,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim(abandoned): %v", err)
	}

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	manager.Now = func() time.Time { return now }
	env, err := manager.BuildEnvironmentWithRuntimeClaims(fixedPortScenario(root, "beta", port), nil, RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      store,
		InstanceID: replacement.InstanceID,
	})
	if err != nil {
		t.Fatalf("BuildEnvironmentWithRuntimeClaims: %v", err)
	}
	if env.AllocatedPorts["api"] != port {
		t.Fatalf("allocated api port = %d, want %d", env.AllocatedPorts["api"], port)
	}
	expired, err := store.GetInstance(ctx, abandoned.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance(abandoned): %v", err)
	}
	if expired.Status != scenarioruntime.StatusStarting {
		t.Fatalf("abandoned instance status = %q, want unchanged starting", expired.Status)
	}
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	statusByID := map[string]string{}
	for _, claim := range claims {
		statusByID[claim.ClaimID] = claim.Status
	}
	if statusByID[oldClaim.ClaimID] != scenarioruntime.ClaimStatusExpired {
		t.Fatalf("old claim status = %q, want expired; claims=%#v", statusByID[oldClaim.ClaimID], claims)
	}
	if got := env.RuntimeClaims["api"].Status; got != scenarioruntime.ClaimStatusReserved {
		t.Fatalf("new runtime claim status = %q, want reserved", got)
	}
}

func TestBoundRuntimeClaimSurvivesExpiredReservationCleanup(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	alpha := createRuntimeInstanceForPortTests(t, store, "alpha")
	beta := createRuntimeInstanceForPortTests(t, store, "beta")
	port := freeLocalPort(t)
	past := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC).Add(-time.Minute)
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID: alpha.InstanceID,
		Scenario:   alpha.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       port,
		ExpiresAt:  &past,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim(alpha): %v", err)
	}
	if _, err := store.BindPortClaim(ctx, claim.ClaimID); err != nil {
		t.Fatalf("BindPortClaim(alpha): %v", err)
	}

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if _, err := manager.BuildEnvironmentWithRuntimeClaims(fixedPortScenario(root, "beta", port), nil, RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      store,
		InstanceID: beta.InstanceID,
	}); err == nil {
		t.Fatal("expected bound active registry claim to block replacement")
	} else if !strings.Contains(err.Error(), "active registry claim already owns port") {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: alpha.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims(alpha): %v", err)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusBound || claims[0].ExpiresAt != nil {
		t.Fatalf("alpha claims = %#v, want bound with cleared expiry", claims)
	}
}

func TestRuntimeClaimReleasedWhenSocketProbeFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	alpha := createRuntimeInstanceForPortTests(t, store, "alpha")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	_, err = manager.BuildEnvironmentWithRuntimeClaims(fixedPortScenario(root, "alpha", port), nil, RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      store,
		InstanceID: alpha.InstanceID,
	})
	if err == nil {
		t.Fatal("expected socket conflict")
	}
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: alpha.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims(alpha): %v", err)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("claims after socket failure = %#v, want released claim", claims)
	}
}

func TestRuntimeClaimsAndLegacyLocksRollbackWhenLaterPortAllocationFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	alpha := createRuntimeInstanceForPortTests(t, store, "alpha")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()
	conflictPort := listener.Addr().(*net.TCPAddr).Port
	firstPort := freeLocalPort(t)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	_, err = manager.BuildEnvironmentWithRuntimeClaims(multiFixedPortScenario(root, "alpha", firstPort, conflictPort), nil, RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      store,
		InstanceID: alpha.InstanceID,
	})
	if err == nil {
		t.Fatal("expected second fixed-port socket conflict")
	}

	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: alpha.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims(alpha): %v", err)
	}
	statusByPort := map[int]string{}
	for _, claim := range claims {
		statusByPort[claim.Port] = claim.Status
	}
	if statusByPort[firstPort] != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("first claim status = %q, want released; claims=%#v", statusByPort[firstPort], claims)
	}
	if statusByPort[conflictPort] != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("conflicting claim status = %q, want released; claims=%#v", statusByPort[conflictPort], claims)
	}
	if lock, exists, err := manager.ReadLock(firstPort); err != nil {
		t.Fatalf("ReadLock(first): %v", err)
	} else if exists {
		t.Fatalf("first legacy lock should be abandoned after partial allocation failure, got %#v", lock)
	}
}

func TestLegacyStaleLockDoesNotOverrideActiveRuntimeClaim(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	alpha := createRuntimeInstanceForPortTests(t, store, "alpha")
	beta := createRuntimeInstanceForPortTests(t, store, "beta")
	port := freeLocalPort(t)
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID: alpha.InstanceID,
		Scenario:   alpha.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       port,
	}); err != nil {
		t.Fatalf("AcquirePortClaim(alpha): %v", err)
	}

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.WriteLock(port, "stale-legacy", 999999); err != nil {
		t.Fatalf("WriteLock(stale-legacy): %v", err)
	}
	_, err = manager.BuildEnvironmentWithRuntimeClaims(fixedPortScenario(root, "beta", port), nil, RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      store,
		InstanceID: beta.InstanceID,
	})
	if err == nil {
		t.Fatal("expected active registry claim to block stale legacy lock replacement")
	}
	if !strings.Contains(err.Error(), "active registry claim already owns port") {
		t.Fatalf("unexpected error: %v", err)
	}
	lock, exists, err := manager.ReadLock(port)
	if err != nil {
		t.Fatalf("ReadLock: %v", err)
	}
	if !exists || lock.Scenario != "stale-legacy" {
		t.Fatalf("legacy lock should remain diagnostic evidence, got exists=%v lock=%#v", exists, lock)
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

func fixedPortScenario(root, slug string, port int) scenario.Scenario {
	return scenario.Scenario{
		Slug: slug,
		Path: filepath.Join(root, "scenarios", slug),
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT", Port: intPtr(port)},
			},
		},
	}
}

func multiFixedPortScenario(root, slug string, apiPort, uiPort int) scenario.Scenario {
	return scenario.Scenario{
		Slug: slug,
		Path: filepath.Join(root, "scenarios", slug),
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT", Port: intPtr(apiPort)},
				"ui":  {EnvVar: "UI_PORT", Port: intPtr(uiPort)},
			},
		},
	}
}

func freeLocalPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen(127.0.0.1:0): %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("Close free-port listener: %v", err)
	}
	return port
}

func newRuntimeClaimStoreForPortTests(t *testing.T) *scenarioruntime.SQLiteStore {
	t.Helper()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close runtime store: %v", err)
		}
	})
	return store
}

func createRuntimeInstanceForPortTests(t *testing.T, store *scenarioruntime.SQLiteStore, scenarioName string) scenarioruntime.Instance {
	t.Helper()
	instance, err := store.CreateInstance(context.Background(), scenarioruntime.Instance{
		Scenario: scenarioName,
		Status:   scenarioruntime.StatusStarting,
	})
	if err != nil {
		t.Fatalf("CreateInstance(%s): %v", scenarioName, err)
	}
	return instance
}

func TestLoadResourceEnvironmentReturnsEmptyWhenMetadataMissing(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	exports, err := resources.LoadResourceEnvironment(root, home, "redis")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(redis): %v", err)
	}
	if len(exports) != 0 {
		t.Fatalf("exports = %#v, want empty", exports)
	}
}

func writePortRegistry(t *testing.T, root string, ports map[string]int) {
	t.Helper()
	ensureTypedResourceMetadata(t, root)
	testresource.WritePortRegistry(t, root, ports)
}

func writeSecrets(t *testing.T, home, root string, payload map[string]string) {
	t.Helper()
	ensureTypedResourceMetadata(t, root)
	t.Setenv(secrets.KeyEnvVar, "test-secret-passphrase")
	store := secrets.NewUserStore(home)
	if err := store.Save(payload); err != nil {
		t.Fatalf("save secrets: %v", err)
	}
}

func ensureTypedResourceMetadata(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "resources", "postgres", "resource.json")
	if _, err := os.Stat(path); err == nil {
		return
	}
	testresource.WriteResourceManifest(t, root, "postgres", manifestpkg.ResourceManifest{
		Name:            "postgres",
		Driver:          "docker-service",
		Template:        "docker-service",
		PortabilityTier: "full",
		Ports:           []manifestpkg.ResourcePort{{Name: "postgresql", Container: 5432, Host: 5433}},
		Runtime: manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
			Env: map[string]string{
				"POSTGRES_DB":   "vrooli",
				"POSTGRES_USER": "vrooli",
			},
		},
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Static:         map[string]string{"POSTGRES_HOST": "localhost", "POSTGRES_SSLMODE": "disable"},
			FromPorts:      map[string]string{"POSTGRES_PORT": "postgresql"},
			FromRuntimeEnv: []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD"},
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"POSTGRES_URL": {Template: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE}"},
				"DATABASE_URL": {Template: "${POSTGRES_URL}"},
			},
		},
	})
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
