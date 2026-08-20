package dependencies

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// countingCommander is a Commander that scripts osv-scanner output and records
// how many times the scanner subprocess was invoked (the cache-effectiveness
// signal under test). LookPath reports osv-scanner present; --version is
// answered so the annotator can fold a stable scanner version into the key.
type countingCommander struct {
	mu        sync.Mutex
	scanCalls int           // osv-scanner *scan* invocations
	inFlight  int32         // current concurrent scan invocations
	maxInFl   int32         // peak observed concurrency
	report    string        // JSON returned by each scan
	delay     time.Duration // artificial scan latency (to observe overlap)
	version   string        // --version stdout
}

func (c *countingCommander) LookPath(name string) (string, error) {
	return "/logical/" + name + "/" + c.version, nil
}

func (c *countingCommander) Run(ctx context.Context, _ string, name string, args ...string) ([]byte, []byte, int, error) {
	if name == "osv-scanner" && len(args) > 0 && args[0] == "--version" {
		v := c.version
		if v == "" {
			v = "osv-scanner version: 1.9.2"
		}
		return []byte(v), nil, 0, nil
	}
	// A scan invocation.
	cur := atomic.AddInt32(&c.inFlight, 1)
	for {
		mx := atomic.LoadInt32(&c.maxInFl)
		if cur <= mx || atomic.CompareAndSwapInt32(&c.maxInFl, mx, cur) {
			break
		}
	}
	c.mu.Lock()
	c.scanCalls++
	c.mu.Unlock()
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	atomic.AddInt32(&c.inFlight, -1)
	rep := c.report
	if rep == "" {
		rep = `{"results":[]}`
	}
	return []byte(rep), nil, 0, nil
}

func (c *countingCommander) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scanCalls
}

func newCachingAnnotator(t *testing.T, repoRoot string, cmd *countingCommander) (*Annotator, *Store) {
	t.Helper()
	store := newTestStore(t)
	a := NewAnnotator(repoRoot, cmd).WithCache(store)
	a.clock = schedule.NewFake(time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC))
	return a, store
}

// writeScenario lays down a minimal scenario tree with a go.mod (the cache-key
// lockfile input) and returns a single record for it.
func writeScenario(t *testing.T, repoRoot, scenario, gomod string) []DependencyRecord {
	t.Helper()
	writeF(t, filepath.Join(repoRoot, "scenarios", scenario, "api", "go.mod"), gomod)
	return []DependencyRecord{{
		Scenario:  scenario,
		Ecosystem: EcosystemGo,
		Name:      "golang.org/x/net",
		Version:   "0.17.0",
	}}
}

func TestAnnotate_CacheHitSkipsSubprocess(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	recs := writeScenario(t, repoRoot, "demo", "module x\ngo 1.24\nrequire golang.org/x/net v0.17.0\n")
	cmd := &countingCommander{}
	a, _ := newCachingAnnotator(t, repoRoot, cmd)

	// First annotate: cold cache → one scan.
	a.Annotate(ctx, cloneRecords(recs))
	if cmd.calls() != 1 {
		t.Fatalf("cold annotate: scan calls = %d, want 1", cmd.calls())
	}
	if st := a.LastScanStats(); st.ScansRun != 1 || st.ScansSkipped != 0 {
		t.Fatalf("cold stats = %+v, want run=1 skipped=0", st)
	}

	// Second annotate, nothing changed: cache hit → no new scan.
	a.Annotate(ctx, cloneRecords(recs))
	if cmd.calls() != 1 {
		t.Fatalf("warm annotate: scan calls = %d, want still 1 (cache hit)", cmd.calls())
	}
	if st := a.LastScanStats(); st.ScansRun != 0 || st.ScansSkipped != 1 {
		t.Fatalf("warm stats = %+v, want run=0 skipped=1", st)
	}
}

