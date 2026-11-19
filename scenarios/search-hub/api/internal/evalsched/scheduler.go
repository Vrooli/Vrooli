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
	"strings"
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
	DefaultCaseTimeout = 5 * time.Second
	// DefaultFederatedCaseTimeout bounds one federated case. A federated case
	// runs the ordinary automatic-classification + fan-out + optional rerank
	// path, whose warm p95 exceeds the provider-direct provider-call budget.
	// Using the provider budget here made scheduled federated runs record
	// no_result for most cases and degrade the integration signal, so the
	// federated tier keeps its own wider budget (matching eval's
	// defaultFederatedCaseTimeout).
	DefaultFederatedCaseTimeout = 30 * time.Second
	DefaultRunOverhead          = 10 * time.Second
	// DefaultRetryBackoff retries transient provider or receipt failures soon
	// enough to recover within a freshness window without hot-looping inference.
	DefaultRetryBackoff  = 15 * time.Minute
	MaxRetryBackoff      = 24 * time.Hour
	DefaultStartupDelay  = 30 * time.Second
	DefaultStartupJitter = 15 * time.Second
	// DefaultValidationCadence keeps label drift visible inside the eval
	// freshness window without probing providers on every scheduler tick.
	DefaultValidationCadence = 24 * time.Hour
	schedulerRunTag          = "scheduler"
	// Tier identifiers are shared by the tick, the run budget and the case
	// timeout selection so a tier cannot silently receive the wrong budget.
	directTier    = "provider_direct"
	federatedTier = "federated"
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

// RunHistoryStore is optional so small in-memory stores can keep their narrow
// contract. Production restores cadence from immutable scheduler receipts so
// a process restart cannot trigger every registered suite at once.
type RunHistoryStore interface {
	ListRuns(context.Context, internaleval.ListRunsFilter) ([]*evalv1.EvalRun, error)
}

// ValidationStore is optional so existing run stores and tests remain small;
// production's eval store implements it to retain the latest three-way label
// verdicts as durable evidence rather than only logging them.
type ValidationStore interface {
	AppendCorpusValidation(context.Context, string, *evalv1.ValidateCorpusResponse, time.Time) error
}

// ValidationHistoryStore restores the separate corpus-validation cadence
// from durable receipts after the scheduler process restarts.
type ValidationHistoryStore interface {
	LatestCorpusValidation(context.Context, string) (*internaleval.CorpusValidationRecord, error)
}

type Options struct {
	Cadence           time.Duration
	ValidationCadence time.Duration
	Concurrency       int
	CaseLimit         int32
	CaseTimeout       time.Duration
	// FederatedCaseTimeout is the per-case budget for the federated tier. It is
	// intentionally larger than CaseTimeout because the federated path runs the
	// ordinary routed query rather than a single provider call.
	FederatedCaseTimeout time.Duration
	RunTimeout           time.Duration
	RetryBackoff         time.Duration
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
	if o.FederatedCaseTimeout <= 0 {
		o.FederatedCaseTimeout = DefaultFederatedCaseTimeout
	}
	if o.RetryBackoff <= 0 {
		o.RetryBackoff = DefaultRetryBackoff
	} else if o.RetryBackoff > MaxRetryBackoff {
		o.RetryBackoff = MaxRetryBackoff
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
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_FEDERATED_TIMEOUT"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			o.FederatedCaseTimeout = value
		}
	}
	if raw := os.Getenv("SEARCH_HUB_EVAL_SCHEDULER_RETRY_BACKOFF"); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil && value > 0 {
			o.RetryBackoff = value
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

// caseTimeoutFor selects the per-case budget for one tier. The federated tier
// owns the ordinary routed query and gets FederatedCaseTimeout; every other
// tier keeps the provider-call CaseTimeout.
func (o Options) caseTimeoutFor(tier string) time.Duration {
	if tier == federatedTier && o.FederatedCaseTimeout > 0 {
		return o.FederatedCaseTimeout
	}
	return o.CaseTimeout
}

// runBudgetFor derives a tier's run budget from its own case timeout so a wide
// federated budget is not capped by the provider-direct run budget (or vice
// versa). An explicitly configured RunTimeout still overrides both tiers.
func (o Options) runBudgetFor(tier string) time.Duration {
	if o.RunTimeout > 0 {
		return o.RunTimeout
	}
	return o.caseTimeoutFor(tier)*time.Duration(o.CaseLimit) + DefaultRunOverhead
}

type Scheduler struct {
	clk       schedule.Clock
	suites    SuiteSource
	direct    TierRunner
	federated TierRunner
	store     RunStore
	validator CorpusValidator
	opts      Options

	mu                sync.Mutex
	lastFire          map[string]time.Time
	lastValidation    map[string]time.Time
	evalRetryAt       map[string]time.Time
	validationRetryAt map[string]time.Time
	historyLoaded     bool
	running           bool
}

func New(clk schedule.Clock, suites SuiteSource, direct, federated TierRunner, store RunStore, opts Options) *Scheduler {
	return &Scheduler{clk: clk, suites: suites, direct: direct, federated: federated, store: store, validator: opts.Validation, opts: opts.withDefaults(), lastFire: make(map[string]time.Time), lastValidation: make(map[string]time.Time), evalRetryAt: make(map[string]time.Time), validationRetryAt: make(map[string]time.Time)}
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
	if err := s.restoreSchedule(ctx, suites); err != nil {
		return fmt.Errorf("restore scheduler history: %w", err)
	}
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
				if s.validateCorpus(ctx, suite, now) {
					s.clearRetry(s.validationRetryAt, suite.GetSuiteId())
				} else {
					s.scheduleRetry(s.validationRetryAt, suite.GetSuiteId(), now)
				}
			}
			if evalDue {
				directOK := s.runTier(ctx, s.direct, suite, directTier)
				federatedOK := s.runTier(ctx, s.federated, suite, federatedTier)
				if directOK && federatedOK {
					s.clearRetry(s.evalRetryAt, suite.GetSuiteId())
				} else {
					s.scheduleRetry(s.evalRetryAt, suite.GetSuiteId(), now)
				}
			}
		}(suite, evalDue, validationDue)
	}
	wg.Wait()
	return nil
}

