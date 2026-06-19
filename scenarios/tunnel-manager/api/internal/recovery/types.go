// Package recovery is the auto-recovery engine — the live actuation
// surface that restarts cloudflared (with exponential backoff + a
// circuit breaker) when the tunnel stops being ready, and the durable
// recovery_events log of every attempt.
//
// Layering mirrors the canonical pattern (see internal/routes for the
// worked reference):
//
//	HTTP → handler → Service (state machine + policy) → Repository (events)
//	                     ↑                                  ↑
//	                     FakeHealthChecker / FakeCmdRunner  FakeRepository
//
// Recovery is the single authoritative owner of cloudflared restart
// (vrooli-autoheal downgrades to alert-only). It is idempotent and
// replay-safe: a restart already in flight is not re-triggered, the
// circuit breaker pauses a restart storm, and every attempt is persisted
// as a recovery_events row so the log survives a process restart.
package recovery

import (
	"fmt"
	"time"
)

// RecoveryStatus is the state-machine phase. String-typed (like
// routes.Tier) so it round-trips through SQLite and reads naturally in
// the CLI; handlers translate to the proto enum at the edge.
type RecoveryStatus string

const (
	StatusIdle        RecoveryStatus = "idle"
	StatusMonitoring  RecoveryStatus = "monitoring"
	StatusRecovering  RecoveryStatus = "recovering"
	StatusCircuitOpen RecoveryStatus = "circuit_open"
)

// EventOutcome is the result of a single recovery attempt.
type EventOutcome string

const (
	OutcomeSuccess EventOutcome = "success"
	OutcomeFailure EventOutcome = "failure"
	OutcomeSkipped EventOutcome = "skipped"
)

// Trigger labels what prompted a recovery attempt.
const (
	TriggerReadyFailure = "ready_failure"
	TriggerHALoss       = "ha_connection_loss"
	TriggerManual       = "manual"
)

// ActionRestart is the one action the engine takes today: restart the
// cloudflared systemd unit. Stored on every event for forward symmetry
// with future actions (e.g. config_repush).
const ActionRestart = "systemctl_restart"

// RecoveryState is the live state-machine snapshot returned by GetState.
type RecoveryState struct {
	Status         RecoveryStatus
	ConsecFailures int
	BackoffLevel   int
	FailedRecovery int
	CircuitOpen    bool
	LastCheck      time.Time
	LastRecovery   time.Time
	NextRetryAfter time.Time
}

// RecoveryEvent is one durable log entry for a recovery attempt.
type RecoveryEvent struct {
	ID        string
	Trigger   string
	Action    string
	Outcome   EventOutcome
	Details   string
	Attempt   int
	CreatedAt time.Time
}

// ErrInvalidRecovery is the typed sentinel for bad input (e.g. negative
// limit). Handlers translate via errors.As into CodeInvalidArgument.
type ErrInvalidRecovery struct {
	Field  string
	Reason string
}

func (e ErrInvalidRecovery) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
