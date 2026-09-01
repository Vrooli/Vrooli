package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLookupPeerRecordAcceptsEnvironmentPortSpelling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	peers := filepath.Join(home, ".vrooli", "peers")
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

func TestLookupRuntimeRegistryAcceptsEnvironmentPortSpelling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dbPath := filepath.Join(home, ".vrooli", "state", "runtime.db")
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

func TestLookupScenarioPortCountersIdentifyAnsweringRung(t *testing.T) {
	oldPeer, oldRegistry, oldRunner := peerRecordLookupFn, runtimeRegistryLookupFn, portLookupRunner
	defer func() {
		peerRecordLookupFn, runtimeRegistryLookupFn, portLookupRunner = oldPeer, oldRegistry, oldRunner
		resetPortDetectorCache()
	}()
	peerRecordLookupFn = func(_, _ string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Port: "15001"}
	}
	runtimeRegistryLookupFn = func(_ context.Context, target, _ string) ScenarioPortOutcome {
		if target == "registry" {
			return ScenarioPortOutcome{Port: "15002"}
		}
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	portLookupRunner = func(_ context.Context, target, _ string) ScenarioPortOutcome {
		if target == "cli" {
			return ScenarioPortOutcome{Port: "15003"}
		}
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	// The peer fixture answers the first evaluation. Make the later cases miss
	// it explicitly so each rung is observed once.
	peerRecordLookupFn = func(_, _ string) ScenarioPortOutcome { return ScenarioPortOutcome{Err: os.ErrNotExist} }
	for _, target := range []string{"registry", "cli"} {
		if out := LookupScenarioPort(context.Background(), target, "API_PORT", PortCachePolicy{}); !out.Resolved() {
			t.Fatalf("%s lookup did not resolve: %#v", target, out)
		}
	}
	peerRecordLookupFn = func(target, _ string) ScenarioPortOutcome {
		if target == "peer" {
			return ScenarioPortOutcome{Port: "15001"}
		}
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	if out := LookupScenarioPort(context.Background(), "peer", "API_PORT", PortCachePolicy{}); !out.Resolved() {
		t.Fatalf("peer lookup did not resolve: %#v", out)
	}
	stats := PortLookupStats()
	if stats.Evaluations != 3 || stats.PeerHits != 1 || stats.RegistryHits != 1 || stats.CLIHits != 1 {
		t.Fatalf("rung counters = %+v, want one evaluation and hit at each rung", stats)
	}
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
		return ScenarioPortOutcome{Output: "not running", Err: errors.New("exit status 1")}
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

func TestCancelledContextIsNotCached(t *testing.T) {
	calls := 0
	defer SetPortLookupRunner(func(ctx context.Context, _, _ string) ScenarioPortOutcome {
		calls++
		return ScenarioPortOutcome{Output: "", Err: ctx.Err()}
	})()

	policy := PortCachePolicy{MaxAge: time.Hour, NegativeMaxAge: time.Hour}
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		LookupScenarioPort(ctx, "svc", "API_PORT", policy)
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2: a cached cancellation would deny an unrelated caller", calls)
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
