// Package fleetscheduler is Test Genie's default-OFF, priority-weighted
// background scheduler. It cycles full test suites across the fleet so that
// fleet-wide health stays reasonably fresh WITHOUT anyone paying an hours-long
// synchronous fleet run: each tick selects the highest-priority, stalest
// scenarios and launches their suites through the durable run manager, which
// owns the one-in-progress-per-scenario invariant.
//
// The scheduler is the write side of the fleet backbone; the fleet ledger
// (internal/selfhealth fleet scope) is the read side that aggregates the runs
// this scheduler (and ordinary on-demand runs) leave behind. It is OFF unless
// explicitly enabled (mirroring the score/self-health sweepers and EM's
// DefaultImportanceAwareScheduling=false), and every cycle is bounded by an
// explicit concurrency cap, a per-cycle run-count budget, and a wall-clock
// budget so an enabled scheduler can never saturate the host.
package fleetscheduler

import (
	"context"
	"errors"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrScenarioBusy is returned by a Launcher when the scenario already has an
// in-progress run (the run manager's one-per-scenario invariant). The scheduler
// treats it as a clean skip for the cycle — never an error.
var ErrScenarioBusy = errors.New("scenario already has an in-progress run")

// ErrCycleInProgress guards against overlapping cycles (a slow cycle must not
// be re-entered by the next tick).
var ErrCycleInProgress = errors.New("fleet scheduler cycle already in progress")

// Candidate is one fleet scenario the scheduler may re-test, carrying the
// priority signals it ranks by. Priority/Importance come from
// scenario-completeness-scoring; LastRunAt/LastStatus are the scenario-level
// test-recency keystone (SCS ScoreRow.last_run_at/last_status).
type Candidate struct {
	Scenario   string
	Importance float64
	Priority   float64
	LastRunAt  time.Time // zero = never tested / unknown
	LastStatus string
}

// PrioritySource enumerates fleet candidates. Production shells
// `scenario-completeness-scoring score list --json` (see priority.go); tests
// wire deterministic fakes.
type PrioritySource interface {
	Candidates(ctx context.Context) ([]Candidate, error)
}

// Launcher starts a suite run for one scenario and waits for it to terminate.
// Production wraps runmanager.Manager (Start + Wait, see launcher.go); the
// per-scenario one-in-progress invariant lives there. Launch must return
// ErrScenarioBusy (not a generic error) when the scenario is already running so
// the scheduler can skip it cleanly.
type Launcher interface {
	Launch(ctx context.Context, scenario string) (runID string, err error)
	Await(ctx context.Context, scenario, runID string) (status string, err error)
}

// Config controls the scheduler. Source and Launcher are required.
type Config struct {
	Source   PrioritySource
	Launcher Launcher
	Logger   *log.Logger
	Now      func() time.Time

	// Interval is the tick cadence; InitialJitter delays the first cycle so a
	// fleet of schedulers doesn't thunder at boot. Interval defaults to 6h.
	Interval      time.Duration
	InitialJitter time.Duration

	// MaxConcurrent bounds simultaneously-launched runs (the global compute
	// cap). Defaults to 1 — deliberately conservative, since each run is a full
	// suite. It NEVER weakens the per-scenario invariant; it caps how many
	// DISTINCT scenarios run at once.
	MaxConcurrent int

	// MaxRunsPerCycle bounds how many scenarios one cycle launches (the count
	// budget). Defaults to 5. The fleet is covered across many cycles, not one.
	MaxRunsPerCycle int

	// CycleBudget bounds one cycle's wall-clock: once exceeded, the cycle stops
	// admitting new runs (in-flight runs still finish). Zero = no time cap.
	CycleBudget time.Duration

	// StalenessHorizon scales the priority weight by test age: a scenario
	// untested for the whole horizon gets up to ~2x its priority, and a
	// never-tested scenario gets the strongest pull. Zero = rank by priority
	// alone (recency ignored).
	StalenessHorizon time.Duration
}

// CycleReport summarizes one scheduler cycle.
type CycleReport struct {
	Candidates int
	Selected   int
	Launched   int
	Passed     int
	Failed     int
	Skipped    int // busy scenarios skipped this cycle
	Errored    int
	Duration   time.Duration
	// BudgetHit is true when the cycle stopped admitting runs because the
	// wall-clock budget elapsed (some selected scenarios were not launched).
	BudgetHit bool
}

// Scheduler launches priority-ordered fleet runs in the background.
type Scheduler struct {
	cfg     Config
	running sync.Mutex // serializes cycles; held for the duration of RunOnce
	active  bool
}

// New validates and constructs a Scheduler.
func New(cfg Config) (*Scheduler, error) {
	if cfg.Source == nil {
		return nil, errors.New("priority source is required")
	}
	if cfg.Launcher == nil {
		return nil, errors.New("launcher is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 6 * time.Hour
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if cfg.MaxRunsPerCycle <= 0 {
		cfg.MaxRunsPerCycle = 5
	}
	return &Scheduler{cfg: cfg}, nil
}

// RunOnce executes a single selection-and-launch cycle. It is single-flight:
// a concurrent call returns ErrCycleInProgress rather than overlapping.
func (s *Scheduler) RunOnce(ctx context.Context) (CycleReport, error) {
	if !s.tryEnter() {
		return CycleReport{}, ErrCycleInProgress
	}
	defer s.leave()

	start := s.cfg.Now()
	report := CycleReport{}

	candidates, err := s.cfg.Source.Candidates(ctx)
	if err != nil {
		return report, err
	}
	report.Candidates = len(candidates)

	selected := s.selectScenarios(candidates)
	report.Selected = len(selected)
	if len(selected) == 0 {
		report.Duration = s.cfg.Now().Sub(start)
		return report, nil
	}

	deadline := time.Time{}
	if s.cfg.CycleBudget > 0 {
		deadline = start.Add(s.cfg.CycleBudget)
	}

	jobs := make(chan Candidate)
	var mu sync.Mutex
	var wg sync.WaitGroup
	workers := s.cfg.MaxConcurrent
	if workers > len(selected) {
		workers = len(selected)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				outcome := s.launchOne(ctx, c)
				mu.Lock()
				report.applyOutcome(outcome)
				mu.Unlock()
			}
		}()
	}

dispatch:
	for _, c := range selected {
		if ctx.Err() != nil {
			break
		}
		if !deadline.IsZero() && !s.cfg.Now().Before(deadline) {
			report.BudgetHit = true
			break dispatch
		}
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- c:
		}
	}
	close(jobs)
	wg.Wait()

	report.Duration = s.cfg.Now().Sub(start)
	return report, nil
}

