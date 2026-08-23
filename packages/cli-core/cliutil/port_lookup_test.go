package cliutil

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

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
