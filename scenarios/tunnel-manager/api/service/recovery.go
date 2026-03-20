package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"tunnel-manager/adapter"
	"tunnel-manager/domain"
)

// RecoveryEngine monitors tunnel health and automatically restarts cloudflared
// when failures are detected. It implements exponential backoff and circuit breaker patterns.
type RecoveryEngine struct {
	recoveryStore RecoveryEventStore
	healthCheck   TunnelChecker
	cmdRunner     adapter.CmdRunner
	mu            sync.Mutex
	state         domain.RecoveryState
	circuitAt     time.Time
	consecHAZero  int // consecutive metric scrapes showing HA connections = 0

	// Configurable thresholds
	ConsecutiveFailures int           // failures before triggering recovery (default 3)
	MaxBackoffRetries   int           // consecutive failures before circuit opens (default 5)
	CircuitCooldown     time.Duration // how long circuit stays open (default 1h)
	BackoffSchedule     []time.Duration
	ReadyPollTimeout    time.Duration // how long to poll /ready after restart (default 30s)
	ReadyPollInterval   time.Duration // poll interval (default 1s)
}

type RecoveryOption func(*RecoveryEngine)

func WithRecoveryCmdRunner(fn adapter.CmdRunner) RecoveryOption {
	return func(re *RecoveryEngine) { re.cmdRunner = fn }
}

func WithConsecutiveFailures(n int) RecoveryOption {
	return func(re *RecoveryEngine) { re.ConsecutiveFailures = n }
}

func WithMaxBackoffRetries(n int) RecoveryOption {
	return func(re *RecoveryEngine) { re.MaxBackoffRetries = n }
}

func WithCircuitCooldown(d time.Duration) RecoveryOption {
	return func(re *RecoveryEngine) { re.CircuitCooldown = d }
}

func NewRecoveryEngine(recoveryStore RecoveryEventStore, healthCheck TunnelChecker, opts ...RecoveryOption) *RecoveryEngine {
	re := &RecoveryEngine{
		recoveryStore:       recoveryStore,
		healthCheck:         healthCheck,
		cmdRunner:           adapter.DefaultCmdRunner,
		ConsecutiveFailures: 3,
		MaxBackoffRetries:   5,
		CircuitCooldown:     1 * time.Hour,
		BackoffSchedule: []time.Duration{
			30 * time.Second,
			1 * time.Minute,
			2 * time.Minute,
			4 * time.Minute,
			8 * time.Minute,
			15 * time.Minute,
		},
		ReadyPollTimeout:  30 * time.Second,
		ReadyPollInterval: 1 * time.Second,
		state:             domain.RecoveryState{Status: "idle"},
	}
	for _, opt := range opts {
		opt(re)
	}
	return re
}

// resetCounters clears circuit breaker, backoff, and failure counters.
// Must be called with re.mu held.
func (re *RecoveryEngine) resetCounters() {
	re.state.CircuitOpen = false
	re.state.BackoffLevel = 0
	re.state.FailedRecovery = 0
	re.state.ConsecFailures = 0
	re.state.NextRetryAfter = time.Time{}
}

// checkGuards returns true if recovery is currently blocked by circuit breaker
// or backoff. When blocked, it updates state.Status accordingly.
// Must be called with re.mu held.
func (re *RecoveryEngine) checkGuards() (blocked bool) {
	if re.state.CircuitOpen {
		if time.Since(re.circuitAt) >= re.CircuitCooldown {
			re.resetCounters()
			slog.Info("recovery: circuit breaker reset after cooldown", "component", "recovery")
		} else {
			re.state.Status = "circuit_open"
			return true
		}
	}
	if !re.state.NextRetryAfter.IsZero() && time.Now().Before(re.state.NextRetryAfter) {
		re.state.Status = "monitoring"
		return true
	}
	return false
}

// Evaluate checks tunnel health and decides whether to trigger recovery.
// Returns the recovery event if one occurred, or nil if no action was needed.
func (re *RecoveryEngine) Evaluate(ctx context.Context) (*domain.RecoveryEvent, error) {
	re.mu.Lock()
	defer re.mu.Unlock()

	re.state.LastCheck = time.Now()

	if re.checkGuards() {
		return nil, nil
	}

	// Run health check
	status := re.healthCheck.Check(ctx)

	// Check for ready endpoint failure
	if status.Ready != "ok" {
		re.state.ConsecFailures++
		if re.state.ConsecFailures >= re.ConsecutiveFailures {
			return re.attemptRecovery(ctx, "ready_failure",
				fmt.Sprintf("ready endpoint returned %q for %d consecutive checks", status.Ready, re.state.ConsecFailures))
		}
		re.state.Status = "monitoring"
		return nil, nil
	}

	// Healthy — reset failure counter
	re.state.ConsecFailures = 0
	re.state.Status = "idle"
	return nil, nil
}

// TriggerManual forces an immediate recovery attempt, bypassing backoff
// but respecting the circuit breaker unless force is true.
func (re *RecoveryEngine) TriggerManual(ctx context.Context, force bool) (*domain.RecoveryEvent, error) {
	re.mu.Lock()

	if re.state.CircuitOpen && !force {
		evt := newEvent("manual", "skipped", "circuit breaker is open; use --force to override")
		re.mu.Unlock()
		re.persistEvent(evt)
		return evt, nil
	}

	if re.state.CircuitOpen && force {
		re.resetCounters()
	}

	re.state.Status = "recovering"
	re.state.LastRecovery = time.Now()
	re.mu.Unlock()

	// Execute recovery without holding the lock
	evt := re.executeRecovery(ctx, "manual")

	// Re-acquire lock to update state
	re.mu.Lock()
	re.applyRecoveryOutcome(evt, "manual recovery trigger")
	re.mu.Unlock()

	return evt, nil
}

