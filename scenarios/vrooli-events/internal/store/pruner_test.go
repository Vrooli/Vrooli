package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// [REQ:REQ-ES-003] Verify pruner loop runs periodically and stops on context cancellation
func TestStartPrunerCancellation(t *testing.T) {
	s := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		StartPruner(ctx, PrunerConfig{
			Interval: 50 * time.Millisecond,
			Store:    s,
		})
		close(done)
	}()

	// Let it run a couple cycles
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK - pruner stopped
	case <-time.After(2 * time.Second):
		t.Fatal("pruner did not stop after context cancellation")
	}
}

// [REQ:REQ-ES-003] Verify pruner actually calls Store.Prune on each tick
func TestStartPrunerCallsStore(t *testing.T) {
	var mu sync.Mutex
	pruneCalls := 0

	mock := &mockPrunerStore{
		pruneFunc: func(_ context.Context) (PruneResult, error) {
			mu.Lock()
			defer mu.Unlock()
			pruneCalls++
			return PruneResult{TimeDeletedCount: 1, SizeDeletedCount: 0}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StartPruner(ctx, PrunerConfig{
		Interval: 20 * time.Millisecond,
		Store:    mock,
	})

	// Wait enough time for at least 3 ticks
	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	got := pruneCalls
	mu.Unlock()

	if got < 3 {
		t.Errorf("expected at least 3 prune calls, got %d", got)
	}
}

// [REQ:REQ-ES-003] Verify pruner logs errors from Store.Prune and continues running
func TestStartPrunerLogsErrorsAndContinues(t *testing.T) {
	var mu sync.Mutex
	pruneCalls := 0
	errReturned := false

	mock := &mockPrunerStore{
		pruneFunc: func(_ context.Context) (PruneResult, error) {
			mu.Lock()
			defer mu.Unlock()
			pruneCalls++
			if pruneCalls == 1 {
				errReturned = true
				return PruneResult{}, fmt.Errorf("db locked")
			}
			return PruneResult{}, nil
		},
	}

	var logMu sync.Mutex
	var logs []string
	logger := func(format string, args ...any) {
		logMu.Lock()
		defer logMu.Unlock()
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StartPruner(ctx, PrunerConfig{
		Interval: 20 * time.Millisecond,
		Store:    mock,
		Logger:   logger,
	})

	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	gotCalls := pruneCalls
	gotErr := errReturned
	mu.Unlock()

	if !gotErr {
		t.Fatal("expected first prune call to return error")
	}
	if gotCalls < 2 {
		t.Errorf("expected pruner to continue after error, got only %d calls", gotCalls)
	}

	logMu.Lock()
	defer logMu.Unlock()
	foundErrLog := false
	for _, l := range logs {
		if strings.Contains(l, "db locked") {
			foundErrLog = true
			break
		}
	}
	if !foundErrLog {
		t.Error("expected error to be logged, but 'db locked' not found in logs")
	}
}

// [REQ:REQ-ES-003] Verify pruner defaults to 6-hour interval when zero
func TestStartPrunerDefaultInterval(t *testing.T) {
	cfg := PrunerConfig{
		Interval: 0,
		Store:    newTestStore(t),
	}

	// The pruner applies defaults internally. We verify by checking that it
	// doesn't prune immediately (6h interval means no tick in 50ms).
	var mu sync.Mutex
	pruneCalls := 0
	mock := &mockPrunerStore{
		pruneFunc: func(_ context.Context) (PruneResult, error) {
			mu.Lock()
			defer mu.Unlock()
			pruneCalls++
			return PruneResult{}, nil
		},
	}
	cfg.Store = mock

	ctx, cancel := context.WithCancel(context.Background())
	go StartPruner(ctx, cfg)
	time.Sleep(50 * time.Millisecond)
	cancel()

	mu.Lock()
	got := pruneCalls
	mu.Unlock()

	if got != 0 {
		t.Errorf("expected 0 prune calls with default 6h interval in 50ms, got %d", got)
	}
}

// [REQ:REQ-ES-003] Verify pruner logs deletion counts when events are pruned
func TestStartPrunerLogsDeletionCounts(t *testing.T) {
	mock := &mockPrunerStore{
		pruneFunc: func(_ context.Context) (PruneResult, error) {
			return PruneResult{TimeDeletedCount: 5, SizeDeletedCount: 3}, nil
		},
	}

	var logMu sync.Mutex
	var logs []string
	logger := func(format string, args ...any) {
		logMu.Lock()
		defer logMu.Unlock()
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StartPruner(ctx, PrunerConfig{
		Interval: 20 * time.Millisecond,
		Store:    mock,
		Logger:   logger,
	})

	time.Sleep(50 * time.Millisecond)
	cancel()

	logMu.Lock()
	defer logMu.Unlock()

	if len(logs) == 0 {
		t.Fatal("expected at least one log entry for pruned events")
	}
	found := false
	for _, l := range logs {
		if strings.Contains(l, "pruned 8 events") && strings.Contains(l, "time: 5") && strings.Contains(l, "size: 3") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected log with deletion counts, got: %v", logs)
	}
}

// mockPrunerStore is a minimal mock for pruner tests that only need Prune().
type mockPrunerStore struct {
	pruneFunc func(ctx context.Context) (PruneResult, error)
}

func (m *mockPrunerStore) Insert(_ context.Context, _ Event) (int64, error) { return 0, nil }
func (m *mockPrunerStore) Query(_ context.Context, _ QueryFilters) ([]Event, error) {
	return nil, nil
}
func (m *mockPrunerStore) GetSince(_ context.Context, _ int64, _ int) ([]Event, error) {
	return nil, nil
}
func (m *mockPrunerStore) Prune(ctx context.Context) (PruneResult, error) {
	return m.pruneFunc(ctx)
}
func (m *mockPrunerStore) Stats(_ context.Context) (Stats, error) { return Stats{}, nil }
func (m *mockPrunerStore) Close() error                           { return nil }
