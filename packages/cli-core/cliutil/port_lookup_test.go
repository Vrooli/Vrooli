package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestLookupPeerRecordAcceptsEnvironmentPortSpelling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	peers := filepath.Join(os.Getenv("VROOLI_HOME"), "peers")
	if err := os.MkdirAll(peers, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"schema_version": 1,
		"scenario":       "fixture-scenario",
		"owner_pid":      os.Getpid(),
		"ports":          map[string]int{"api": 18888},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peers, "fixture-scenario.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	out := lookupPeerRecord("fixture-scenario", "API_PORT")
	if !out.Resolved() || out.Port != "18888" {
		t.Fatalf("outcome = %#v, want resolved api claim", out)
	}
}

func TestLookupPeerRecordSelectsOnlyPublishedPortCapabilities(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	peers := filepath.Join(os.Getenv("VROOLI_HOME"), "peers")
	if err := os.MkdirAll(peers, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"schema_version": 1, "scenario": "fixture-scenario", "owner_pid": os.Getpid(), "version": "1.2.3", "provides": []string{"ui"}, "ports": map[string]int{"api": 18888, "ui": 18889}}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peers, "fixture-scenario.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if out := lookupPeerRecord("fixture-scenario", "API_PORT"); out.Resolved() {
		t.Fatalf("unpublished api capability resolved: %#v", out)
	}
	if out := lookupPeerRecord("fixture-scenario", "UI_PORT"); !out.Resolved() || out.Port != "18889" {
		t.Fatalf("published ui capability = %#v", out)
	}
}

func TestLookupPeerRecordTreatsExplicitEmptyProvidesAsNoCapabilities(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	peers := filepath.Join(os.Getenv("VROOLI_HOME"), "peers")
	if err := os.MkdirAll(peers, 0o700); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"scenario":       "fixture-scenario",
		"owner_pid":      os.Getpid(),
		"provides":       []string{},
		"ports":          map[string]int{"api": 18888},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peers, "fixture-scenario.json"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if out := lookupPeerRecord("fixture-scenario", "API_PORT"); out.Resolved() {
		t.Fatalf("empty capability declaration resolved api: %#v", out)
	}
}

func TestLookupPeerRecordSeparatesLiveAndShadowInstances(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	peers := filepath.Join(os.Getenv("VROOLI_HOME"), "peers")
	if err := os.MkdirAll(peers, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		target   string
		instance string
		port     int
	}{{"browser-automation-studio", "live", 17116}, {"browser-automation-studio@shadow", "shadow", 15372}} {
		payload, err := json.Marshal(map[string]any{
			"schema_version": 1,
			"scenario":       "browser-automation-studio",
			"instance":       tc.instance,
			"owner_pid":      os.Getpid(),
			"ports":          map[string]int{"api": tc.port},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(peers, tc.target+".json"), payload, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		target string
		port   string
	}{{"browser-automation-studio", "17116"}, {"browser-automation-studio@live", "17116"}, {"browser-automation-studio@shadow", "15372"}} {
		out := lookupPeerRecord(tc.target, "API_PORT")
		if !out.Resolved() || out.Port != tc.port {
			t.Errorf("lookupPeerRecord(%q) = %#v, want port %s", tc.target, out, tc.port)
		}
	}
}

func TestMisfiledShadowPeerFallsThroughToLiveRegistry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	peers := filepath.Join(os.Getenv("VROOLI_HOME"), "peers")
	if err := os.MkdirAll(peers, 0o700); err != nil {
		t.Fatal(err)
	}
	misfiled, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"scenario":       "misfiled-peer",
		"instance":       "shadow",
		"owner_pid":      os.Getpid(),
		"ports":          map[string]int{"api": 15372},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peers, "misfiled-peer.json"), misfiled, 0o600); err != nil {
		t.Fatal(err)
	}

	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	peerRecordLookupFn = lookupPeerRecord
	registryCalls := 0
	runtimeRegistryLookupFn = func(_ context.Context, target, _ string) ScenarioPortOutcome {
		registryCalls++
		if target != "misfiled-peer" {
			t.Fatalf("registry target = %q, want bare live target", target)
		}
		return ScenarioPortOutcome{Port: "17116"}
	}
	defer func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	}()
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		t.Fatal("CLI fallback should not run after the live registry resolves")
		return ScenarioPortOutcome{}
	})()

	resetPortDetectorCache()
	out := LookupScenarioPort(context.Background(), "misfiled-peer", "API_PORT", PortCachePolicy{})
	if !out.Resolved() || out.Port != "17116" {
		t.Fatalf("live lookup = %#v, want registry port 17116", out)
	}
	if registryCalls != 1 {
		t.Fatalf("registry calls = %d, want 1 after rejecting the misfiled shadow peer", registryCalls)
	}
}

