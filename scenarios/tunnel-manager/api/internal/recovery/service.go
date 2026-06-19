package recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
)

// HealthChecker is the readiness seam the engine consults before and
// after a restart. Declared at the consumer (recovery) so it never
// imports the tunnel domain; main.go adapts the tunnel service to it.
// Ready reports whether the cloudflared /ready probe currently passes.
type HealthChecker interface {
	Ready(ctx context.Context) bool
}

// Service is the recovery engine surface the handlers depend on.
type Service interface {
	// GetState returns the live state-machine snapshot.
	GetState(ctx context.Context) (RecoveryState, error)

	// ListEvents returns the recovery event log, newest first.
	ListEvents(ctx context.Context, limit int) ([]RecoveryEvent, error)

	// Recover triggers a manual recovery attempt. Idempotent while a
	// recovery is already in flight (returns SKIPPED rather than
	// restarting twice); rejected while the circuit breaker is open
	// unless force is set.
	Recover(ctx context.Context, force bool) (EventOutcome, RecoveryEvent, error)

	// Evaluate runs one health-driven evaluation: on a failed /ready it
	// counts a consecutive failure and, once the threshold is crossed,
	// attempts recovery. Returns the event when an attempt occurred, or
	// a zero event when no action was needed. Wired to a background
	// timer in live operation (Stage 5); exported and tested now.
	Evaluate(ctx context.Context) (RecoveryEvent, bool, error)
}

// Config tunes the engine. Zero values fall back to defaults in
// NewService.
type Config struct {
	ConsecutiveFailures int
	MaxBackoffRetries   int
	CircuitCooldown     time.Duration
	BackoffSchedule     []time.Duration
	ReadyPollAttempts   int
	ReadyPollInterval   time.Duration
}

type engine struct {
	repo    Repository
	health  HealthChecker
	runner  cmdrunner.Runner
	clock   clock.Clock
	sleep   func(time.Duration)
	cfg     Config
	circuit time.Time // wall time the circuit opened

	mu             sync.Mutex
	state          RecoveryState
	recovering     bool
	attemptCounter int
}

