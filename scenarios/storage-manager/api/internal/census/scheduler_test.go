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
	})
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
