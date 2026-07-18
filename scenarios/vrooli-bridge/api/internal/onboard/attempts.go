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
