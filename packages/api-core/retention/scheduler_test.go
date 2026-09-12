package retention

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// fakeClock drives scheduler cycles without sleeping, so a 15-minute interval
// costs nothing to test.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
	asked  []time.Duration
}

type fakeTimer struct {
	ch      chan time.Time
	stopped bool
}

func (t *fakeTimer) C() <-chan time.Time { return t.ch }

func (t *fakeTimer) Stop() bool {
	t.stopped = true
	return true
}
func (t *fakeTimer) Reset(time.Duration) bool { t.stopped = false; return true }

func newFakeClock(start time.Time) *fakeClock { return &fakeClock{now: start} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) NewTimer(d time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{ch: make(chan time.Time, 1)}
	c.timers = append(c.timers, t)
	c.asked = append(c.asked, d)
	return t
}
func (c *fakeClock) NewTicker(d time.Duration) schedule.Ticker { return schedule.System().NewTicker(d) }
func (c *fakeClock) Sleep(d time.Duration)                     { c.now = c.now.Add(d) }

// fire advances the clock and fires the most recently created timer, blocking
// until one exists so the test never races scheduler startup.
func (c *fakeClock) fire(t *testing.T, d time.Duration) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		c.mu.Lock()
		if len(c.timers) > 0 {
			timer := c.timers[len(c.timers)-1]
			c.timers = c.timers[:len(c.timers)-1]
			c.now = c.now.Add(d)
			now := c.now
			c.mu.Unlock()
			timer.ch <- now
			return
		}
		c.mu.Unlock()
		select {
		case <-deadline:
			t.Fatal("no timer was created; the scheduler never armed one")
		case <-time.After(time.Millisecond):
		}
	}
}

func (c *fakeClock) intervalsAsked() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.asked...)
}

func TestSchedulerRequiresAnEngine(t *testing.T) {
	if _, err := NewScheduler(SchedulerConfig{}); err == nil {
		t.Fatal("expected a scheduler without an engine to be rejected")
	}
}

func TestSchedulerDefaultsInterval(t *testing.T) {
	engine, err := NewEngine(EngineConfig{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	s, err := NewScheduler(SchedulerConfig{Engine: engine})
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	if s.Interval() != DefaultInterval {
		t.Fatalf("Interval() = %v, want %v", s.Interval(), DefaultInterval)
	}
}

func TestSchedulerRunsOnStartAndOnEveryTick(t *testing.T) {
	var mu sync.Mutex
	cycles := 0

	pruner := &fakePruner{before: Usage{Bytes: 10}, after: Usage{Bytes: 5}, result: Result{Deleted: 1, After: Usage{Bytes: 5}}}
	engine := newTestEngine(t, specFor("b", Budget{MaxBytes: 100}, PrunerBuiltin), pruner)

	clock := newFakeClock(time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	done := make(chan struct{})
	s, err := NewScheduler(SchedulerConfig{
		Engine:     engine,
		Interval:   15 * time.Minute,
		Clock:      clock,
		RunOnStart: true,
		OnCycle: func(results []Result, err error) {
			mu.Lock()
			cycles++
			mu.Unlock()
			if err != nil {
				t.Errorf("cycle error: %v", err)
			}
			if len(results) != 1 {
				t.Errorf("cycle produced %d results, want 1", len(results))
			}
			done <- struct{}{}
		},
	})
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)

	<-done // the RunOnStart cycle
	clock.fire(t, 15*time.Minute)
	<-done // the first tick
	clock.fire(t, 15*time.Minute)
	<-done // the second tick

	cancel()
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after its context was cancelled")
	}

	mu.Lock()
	got := cycles
	mu.Unlock()
	if got < 3 {
		t.Fatalf("ran %d cycles, want at least 3", got)
	}
	for _, d := range clock.intervalsAsked() {
		if d != 15*time.Minute {
			t.Fatalf("armed a timer for %v, want the configured 15m interval", d)
		}
	}
}

func TestSchedulerStopsOnContextCancel(t *testing.T) {
	engine, err := NewEngine(EngineConfig{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	s, err := NewScheduler(SchedulerConfig{Engine: engine, Interval: time.Hour, Clock: newFakeClock(time.Now())})
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)
	cancel()
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after its context was cancelled")
	}
}

func TestSchedulerSurvivesACycleError(t *testing.T) {
	// A transient failure to open a database must not silently end retention for
	// the life of the process.
	var mu sync.Mutex
	errs := 0
	pruner := &fakePruner{measureErr: context.DeadlineExceeded}
	engine := newTestEngine(t, specFor("b", Budget{MaxBytes: 1}, PrunerBuiltin), pruner)

	clock := newFakeClock(time.Now())
	done := make(chan struct{}, 4)
	s, err := NewScheduler(SchedulerConfig{
		Engine:     engine,
		Interval:   time.Minute,
		Clock:      clock,
		RunOnStart: true,
		OnCycle: func(_ []Result, err error) {
			mu.Lock()
			if err != nil {
				errs++
			}
			mu.Unlock()
			done <- struct{}{}
		},
	})
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx)

	<-done
	clock.fire(t, time.Minute)
	<-done

	mu.Lock()
	got := errs
	mu.Unlock()
	if got < 2 {
		t.Fatalf("observed %d cycle errors, want the scheduler to keep cycling after a failure", got)
	}
}
