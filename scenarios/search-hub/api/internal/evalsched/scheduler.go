// Package evalsched runs registered golden suites independently of operator
// commands. It owns scheduling policy only; suite storage and both eval tiers
// remain injected seams, so a new registrant needs no scheduler change.
package evalsched

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	internaleval "search-hub/internal/eval"

	"github.com/vrooli/api-core/schedule"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

const (
	// DefaultCadence is comfortably inside the 30-day eval freshness window.
	DefaultCadence     = 7 * 24 * time.Hour
	DefaultConcurrency = 2
	DefaultCaseLimit   = 10
	// DefaultCaseTimeout bounds one provider call. It is intentionally separate
	// from the run budget: a ten-case suite must not be canceled after its first
	// slow-but-valid query.
	DefaultCaseTimeout   = 5 * time.Second
	DefaultRunOverhead   = 10 * time.Second
	DefaultStartupDelay  = 30 * time.Second
	DefaultStartupJitter = 15 * time.Second
	// DefaultValidationCadence keeps label drift visible inside the eval
	// freshness window without probing providers on every scheduler tick.
	DefaultValidationCadence = 24 * time.Hour
	schedulerRunTag          = "scheduler"
)

type SuiteSource interface {
	ListSuites(context.Context, internaleval.ListSuitesFilter) ([]*evalv1.EvalSuite, error)
}

type TierRunner interface {
	Run(context.Context, *evalv1.EvalSuite, string, int32) (*evalv1.EvalRun, error)
}

type CorpusValidator interface {
	ValidateCorpus(context.Context, *evalv1.EvalSuite, int32) (*evalv1.ValidateCorpusResponse, error)
}

type RunStore interface {
	AppendRun(context.Context, *evalv1.EvalRun) error
}

// ValidationStore is optional so existing run stores and tests remain small;
// production's eval store implements it to retain the latest three-way label
// verdicts as durable evidence rather than only logging them.
type ValidationStore interface {
	AppendCorpusValidation(context.Context, string, *evalv1.ValidateCorpusResponse, time.Time) error
}

type Options struct {
	Cadence           time.Duration
	ValidationCadence time.Duration
	Concurrency       int
	CaseLimit         int32
	CaseTimeout       time.Duration
	RunTimeout        time.Duration
	// TierTimeout is retained as a source-compatible alias for operators that
	// already set SEARCH_HUB_EVAL_SCHEDULER_TIMEOUT; it now means per-case.
	TierTimeout   time.Duration
	StartupDelay  time.Duration
	StartupJitter time.Duration
	Sleep         func(context.Context, time.Duration) error
	Validation    CorpusValidator
	Logger        *log.Logger
}

func (o Options) withDefaults() Options {
	if o.Cadence <= 0 {
		o.Cadence = DefaultCadence
	}
	if o.ValidationCadence <= 0 {
		o.ValidationCadence = DefaultValidationCadence
	}
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	if o.CaseLimit <= 0 {
		o.CaseLimit = DefaultCaseLimit
	}
	if o.CaseTimeout <= 0 {
		o.CaseTimeout = o.TierTimeout
	}
	if o.CaseTimeout <= 0 {
		o.CaseTimeout = DefaultCaseTimeout
	}
	if o.RunTimeout <= 0 {
		o.RunTimeout = o.CaseTimeout*time.Duration(o.CaseLimit) + DefaultRunOverhead
	}
	if o.StartupDelay < 0 {
		o.StartupDelay = 0
	}
	if o.StartupDelay == 0 {
		o.StartupDelay = DefaultStartupDelay
	}
	if o.StartupJitter < 0 {
		o.StartupJitter = 0
	}
	if o.StartupJitter == 0 {
		o.StartupJitter = DefaultStartupJitter
	}
	if o.Sleep == nil {
		o.Sleep = func(ctx context.Context, d time.Duration) error {
			if d <= 0 {
				return nil
			}
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	if o.Logger == nil {
		o.Logger = log.Default()
	}
	return o
}

// OptionsFromEnv keeps operator tuning explicit and bounded. Invalid values
// fall back to safe defaults rather than disabling unattended evaluation.
func OptionsFromEnv(logger *log.Logger) Options {
	o := Options{Logger: logger}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_INTERVAL"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			o.Cadence = value
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_VALIDATION_INTERVAL"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			o.ValidationCadence = value
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_CONCURRENCY"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			o.Concurrency = value
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_CASE_LIMIT"); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 32); err == nil && value >= 0 {
			o.CaseLimit = int32(value)
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_TIMEOUT"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			o.CaseTimeout = value
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_STARTUP_DELAY"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			o.StartupDelay = value
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_STARTUP_JITTER"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			o.StartupJitter = value
		}
	}
	return o.withDefaults()
}

type Scheduler struct {
	clk       schedule.Clock
	suites    SuiteSource
	direct    TierRunner
	federated TierRunner
	store     RunStore
	validator CorpusValidator
	opts      Options

	mu             sync.Mutex
	lastFire       map[string]time.Time
	lastValidation map[string]time.Time
	running        bool
}

func New(clk schedule.Clock, suites SuiteSource, direct, federated TierRunner, store RunStore, opts Options) *Scheduler {
	return &Scheduler{clk: clk, suites: suites, direct: direct, federated: federated, store: store, validator: opts.Validation, opts: opts.withDefaults(), lastFire: make(map[string]time.Time), lastValidation: make(map[string]time.Time)}
}

