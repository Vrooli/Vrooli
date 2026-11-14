package vrooli

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type snapshotExecutor struct {
	mu        sync.Mutex
	responses map[string]struct {
		output []byte
		err    error
	}
	calls map[string]int
	block chan struct{}
}

func (e *snapshotExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	e.mu.Lock()
	e.calls[key]++
	response := e.responses[key]
	block := e.block
	e.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return response.output, response.err
}

func (e *snapshotExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.Output(ctx, name, args...)
}

func (e *snapshotExecutor) Run(ctx context.Context, name string, args ...string) error {
	_, err := e.Output(ctx, name, args...)
	return err
}

func fleetJSON() []byte {
	return []byte(`{"success":true,"resources":[{"resource":{"name":"postgres"},"installed":true,"running":true,"healthy":true,"message":"healthy"}]}`)
}

func TestSlowFleetRefreshIsStaleAndBackedOff(t *testing.T) {
	executor := &snapshotExecutor{
		responses: map[string]struct {
			output []byte
			err    error
		}{
			"vrooli resource status --health-only --json": {output: fleetJSON()},
		},
		calls: make(map[string]int),
		block: make(chan struct{}),
	}
	var clock atomic.Int64
	clock.Store(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC).UnixNano())
	provider := NewSnapshotProvider(NewClient(executor), WithSnapshotTTL(20*time.Second), WithSnapshotRetryBackoff(time.Minute), WithSnapshotClock(func() time.Time { return time.Unix(0, clock.Load()).UTC() }))
	type result struct {
		snapshot ResourceStatusSnapshot
		err      error
	}
	finished := make(chan result, 1)
	go func() {
		s, err := provider.Snapshot(context.Background(), []string{"postgres"})
		finished <- result{s, err}
	}()
	for {
		executor.mu.Lock()
		started := executor.calls["vrooli resource status --health-only --json"] > 0
		executor.mu.Unlock()
		if started {
			break
		}
		time.Sleep(time.Millisecond)
	}
	waiters := make(chan result, 4)
	for range 4 {
		go func() {
			s, err := provider.Snapshot(context.Background(), []string{"postgres"})
			waiters <- result{s, err}
		}()
	}
	deadline := time.Now().Add(time.Second)
	for provider.SnapshotMetrics().CoalescedCalls < 4 {
		if time.Now().After(deadline) {
			close(executor.block)
			t.Fatal("callers did not join the in-flight refresh")
		}
		time.Sleep(time.Millisecond)
	}
	clock.Add(int64(21 * time.Second))
	close(executor.block)
	first := <-finished
	if first.err == nil || first.snapshot.Complete || first.snapshot.Source != SnapshotSourceFallback {
		t.Fatalf("expired snapshot = %+v, err = %v", first.snapshot, first.err)
	}
	for range 4 {
		joined := <-waiters
		if joined.err == nil || joined.snapshot.Complete || joined.snapshot.Source != SnapshotSourceFallback {
			t.Fatalf("coalesced stale snapshot = %+v, err = %v", joined.snapshot, joined.err)
		}
	}
	second, err := provider.Snapshot(context.Background(), []string{"postgres"})
	if err == nil || second.Complete || second.Source != SnapshotSourceFallback {
		t.Fatalf("backoff snapshot = %+v, err = %v", second, err)
	}
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if got := executor.calls["vrooli resource status --health-only --json"]; got != 1 {
		t.Fatalf("fleet calls = %d, want one scan during backoff", got)
	}
}