// State returns the current recovery engine state.
func (re *RecoveryEngine) State() domain.RecoveryState {
	re.mu.Lock()
	defer re.mu.Unlock()
	return re.state
}

// ResetCircuit manually resets the circuit breaker.
func (re *RecoveryEngine) ResetCircuit() {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.resetCounters()
	re.state.Status = "idle"
}

// ListEvents returns recent recovery events from the database.
func (re *RecoveryEngine) ListEvents(limit int) ([]domain.RecoveryEvent, error) {
	return re.recoveryStore.ListEvents(limit)
}

func (re *RecoveryEngine) attemptRecovery(ctx context.Context, triggerType, details string) (*domain.RecoveryEvent, error) {
	re.state.Status = "recovering"
	re.state.LastRecovery = time.Now()

	// Release lock during the actual recovery operation (restart + polling)
	re.mu.Unlock()
	evt := re.executeRecovery(ctx, triggerType)
	re.mu.Lock()

	re.applyRecoveryOutcome(evt, details)
	return evt, nil
}

// applyRecoveryOutcome updates engine state based on recovery result.
// Must be called with re.mu held.
func (re *RecoveryEngine) applyRecoveryOutcome(evt *domain.RecoveryEvent, details string) {
	if evt.Outcome == "success" {
		re.resetCounters()
		re.state.Status = "idle"
	} else {
		re.state.FailedRecovery++

		if re.state.FailedRecovery >= re.MaxBackoffRetries {
			re.state.CircuitOpen = true
			re.circuitAt = time.Now()
			re.state.Status = "circuit_open"
			slog.Warn("recovery: circuit breaker OPEN", "component", "recovery", "failures", re.state.FailedRecovery)
		} else {
			backoffIdx := min(re.state.BackoffLevel, len(re.BackoffSchedule)-1)
			re.state.NextRetryAfter = time.Now().Add(re.BackoffSchedule[backoffIdx])
			re.state.BackoffLevel++
			re.state.Status = "monitoring"
		}
	}

	evt.Details = details
}

// newEvent creates a RecoveryEvent with the standard action field pre-filled.
func newEvent(triggerType, outcome, details string) *domain.RecoveryEvent {
	return &domain.RecoveryEvent{
		TriggerType: triggerType,
		Action:      "systemctl_restart",
		Outcome:     outcome,
		Details:     details,
	}
}

// executeRecovery performs the restart and polls for readiness.
// Does NOT hold re.mu — safe for concurrent state reads.
func (re *RecoveryEngine) executeRecovery(ctx context.Context, triggerType string) *domain.RecoveryEvent {
	restartStart := time.Now()

	// Execute systemctl restart cloudflared
	_, err := re.cmdRunner(ctx, "sudo", "systemctl", "restart", "cloudflared")
	if err != nil {
		evt := newEvent(triggerType, "failure", fmt.Sprintf("restart failed: %v", err))
		re.persistEvent(evt)
		return evt
	}

	// Poll /ready until timeout
	evt := re.pollReady(ctx, triggerType, restartStart)
	re.persistEvent(evt)
	return evt
}

// pollReady polls the health check endpoint until ready or timeout.
func (re *RecoveryEngine) pollReady(ctx context.Context, triggerType string, restartStart time.Time) *domain.RecoveryEvent {
	deadline := time.Now().Add(re.ReadyPollTimeout)
	for time.Now().Before(deadline) {
		status := re.healthCheck.Check(ctx)
		if status.Ready == "ok" {
			return newEvent(triggerType, "success",
				fmt.Sprintf("tunnel recovered after restart, ready in %v", time.Since(restartStart).Round(time.Millisecond)))
		}
		time.Sleep(re.ReadyPollInterval)
	}

	return newEvent(triggerType, "failure",
		fmt.Sprintf("tunnel did not become ready within %v after restart", re.ReadyPollTimeout))
}

// persistEvent logs errors from event persistence instead of silently discarding them.
func (re *RecoveryEngine) persistEvent(evt *domain.RecoveryEvent) {
	if err := re.recoveryStore.PersistEvent(evt); err != nil {
		slog.Error("persist recovery event", "component", "recovery", "trigger", evt.TriggerType, "error", err)
	}
}

// EvaluateHA checks HA connection metrics and triggers recovery if connections
// drop to 0 for ConsecutiveFailures consecutive scrapes.
func (re *RecoveryEngine) EvaluateHA(ctx context.Context, scraper *MetricsScraper) (*domain.RecoveryEvent, error) {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.checkGuards() {
		return nil, nil
	}

	metrics, err := scraper.Scrape(ctx)
	if err != nil {
		// Can't scrape metrics — don't trigger, just log
		return nil, nil
	}

	if metrics.HAConnections > 0 {
		// Healthy — reset consecutive counter
		re.consecHAZero = 0
		return nil, nil
	}

	// HA = 0
	re.consecHAZero++
	if re.consecHAZero < re.ConsecutiveFailures {
		return nil, nil
	}

	// Trigger recovery
	details := fmt.Sprintf("HA connections at 0 for %d consecutive scrapes", re.consecHAZero)
	evt, err := re.attemptRecovery(ctx, "ha_connection_loss", details)
	if evt != nil && evt.Outcome == "success" {
		re.consecHAZero = 0
	}
	return evt, err
}
