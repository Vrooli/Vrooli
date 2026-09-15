package fleetscheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeSource struct {
	candidates []Candidate
	err        error
}

func (f fakeSource) Candidates(context.Context) ([]Candidate, error) {
	return f.candidates, f.err
}

// fakeLauncher records launch order and lets tests script per-scenario outcomes.
type fakeLauncher struct {
	mu          sync.Mutex
	launched    []string
	busy        map[string]bool
	status      map[string]string // scenario -> terminal status; default "passed"
	launchErr   map[string]error
	maxInFlight int
	inFlight    int
	holdUntil   chan struct{} // when set, Await blocks until closed
}

func (l *fakeLauncher) Launch(_ context.Context, scenario string) (string, error) {
	l.mu.Lock()
	if l.busy[scenario] {
		l.mu.Unlock()
		return "", ErrScenarioBusy
	}
	if err := l.launchErr[scenario]; err != nil {
		l.mu.Unlock()
		return "", err
	}
	l.launched = append(l.launched, scenario)
	l.inFlight++
	if l.inFlight > l.maxInFlight {
		l.maxInFlight = l.inFlight
	}
	l.mu.Unlock()
	return scenario + "-run", nil
}

func (l *fakeLauncher) Await(_ context.Context, scenario, _ string) (string, error) {
	if l.holdUntil != nil {
		<-l.holdUntil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inFlight--
	st := l.status[scenario]
	if st == "" {
		st = "passed"
	}
	return st, nil
}

func newScheduler(t *testing.T, cfg Config) *Scheduler {
	t.Helper()
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return s
}

func TestSelectByStalenessWeightedPriority(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	src := fakeSource{candidates: []Candidate{
		{Scenario: "fresh-important", Priority: 10, LastRunAt: now.Add(-time.Hour)},
		{Scenario: "stale-mid", Priority: 6, LastRunAt: now.Add(-7 * 24 * time.Hour)},
		{Scenario: "never-tested", Priority: 4},
		{Scenario: "fresh-low", Priority: 1, LastRunAt: now.Add(-time.Hour)},
	}}
	lr := &fakeLauncher{}
	s := newScheduler(t, Config{
		Source: src, Launcher: lr, Now: func() time.Time { return now },
		MaxRunsPerCycle: 3, MaxConcurrent: 1, StalenessHorizon: 7 * 24 * time.Hour,
	})
	rep, err := s.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if rep.Selected != 3 || rep.Launched != 3 || rep.Passed != 3 {
		t.Fatalf("report = %+v, want 3 selected/launched/passed", rep)
	}
	// never-tested (4*2.5=10) and stale-mid (6*2.0=12) outrank fresh-important
	// (10*1.0=10 ... ties broken by name, but stale-mid=12 wins outright);
	// fresh-low (1) is dropped. Expect stale-mid first, then never-tested or
	// fresh-important. fresh-low must NOT be launched.
	for _, got := range lr.launched {
		if got == "fresh-low" {
			t.Fatalf("fresh-low was launched but should be dropped: %v", lr.launched)
		}
	}
	if lr.launched[0] != "stale-mid" {
		t.Fatalf("highest-weight first = %q, want stale-mid (order=%v)", lr.launched[0], lr.launched)
	}
}

func TestBusyScenarioSkippedNotErrored(t *testing.T) {
	now := time.Now()
	lr := &fakeLauncher{busy: map[string]bool{"busy-one": true}}
	s := newScheduler(t, Config{
		Source: fakeSource{candidates: []Candidate{
			{Scenario: "busy-one", Priority: 9},
			{Scenario: "ok-two", Priority: 8},
		}},
		Launcher: lr, Now: func() time.Time { return now },
		MaxRunsPerCycle: 5, MaxConcurrent: 1,
	})
	rep, err := s.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if rep.Skipped != 1 || rep.Errored != 0 {
		t.Fatalf("report = %+v, want skipped=1 errored=0", rep)
	}
	if rep.Launched != 1 || rep.Passed != 1 {
		t.Fatalf("report = %+v, want launched=1 passed=1 (ok-two)", rep)
	}
}

func TestFailedRunCounted(t *testing.T) {
	now := time.Now()
	lr := &fakeLauncher{status: map[string]string{"flaky": "failed"}}
	s := newScheduler(t, Config{
		Source:   fakeSource{candidates: []Candidate{{Scenario: "flaky", Priority: 5}}},
		Launcher: lr, Now: func() time.Time { return now }, MaxRunsPerCycle: 5, MaxConcurrent: 1,
	})
	rep, _ := s.RunOnce(context.Background())
	if rep.Launched != 1 || rep.Failed != 1 || rep.Passed != 0 {
		t.Fatalf("report = %+v, want launched=1 failed=1", rep)
	}
}

func TestConcurrencyCapHonored(t *testing.T) {
	now := time.Now()
	hold := make(chan struct{})
	lr := &fakeLauncher{holdUntil: hold}
	candidates := make([]Candidate, 6)
	for i := range candidates {
		candidates[i] = Candidate{Scenario: fmt.Sprintf("s%d", i), Priority: float64(10 - i)}
	}
	s := newScheduler(t, Config{
		Source: fakeSource{candidates: candidates}, Launcher: lr, Now: func() time.Time { return now },
		MaxRunsPerCycle: 6, MaxConcurrent: 2,
	})
	done := make(chan CycleReport, 1)
	go func() {
		rep, _ := s.RunOnce(context.Background())
		done <- rep
	}()
	// Give the workers a moment to saturate, then release.
	time.Sleep(50 * time.Millisecond)
	close(hold)
	rep := <-done
	if rep.Launched != 6 {
		t.Fatalf("launched = %d, want 6", rep.Launched)
	}
	if lr.maxInFlight > 2 {
		t.Fatalf("maxInFlight = %d, want <= 2 (concurrency cap breached)", lr.maxInFlight)
	}
}

func TestSingleFlightCycle(t *testing.T) {
	now := time.Now()
	hold := make(chan struct{})
	lr := &fakeLauncher{holdUntil: hold}
	s := newScheduler(t, Config{
		Source:   fakeSource{candidates: []Candidate{{Scenario: "s", Priority: 1}}},
		Launcher: lr, Now: func() time.Time { return now }, MaxRunsPerCycle: 1, MaxConcurrent: 1,
	})
	go func() { _, _ = s.RunOnce(context.Background()) }()
	time.Sleep(30 * time.Millisecond)
	_, err := s.RunOnce(context.Background())
	if err != ErrCycleInProgress {
		t.Fatalf("concurrent RunOnce err = %v, want ErrCycleInProgress", err)
	}
	close(hold)
}

func TestEmptyFleetIsNoop(t *testing.T) {
	s := newScheduler(t, Config{Source: fakeSource{}, Launcher: &fakeLauncher{}})
	rep, err := s.RunOnce(context.Background())
	if err != nil || rep.Launched != 0 || rep.Selected != 0 {
		t.Fatalf("empty fleet rep=%+v err=%v", rep, err)
	}
}
