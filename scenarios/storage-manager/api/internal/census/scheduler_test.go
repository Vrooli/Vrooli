package census

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunOnceDelegatesWithoutStartingAHostScan(t *testing.T) {
	var calls atomic.Int32
	scheduler := NewScheduler(time.Hour, func(context.Context) error {
		calls.Add(1)
		return nil
	})
	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
}

func TestSchedulerWaitsForItsIntervalAndStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := make(chan struct{}, 1)
	scheduler := NewScheduler(10*time.Millisecond, func(context.Context) error {
		called <- struct{}{}
		return nil
	})
	scheduler.Start(ctx)
	select {
	case <-called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("scheduler did not run after its interval")
	}
	cancel()
}

// A census that runs longer than its interval must still leave the host a full
// interval of quiet before the next walk. This is the regression guard for the
// time.Ticker implementation, whose buffered pending tick restarted an
// overrunning census immediately and kept the host under a continuous
// metadata walk.
func TestSchedulerYieldsAFullIntervalAfterAnOverrunningCycle(t *testing.T) {
	const (
		interval    = 50 * time.Millisecond
		cycleLength = 200 * time.Millisecond // deliberately 4x the interval
		wantCycles  = 3
	)

	type span struct{ startedAt, endedAt time.Time }

	var (
		mu    sync.Mutex
		spans []span
	)
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The duty-cycle cap is lifted so this test isolates the wait-from-end
	// guarantee; TestSchedulerIdlesLongerAfterALongPass covers the cap.
	scheduler := NewScheduler(interval, func(context.Context) error {
		startedAt := time.Now()
		time.Sleep(cycleLength)
		mu.Lock()
		spans = append(spans, span{startedAt: startedAt, endedAt: time.Now()})
		count := len(spans)
		mu.Unlock()
		if count == wantCycles {
			close(done)
		}
		return nil
	}).WithDutyCycle(1)
	scheduler.Start(ctx)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not complete enough cycles")
	}
	cancel()

	mu.Lock()
	defer mu.Unlock()

	// Timer wakeups can be late but never early; allow scheduling slack below
	// the interval while still failing decisively on the ~0ms gap the ticker
	// implementation produced.
	const minGap = 40 * time.Millisecond
	for i := 1; i < wantCycles; i++ {
		gap := spans[i].startedAt.Sub(spans[i-1].endedAt)
		if gap < minGap {
			t.Fatalf("cycle %d started %v after the previous cycle ended, want at least %v: an overrunning census must not restart immediately",
				i, gap, minGap)
		}
	}
}

func TestSchedulerCountsOverrunsAndRecordsTheLastCycle(t *testing.T) {
	const interval = 20 * time.Millisecond
	wantErr := errors.New("census failed")

	cycles := make(chan Cycle, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := NewScheduler(interval, func(context.Context) error {
		time.Sleep(3 * interval) // every cycle overruns
		return wantErr
	}).WithObserver(func(c Cycle) { cycles <- c })
	scheduler.Start(ctx)

	var observed Cycle
	select {
	case observed = <-cycles:
	case <-time.After(3 * time.Second):
		t.Fatal("observer never saw a cycle")
	}
	cancel()

	if !observed.Overran {
		t.Errorf("Overran = false, want true for a cycle of %v against a %v interval", observed.Duration, interval)
	}
	if !errors.Is(observed.Err, wantErr) {
		t.Errorf("Err = %v, want %v", observed.Err, wantErr)
	}
	if observed.Duration < interval {
		t.Errorf("Duration = %v, want at least %v", observed.Duration, interval)
	}

	stats := scheduler.Stats()
	if stats.Cycles < 1 {
		t.Errorf("Cycles = %d, want at least 1", stats.Cycles)
	}
	if stats.Overruns != stats.Cycles {
		t.Errorf("Overruns = %d, Cycles = %d; every cycle in this test overruns", stats.Overruns, stats.Cycles)
	}
	if !errors.Is(stats.Last.Err, wantErr) {
		t.Errorf("Last.Err = %v, want %v", stats.Last.Err, wantErr)
	}
}

func TestSchedulerStopsWhenContextIsCancelledDuringACycle(t *testing.T) {
	const interval = 10 * time.Millisecond
	var calls atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := NewScheduler(interval, func(context.Context) error {
		calls.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	scheduler.Start(ctx)

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler never entered a cycle")
	}

	// Cancel while the cycle is still in flight, then let it finish.
	cancel()
	close(release)

	time.Sleep(20 * interval)
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1: cancellation during a cycle must stop the loop", got)
	}
}

func TestIdleAfterKeepsCensusWorkUnderTheDutyCycleCap(t *testing.T) {
	for _, tc := range []struct {
		name               string
		duration, interval time.Duration
		duty               float64
		want               time.Duration
	}{
		{name: "short pass waits one interval", duration: time.Minute, interval: 6 * time.Hour, duty: 0.1, want: 6 * time.Hour},
		{name: "85 minute pass on a 30 minute interval idles 9x the pass", duration: 85 * time.Minute, interval: 30 * time.Minute, duty: 0.1, want: 765 * time.Minute},
		{name: "cap of one is plain wait-from-end", duration: time.Hour, interval: 10 * time.Minute, duty: 1, want: 10 * time.Minute},
		{name: "invalid cap falls back to the default", duration: time.Hour, interval: time.Minute, duty: -3, want: 9 * time.Hour},
		{name: "zero duration waits one interval", duration: 0, interval: time.Minute, duty: 0.1, want: time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := IdleAfter(tc.duration, tc.interval, tc.duty)
			if got != tc.want {
				t.Fatalf("IdleAfter(%s, %s, %v) = %s, want %s", tc.duration, tc.interval, tc.duty, got, tc.want)
			}
			if tc.duty > 0 && tc.duty <= 1 && tc.duration > 0 {
				share := float64(tc.duration) / float64(tc.duration+got)
				if share > tc.duty+1e-9 {
					t.Fatalf("census share of wall time = %.3f, exceeds cap %.3f", share, tc.duty)
				}
			}
		})
	}
}

