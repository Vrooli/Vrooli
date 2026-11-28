package validation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryEvidenceStore struct {
	mu      sync.Mutex
	records map[EvidenceKey]EvidenceRecord
	loads   int
	stores  int
	err     error
}

func (s *memoryEvidenceStore) Load(_ context.Context, key EvidenceKey, now time.Time) (EvidenceRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loads++
	if s.err != nil {
		return EvidenceRecord{}, false, s.err
	}
	record, ok := s.records[key]
	if !ok || !record.ExpiresAt.After(now) {
		return EvidenceRecord{}, false, nil
	}
	return record, true, nil
}

func (s *memoryEvidenceStore) Store(_ context.Context, record EvidenceRecord, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	if s.records == nil {
		s.records = make(map[EvidenceKey]EvidenceRecord)
	}
	s.records[record.Key] = record
	s.stores++
	return nil
}

func (s *memoryEvidenceStore) DeleteExpired(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func TestEvidenceCoordinatorReusesSuccessfulEvidence(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	store := &memoryEvidenceStore{}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{
		Store: store, Clock: func() time.Time { return now }, Capacity: 2,
	})
	key := EvidenceKey{Scenario: "demo", Scanner: "gosec", Fingerprint: "sha256:abc"}
	want := []Finding{{RuleID: "gosec.G101", Scanner: "gosec", Severity: SeverityError}}
	var runs atomic.Int32
	run := func(context.Context) ([]Finding, error) {
		runs.Add(1)
		return want, nil
	}

	first, firstOutcome, err := coordinator.Execute(context.Background(), key, 2, time.Hour, run)
	if err != nil {
		t.Fatal(err)
	}
	second, secondOutcome, err := coordinator.Execute(context.Background(), key, 2, time.Hour, run)
	if err != nil {
		t.Fatal(err)
	}
	if runs.Load() != 1 {
		t.Fatalf("scanner runs = %d, want 1", runs.Load())
	}
	if firstOutcome.Source != EvidenceSourceExecution || secondOutcome.Source != EvidenceSourceCache {
		t.Fatalf("sources = %q, %q", firstOutcome.Source, secondOutcome.Source)
	}
	if len(first) != 1 || len(second) != 1 || first[0].RuleID != second[0].RuleID {
		t.Fatalf("findings differ: first=%v second=%v", first, second)
	}
	second[0].RuleID = "mutated"
	third, _, err := coordinator.Execute(context.Background(), key, 2, time.Hour, run)
	if err != nil {
		t.Fatal(err)
	}
	if third[0].RuleID != want[0].RuleID {
		t.Fatal("caller mutated cached evidence")
	}
	metrics := coordinator.Metrics().Scanners["gosec"]
	if metrics.Hits != 2 || metrics.Misses != 1 || metrics.Executions != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

// [REQ:REQ-P0-023]
func TestEvidenceCoordinatorCoalescesIdenticalMisses(t *testing.T) {
	t.Log("[REQ:REQ-P0-023]")
	store := &memoryEvidenceStore{}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: store, Capacity: 2})
	key := EvidenceKey{Scenario: "demo", Scanner: "gitleaks", Fingerprint: "sha256:def"}
	started := make(chan struct{})
	release := make(chan struct{})
	var runs atomic.Int32
	run := func(context.Context) ([]Finding, error) {
		if runs.Add(1) == 1 {
			close(started)
		}
		<-release
		return []Finding{{RuleID: "gitleaks.generic", Scanner: "gitleaks"}}, nil
	}

	type result struct {
		outcome EvidenceOutcome
		err     error
	}
	results := make(chan result, 2)
	go func() {
		_, outcome, err := coordinator.Execute(context.Background(), key, 2, time.Hour, run)
		results <- result{outcome: outcome, err: err}
	}()
	<-started
	go func() {
		_, outcome, err := coordinator.Execute(context.Background(), key, 2, time.Hour, run)
		results <- result{outcome: outcome, err: err}
	}()

	deadline := time.Now().Add(time.Second)
	for coordinator.Metrics().Scanners["gitleaks"].Coalesced == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(release)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("errors = %v, %v", first.err, second.err)
	}
	if runs.Load() != 1 {
		t.Fatalf("scanner runs = %d, want 1", runs.Load())
	}
	if first.outcome.Source == second.outcome.Source {
		t.Fatalf("sources = %q and %q, want execution + coalesced", first.outcome.Source, second.outcome.Source)
	}
}