func TestLookupPeerRecordRejectsCrossPlatformPathSeparators(t *testing.T) {
	for _, target := range []string{`demo\..\outside`, `demo@..\..\outside`} {
		out := lookupPeerRecord(target, "API_PORT")
		if out.Resolved() {
			t.Errorf("lookupPeerRecord(%q) resolved unsafe target to port %s", target, out.Port)
		}
		if out.Err == nil || out.Err.Error() != "invalid peer record target" {
			t.Errorf("lookupPeerRecord(%q) error = %v, want early invalid-target rejection", target, out.Err)
		}
	}
}

func TestLookupRuntimeRegistryAcceptsEnvironmentPortSpelling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	dbPath, err := repocontract.RuntimeHomeEntryPathAtRoot(os.Getenv("VROOLI_HOME"), repocontract.HomeKeyRuntimeDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	sql := `CREATE TABLE runtime_instances (instance_id TEXT, scenario TEXT, variant TEXT, status TEXT, generation INTEGER);
CREATE TABLE runtime_port_claims (instance_id TEXT, port INTEGER, port_name TEXT, status TEXT);
INSERT INTO runtime_instances VALUES ('instance-1', 'fixture-scenario', 'live', 'running', 1);
INSERT INTO runtime_port_claims VALUES ('instance-1', 18889, 'api', 'bound');`
	if err := exec.Command("sqlite3", dbPath, sql).Run(); err != nil {
		t.Skipf("sqlite3 unavailable: %v", err)
	}

	out := lookupRuntimeRegistry(context.Background(), "fixture-scenario", "API_PORT")
	if !out.Resolved() || out.Port != "18889" {
		t.Fatalf("outcome = %#v, want resolved api claim", out)
	}
}

func fakeLookup(counter *int) func(context.Context, string, string) ScenarioPortOutcome {
	return func(context.Context, string, string) ScenarioPortOutcome {
		*counter++
		return ScenarioPortOutcome{Port: "15000", Output: "15000"}
	}
}

func TestLookupScenarioPortCachesWithinPolicy(t *testing.T) {
	calls := 0
	defer SetPortLookupRunner(fakeLookup(&calls))()

	policy := PortCachePolicy{MaxAge: time.Minute}
	for i := 0; i < 4; i++ {
		if got := LookupScenarioPort(context.Background(), "svc", "API_PORT", policy); got.Port != "15000" {
			t.Fatalf("port=%q", got.Port)
		}
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
}

func TestInvalidateScenarioPortCacheClearsTypedStoppedEntry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	var calls int
	restore := SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		calls++
		if calls == 1 {
			return ScenarioPortOutcome{Err: &ScenarioPortError{Code: ScenarioPortCodeNotRunning}}
		}
		return ScenarioPortOutcome{Port: "19191"}
	})
	t.Cleanup(restore)
	var observations []bool
	policy := PortCachePolicy{MaxAge: time.Minute, NegativeMaxAge: time.Minute, OnCacheLookup: func(hit bool) {
		observations = append(observations, hit)
	}}
	ctx := context.Background()

	if first := LookupScenarioPort(ctx, "invalidate-peer", "API_PORT", policy); !IsScenarioPortNotRunning(first.Err) {
		t.Fatalf("first lookup = %#v, want typed stopped result", first)
	}
	if cached := LookupScenarioPort(ctx, "invalidate-peer", "API_PORT", policy); !IsScenarioPortNotRunning(cached.Err) {
		t.Fatalf("cached lookup = %#v, want typed stopped result", cached)
	}
	if err := InvalidateScenarioPortCache(ctx, "invalidate-peer", "API_PORT"); err != nil {
		t.Fatalf("invalidate cached stopped result: %v", err)
	}
	if fresh := LookupScenarioPort(ctx, "invalidate-peer", "API_PORT", policy); !fresh.Resolved() || fresh.Port != "19191" {
		t.Fatalf("lookup after invalidation = %#v, want fresh port 19191", fresh)
	}
	if calls != 2 {
		t.Fatalf("lookup runner calls = %d, want 2", calls)
	}
	if !reflect.DeepEqual(observations, []bool{false, true, false}) {
		t.Fatalf("cache observations = %v, want miss, hit, miss", observations)
	}
}

