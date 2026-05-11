package ports

import (
	"context"
	"fmt"
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

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-05-11

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

	env, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	if env.AllocatedPorts["ui"] != 21223 {
		t.Fatalf("fixed UI port = %d, want 21223", env.AllocatedPorts["ui"])
	}
	if env.AllocatedPorts["api"] < 15000 || env.AllocatedPorts["api"] > 19999 {
		t.Fatalf("API port = %d outside expected range", env.AllocatedPorts["api"])
	}
	if _, ok := env.AllocatedPorts["websocket"]; ok {
		t.Fatalf("unexpected websocket port allocation for test-genie")
	}
	if _, ok := env.EnvVars["WS_PORT"]; ok {
		t.Fatalf("unexpected WS_PORT env var for test-genie")
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

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: filepath.Join(root, "scenarios", "alpha"),
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT", Range: "15000-15099"},
				"ui":  {EnvVar: "UI_PORT", Range: "15100-15199"},
			},
			Environment: map[string]string{
				"DERIVED_URL": "http://localhost:${API_PORT}",
			},
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

	env, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	if env.AllocatedPorts["api"] < 15000 || env.AllocatedPorts["api"] > 15099 {
		t.Fatalf("api allocated outside range: %d", env.AllocatedPorts["api"])
	}
	if env.AllocatedPorts["ui"] < 15100 || env.AllocatedPorts["ui"] > 15199 {
		t.Fatalf("ui allocated outside range: %d", env.AllocatedPorts["ui"])
	}
	wantDerived := fmt.Sprintf("http://localhost:%d", env.AllocatedPorts["api"])
	if env.EnvVars["DERIVED_URL"] != wantDerived {
		t.Fatalf("DERIVED_URL = %q, want %q", env.EnvVars["DERIVED_URL"], wantDerived)
	}

	// Re-building must produce identical allocations: the hash-derived offset
	// is stable when there is no contention.
	env2, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment(retry): %v", err)
	}
	if env2.AllocatedPorts["api"] != env.AllocatedPorts["api"] || env2.AllocatedPorts["ui"] != env.AllocatedPorts["ui"] {
		t.Fatalf("retry allocations differ: %v vs %v", env2.AllocatedPorts, env.AllocatedPorts)
	}
}

// TestCleanStaleLocksRemovesLegacyArtifacts proves that CleanStaleLocks
// remains as a one-time cleanup utility for legacy `.port_<port>.lock` files,
// but that the files themselves no longer participate in port ownership.
func TestCleanStaleLocksRemovesLegacyArtifacts(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.EnsureStateDir(); err != nil {
		t.Fatalf("EnsureStateDir: %v", err)
	}
	writeLegacyLockFile(t, manager, 21234, "alpha", 999999, time.Unix(1_700_000_000, 0))
	writeLegacyLockFile(t, manager, 21235, "beta", os.Getpid(), time.Now())

	if err := manager.CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if _, err := os.Stat(manager.lockPath(21234)); !os.IsNotExist(err) {
		t.Fatalf("expected stale legacy lock removal, stat err=%v", err)
	}
	if _, err := os.Stat(manager.lockPath(21235)); err != nil {
		t.Fatalf("expected live-owner legacy lock to remain as diagnostic: %v", err)
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
	if err := manager.EnsureStateDir(); err != nil {
		t.Fatalf("EnsureStateDir: %v", err)
	}
	writeLegacyLockFile(t, manager, 21234, "alpha", os.Getpid(), time.Now())
	writeLegacyLockFile(t, manager, 21235, "beta", os.Getpid(), time.Now())
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

	if _, err := os.Stat(manager.lockPath(21234)); !os.IsNotExist(err) {
		t.Fatalf("expected alpha lock removal, stat err=%v", err)
	}
	if _, err := os.Stat(manager.lockPath(21235)); err != nil {
		t.Fatalf("expected beta lock to remain: %v", err)
	}
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Fatalf("expected scenario state file removal, stat err=%v", err)
	}
}