// Tick performs one non-overlapping best-effort cycle. A provider or suite
// failure is logged and does not prevent other registered suites from running.
func (s *Scheduler) Tick(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	suites, err := s.suites.ListSuites(ctx, internaleval.ListSuitesFilter{})
	if err != nil {
		return fmt.Errorf("list registered eval suites: %w", err)
	}
	now := s.clk.Now()
	sem := make(chan struct{}, s.opts.Concurrency)
	var wg sync.WaitGroup
	for _, suite := range suites {
		if suite == nil {
			continue
		}
		evalDue := s.due(suite.GetSuiteId(), now)
		validationDue := s.validationDue(suite.GetSuiteId(), now)
		if !evalDue && !validationDue {
			continue
		}
		if evalDue {
			s.mu.Lock()
			s.lastFire[suite.GetSuiteId()] = now
			s.mu.Unlock()
		}
		if validationDue {
			s.mu.Lock()
			s.lastValidation[suite.GetSuiteId()] = now
			s.mu.Unlock()
		}
		wg.Add(1)
		go func(suite *evalv1.EvalSuite, evalDue, validationDue bool) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if validationDue {
				s.validateCorpus(ctx, suite, now)
			}
			if evalDue {
				s.runTier(ctx, s.direct, suite, "provider_direct")
				s.runTier(ctx, s.federated, suite, "federated")
			}
		}(suite, evalDue, validationDue)
	}
	wg.Wait()
	return nil
}

func (s *Scheduler) validateCorpus(ctx context.Context, suite *evalv1.EvalSuite, observedAt time.Time) {
	if s.validator == nil {
		return
	}
	validationCtx, cancel := context.WithTimeout(ctx, s.opts.CaseTimeout)
	defer cancel()
	result, err := s.validator.ValidateCorpus(validationCtx, suite, s.opts.CaseLimit)
	if err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q corpus validation failed: %v", suite.GetSuiteId(), err)
		return
	}
	if store, ok := s.store.(ValidationStore); ok {
		if err := store.AppendCorpusValidation(ctx, suite.GetSuiteId(), result, observedAt); err != nil {
			s.opts.Logger.Printf("eval scheduler: suite %q corpus validation persist failed: %v", suite.GetSuiteId(), err)
		}
	}
	rollup := result.GetRollup()
	if rollup == nil {
		s.opts.Logger.Printf("eval scheduler: suite %q corpus validation returned no rollup", suite.GetSuiteId())
		return
	}
	s.opts.Logger.Printf("eval scheduler: suite %q labels live=%d stale=%d provider_error=%d", suite.GetSuiteId(), rollup.GetLive(), rollup.GetStale(), rollup.GetProviderErrors())
}

func (s *Scheduler) due(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.lastFire[id]
	return !ok || now.Sub(last) >= s.opts.Cadence
}

func (s *Scheduler) validationDue(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.lastValidation[id]
	return !ok || now.Sub(last) >= s.opts.ValidationCadence
}

func (s *Scheduler) runTier(ctx context.Context, runner TierRunner, suite *evalv1.EvalSuite, tier string) {
	if runner == nil {
		return
	}
	runBudget := s.opts.RunTimeout
	if runBudget <= 0 {
		runBudget = s.opts.CaseTimeout*time.Duration(s.opts.CaseLimit) + DefaultRunOverhead
	}
	tierCtx, cancel := context.WithTimeout(ctx, runBudget)
	defer cancel()
	run, err := runner.Run(internaleval.WithCaseTimeout(tierCtx, s.opts.CaseTimeout), suite, schedulerRunTag+":"+tier, s.opts.CaseLimit)
	if err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s failed: %v", suite.GetSuiteId(), tier, err)
		s.persistDegraded(ctx, suite, tier, err)
		return
	}
	if run == nil {
		err := fmt.Errorf("runner returned no run")
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s returned no run", suite.GetSuiteId(), tier)
		s.persistDegraded(ctx, suite, tier, err)
		return
	}
	if err := s.store.AppendRun(ctx, run); err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s persist failed: %v", suite.GetSuiteId(), tier, err)
	}
}

func (s *Scheduler) persistDegraded(ctx context.Context, suite *evalv1.EvalSuite, tier string, cause error) {
	run := &evalv1.EvalRun{
		RunId:          uuid.NewString(),
		SuiteId:        suite.GetSuiteId(),
		Tag:            schedulerRunTag + ":" + tier,
		Tier:           tier,
		Degraded:       true,
		DegradedReason: cause.Error(),
	}
	if err := s.store.AppendRun(ctx, run); err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s degraded run persist failed: %v", suite.GetSuiteId(), tier, err)
	}
}

// Run owns the production ticker. Tests should call Tick directly with a fake
// clock so time and cycle overlap are deterministic.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.opts.Cadence)
	defer ticker.Stop()
	delay := s.opts.StartupDelay
	if s.opts.StartupJitter > 0 {
		delay += time.Duration(s.clk.Now().UnixNano() % int64(s.opts.StartupJitter))
	}
	if err := s.opts.Sleep(ctx, delay); err != nil {
		return
	}
	_ = s.Tick(ctx) // first cycle is delayed to let dependencies settle
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Tick(ctx); err != nil {
				s.opts.Logger.Printf("eval scheduler cycle failed: %v", err)
			}
		}
	}
}