func TestInvalidateScenarioPortCacheHonorsCallerDeadline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime-home"))
	started := make(chan struct{})
	release := make(chan struct{})
	restore := SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		close(started)
		<-release
		return ScenarioPortOutcome{Port: "19192"}
	})
	t.Cleanup(restore)
	lookupDone := make(chan struct{})
	go func() {
		defer close(lookupDone)
		LookupScenarioPort(context.Background(), "invalidate-wait", "API_PORT", PortCachePolicy{MaxAge: time.Minute})
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := InvalidateScenarioPortCache(ctx, "invalidate-wait", "API_PORT")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("invalidation error = %v, want caller deadline", err)
	}
	close(release)
	<-lookupDone
}

// The point of the shared cache: freshness is the caller's property, so a
// tolerant caller reusing an entry must not stop a strict caller from
// re-looking-up, and vice versa.
func TestPolicyIsPerCallerNotPerEntry(t *testing.T) {
	calls := 0
	defer SetPortLookupRunner(fakeLookup(&calls))()

	base := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	now := base
	SetPortCacheNowForTest(func() time.Time { return now })
	defer SetPortCacheNowForTest(nil)

	tolerant := PortCachePolicy{MaxAge: time.Minute}
	strict := PortCachePolicy{MaxAge: time.Second}

	LookupScenarioPort(context.Background(), "svc", "API_PORT", tolerant)
	if calls != 1 {
		t.Fatalf("calls=%d after first lookup", calls)
	}

	now = base.Add(5 * time.Second)
	// Still fresh for the tolerant caller.
	LookupScenarioPort(context.Background(), "svc", "API_PORT", tolerant)
	if calls != 1 {
		t.Fatalf("tolerant caller re-looked-up: calls=%d, want 1", calls)
	}
	// Stale for the strict caller, which must not inherit the other's tolerance.
	LookupScenarioPort(context.Background(), "svc", "API_PORT", strict)
	if calls != 2 {
		t.Fatalf("strict caller reused a stale entry: calls=%d, want 2", calls)
	}
}

func TestZeroMaxAgeDisablesReuse(t *testing.T) {
	calls := 0
	defer SetPortLookupRunner(fakeLookup(&calls))()

	for i := 0; i < 3; i++ {
		LookupScenarioPort(context.Background(), "svc", "API_PORT", PortCachePolicy{})
	}
	if calls != 3 {
		t.Fatalf("calls=%d, want 3 when reuse is disabled", calls)
	}
}

func TestFailedLookupUsesNegativeMaxAge(t *testing.T) {
	calls := 0
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Output: `{"error_code":"scenario_not_running"}`, Err: &ScenarioPortError{Code: ScenarioPortCodeNotRunning}}
	})()

	base := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	now := base
	SetPortCacheNowForTest(func() time.Time { return now })
	defer SetPortCacheNowForTest(nil)

	policy := PortCachePolicy{MaxAge: time.Hour, NegativeMaxAge: 3 * time.Second}
	for i := 0; i < 4; i++ {
		if out := LookupScenarioPort(context.Background(), "stopped", "API_PORT", policy); out.Resolved() {
			t.Fatal("failed lookup reported as resolved")
		}
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1 within the negative window", calls)
	}

	// A failure must expire on the negative window, not the long positive one —
	// otherwise a scenario that has since started stays unreachable for an hour.
	now = base.Add(4 * time.Second)
	LookupScenarioPort(context.Background(), "stopped", "API_PORT", policy)
	if calls != 2 {
		t.Fatalf("calls=%d, want 2 after the negative window expired", calls)
	}
}

