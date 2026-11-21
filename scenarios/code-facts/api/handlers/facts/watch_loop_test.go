package facts

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// countingRefresh records every refresh run with its wall-clock instant.
type countingRefresh struct {
	count atomic.Int64
	times chan time.Time
}

func newCountingRefresh() *countingRefresh {
	return &countingRefresh{times: make(chan time.Time, 64)}
}

func (c *countingRefresh) run() {
	c.count.Add(1)
	c.times <- time.Now()
}

func (c *countingRefresh) next(t *testing.T, within time.Duration) time.Time {
	t.Helper()
	select {
	case at := <-c.times:
		return at
	case <-time.After(within):
		t.Fatalf("no refresh within %s", within)
		return time.Time{}
	}
}

func (c *countingRefresh) expectNone(t *testing.T, within time.Duration) {
	t.Helper()
	select {
	case at := <-c.times:
		t.Fatalf("unexpected refresh at %s", at.Format(time.RFC3339Nano))
	case <-time.After(within):
	}
}

func startLoop(t *testing.T, config watchLoopConfig, refresh *countingRefresh) (chan struct{}, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runWatchLoop(ctx, events, config, refresh.run, func() {})
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return events, cancel
}

func fireBurst(events chan struct{}, n int) {
	for i := 0; i < n; i++ {
		select {
		case events <- struct{}{}:
		default:
		}
	}
}

func TestWatchLoopRunsOnceAtStartup(t *testing.T) {
	refresh := newCountingRefresh()
	startLoop(t, watchLoopConfig{debounce: 20 * time.Millisecond, minInterval: 100 * time.Millisecond, backstop: time.Hour}, refresh)
	refresh.next(t, time.Second)
	refresh.expectNone(t, 300*time.Millisecond)
}

func TestWatchLoopCoalescesBurstIntoOneRefresh(t *testing.T) {
	refresh := newCountingRefresh()
	const debounce = 30 * time.Millisecond
	const minInterval = 300 * time.Millisecond
	events, _ := startLoop(t, watchLoopConfig{debounce: debounce, minInterval: minInterval, backstop: time.Hour}, refresh)
	first := refresh.next(t, time.Second)

	// A burst of events spread across the debounce window becomes one run.
	for i := 0; i < 10; i++ {
		fireBurst(events, 5)
		time.Sleep(5 * time.Millisecond)
	}
	second := refresh.next(t, 2*time.Second)
	if spacing := second.Sub(first); spacing < minInterval-10*time.Millisecond {
		t.Fatalf("second refresh ran %s after the startup run, want at least the %s minimum interval", spacing, minInterval)
	}
	refresh.expectNone(t, minInterval/2)
	if got := refresh.count.Load(); got != 2 {
		t.Fatalf("burst produced %d refreshes, want exactly 2 (startup + one coalesced run)", got)
	}
}

func TestWatchLoopSpacesRefreshesByMinimumInterval(t *testing.T) {
	refresh := newCountingRefresh()
	const debounce = 20 * time.Millisecond
	const minInterval = 300 * time.Millisecond
	events, _ := startLoop(t, watchLoopConfig{debounce: debounce, minInterval: minInterval, backstop: time.Hour}, refresh)
	refresh.next(t, time.Second)

	// Wait out the startup spacing so the next event is only debounced.
	time.Sleep(minInterval + 20*time.Millisecond)
	fireBurst(events, 1)
	first := refresh.next(t, time.Second)

	// A continuous stream of events must not run a second refresh inside the
	// minimum interval, and must run exactly one follow-up once it elapses.
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fireBurst(events, 1)
			}
		}
	}()
	refresh.expectNone(t, minInterval-50*time.Millisecond)
	second := refresh.next(t, time.Second)
	close(stop)
	if spacing := second.Sub(first); spacing < minInterval-10*time.Millisecond {
		t.Fatalf("second refresh ran %s after the first, want at least %s", spacing, minInterval)
	}
	if spacing := second.Sub(first); spacing > minInterval+150*time.Millisecond {
		t.Fatalf("follow-up refresh ran %s after the first, want close to %s", spacing, minInterval)
	}
}

func TestWatchLoopEventsDuringRefreshProduceOneFollowUp(t *testing.T) {
	const debounce = 20 * time.Millisecond
	const minInterval = 200 * time.Millisecond
	var calls atomic.Int64
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runWatchLoop(ctx, events, watchLoopConfig{debounce: debounce, minInterval: minInterval, backstop: time.Hour}, func() {
			n := calls.Add(1)
			started <- struct{}{}
			if n == 2 {
				<-release
			}
		}, func() {})
	}()
	<-started // startup run
	time.Sleep(minInterval + 20*time.Millisecond)
	fireBurst(events, 1)
	<-started // second run, now blocked on release
	for i := 0; i < 20; i++ {
		fireBurst(events, 3)
		time.Sleep(2 * time.Millisecond)
	}
	close(release)
	select {
	case <-started: // exactly one follow-up run
	case <-time.After(2 * time.Second):
		t.Fatal("events during a refresh did not produce a follow-up run")
	}
	select {
	case <-started:
		t.Fatal("events during a refresh produced more than one follow-up run")
	case <-time.After(minInterval + 100*time.Millisecond):
	}
	cancel()
	<-done
	if got := calls.Load(); got != 3 {
		t.Fatalf("refresh ran %d times, want 3 (startup, triggered, one coalesced follow-up)", got)
	}
}
