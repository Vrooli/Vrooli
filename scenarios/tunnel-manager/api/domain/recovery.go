package domain

import "time"

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