func TestDefaultNegativeMaxAgeIsHalfSecond(t *testing.T) {
	if DefaultPortCachePolicy.NegativeMaxAge != 500*time.Millisecond {
		t.Fatalf("default negative max age=%s, want 500ms", DefaultPortCachePolicy.NegativeMaxAge)
	}
}

func TestDefaultPortPolicyRefreshesAfterPortDrift(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	base := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	now := base
	SetPortCacheNowForTest(func() time.Time { return now })
	t.Cleanup(func() { SetPortCacheNowForTest(nil) })
	lookupCount := 0
	peerRecordLookupFn = func(_, _ string) ScenarioPortOutcome {
		lookupCount++
		if lookupCount == 1 {
			return ScenarioPortOutcome{Port: "19001"}
		}
		return ScenarioPortOutcome{Port: "19002"}
	}
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		t.Fatal("runtime registry called after peer hit")
		return ScenarioPortOutcome{}
	}
	resetPortDetectorCache()

	if DefaultPortCachePolicy.MaxAge != 2*time.Second {
		t.Fatalf("default positive max age=%s, want 2s", DefaultPortCachePolicy.MaxAge)
	}
	if got := LookupScenarioPort(context.Background(), "bridge-target", "API_PORT", DefaultPortCachePolicy); got.Port != "19001" {
		t.Fatalf("initial port=%q, want 19001", got.Port)
	}
	now = base.Add(3 * time.Second)
	if got := LookupScenarioPort(context.Background(), "bridge-target", "API_PORT", DefaultPortCachePolicy); got.Port != "19002" {
		t.Fatalf("port after lifecycle restart=%q, want 19002", got.Port)
	}
	if lookupCount != 2 {
		t.Fatalf("peer lookups=%d, want 2 after positive cache expiry", lookupCount)
	}
}

func TestTransientLookupFailureIsNotNegativeCached(t *testing.T) {
	oldPeer, oldRegistry, oldRunner := peerRecordLookupFn, runtimeRegistryLookupFn, portLookupRunner
	defer func() {
		peerRecordLookupFn, runtimeRegistryLookupFn, portLookupRunner = oldPeer, oldRegistry, oldRunner
		resetPortDetectorCache()
	}()
	peerRecordLookupFn = func(_, _ string) ScenarioPortOutcome { return ScenarioPortOutcome{Err: os.ErrNotExist} }
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	calls := 0
	portLookupRunner = func(context.Context, string, string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Output: "permission denied", Err: errors.New("permission denied")}
	}
	resetPortDetectorCache()
	policy := PortCachePolicy{MaxAge: time.Hour, NegativeMaxAge: time.Hour}
	for i := 0; i < 2; i++ {
		out := LookupScenarioPort(context.Background(), "transient", "API_PORT", policy)
		if out.Resolved() {
			t.Fatal("transient error unexpectedly resolved")
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2: transient failures must not be cached", calls)
	}
}

func TestDesktopRuntimeFailureIsNotNegativeCached(t *testing.T) {
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
	oldDesktop := desktopPortLookupFn
	defer func() {
		desktopPortLookupFn = oldDesktop
		resetPortDetectorCache()
	}()
	calls := 0
	desktopPortLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Output: "no running runtime ports found: IPC unavailable", Err: errors.New("IPC unavailable")}
	}
	resetPortDetectorCache()
	policy := PortCachePolicy{MaxAge: time.Hour, NegativeMaxAge: time.Hour}
	for i := 0; i < 2; i++ {
		out := LookupScenarioPort(context.Background(), "desktop-target", "API_PORT", policy)
		if out.Resolved() {
			t.Fatal("runtime IPC error unexpectedly resolved")
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2: desktop IPC failures must not be cached", calls)
	}
}