// restoreSchedule hydrates only once per scheduler instance. Immutable run and
// validation records are the durable clock; retries remain bounded and do
// not cause a restart to fan out a fresh batch before its prior cadence.
func (s *Scheduler) restoreSchedule(ctx context.Context, suites []*evalv1.EvalSuite) error {
	s.mu.Lock()
	if s.historyLoaded {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	runHistory, hasRunHistory := s.store.(RunHistoryStore)
	validationHistory, hasValidationHistory := s.store.(ValidationHistoryStore)
	lastFire := make(map[string]time.Time)
	lastValidation := make(map[string]time.Time)
	evalRetryAt := make(map[string]time.Time)
	validationRetryAt := make(map[string]time.Time)
	for _, suite := range suites {
		if suite == nil || strings.TrimSpace(suite.GetSuiteId()) == "" {
			continue
		}
		id := suite.GetSuiteId()
		if hasRunHistory {
			var tierTimes []time.Time
			var latestFailure time.Time
			complete := true
			for _, tier := range []string{directTier, federatedTier} {
				runs, err := runHistory.ListRuns(ctx, internaleval.ListRunsFilter{SuiteID: id, Tag: schedulerRunTag + ":" + tier, Limit: 1})
				if err != nil {
					return fmt.Errorf("read %s history for suite %q: %w", tier, id, err)
				}
				if len(runs) == 0 || runs[0] == nil {
					complete = false
					continue
				}
				createdAt, err := time.Parse(time.RFC3339Nano, runs[0].GetCreatedAt())
				if err != nil {
					return fmt.Errorf("parse %s history timestamp for suite %q: %w", tier, id, err)
				}
				tierTimes = append(tierTimes, createdAt)
				if runs[0].GetDegraded() && createdAt.After(latestFailure) {
					latestFailure = createdAt
				}
			}
			if !latestFailure.IsZero() {
				evalRetryAt[id] = latestFailure.Add(s.opts.RetryBackoff)
			} else if complete && len(tierTimes) == 2 {
				lastFire[id] = earlier(tierTimes[0], tierTimes[1])
			}
		}
		if hasValidationHistory {
			record, err := validationHistory.LatestCorpusValidation(ctx, id)
			if err != nil {
				return fmt.Errorf("read corpus validation history for suite %q: %w", id, err)
			}
			if record != nil && record.Result != nil {
				if record.CreatedAt.IsZero() {
					return fmt.Errorf("corpus validation history for suite %q has no timestamp", id)
				}
				if rollup := record.Result.GetRollup(); rollup == nil || rollup.GetProviderErrors() > 0 {
					validationRetryAt[id] = record.CreatedAt.Add(s.opts.RetryBackoff)
				} else {
					lastValidation[id] = record.CreatedAt
				}
			}
		}
	}
	s.mu.Lock()
	if !s.historyLoaded {
		for id, at := range lastFire {
			s.lastFire[id] = at
		}
		for id, at := range lastValidation {
			s.lastValidation[id] = at
		}
		for id, at := range evalRetryAt {
			s.evalRetryAt[id] = at
		}
		for id, at := range validationRetryAt {
			s.validationRetryAt[id] = at
		}
		s.historyLoaded = true
	}
	s.mu.Unlock()
	return nil
}

func earlier(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (s *Scheduler) validateCorpus(ctx context.Context, suite *evalv1.EvalSuite, observedAt time.Time) bool {
	if s.validator == nil {
		return true
	}
	validationCtx, cancel := context.WithTimeout(ctx, s.opts.CaseTimeout)
	defer cancel()
	result, err := s.validator.ValidateCorpus(validationCtx, suite, s.opts.CaseLimit)
	if err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q corpus validation failed: %v", suite.GetSuiteId(), err)
		return false
	}
	if store, ok := s.store.(ValidationStore); ok {
		if err := store.AppendCorpusValidation(ctx, suite.GetSuiteId(), result, observedAt); err != nil {
			s.opts.Logger.Printf("eval scheduler: suite %q corpus validation persist failed: %v", suite.GetSuiteId(), err)
			return false
		}
	}
	rollup := result.GetRollup()
	if rollup == nil {
		s.opts.Logger.Printf("eval scheduler: suite %q corpus validation returned no rollup", suite.GetSuiteId())
		return false
	}
	s.opts.Logger.Printf("eval scheduler: suite %q labels live=%d stale=%d provider_error=%d", suite.GetSuiteId(), rollup.GetLive(), rollup.GetStale(), rollup.GetProviderErrors())
	// The validator records per-case provider failures in a successful response
	// so the partial receipt is still valuable. Treat it as incomplete for
	// scheduling: otherwise transient provider outages become a fresh 24-hour
	// validation interval and the evidence remains degraded until the next day.
	if rollup.GetProviderErrors() > 0 {
		return false
	}
	return true
}

func (s *Scheduler) due(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if retryAt, ok := s.evalRetryAt[id]; ok {
		return !now.Before(retryAt)
	}
	last, ok := s.lastFire[id]
	return !ok || now.Sub(last) >= s.opts.Cadence
}

func (s *Scheduler) validationDue(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if retryAt, ok := s.validationRetryAt[id]; ok {
		return !now.Before(retryAt)
	}
	last, ok := s.lastValidation[id]
	return !ok || now.Sub(last) >= s.opts.ValidationCadence
}

func (s *Scheduler) runTier(ctx context.Context, runner TierRunner, suite *evalv1.EvalSuite, tier string) bool {
	if runner == nil {
		err := fmt.Errorf("%s evaluation runner is unavailable", tier)
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s failed: %v", suite.GetSuiteId(), tier, err)
		s.persistDegraded(ctx, suite, tier, err)
		return false
	}
	caseTimeout := s.opts.caseTimeoutFor(tier)
	runBudget := s.opts.runBudgetFor(tier)
	tierCtx, cancel := context.WithTimeout(ctx, runBudget)
	defer cancel()
	run, err := runner.Run(internaleval.WithCaseTimeout(tierCtx, caseTimeout), suite, schedulerRunTag+":"+tier, s.opts.CaseLimit)
	if err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s failed: %v", suite.GetSuiteId(), tier, err)
		s.persistDegraded(ctx, suite, tier, err)
		return false
	}
	if run == nil {
		err := fmt.Errorf("runner returned no run")
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s returned no run", suite.GetSuiteId(), tier)
		s.persistDegraded(ctx, suite, tier, err)
		return false
	}
	if err := s.store.AppendRun(ctx, run); err != nil {
		s.opts.Logger.Printf("eval scheduler: suite %q tier %s persist failed: %v", suite.GetSuiteId(), tier, err)
		return false
	}
	return true
}

func (s *Scheduler) scheduleRetry(retries map[string]time.Time, suiteID string, now time.Time) {
	s.mu.Lock()
	retries[suiteID] = now.Add(s.opts.RetryBackoff)
	s.mu.Unlock()
}

func (s *Scheduler) clearRetry(retries map[string]time.Time, suiteID string) {
	s.mu.Lock()
	delete(retries, suiteID)
	s.mu.Unlock()
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