// NewService constructs the production engine. sleep is injectable so
// tests drive the ready-poll loop without real delays; pass nil for
// time.Sleep.
func NewService(repo Repository, health HealthChecker, runner cmdrunner.Runner, clk clock.Clock, cfg Config, sleep func(time.Duration)) Service {
	if cfg.ConsecutiveFailures <= 0 {
		cfg.ConsecutiveFailures = 3
	}
	if cfg.MaxBackoffRetries <= 0 {
		cfg.MaxBackoffRetries = 5
	}
	if cfg.CircuitCooldown <= 0 {
		cfg.CircuitCooldown = time.Hour
	}
	if len(cfg.BackoffSchedule) == 0 {
		cfg.BackoffSchedule = []time.Duration{
			30 * time.Second, time.Minute, 2 * time.Minute,
			4 * time.Minute, 8 * time.Minute, 15 * time.Minute,
		}
	}
	if cfg.ReadyPollAttempts <= 0 {
		cfg.ReadyPollAttempts = 30
	}
	if cfg.ReadyPollInterval <= 0 {
		cfg.ReadyPollInterval = time.Second
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	return &engine{
		repo:   repo,
		health: health,
		runner: runner,
		clock:  clk,
		sleep:  sleep,
		cfg:    cfg,
		state:  RecoveryState{Status: StatusIdle},
	}
}

var _ Service = (*engine)(nil)

func (e *engine) GetState(_ context.Context) (RecoveryState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state, nil
}

func (e *engine) ListEvents(ctx context.Context, limit int) ([]RecoveryEvent, error) {
	if limit < 0 {
		return nil, ErrInvalidRecovery{Field: "limit", Reason: "must not be negative"}
	}
	return e.repo.ListEvents(ctx, limit)
}

func (e *engine) Recover(ctx context.Context, force bool) (EventOutcome, RecoveryEvent, error) {
	e.mu.Lock()

	// Idempotency: a recovery already in flight is not re-triggered.
	// The second caller observes a SKIPPED outcome — twice == once.
	if e.recovering {
		e.mu.Unlock()
		evt, err := e.persistSkipped(ctx, TriggerManual, "recovery already in flight")
		return OutcomeSkipped, evt, err
	}

	if e.circuitGuarded(force) {
		e.mu.Unlock()
		evt, err := e.persistSkipped(ctx, TriggerManual, "circuit breaker is open; use --force to override")
		return OutcomeSkipped, evt, err
	}

	e.recovering = true
	e.state.Status = StatusRecovering
	e.state.LastRecovery = e.clock.Now().UTC()
	attempt := e.nextAttemptLocked()
	e.mu.Unlock()

	evt := e.executeRecovery(ctx, TriggerManual, attempt)

	e.mu.Lock()
	e.recovering = false
	e.applyOutcomeLocked(evt)
	e.mu.Unlock()

	stored, err := e.repo.PersistEvent(ctx, evt)
	if err != nil {
		return evt.Outcome, evt, err
	}
	return stored.Outcome, stored, nil
}

func (e *engine) Evaluate(ctx context.Context) (RecoveryEvent, bool, error) {
	e.mu.Lock()
	e.state.LastCheck = e.clock.Now().UTC()

	if e.recovering || e.blockedLocked() {
		e.mu.Unlock()
		return RecoveryEvent{}, false, nil
	}

	if e.health.Ready(ctx) {
		e.state.ConsecFailures = 0
		e.state.Status = StatusIdle
		e.mu.Unlock()
		return RecoveryEvent{}, false, nil
	}

	e.state.ConsecFailures++
	if e.state.ConsecFailures < e.cfg.ConsecutiveFailures {
		e.state.Status = StatusMonitoring
		e.mu.Unlock()
		return RecoveryEvent{}, false, nil
	}

	e.recovering = true
	e.state.Status = StatusRecovering
	e.state.LastRecovery = e.clock.Now().UTC()
	attempt := e.nextAttemptLocked()
	failures := e.state.ConsecFailures
	e.mu.Unlock()

	evt := e.executeRecovery(ctx, TriggerReadyFailure, attempt)
	evt.Details = fmt.Sprintf("%s (after %d consecutive ready failures)", evt.Details, failures)

	e.mu.Lock()
	e.recovering = false
	e.applyOutcomeLocked(evt)
	e.mu.Unlock()

	stored, err := e.repo.PersistEvent(ctx, evt)
	if err != nil {
		return evt, true, err
	}
	return stored, true, nil
}

// circuitGuarded reports whether the circuit breaker blocks this attempt.
// Resets the breaker after cooldown; force bypasses (and resets) it.
// Must hold e.mu.
func (e *engine) circuitGuarded(force bool) bool {
	if !e.state.CircuitOpen {
		return false
	}
	if force {
		e.resetCountersLocked()
		return false
	}
	if e.clock.Now().Sub(e.circuit) >= e.cfg.CircuitCooldown {
		e.resetCountersLocked()
		return false
	}
	e.state.Status = StatusCircuitOpen
	return true
}

// blockedLocked reports whether automatic evaluation is paused by the
// circuit breaker or an unexpired backoff window. Must hold e.mu.
func (e *engine) blockedLocked() bool {
	if e.state.CircuitOpen {
		if e.clock.Now().Sub(e.circuit) >= e.cfg.CircuitCooldown {
			e.resetCountersLocked()
		} else {
			e.state.Status = StatusCircuitOpen
			return true
		}
	}
	if !e.state.NextRetryAfter.IsZero() && e.clock.Now().Before(e.state.NextRetryAfter) {
		e.state.Status = StatusMonitoring
		return true
	}
	return false
}

func (e *engine) nextAttemptLocked() int {
	e.attemptCounter++
	return e.attemptCounter
}

// executeRecovery restarts cloudflared and polls /ready. Runs WITHOUT
// holding e.mu so concurrent state reads stay responsive.
func (e *engine) executeRecovery(ctx context.Context, trigger string, attempt int) RecoveryEvent {
	base := RecoveryEvent{Trigger: trigger, Action: ActionRestart, Attempt: attempt}

	if _, err := e.runner(ctx, "sudo", "systemctl", "restart", "cloudflared"); err != nil {
		base.Outcome = OutcomeFailure
		base.Details = fmt.Sprintf("restart failed: %v", err)
		return base
	}

	for i := 0; i < e.cfg.ReadyPollAttempts; i++ {
		if e.health.Ready(ctx) {
			base.Outcome = OutcomeSuccess
			base.Details = "tunnel recovered after restart"
			return base
		}
		e.sleep(e.cfg.ReadyPollInterval)
	}
	base.Outcome = OutcomeFailure
	base.Details = "tunnel did not become ready after restart"
	return base
}

// applyOutcomeLocked updates engine state from a finished attempt. Must
// hold e.mu.
func (e *engine) applyOutcomeLocked(evt RecoveryEvent) {
	if evt.Outcome == OutcomeSuccess {
		e.resetCountersLocked()
		e.state.Status = StatusIdle
		return
	}
	e.state.FailedRecovery++
	if e.state.FailedRecovery >= e.cfg.MaxBackoffRetries {
		e.state.CircuitOpen = true
		e.circuit = e.clock.Now()
		e.state.Status = StatusCircuitOpen
		return
	}
	idx := e.state.BackoffLevel
	if idx > len(e.cfg.BackoffSchedule)-1 {
		idx = len(e.cfg.BackoffSchedule) - 1
	}
	e.state.NextRetryAfter = e.clock.Now().Add(e.cfg.BackoffSchedule[idx])
	e.state.BackoffLevel++
	e.state.Status = StatusMonitoring
}

func (e *engine) resetCountersLocked() {
	e.state.CircuitOpen = false
	e.state.BackoffLevel = 0
	e.state.FailedRecovery = 0
	e.state.ConsecFailures = 0
	e.state.NextRetryAfter = time.Time{}
}

func (e *engine) persistSkipped(ctx context.Context, trigger, details string) (RecoveryEvent, error) {
	e.mu.Lock()
	attempt := e.attemptCounter
	e.mu.Unlock()
	return e.repo.PersistEvent(ctx, RecoveryEvent{
		Trigger: trigger,
		Action:  ActionRestart,
		Outcome: OutcomeSkipped,
		Details: details,
		Attempt: attempt,
	})
}