func TestCancelledContextDoesNotStartOrPoisonLookup(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	calls := 0
	runtimeRegistryLookupFn = func(ctx context.Context, _, _ string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Port: "15000"}
	}
	resetPortDetectorCache()

	policy := PortCachePolicy{MaxAge: time.Hour, NegativeMaxAge: time.Hour}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if out := LookupScenarioPort(cancelled, "svc", "API_PORT", policy); !errors.Is(out.Err, context.Canceled) {
		t.Fatalf("cancelled lookup = %#v, want context cancellation", out)
	}
	if calls != 0 {
		t.Fatalf("registry calls=%d after cancellation, want 0", calls)
	}
	if out := LookupScenarioPort(context.Background(), "svc", "API_PORT", policy); !out.Resolved() {
		t.Fatalf("fresh caller was affected by prior cancellation: %#v", out)
	}
	if calls != 1 {
		t.Fatalf("registry calls=%d for fresh caller, want 1", calls)
	}
}

func TestLookupScenarioPortSharesOneBoundAcrossRegistryAndCLIFallback(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	var registryDeadline, cliDeadline time.Time
	runtimeRegistryLookupFn = func(ctx context.Context, _, _ string) ScenarioPortOutcome {
		registryDeadline, _ = ctx.Deadline()
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	defer SetPortLookupRunner(func(ctx context.Context, _, _ string) ScenarioPortOutcome {
		cliDeadline, _ = ctx.Deadline()
		return ScenarioPortOutcome{Err: errors.New("expected fallback miss")}
	})()

	resetPortDetectorCache()
	LookupScenarioPort(context.Background(), "bounded-target", "API_PORT", PortCachePolicy{})
	remaining := time.Until(registryDeadline)
	if registryDeadline.IsZero() || remaining <= 0 || remaining > portLookupTimeout {
		t.Fatalf("registry deadline remaining = %s, want a future deadline within %s", remaining, portLookupTimeout)
	}
	if !registryDeadline.Equal(cliDeadline) {
		t.Fatalf("registry deadline %v differs from CLI fallback deadline %v", registryDeadline, cliDeadline)
	}
}

func TestLookupScenarioPortPreservesEarlierCallerDeadlineAcrossRegistry(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	registryCalled := false
	runtimeRegistryLookupFn = func(ctx context.Context, _, _ string) ScenarioPortOutcome {
		registryCalled = true
		<-ctx.Done()
		return ScenarioPortOutcome{Err: ctx.Err()}
	}
	cliCalled := false
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		cliCalled = true
		return ScenarioPortOutcome{Err: errors.New("CLI must not run after lookup deadline")}
	})()

	resetPortDetectorCache()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	outcome := LookupScenarioPort(ctx, "deadline-target", "API_PORT", PortCachePolicy{})
	if !registryCalled {
		t.Fatal("runtime registry was not given the caller context")
	}
	if !errors.Is(outcome.Err, context.DeadlineExceeded) {
		t.Fatalf("registry timeout outcome = %#v, want caller deadline error", outcome)
	}
	if cliCalled {
		t.Fatal("CLI fallback ran after the shared lookup deadline expired")
	}
}

func TestLookupScenarioPortReturnsDeadlineWhenRegistryReturnsLateMiss(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	registryCalls := 0
	lateOutcomes := []ScenarioPortOutcome{
		{Err: os.ErrNotExist},
		{Port: "14999"},
	}
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		registryCalls++
		if registryCalls == 1 {
			time.Sleep(30 * time.Millisecond) // Simulate a callee that returns just after cancellation.
			return lateOutcomes[0]
		}
		return ScenarioPortOutcome{Port: "15000"}
	}
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		t.Fatal("CLI fallback ran after the lookup deadline")
		return ScenarioPortOutcome{}
	})()
	policy := PortCachePolicy{MaxAge: time.Hour, NegativeMaxAge: time.Hour}
	for _, late := range lateOutcomes {
		registryCalls = 0
		lateOutcomes[0] = late
		resetPortDetectorCache()
		target := "late-registry-result"
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		first := LookupScenarioPort(ctx, target, "API_PORT", policy)
		cancel()
		if !errors.Is(first.Err, context.DeadlineExceeded) || first.Port != "" {
			t.Fatalf("late registry result %#v = %#v, want deadline error without a port", late, first)
		}
		if second := LookupScenarioPort(context.Background(), target, "API_PORT", policy); !second.Resolved() {
			t.Fatalf("fresh lookup was affected by late result %#v: %#v", late, second)
		}
		if registryCalls != 2 {
			t.Fatalf("registry calls=%d after late result %#v, want 2 because timeout is not cached", registryCalls, late)
		}
	}
}

