package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AttemptState is independent from SSH/pairing checkpoints. Terminal attempts
// are immutable; a retry always gets a new identity.
type AttemptState string

const (
	AttemptCreated     AttemptState = "created"
	AttemptRunning     AttemptState = "running"
	AttemptSucceeded   AttemptState = "succeeded"
	AttemptFailed      AttemptState = "failed"
	AttemptInterrupted AttemptState = "interrupted"
)

func (s AttemptState) Terminal() bool {
	return s == AttemptSucceeded || s == AttemptFailed || s == AttemptInterrupted
}

func (s AttemptState) Valid() bool {
	switch s {
	case AttemptCreated, AttemptRunning, AttemptSucceeded, AttemptFailed, AttemptInterrupted:
		return true
	default:
		return false
	}
}

func (s AttemptState) CanTransitionTo(next AttemptState) bool {
	if !s.Valid() || !next.Terminal() {
		return false
	}
	return s == AttemptCreated || s == AttemptRunning
}

type ErrInvalidAttemptTransition struct {
	Current   AttemptState
	Attempted AttemptState
}

func (e ErrInvalidAttemptTransition) Error() string {
	return fmt.Sprintf("invalid enrollment attempt transition %q -> %q", e.Current, e.Attempted)
}

type ErrInvalidAttemptState struct{ State AttemptState }

func (e ErrInvalidAttemptState) Error() string {
	return fmt.Sprintf("unknown persisted enrollment attempt state %q", e.State)
}

type EnrollmentAttempt struct {
	ID               string
	MachineID        string
	RetryOfAttemptID string
	CorrelationID    string
	State            AttemptState
	InputSnapshot    map[string]string
	TerminalResult   string
	Diagnostics      string
	CreatedAt        time.Time
	TerminalAt       time.Time
}

func (a *EnrollmentAttempt) TransitionState(next AttemptState) error {
	if a == nil || !a.State.CanTransitionTo(next) {
		current := AttemptState("")
		if a != nil {
			current = a.State
		}
		return ErrInvalidAttemptTransition{Current: current, Attempted: next}
	}
	a.State = next
	return nil
}

type AttemptStore interface {
	CreateAttempt(context.Context, EnrollmentAttempt) (EnrollmentAttempt, error)
	GetAttempt(context.Context, string) (EnrollmentAttempt, error)
	GetAttemptByCorrelation(context.Context, string) (EnrollmentAttempt, error)
	ListAttemptsForMachine(context.Context, string) ([]EnrollmentAttempt, error)
	RetryAttempt(context.Context, string, map[string]string) (EnrollmentAttempt, error)
	CompleteAttempt(context.Context, string, AttemptState, string, string) (EnrollmentAttempt, error)
	RecordCheckpoint(context.Context, string, string, string) error
}

// NewAttempt creates pre-contact enrollment intent. The snapshot is copied and
// serialized once so later retries cannot rewrite historic input.
func NewAttempt(machineID string, snapshot map[string]string) (EnrollmentAttempt, error) {
	if machineID == "" {
		return EnrollmentAttempt{}, ErrInvalid{Field: "machine_id", Reason: "required"}
	}
	copySnapshot := make(map[string]string, len(snapshot))
	for k, v := range snapshot {
		copySnapshot[k] = v
	}
	return EnrollmentAttempt{ID: uuid.NewString(), MachineID: machineID, CorrelationID: uuid.NewString(), State: AttemptCreated, InputSnapshot: copySnapshot}, nil
}

func marshalAttemptSnapshot(snapshot map[string]string) (string, error) {
	b, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("encode attempt snapshot: %w", err)
	}
	return string(b), nil
}
