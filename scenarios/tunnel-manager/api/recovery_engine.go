package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// RecoveryEngine monitors tunnel health and automatically restarts cloudflared
// when failures are detected. It implements exponential backoff and circuit breaker patterns.
type RecoveryEngine struct {
	db           *sql.DB
	healthCheck  *TunnelHealthChecker
	cmdRunner    func(ctx context.Context, name string, args ...string) ([]byte, error)
	mu           sync.Mutex
	state        RecoveryState
	circuitAt    time.Time
	consecHAZero int // consecutive metric scrapes showing HA connections = 0

	// Configurable thresholds
	ConsecutiveFailures int           // failures before triggering recovery (default 3)
	MaxBackoffRetries   int           // consecutive failures before circuit opens (default 5)
	CircuitCooldown     time.Duration // how long circuit stays open (default 1h)
	BackoffSchedule     []time.Duration
	ReadyPollTimeout    time.Duration // how long to poll /ready after restart (default 30s)
	ReadyPollInterval   time.Duration // poll interval (default 1s)
}

// RecoveryState tracks the current state of the recovery engine.
type RecoveryState struct {
	Status         string    `json:"status"`          // "idle", "monitoring", "recovering", "circuit_open"
	ConsecFailures int       `json:"consec_failures"` // consecutive health check failures
	BackoffLevel   int       `json:"backoff_level"`   // current backoff level
	FailedRecovery int       `json:"failed_recoveries"`
	CircuitOpen    bool      `json:"circuit_open"`
	LastCheck      time.Time `json:"last_check"`
	LastRecovery   time.Time `json:"last_recovery,omitempty"`
	NextRetryAfter time.Time `json:"next_retry_after,omitempty"`
}

// RecoveryEvent records a recovery attempt.
type RecoveryEvent struct {
	ID          int       `json:"id"`
	TriggerType string    `json:"trigger_type"` // "ready_failure", "ha_connection_loss", "manual"
	Action      string    `json:"action"`       // "systemctl_restart"
	Outcome     string    `json:"outcome"`      // "success", "failure", "skipped"
	Details     string    `json:"details,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type RecoveryOption func(*RecoveryEngine)

func WithRecoveryCmdRunner(fn func(ctx context.Context, name string, args ...string) ([]byte, error)) RecoveryOption {
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

func NewRecoveryEngine(db *sql.DB, healthCheck *TunnelHealthChecker, opts ...RecoveryOption) *RecoveryEngine {
	re := &RecoveryEngine{
		db:                  db,
		healthCheck:         healthCheck,
		cmdRunner:           defaultCmdRunner,
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
		state:             RecoveryState{Status: "idle"},
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
			log.Printf("recovery: circuit breaker reset after cooldown")
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
func (re *RecoveryEngine) Evaluate(ctx context.Context) (*RecoveryEvent, error) {
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
func (re *RecoveryEngine) TriggerManual(ctx context.Context, force bool) (*RecoveryEvent, error) {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.state.CircuitOpen && !force {
		evt := newEvent("manual", "skipped", "circuit breaker is open; use --force to override")
		re.persistEvent(evt)
		return evt, nil
	}

	if re.state.CircuitOpen && force {
		re.resetCounters()
	}

	return re.doRecovery(ctx, "manual")
}

// State returns the current recovery engine state.
func (re *RecoveryEngine) State() RecoveryState {
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

func (re *RecoveryEngine) attemptRecovery(ctx context.Context, triggerType, details string) (*RecoveryEvent, error) {
	re.state.Status = "recovering"

	evt, err := re.doRecovery(ctx, triggerType)
	if err != nil {
		return evt, err
	}

	if evt.Outcome == "success" {
		re.resetCounters()
		re.state.Status = "idle"
	} else {
		re.state.FailedRecovery++

		if re.state.FailedRecovery >= re.MaxBackoffRetries {
			re.state.CircuitOpen = true
			re.circuitAt = time.Now()
			re.state.Status = "circuit_open"
			log.Printf("recovery: circuit breaker OPEN after %d failures", re.state.FailedRecovery)
		} else {
			backoffIdx := min(re.state.BackoffLevel, len(re.BackoffSchedule)-1)
			re.state.NextRetryAfter = time.Now().Add(re.BackoffSchedule[backoffIdx])
			re.state.BackoffLevel++
			re.state.Status = "monitoring"
		}
	}

	evt.Details = details
	return evt, nil
}

// newEvent creates a RecoveryEvent with the standard action field pre-filled.
func newEvent(triggerType, outcome, details string) *RecoveryEvent {
	return &RecoveryEvent{
		TriggerType: triggerType,
		Action:      "systemctl_restart",
		Outcome:     outcome,
		Details:     details,
	}
}

func (re *RecoveryEngine) doRecovery(ctx context.Context, triggerType string) (*RecoveryEvent, error) {
	re.state.LastRecovery = time.Now()

	// Execute systemctl restart cloudflared
	_, err := re.cmdRunner(ctx, "sudo", "systemctl", "restart", "cloudflared")
	if err != nil {
		evt := newEvent(triggerType, "failure", fmt.Sprintf("restart failed: %v", err))
		re.persistEvent(evt)
		return evt, nil
	}

	// Poll /ready until timeout (uses injected healthCheck for testability)
	deadline := time.Now().Add(re.ReadyPollTimeout)
	for time.Now().Before(deadline) {
		status := re.healthCheck.Check(ctx)
		if status.Ready == "ok" {
			evt := newEvent(triggerType, "success",
				fmt.Sprintf("tunnel recovered after restart, ready in %v", time.Since(re.state.LastRecovery).Round(time.Millisecond)))
			re.persistEvent(evt)
			return evt, nil
		}
		time.Sleep(re.ReadyPollInterval)
	}

	evt := newEvent(triggerType, "failure",
		fmt.Sprintf("tunnel did not become ready within %v after restart", re.ReadyPollTimeout))
	re.persistEvent(evt)
	return evt, nil
}

func (re *RecoveryEngine) persistEvent(evt *RecoveryEvent) {
	if re.db == nil {
		return
	}
	var details *string
	if evt.Details != "" {
		details = &evt.Details
	}
	_, _ = re.db.Exec(
		`INSERT INTO recovery_events (trigger_type, action, outcome, details) VALUES ($1, $2, $3, $4)`,
		evt.TriggerType, evt.Action, evt.Outcome, details,
	)
}

// EvaluateHA checks HA connection metrics and triggers recovery if connections
// drop to 0 for ConsecutiveFailures consecutive scrapes.
func (re *RecoveryEngine) EvaluateHA(ctx context.Context, scraper *MetricsScraper) (*RecoveryEvent, error) {
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

// ListEvents returns recent recovery events from the database.
func (re *RecoveryEngine) ListEvents(limit int) ([]RecoveryEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := re.db.Query(
		`SELECT id, trigger_type, action, outcome, COALESCE(details, ''), created_at FROM recovery_events ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []RecoveryEvent
	for rows.Next() {
		var e RecoveryEvent
		if err := rows.Scan(&e.ID, &e.TriggerType, &e.Action, &e.Outcome, &e.Details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