func TestLookupScenarioPortStopsBeforeRegistryWhenPeerReadExceedsDeadline(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerCalls := 0
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		peerCalls++
		if peerCalls == 1 {
			time.Sleep(30 * time.Millisecond)
			return ScenarioPortOutcome{Err: os.ErrNotExist}
		}
		return ScenarioPortOutcome{Port: "15000"}
	}
	registryCalls := 0
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		registryCalls++
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		t.Fatal("CLI fallback ran after the peer rung exhausted the deadline")
		return ScenarioPortOutcome{}
	})()
	resetPortDetectorCache()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	first := LookupScenarioPort(ctx, "late-peer-miss", "API_PORT", PortCachePolicy{})
	cancel()
	if !errors.Is(first.Err, context.DeadlineExceeded) {
		t.Fatalf("late peer miss = %#v, want deadline error", first)
	}
	if registryCalls != 0 {
		t.Fatalf("registry calls=%d after peer deadline, want 0", registryCalls)
	}
	if second := LookupScenarioPort(context.Background(), "late-peer-miss", "API_PORT", PortCachePolicy{}); !second.Resolved() {
		t.Fatalf("fresh peer lookup = %#v, want recovery after expired miss", second)
	}
	if registryCalls != 0 {
		t.Fatalf("registry calls=%d after peer recovery, want 0", registryCalls)
	}
}

func TestLookupScenarioPortRejectsLateCLISuccess(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		time.Sleep(30 * time.Millisecond)
		return ScenarioPortOutcome{Port: "15000"}
	})()
	resetPortDetectorCache()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	outcome := LookupScenarioPort(ctx, "late-cli", "API_PORT", PortCachePolicy{})
	cancel()
	if !errors.Is(outcome.Err, context.DeadlineExceeded) || outcome.Port != "" {
		t.Fatalf("late CLI success = %#v, want deadline error without a port", outcome)
	}
}

func TestLookupScenarioPortClearsResultIfDeadlineExpiresWhileCaching(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		SetPortCacheNowForTest(nil)
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	calls := 0
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Port: "15000"}
	}
	resetPortDetectorCache()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	SetPortCacheNowForTest(func() time.Time {
		<-ctx.Done()
		return time.Now()
	})

	late := LookupScenarioPort(ctx, "cache-deadline", "API_PORT", PortCachePolicy{MaxAge: time.Hour})
	if !errors.Is(late.Err, context.DeadlineExceeded) || late.Port != "" {
		t.Fatalf("result whose cache write crossed its deadline = %#v, want deadline error", late)
	}
	SetPortCacheNowForTest(nil)
	if fresh := LookupScenarioPort(context.Background(), "cache-deadline", "API_PORT", PortCachePolicy{MaxAge: time.Hour}); !fresh.Resolved() {
		t.Fatalf("fresh lookup was affected by expired cache write: %#v", fresh)
	}
	if calls != 2 {
		t.Fatalf("registry calls=%d, want 2 because expired result must be removed from cache", calls)
	}
}