func TestAnnotate_InputChangeForcesRescan(t *testing.T) {
	type mutate func(t *testing.T, repoRoot string, cmd *countingCommander)
	cases := []struct {
		name string
		fn   mutate
	}{
		{
			name: "go.mod content changes",
			fn: func(t *testing.T, repoRoot string, _ *countingCommander) {
				writeF(t, filepath.Join(repoRoot, "scenarios", "demo", "api", "go.mod"),
					"module x\ngo 1.24\nrequire golang.org/x/net v0.18.0\n")
			},
		},
		{
			name: "go.sum appears",
			fn: func(t *testing.T, repoRoot string, _ *countingCommander) {
				writeF(t, filepath.Join(repoRoot, "scenarios", "demo", "api", "go.sum"),
					"golang.org/x/net v0.17.0 h1:abc=\n")
			},
		},
		{
			name: "pnpm-lock appears",
			fn: func(t *testing.T, repoRoot string, _ *countingCommander) {
				writeF(t, filepath.Join(repoRoot, "scenarios", "demo", "ui", "pnpm-lock.yaml"),
					"lockfileVersion: '9.0'\npackages:\n  esbuild@0.21.5: {}\n")
			},
		},
		{
			// npm versions can be pinned by package-lock.json (the repo has 55+),
			// which osv-scanner reads under -r; a change there must invalidate the
			// cache or it's a false skip.
			name: "package-lock.json appears",
			fn: func(t *testing.T, repoRoot string, _ *countingCommander) {
				writeF(t, filepath.Join(repoRoot, "scenarios", "demo", "ui", "package-lock.json"),
					"{\"name\":\"demo\",\"lockfileVersion\":3,\"packages\":{}}\n")
			},
		},
		{
			name: "scanner version changes",
			fn: func(t *testing.T, _ string, cmd *countingCommander) {
				cmd.version = "osv-scanner version: 2.0.0"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repoRoot := t.TempDir()
			recs := writeScenario(t, repoRoot, "demo", "module x\ngo 1.24\nrequire golang.org/x/net v0.17.0\n")
			cmd := &countingCommander{version: "osv-scanner version: 1.9.2"}
			a, _ := newCachingAnnotator(t, repoRoot, cmd)

			a.Annotate(ctx, cloneRecords(recs))
			if cmd.calls() != 1 {
				t.Fatalf("cold scan calls = %d, want 1", cmd.calls())
			}

			tc.fn(t, repoRoot, cmd)

			a.Annotate(ctx, cloneRecords(recs))
			if cmd.calls() != 2 {
				t.Fatalf("after %q: scan calls = %d, want 2 (re-scan forced)", tc.name, cmd.calls())
			}
			if st := a.LastScanStats(); st.ScansRun != 1 || st.ScansSkipped != 0 {
				t.Fatalf("after %q: stats = %+v, want run=1 skipped=0", tc.name, st)
			}
		})
	}
}

// TestAnnotate_DayEpochChangeForcesRescan exercises the day-epoch arm of the
// key: a cached entry written on one UTC day must miss the next day, even though
// the lockfiles and scanner version are identical, so a scenario with unchanged
// dependencies still re-scans daily and picks up newly-published CVEs.
func TestAnnotate_DayEpochChangeForcesRescan(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	recs := writeScenario(t, repoRoot, "demo", "module x\ngo 1.24\nrequire golang.org/x/net v0.17.0\n")
	cmd := &countingCommander{version: "osv-scanner version: 1.9.2"}
	a, store := newCachingAnnotator(t, repoRoot, cmd)

	a.Annotate(ctx, cloneRecords(recs))
	if cmd.calls() != 1 {
		t.Fatalf("cold scan calls = %d, want 1", cmd.calls())
	}

	// Same day → cache hit, no new scan.
	a.Annotate(ctx, cloneRecords(recs))
	if cmd.calls() != 1 {
		t.Fatalf("same-day re-annotate scanned again: calls = %d, want 1 (cache hit)", cmd.calls())
	}

	// Day rollover → the previously-cached key no longer matches, forcing a
	// re-scan that catches newly-published vulnerabilities.
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo")
	clock := a.clock.(*schedule.Fake)
	oldKey := a.scenarioCacheKey(ctx, scenarioDir)
	clock.Advance(24 * time.Hour)
	newKey := a.scenarioCacheKey(ctx, scenarioDir)
	if oldKey == newKey {
		t.Fatal("cache key did not change across a day boundary")
	}
	if _, ok := store.GetOSVScanCache(ctx, "demo", newKey); ok {
		t.Fatal("cache hit across a day boundary: a day-old result would be served")
	}
}