func TestEvidenceCoordinatorDoesNotCacheFailures(t *testing.T) {
	t.Log("[REQ:REQ-P0-023]")
	store := &memoryEvidenceStore{}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: store, Capacity: 1})
	key := EvidenceKey{Scenario: "demo", Scanner: "osv", Fingerprint: "sha256:ghi"}
	wantErr := errors.New("scanner failed")
	var runs atomic.Int32
	run := func(context.Context) ([]Finding, error) {
		runs.Add(1)
		return nil, wantErr
	}
	for range 2 {
		if _, _, err := coordinator.Execute(context.Background(), key, 1, time.Hour, run); !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	}
	if runs.Load() != 2 || store.stores != 0 {
		t.Fatalf("runs=%d stores=%d, want 2/0", runs.Load(), store.stores)
	}
}

func TestEvidenceCoordinatorDoesNotCacheCancellation(t *testing.T) {
	store := &memoryEvidenceStore{}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: store, Capacity: 1})
	key := EvidenceKey{Scenario: "demo", Scanner: "govulncheck", Fingerprint: "sha256:cancelled"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := coordinator.Execute(ctx, key, 1, time.Hour, func(ctx context.Context) ([]Finding, error) {
		return nil, ctx.Err()
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if store.stores != 0 {
		t.Fatalf("cancelled result stores = %d, want 0", store.stores)
	}
}

func TestEvidenceCoordinatorCacheFailureRunsScanner(t *testing.T) {
	store := &memoryEvidenceStore{err: errors.New("cache unavailable")}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: store, Capacity: 1})
	key := EvidenceKey{Scenario: "demo", Scanner: "gosec", Fingerprint: "sha256:jkl"}
	var runs atomic.Int32
	_, _, err := coordinator.Execute(context.Background(), key, 1, time.Hour, func(context.Context) ([]Finding, error) {
		runs.Add(1)
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if runs.Load() != 1 {
		t.Fatalf("scanner runs = %d, want 1", runs.Load())
	}
	metrics := coordinator.Metrics().Scanners["gosec"]
	if metrics.CacheErrors != 2 || metrics.Executions != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

// [REQ:REQ-P0-023]
func TestWeightedGateBoundsUsageAndReleasesCancellation(t *testing.T) {
	t.Log("[REQ:REQ-P0-023]")
	gate := NewWeightedGate(3)
	if err := gate.Acquire(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gate.Acquire(ctx, 2) }()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
	gate.Release(2)
	if err := gate.Acquire(context.Background(), 3); err != nil {
		t.Fatalf("capacity was not released: %v", err)
	}
	capacity, inUse, peak := gate.Snapshot()
	if capacity != 3 || inUse != 3 || peak > capacity {
		t.Fatalf("gate snapshot = capacity=%d inUse=%d peak=%d", capacity, inUse, peak)
	}
	gate.Release(3)
}

func TestEvidenceCoordinatorSharesCapacityAcrossScannerKeys(t *testing.T) {
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Capacity: 3})
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	result := make(chan error, 2)
	run := func(started chan struct{}, release <-chan struct{}) EvidenceRun {
		return func(context.Context) ([]Finding, error) {
			close(started)
			if release != nil {
				<-release
			}
			return nil, nil
		}
	}
	go func() {
		_, _, err := coordinator.Execute(context.Background(), EvidenceKey{
			Scenario: "demo", Scanner: "gosec", Fingerprint: "sha256:one",
		}, 2, time.Hour, run(firstStarted, releaseFirst))
		result <- err
	}()
	<-firstStarted
	go func() {
		_, _, err := coordinator.Execute(context.Background(), EvidenceKey{
			Scenario: "demo", Scanner: "gitleaks", Fingerprint: "sha256:two",
		}, 2, time.Hour, run(secondStarted, nil))
		result <- err
	}()
	select {
	case <-secondStarted:
		t.Fatal("second scanner exceeded the shared weighted capacity")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("second scanner did not start after capacity was released")
	}
	for range 2 {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	metrics := coordinator.Metrics()
	if metrics.PeakUse > metrics.Capacity {
		t.Fatalf("peak use %d exceeded capacity %d", metrics.PeakUse, metrics.Capacity)
	}
}

func TestEvidenceCoordinatorRejectsIncompleteCorrectnessKey(t *testing.T) {
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Capacity: 1})
	_, _, err := coordinator.Execute(context.Background(), EvidenceKey{Scenario: "demo", Scanner: "gosec"}, 1, time.Hour, func(context.Context) ([]Finding, error) {
		t.Fatal("scanner ran without a correctness fingerprint")
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected incomplete key error")
	}
}