// TestEnsurePortBindableRejectsLiveForeignListener confirms the bind-probe
// path: when a TCP listener exists from a foreign Vrooli scenario, the
// allocator surfaces a structured conflict message.
func TestEnsurePortBindableRejectsLiveForeignListener(t *testing.T) {
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
	isTCPPortInUseFn = func(int) (bool, error) { return true, nil }
	inspectPortListenersFn = func(int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
			Listeners:  []network.PortListener{{PID: 4242}},
		}, nil
	}
	readProcessEnvironmentPortFn = func(int) (map[string]string, error) {
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

	if err := manager.ensurePortBindable(21236, "alpha"); err == nil {
		t.Fatal("expected Vrooli listener conflict")
	} else if !strings.Contains(err.Error(), `Vrooli scenario "beta"`) || !strings.Contains(err.Error(), "4242") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestEnsurePortBindableAllowsSameScenarioRestart confirms that a TCP listener
// from the *same* scenario (e.g. a restart-in-progress where the registry
// claim has already been re-acquired) does not block allocation.
func TestEnsurePortBindableAllowsSameScenarioRestart(t *testing.T) {
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
	isTCPPortInUseFn = func(int) (bool, error) { return true, nil }
	inspectPortListenersFn = func(int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
			Listeners:  []network.PortListener{{PID: 5151}},
		}, nil
	}
	readProcessEnvironmentPortFn = func(int) (map[string]string, error) {
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

	if err := manager.ensurePortBindable(21236, "alpha"); err != nil {
		t.Fatalf("ensurePortBindable(same scenario): %v", err)
	}
}

func TestEnsurePortBindableFallsBackToGenericConflictForNonVrooliListeners(t *testing.T) {
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
	isTCPPortInUseFn = func(int) (bool, error) { return true, nil }
	inspectPortListenersFn = func(int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
			Listeners:  []network.PortListener{{PID: 6161}},
		}, nil
	}
	readProcessEnvironmentPortFn = func(int) (map[string]string, error) {
		return map[string]string{}, nil
	}
	t.Cleanup(func() {
		isTCPPortInUseFn = previousInUse
		inspectPortListenersFn = previousInspect
		readProcessEnvironmentPortFn = previousReadEnv
	})

	if err := manager.ensurePortBindable(21236, "alpha"); err == nil {
		t.Fatal("expected generic listener conflict")
	} else if !strings.Contains(err.Error(), "port already in use") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestEnsurePortBindableRejectsResourceReservedPort verifies the resource
// reservation check survives the lock-ownership removal.
func TestEnsurePortBindableRejectsResourceReservedPort(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, map[string]int{"postgres": 5433})

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	if err := manager.ensurePortBindable(5433, "alpha"); err == nil {
		t.Fatal("expected reserved resource port to fail")
	} else if !strings.Contains(err.Error(), "reserved for resource") {
		t.Fatalf("unexpected reserved-port error: %v", err)
	}
}

// TestEnsurePortBindableIgnoresLegacyLockFiles is the load-bearing assertion
// for Phase 9.3: a leftover `.port_<port>.lock` from a previous release is
// pure diagnostic clutter — it does not block a registry-authorized
// allocation that has already acquired its claim.
func TestEnsurePortBindableIgnoresLegacyLockFiles(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.EnsureStateDir(); err != nil {
		t.Fatalf("EnsureStateDir: %v", err)
	}
	writeLegacyLockFile(t, manager, 21999, "ghost-scenario", os.Getpid(), time.Now())

	if err := manager.ensurePortBindable(21999, "alpha"); err != nil {
		t.Fatalf("ensurePortBindable should ignore legacy lock; got %v", err)
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
		if !strings.Contains(err.Error(), "active registry claim already owns port") {
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

func TestRuntimeClaimsRollbackWhenLaterPortAllocationFails(t *testing.T) {
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
}

// TestStaleLegacyLockDoesNotOverrideActiveRuntimeClaim proves that even when
// a `.port_<port>.lock` file from a prior install is present, the registry
// claim remains the source of truth. The lock is a diagnostic artifact;
// allocation neither honors it nor mutates it.
func TestStaleLegacyLockDoesNotOverrideActiveRuntimeClaim(t *testing.T) {
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
	if err := manager.EnsureStateDir(); err != nil {
		t.Fatalf("EnsureStateDir: %v", err)
	}
	writeLegacyLockFile(t, manager, port, "stale-legacy", 999999, time.Now())

	_, err = manager.BuildEnvironmentWithRuntimeClaims(fixedPortScenario(root, "beta", port), nil, RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      store,
		InstanceID: beta.InstanceID,
	})
	if err == nil {
		t.Fatal("expected active registry claim to block replacement allocation")
	}
	if !strings.Contains(err.Error(), "active registry claim already owns port") {
		t.Fatalf("unexpected error: %v", err)
	}
	lock, exists, err := manager.ReadLock(port)
	if err != nil {
		t.Fatalf("ReadLock: %v", err)
	}
	if !exists || lock.Scenario != "stale-legacy" {
		t.Fatalf("legacy lock should remain as diagnostic evidence, got exists=%v lock=%#v", exists, lock)
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

func writeLegacyLockFile(t *testing.T, m *Manager, port int, scenarioName string, pid int, when time.Time) {
	t.Helper()
	if err := m.EnsureStateDir(); err != nil {
		t.Fatalf("EnsureStateDir: %v", err)
	}
	content := []byte(fmt.Sprintf("%s:%d:%d\n", scenarioName, pid, when.Unix()))
	if err := os.WriteFile(m.lockPath(port), content, 0o644); err != nil {
		t.Fatalf("write legacy lock file: %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}

// Compile-time guard: process.Record is no longer used by allocator authority
// but remains imported elsewhere; force-import here so test file imports
// match the package set.
var _ = process.Record{}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