// [REQ:RES-MON-007]
func TestSnapshotProviderUsesOneTypedFleetRefreshForConcurrentCallers(t *testing.T) {
	executor := &snapshotExecutor{
		responses: map[string]struct {
			output []byte
			err    error
		}{"vrooli resource status --health-only --json": {output: fleetJSON()}},
		calls: make(map[string]int),
		block: make(chan struct{}),
	}
	provider := NewSnapshotProvider(NewClient(executor), WithSnapshotTTL(time.Minute))

	const callers = 8
	results := make(chan ResourceStatusSnapshot, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snapshot, err := provider.Snapshot(context.Background(), []string{"postgres"})
			results <- snapshot
			errs <- err
		}()
	}
	// Let every caller reach the provider before releasing the one refresh.
	time.Sleep(10 * time.Millisecond)
	close(executor.block)
	wg.Wait()
	close(results)
	close(errs)

	executor.mu.Lock()
	gotCalls := executor.calls["vrooli resource status --health-only --json"]
	executor.mu.Unlock()
	if gotCalls != 1 {
		t.Fatalf("fleet refresh calls = %d, want 1", gotCalls)
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("Snapshot error: %v", err)
		}
	}
	for snapshot := range results {
		if !snapshot.Complete || snapshot.Source != SnapshotSourceFleet || snapshot.ProbeLevel != ProbeLevelFast {
			t.Fatalf("snapshot metadata = %+v", snapshot)
		}
	}
}

// [REQ:RES-MON-007]
func TestSnapshotProviderCoalescesConcurrentForcedDeepReaders(t *testing.T) {
	executor := &snapshotExecutor{
		responses: map[string]struct {
			output []byte
			err    error
		}{"vrooli resource status postgres --health-only --json": {output: []byte(`{"success":true,"name":"postgres","installed":true,"running":true,"healthy":true,"status":"healthy"}`)}},
		calls: make(map[string]int),
		block: make(chan struct{}),
	}
	provider := NewSnapshotProvider(NewClient(executor), WithSnapshotTTL(time.Minute))

	const callers = 8
	results := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := provider.ForceDeepStatus(context.Background(), "postgres")
			results <- err
		}()
	}
	time.Sleep(10 * time.Millisecond)
	close(executor.block)
	wg.Wait()
	close(results)

	executor.mu.Lock()
	gotCalls := executor.calls["vrooli resource status postgres --health-only --json"]
	executor.mu.Unlock()
	if gotCalls != 1 {
		t.Fatalf("forced deep calls = %d, want 1", gotCalls)
	}
	for err := range results {
		if err != nil {
			t.Fatalf("ForceDeepStatus error: %v", err)
		}
	}
}

// [REQ:RES-MON-006]
func TestSnapshotProviderClonesResultsAndPreservesPartialPopulation(t *testing.T) {
	executor := &snapshotExecutor{
		responses: map[string]struct {
			output []byte
			err    error
		}{"vrooli resource status --health-only --json": {output: []byte(`{"success":true,"resources":[{"resource":{"name":"postgres"},"installed":true,"running":true,"healthy":true,"message":"healthy"}],"discovery_failures":[{"name":"redis","error":"timeout"}]}`)}},
		calls: make(map[string]int),
	}
	provider := NewSnapshotProvider(NewClient(executor), WithSnapshotTTL(time.Minute))
	one, err := provider.Snapshot(context.Background(), []string{"postgres"})
	if err != nil {
		t.Fatalf("first Snapshot: %v", err)
	}
	if one.Complete {
		t.Fatal("partial fleet response must not be complete")
	}
	healthy := false
	one.Resources["postgres"] = ResourceStatus{Healthy: &healthy}
	two, err := provider.Snapshot(context.Background(), []string{"postgres"})
	if err != nil {
		t.Fatalf("second Snapshot: %v", err)
	}
	if two.Resources["postgres"].Healthy == nil || !*two.Resources["postgres"].Healthy {
		t.Fatalf("snapshot was mutated through caller copy: %+v", two.Resources["postgres"])
	}
}