func TestAnnotate_ConcurrencyCapRespected(t *testing.T) {
	t.Setenv(EnvScanConcurrency, "2")
	ctx := context.Background()
	repoRoot := t.TempDir()
	var recs []DependencyRecord
	for i := 0; i < 8; i++ {
		recs = append(recs, writeScenario(t, repoRoot, fmt.Sprintf("s%d", i),
			fmt.Sprintf("module x%d\ngo 1.24\nrequire golang.org/x/net v0.17.0\n", i))...)
	}
	cmd := &countingCommander{delay: 20 * time.Millisecond}
	a, _ := newCachingAnnotator(t, repoRoot, cmd)

	a.Annotate(ctx, recs)
	if cmd.calls() != 8 {
		t.Fatalf("scan calls = %d, want 8 (one per scenario)", cmd.calls())
	}
	if peak := atomic.LoadInt32(&cmd.maxInFl); peak > 2 {
		t.Fatalf("peak concurrency = %d, want <= 2 (cap respected)", peak)
	}
}

func cloneRecords(in []DependencyRecord) []DependencyRecord {
	out := make([]DependencyRecord, len(in))
	copy(out, in)
	return out
}

// gateCommander blocks each scan until released, recording the peak number of
// concurrent scans so a test can assert the reconcile overlap lock serializes
// the full-fleet timer path and the full-fleet reindex path.
type gateCommander struct {
	mu       sync.Mutex
	inFlight int
	maxInFl  int
	enter    chan struct{} // signalled each time a scan starts
	release  chan struct{} // scans block until this closes
}

func (g *gateCommander) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }

func (g *gateCommander) Run(ctx context.Context, _ string, name string, args ...string) ([]byte, []byte, int, error) {
	if name == "osv-scanner" && len(args) > 0 && args[0] == "--version" {
		return []byte("osv-scanner version: 1.9.2"), nil, 0, nil
	}
	g.mu.Lock()
	g.inFlight++
	if g.inFlight > g.maxInFl {
		g.maxInFl = g.inFlight
	}
	g.mu.Unlock()
	select {
	case g.enter <- struct{}{}:
	default:
	}
	select {
	case <-g.release:
	case <-ctx.Done():
	}
	g.mu.Lock()
	g.inFlight--
	g.mu.Unlock()
	return []byte(`{"results":[]}`), nil, 0, nil
}

func (g *gateCommander) peak() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.maxInFl
}

func TestService_ReconcileReindexOverlapSerialized(t *testing.T) {
	t.Setenv(EnvScanConcurrency, "1") // force one scan at a time within a pass
	ctx := context.Background()
	store := newTestStore(t)
	repoRoot := t.TempDir()
	writeF(t, filepath.Join(repoRoot, "scenarios", "demo", "api", "go.mod"),
		"module x\ngo 1.24\nrequire golang.org/x/net v0.17.0\n")

	cmd := &gateCommander{enter: make(chan struct{}, 4), release: make(chan struct{})}
	annot := NewAnnotator(repoRoot, cmd).WithCache(store)
	svc := NewService(Deps{RepoRoot: repoRoot, Store: store, Annotator: annot, Clock: schedule.System()})

	// Start a periodic reconcile; it will block inside the scan holding the
	// overlap lock.
	reconcileDone := make(chan struct{})
	go func() {
		_ = svc.RunReconcileOnce(ctx)
		close(reconcileDone)
	}()

	// Wait until the reconcile's scan is in flight.
	select {
	case <-cmd.enter:
	case <-time.After(2 * time.Second):
		t.Fatal("reconcile scan never started")
	}

	// Kick off a full-fleet reindex; it must NOT start its scan while the
	// reconcile still holds the lock.
	res, err := svc.Reindex(ctx, "", false)
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	select {
	case <-cmd.enter:
		t.Fatal("reindex scan started while reconcile held the overlap lock (scan storm)")
	case <-time.After(150 * time.Millisecond):
		// good: reindex is blocked on the lock
	}

	// Release scans; both passes drain, never overlapping.
	close(cmd.release)
	<-reconcileDone
	// Drain the reindex job to completion.
	deadline := time.After(3 * time.Second)
	for {
		state, _, _, _, ok := svc.ReindexStatus(res.JobID)
		if ok && isTerminal(state) {
			break
		}
		select {
		case <-deadline:
			t.Fatal("reindex job never terminated")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if peak := cmd.peak(); peak > 1 {
		t.Fatalf("peak concurrent scans = %d, want 1 (overlap lock serializes the two fleet passes)", peak)
	}
}
