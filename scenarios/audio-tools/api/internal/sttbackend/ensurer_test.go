package sttbackend

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"audio-tools/internal/controlplane"
)

// newTestEnsurer builds a CLIEnsurer with a fake binary + injectable exec/clock
// so no real `vrooli` is shelled.
func newTestEnsurer(run func(ctx context.Context, bin string, args ...string) error, now func() time.Time) *CLIEnsurer {
	return &CLIEnsurer{
		controlPlane: controlplane.NewForTest("/fake/vrooli", func(ctx context.Context, bin string, args ...string) ([]byte, error) {
			return nil, run(ctx, bin, args...)
		}),
		now:      now,
		timeout:  DefaultEnsureTimeout,
		cooldown: DefaultCooldown,
		inflight: map[string]*ensureCall{},
		last:     map[string]ensureOutcome{},
	}
}

// N concurrent EnsureRunning calls for the same resource trigger exactly ONE
// `resource start` (single-flight).
func TestEnsureRunningSingleFlight(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	entered := make(chan struct{})
	release := make(chan struct{})
	run := func(_ context.Context, _ string, args ...string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		close(entered)
		<-release // hold the start open while the other callers pile up
		return nil
	}
	e := newTestEnsurer(run, time.Now)

	// First caller acquires the in-flight slot and enters the (blocked) start.
	firstDone := make(chan error, 1)
	go func() { firstDone <- e.EnsureRunning(context.Background(), "whisper") }()
	<-entered

	// Now launch a burst that must all JOIN the in-flight start, not spawn new ones.
	const burst = 8
	var wg sync.WaitGroup
	errs := make([]error, burst)
	for i := range burst {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = e.EnsureRunning(context.Background(), "whisper")
		}(i)
	}
	// Give the burst a moment to reach the join point, then release the start.
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if err := <-firstDone; err != nil {
		t.Fatalf("first EnsureRunning err = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("resource start invoked %d times, want exactly 1 (single-flight)", calls)
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("joined caller %d err = %v, want nil", i, err)
		}
	}
}

// A failure within the cooldown window is cached and not re-shelled.
func TestEnsureRunningCooldownSuppressesRetry(t *testing.T) {
	calls := 0
	run := func(_ context.Context, _ string, _ ...string) error {
		calls++
		return errors.New("start failed")
	}
	clk := &fakeClock{t: time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)}
	e := newTestEnsurer(run, clk.now)

	if err := e.EnsureRunning(context.Background(), "whisper"); err == nil {
		t.Fatal("first EnsureRunning should fail")
	}
	// Within cooldown: cached failure, no new exec.
	clk.advance(DefaultCooldown / 2)
	if err := e.EnsureRunning(context.Background(), "whisper"); err == nil {
		t.Fatal("cooldown EnsureRunning should return the cached failure")
	}
	if calls != 1 {
		t.Fatalf("exec called %d times within cooldown, want 1", calls)
	}
	// Past cooldown: a fresh attempt.
	clk.advance(DefaultCooldown + time.Second)
	_ = e.EnsureRunning(context.Background(), "whisper")
	if calls != 2 {
		t.Fatalf("exec called %d times after cooldown, want 2", calls)
	}
}

// A resource outside the allowlist is rejected without shelling anything.
func TestEnsureRunningAllowlist(t *testing.T) {
	calls := 0
	run := func(_ context.Context, _ string, _ ...string) error { calls++; return nil }
	e := newTestEnsurer(run, time.Now)
	err := e.EnsureRunning(context.Background(), "postgres")
	if !errors.Is(err, ErrResourceNotAllowed) {
		t.Fatalf("err = %v, want ErrResourceNotAllowed", err)
	}
	if calls != 0 {
		t.Fatalf("exec called %d times for a disallowed resource, want 0", calls)
	}
}

// A missing vrooli binary returns an error rather than panicking.
func TestEnsureRunningMissingBinary(t *testing.T) {
	e := &CLIEnsurer{controlPlane: controlplane.NewForTest("", nil), now: time.Now, inflight: map[string]*ensureCall{}, last: map[string]ensureOutcome{}}
	if err := e.EnsureRunning(context.Background(), "whisper"); err == nil {
		t.Fatal("EnsureRunning with no vrooli binary should error")
	}
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}