func TestLookupScenarioPortRechecksContextAfterCachedAgeEvaluation(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		SetPortCacheNowForTest(nil)
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	calls := 0
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Port: "15000"}
	}
	resetPortDetectorCache()
	base := time.Now()
	SetPortCacheNowForTest(func() time.Time { return base })
	policy := PortCachePolicy{MaxAge: time.Hour}
	if first := LookupScenarioPort(context.Background(), "cached-deadline", "API_PORT", policy); !first.Resolved() {
		t.Fatalf("initial lookup = %#v, want a cached port", first)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	SetPortCacheNowForTest(func() time.Time {
		<-ctx.Done()
		return base.Add(time.Millisecond)
	})
	lateHit := LookupScenarioPort(ctx, "cached-deadline", "API_PORT", policy)
	cancel()
	if !errors.Is(lateHit.Err, context.DeadlineExceeded) || lateHit.Port != "" {
		t.Fatalf("cache hit whose age evaluation crossed its deadline = %#v, want deadline error", lateHit)
	}
	SetPortCacheNowForTest(func() time.Time { return base.Add(2 * time.Millisecond) })
	if next := LookupScenarioPort(context.Background(), "cached-deadline", "API_PORT", policy); next.Port != "15000" {
		t.Fatalf("valid cache entry was removed by another caller's timeout: %#v", next)
	}
	if calls != 1 {
		t.Fatalf("registry calls=%d, want 1: context expiry must preserve the shared cached result", calls)
	}
}

func TestDesktopLookupReturnsDeadlineForLateOutcome(t *testing.T) {
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
	oldDesktop := desktopPortLookupFn
	t.Cleanup(func() {
		desktopPortLookupFn = oldDesktop
		resetPortDetectorCache()
	})
	for _, late := range []ScenarioPortOutcome{
		{Err: errors.New("runtime unavailable")},
		{Port: "14999"},
	} {
		desktopPortLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
			time.Sleep(30 * time.Millisecond)
			return late
		}
		resetPortDetectorCache()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		outcome := LookupScenarioPort(ctx, "desktop-late-outcome", "API_PORT", PortCachePolicy{})
		cancel()
		if !errors.Is(outcome.Err, context.DeadlineExceeded) || outcome.Port != "" {
			t.Fatalf("late desktop result %#v = %#v, want deadline error without a port", late, outcome)
		}
	}
}

func TestLookupScenarioPortDeadlineAlsoBoundsConcurrentWait(t *testing.T) {
	oldPeer, oldRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn = oldPeer, oldRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	started := make(chan struct{})
	runtimeRegistryLookupFn = func(ctx context.Context, _, _ string) ScenarioPortOutcome {
		close(started)
		<-ctx.Done()
		return ScenarioPortOutcome{Err: ctx.Err()}
	}
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: errors.New("CLI must not run after shared lookup is cancelled")}
	})()

	resetPortDetectorCache()
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstDone := make(chan ScenarioPortOutcome, 1)
	go func() {
		firstDone <- LookupScenarioPort(firstCtx, "contended-target", "API_PORT", PortCachePolicy{})
	}()
	<-started

	secondCtx, cancelSecond := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelSecond()
	second := LookupScenarioPort(secondCtx, "contended-target", "API_PORT", PortCachePolicy{})
	if !errors.Is(second.Err, context.DeadlineExceeded) {
		t.Fatalf("waiting caller outcome = %#v, want its own deadline", second)
	}

	cancelFirst()
	if first := <-firstDone; !errors.Is(first.Err, context.Canceled) {
		t.Fatalf("first caller outcome = %#v, want cancellation", first)
	}
}

func TestConcurrentCallersCollapseToOneLookup(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	defer SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return ScenarioPortOutcome{Port: "15000", Output: "15000"}
	})()

	policy := PortCachePolicy{MaxAge: time.Minute}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if out := LookupScenarioPort(context.Background(), "svc", "API_PORT", policy); out.Port != "15000" {
				t.Errorf("port=%q", out.Port)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("64 concurrent callers performed %d lookups, want 1", calls)
	}
}

func TestResolvedRequiresBothPortAndNoError(t *testing.T) {
	if (ScenarioPortOutcome{Port: "80"}).Resolved() != true {
		t.Fatal("a port with no error should be resolved")
	}
	if (ScenarioPortOutcome{}).Resolved() {
		t.Fatal("an empty outcome is not resolved")
	}
	if (ScenarioPortOutcome{Port: "80", Err: errors.New("x")}).Resolved() {
		t.Fatal("an errored outcome is not resolved")
	}
}
