package graphsweeper

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/graphingest"
)

type manualClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *manualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type fakeLister struct{ scenarios []Scenario }

func (f fakeLister) ListScenarios(string) ([]Scenario, error) { return f.scenarios, nil }

type fakeDigester struct {
	mu      sync.Mutex
	digests map[string]string
}

func (f *fakeDigester) ComputeDigest(root string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.digests[root], nil
}

type memDigestStore struct {
	mu sync.Mutex
	m  map[string]string
}

func newMemDigestStore() *memDigestStore { return &memDigestStore{m: map[string]string{}} }
func (s *memDigestStore) GetIngestDigest(scenario string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[scenario]
	return v, ok, nil
}

func (s *memDigestStore) SetIngestDigest(scenario, digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[scenario] = digest
	return nil
}

type fakeIngestor struct {
	mu       sync.Mutex
	calls    []string
	failFor  map[string]bool
	degraded bool
	clock    *manualClock
	advance  time.Duration
}

func (f *fakeIngestor) IngestScenario(_ context.Context, _, scenario string, _ bool) (graphingest.ScenarioResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, scenario)
	clock, advance := f.clock, f.advance
	fail := f.failFor[scenario]
	degraded := f.degraded
	f.mu.Unlock()
	if clock != nil && advance > 0 {
		clock.Advance(advance)
	}
	if fail {
		return graphingest.ScenarioResult{Scenario: scenario, Degraded: degraded}, errors.New("boom")
	}
	return graphingest.ScenarioResult{Scenario: scenario, EdgesPersisted: 1}, nil
}

func newTestSweeper(cfg Config, ing Ingestor, digests DigestStore, lister Lister, digester Digester, clock Clock) *Sweeper {
	return New(cfg, ing, digests, WithLister(lister), WithDigester(digester), WithClock(clock))
}

func TestRunOnceFreshnessSkipsUnchanged(t *testing.T) {
	clock := &manualClock{t: time.Unix(1000, 0)}
	digester := &fakeDigester{digests: map[string]string{"/r/a": "td:1", "/r/b": "td:2"}}
	store := newMemDigestStore()
	store.m["a"] = "td:1" // a is fresh, b is new
	ing := &fakeIngestor{failFor: map[string]bool{}}
	cfg := Config{Concurrency: 1, ScenariosRoot: "/r"}
	sw := newTestSweeper(cfg, ing, store, fakeLister{scenarios: []Scenario{{Name: "a", Root: "/r/a"}, {Name: "b", Root: "/r/b"}}}, digester, clock)

	report := sw.RunOnce(context.Background())
	if report.SkippedFresh != 1 {
		t.Fatalf("skipped_fresh = %d, want 1", report.SkippedFresh)
	}
	if report.Ingested != 1 {
		t.Fatalf("ingested = %d, want 1", report.Ingested)
	}
	if len(ing.calls) != 1 || ing.calls[0] != "b" {
		t.Fatalf("expected only b ingested, got %v", ing.calls)
	}
	if got, _, _ := store.GetIngestDigest("b"); got != "td:2" {
		t.Fatalf("digest for b not recorded, got %q", got)
	}
}

func TestRunOnceBreakerOpensUnderRepeatedFailure(t *testing.T) {
	clock := &manualClock{t: time.Unix(1000, 0)}
	scenarios := []Scenario{}
	digests := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		scenarios = append(scenarios, Scenario{Name: name, Root: "/r/" + name})
		digests["/r/"+name] = "td:" + name
	}
	digester := &fakeDigester{digests: digests}
	ing := &fakeIngestor{failFor: map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true}, degraded: true}
	cfg := Config{Concurrency: 1, BreakerThreshold: 2, BreakerCooldown: time.Hour, ScenariosRoot: "/r"}
	sw := newTestSweeper(cfg, ing, newMemDigestStore(), fakeLister{scenarios: scenarios}, digester, clock)

	report := sw.RunOnce(context.Background())
	if report.BreakerState != BreakerOpen {
		t.Fatalf("breaker state = %q, want open", report.BreakerState)
	}
	// After 2 failures the breaker opens and remaining scenarios are skipped.
	if len(ing.calls) > 3 {
		t.Fatalf("breaker should have stopped ingest attempts, got %d calls", len(ing.calls))
	}
	if report.BreakerSkipped == 0 {
		t.Fatalf("expected some breaker_skipped, got 0")
	}
}

func TestRunOnceBudgetHit(t *testing.T) {
	clock := &manualClock{t: time.Unix(1000, 0)}
	digester := &fakeDigester{digests: map[string]string{"/r/a": "td:a", "/r/b": "td:b", "/r/c": "td:c"}}
	// Each ingest advances the clock past the cycle budget so the second
	// iteration's pre-launch budget check trips. Concurrency=1 serializes the
	// ingest before the next iteration is considered.
	ing := &fakeIngestor{failFor: map[string]bool{}, clock: clock, advance: 2 * time.Millisecond}
	cfg := Config{Concurrency: 1, CycleBudget: time.Millisecond, ScenariosRoot: "/r"}
	sw := newTestSweeper(cfg, ing, newMemDigestStore(),
		fakeLister{scenarios: []Scenario{{Name: "a", Root: "/r/a"}, {Name: "b", Root: "/r/b"}, {Name: "c", Root: "/r/c"}}}, digester, clock)

	report := sw.RunOnce(context.Background())
	if !report.BudgetHit {
		t.Fatalf("expected budget hit")
	}
	// The budget gate is best-effort (checked before acquiring the concurrency
	// slot), so it bounds the cycle without guaranteeing an exact count — but it
	// must stop the sweep short of the full set.
	if report.Ingested >= 3 {
		t.Fatalf("budget should have stopped the sweep short of all 3, ingested %d", report.Ingested)
	}
}

func TestBreakerStateMachine(t *testing.T) {
	clock := &manualClock{t: time.Unix(0, 0)}
	b := newBreaker(2, time.Minute, clock)
	if !b.Allow() {
		t.Fatalf("closed breaker should allow")
	}
	b.Failure()
	if b.State() != BreakerClosed {
		t.Fatalf("one failure below threshold should stay closed")
	}
	b.Failure()
	if b.State() != BreakerOpen {
		t.Fatalf("threshold failures should open")
	}
	if b.Allow() {
		t.Fatalf("open breaker should deny before cooldown")
	}
	clock.Advance(2 * time.Minute)
	if !b.Allow() {
		t.Fatalf("after cooldown should admit half-open probe")
	}
	if b.State() != BreakerHalfOpen {
		t.Fatalf("state should be half-open after probe admitted")
	}
	b.Success()
	if b.State() != BreakerClosed {
		t.Fatalf("success should close breaker")
	}
}