// launchOutcome is one scenario's result within a cycle.
type launchOutcome int

const (
	outcomePassed launchOutcome = iota
	outcomeFailed
	outcomeBusy
	outcomeErrored
)

func (r *CycleReport) applyOutcome(o launchOutcome) {
	switch o {
	case outcomePassed:
		r.Launched++
		r.Passed++
	case outcomeFailed:
		r.Launched++
		r.Failed++
	case outcomeBusy:
		r.Skipped++
	case outcomeErrored:
		r.Errored++
	}
}

func (s *Scheduler) launchOne(ctx context.Context, c Candidate) launchOutcome {
	runID, err := s.cfg.Launcher.Launch(ctx, c.Scenario)
	if err != nil {
		if errors.Is(err, ErrScenarioBusy) {
			return outcomeBusy
		}
		s.cfg.Logger.Printf("fleet scheduler: launch %s failed: %v", c.Scenario, err)
		return outcomeErrored
	}
	status, err := s.cfg.Launcher.Await(ctx, c.Scenario, runID)
	if err != nil {
		s.cfg.Logger.Printf("fleet scheduler: await %s/%s failed: %v", c.Scenario, runID, err)
		return outcomeErrored
	}
	if strings.EqualFold(strings.TrimSpace(status), "passed") {
		return outcomePassed
	}
	return outcomeFailed
}

// selectScenarios ranks candidates by staleness-weighted priority and returns
// the top MaxRunsPerCycle. Deterministic: ties break by scenario name.
func (s *Scheduler) selectScenarios(candidates []Candidate) []Candidate {
	now := s.cfg.Now()
	ranked := append([]Candidate(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		wi, wj := s.weight(ranked[i], now), s.weight(ranked[j], now)
		if wi != wj {
			return wi > wj
		}
		return ranked[i].Scenario < ranked[j].Scenario
	})
	if len(ranked) > s.cfg.MaxRunsPerCycle {
		ranked = ranked[:s.cfg.MaxRunsPerCycle]
	}
	return ranked
}

// weight is the selection score: base priority scaled by test staleness. A
// never-tested scenario gets the strongest pull (so the fleet gets a first
// signal for every scenario); otherwise older = heavier, capped at ~2x.
func (s *Scheduler) weight(c Candidate, now time.Time) float64 {
	base := c.Priority
	if base <= 0 {
		base = c.Importance
	}
	if base <= 0 {
		// No priority signal at all: keep it selectable but lowest, so a
		// never-tested no-signal scenario still ranks above a fresh one.
		base = 0.0001
	}
	stale := 1.0
	switch {
	case c.LastRunAt.IsZero():
		stale = 2.5 // never tested: strongest pull
	case s.cfg.StalenessHorizon > 0:
		age := now.Sub(c.LastRunAt)
		if age < 0 {
			age = 0
		}
		stale = 1.0 + math.Min(age.Seconds()/s.cfg.StalenessHorizon.Seconds(), 1.0)
	}
	return base * stale
}

func (s *Scheduler) tryEnter() bool {
	s.running.Lock()
	defer s.running.Unlock()
	if s.active {
		return false
	}
	s.active = true
	return true
}

func (s *Scheduler) leave() {
	s.running.Lock()
	defer s.running.Unlock()
	s.active = false
}

// RunLoop ticks RunOnce at the configured interval until ctx is cancelled,
// after an optional initial jitter. It mirrors the score/self-health sweepers:
// a missed/failed cycle is logged and the loop continues.
func (s *Scheduler) RunLoop(ctx context.Context) {
	if s.cfg.InitialJitter > 0 {
		timer := time.NewTimer(s.cfg.InitialJitter)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}

	run := func() {
		report, err := s.RunOnce(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrCycleInProgress) {
				return
			}
			s.cfg.Logger.Printf("fleet scheduler cycle failed: %v", err)
			return
		}
		s.cfg.Logger.Printf("fleet scheduler cycle: candidates=%d selected=%d launched=%d passed=%d failed=%d skipped=%d errored=%d budget_hit=%t duration=%s",
			report.Candidates, report.Selected, report.Launched, report.Passed, report.Failed, report.Skipped, report.Errored, report.BudgetHit, report.Duration)
	}

	run()
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