// A pass that would exceed the duty-cycle cap on a plain wait-from-end
// schedule must push the next pass out until the cap holds, and the observer
// must see the chosen idle gap once per pass.
func TestSchedulerIdlesLongerAfterALongPass(t *testing.T) {
	const (
		interval = 10 * time.Millisecond
		pass     = 50 * time.Millisecond
		duty     = 0.2 // idle must be at least 4x the pass: 200ms
	)
	type span struct{ startedAt, endedAt time.Time }
	var (
		mu    sync.Mutex
		spans []span
	)
	cycles := make(chan Cycle, 4)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := NewScheduler(interval, func(context.Context) error {
		startedAt := time.Now()
		time.Sleep(pass)
		mu.Lock()
		spans = append(spans, span{startedAt: startedAt, endedAt: time.Now()})
		if len(spans) == 2 {
			close(done)
		}
		mu.Unlock()
		return nil
	}).WithDutyCycle(duty).WithObserver(func(c Cycle) { cycles <- c })
	scheduler.Start(ctx)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not complete two cycles")
	}
	cancel()

	first := <-cycles
	if !first.Overran {
		t.Fatalf("a %s pass on a %s interval must be reported as an overrun: %+v", pass, interval, first)
	}
	wantIdle := IdleAfter(first.Duration, interval, duty)
	if first.IdleFor != wantIdle || first.IdleFor < 4*pass {
		t.Fatalf("IdleFor = %s, want %s (at least 4x the %s pass)", first.IdleFor, wantIdle, pass)
	}
	mu.Lock()
	gap := spans[1].startedAt.Sub(spans[0].endedAt)
	mu.Unlock()
	if gap < 4*pass-5*time.Millisecond {
		t.Fatalf("second pass started %s after the first ended; the duty-cycle cap requires at least %s", gap, 4*pass)
	}
	if got := len(cycles); got > 1 {
		t.Fatalf("observer saw %d extra cycles for two passes; the overrun must be reported once per pass", got)
	}
}

func TestSchedulerStartIsIdempotent(t *testing.T) {
	const interval = 10 * time.Millisecond
	var calls atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := NewScheduler(interval, func(context.Context) error {
		calls.Add(1)
		return nil
	})
	scheduler.Start(ctx)
	scheduler.Start(ctx)
	scheduler.Start(ctx)

	time.Sleep(15 * interval)
	cancel()

	// One loop over ~15 intervals lands well under the count three concurrent
	// loops would produce; the guard is that Start did not fan out.
	if got := calls.Load(); got > 15 {
		t.Fatalf("calls = %d, want a single loop's worth: Start must not spawn concurrent loops", got)
	}
}