// [REQ:RES-MON-006] [REQ:RES-MON-007]
func TestSnapshotProviderFailureIsFallbackAndBackedOff(t *testing.T) {
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	current := now
	executor := &snapshotExecutor{
		responses: map[string]struct {
			output []byte
			err    error
		}{"vrooli resource status --health-only --json": {output: fleetJSON()}},
		calls: make(map[string]int),
	}
	provider := NewSnapshotProvider(NewClient(executor),
		WithSnapshotTTL(time.Second),
		WithSnapshotRetryBackoff(10*time.Second),
		WithSnapshotClock(func() time.Time { return current }),
	)
	if _, err := provider.Snapshot(context.Background(), []string{"postgres"}); err != nil {
		t.Fatalf("initial Snapshot: %v", err)
	}
	executor.mu.Lock()
	executor.responses["vrooli resource status --health-only --json"] = struct {
		output []byte
		err    error
	}{err: errors.New("fixture timeout")}
	executor.mu.Unlock()
	current = current.Add(2 * time.Second)
	fallback, err := provider.Snapshot(context.Background(), []string{"postgres"})
	if err == nil {
		t.Fatal("expected refresh failure")
	}
	if fallback.Source != SnapshotSourceFallback || fallback.Complete || fallback.ProbeLevel != ProbeLevelUndetermined {
		t.Fatalf("fallback metadata = %+v", fallback)
	}
	if _, err := provider.Snapshot(context.Background(), []string{"postgres"}); err == nil {
		t.Fatal("backoff result should remain unavailable")
	}
	executor.mu.Lock()
	gotCalls := executor.calls["vrooli resource status --health-only --json"]
	executor.mu.Unlock()
	if gotCalls != 2 {
		t.Fatalf("refresh calls during backoff = %d, want 2", gotCalls)
	}
	metrics := provider.SnapshotMetrics()
	if metrics.RefreshFailures != 1 || metrics.StaleReads != 1 {
		t.Fatalf("failure metrics = %+v", metrics)
	}
	if !fallback.NextRefreshAt.Equal(now.Add(12 * time.Second)) {
		t.Fatalf("fallback next refresh = %s, want %s", fallback.NextRefreshAt, now.Add(12*time.Second))
	}
}

// [REQ:RES-MON-001] [REQ:RES-MON-007]
func TestSnapshotProviderStablePopulationUsesFleetOnly(t *testing.T) {
	executor := &snapshotExecutor{
		responses: map[string]struct {
			output []byte
			err    error
		}{"vrooli resource status --health-only --json": {output: []byte(`{"success":true,"resources":[
			{"resource":{"name":"postgres"},"installed":true,"running":true,"healthy":true,"message":"healthy"},
			{"resource":{"name":"redis"},"installed":true,"running":true,"healthy":true,"message":"healthy"},
			{"resource":{"name":"qdrant"},"installed":true,"running":true,"healthy":true,"message":"healthy"}]}`)}},
		calls: make(map[string]int),
	}
	provider := NewSnapshotProvider(NewClient(executor), WithSnapshotTTL(time.Minute))
	for _, name := range []string{"postgres", "redis", "qdrant"} {
		snapshot, err := provider.Snapshot(context.Background(), []string{name})
		if err != nil {
			t.Fatalf("Snapshot(%q): %v", name, err)
		}
		if !snapshot.FreshAt(time.Now()) || snapshot.ProbeLevel != ProbeLevelFast {
			t.Fatalf("stable snapshot for %q = %+v", name, snapshot)
		}
	}
	executor.mu.Lock()
	fleetCalls := executor.calls["vrooli resource status --health-only --json"]
	namedCalls := executor.calls["vrooli resource status postgres --health-only --json"] +
		executor.calls["vrooli resource status redis --health-only --json"] +
		executor.calls["vrooli resource status qdrant --health-only --json"]
	executor.mu.Unlock()
	if fleetCalls != 1 || namedCalls != 0 {
		t.Fatalf("stable population calls = fleet:%d named:%d, want fleet:1 named:0", fleetCalls, namedCalls)
	}
	metrics := provider.SnapshotMetrics()
	if metrics.FleetRefreshes != 1 || metrics.NamedDeepChecks != 0 {
		t.Fatalf("stable population metrics = %+v", metrics)
	}
}
