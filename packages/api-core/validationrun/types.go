// Package validationrun contains provider-neutral rules for durable validation
// work. Providers retain persistence, execution, recovery, and artifact policy.
package validationrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type State string

const (
	StateQueued         State = "queued"
	StateRunning        State = "running"
	StateSucceeded      State = "succeeded"
	StateFailed         State = "failed"
	StateCanceled       State = "canceled"
	StateRecoveryFailed State = "recovery_failed"
)

func (s State) Terminal() bool {
	return s == StateSucceeded || s == StateFailed || s == StateCanceled || s == StateRecoveryFailed
}

type ErrorCode string

const (
	ErrorInvalidTransition   ErrorCode = "invalid_transition"
	ErrorNotFound            ErrorCode = "not_found"
	ErrorIdempotencyConflict ErrorCode = "idempotency_conflict"
	ErrorAbortRejected       ErrorCode = "abort_rejected"
	ErrorWaitTimeout         ErrorCode = "wait_timeout"
	ErrorExecutionFailed     ErrorCode = "execution_failed"
	ErrorRecoveryFailed      ErrorCode = "recovery_failed"
)

// LifecycleError is safe for API mapping and recovery decisions. Details stay
// provider-owned; this type only carries reusable lifecycle vocabulary.
type LifecycleError struct {
	Code      ErrorCode
	Operation string
	State     State
	Cause     error
}

func (e *LifecycleError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("validation run %s (%s): %v", e.Operation, e.Code, e.Cause)
	}
	return fmt.Sprintf("validation run %s (%s)", e.Operation, e.Code)
}

func (e *LifecycleError) Unwrap() error { return e.Cause }

func IsCode(err error, code ErrorCode) bool {
	var lifecycle *LifecycleError
	return errors.As(err, &lifecycle) && lifecycle.Code == code
}

type Target struct {
	Scenario string
	Path     string
}

func (t Target) Equal(other Target) bool { return t.Scenario == other.Scenario && t.Path == other.Path }

func (t Target) Validate() error {
	if strings.TrimSpace(t.Scenario) == "" && strings.TrimSpace(t.Path) == "" {
		return &LifecycleError{Code: ErrorInvalidTransition, Operation: "start", Cause: errors.New("scenario or path is required")}
	}
	return nil
}

type Run struct {
	ID                    string
	Target                Target
	IdempotencyKey        string
	ParentRunID           string
	State                 State
	CreatedAt             time.Time
	StartedAt             time.Time
	CompletedAt           time.Time
	CancellationRequested bool
	Version               int64
}

func (r Run) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("run id is required")
	}
	if err := r.Target.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return errors.New("idempotency key is required")
	}
	if r.State == "" {
		return errors.New("run state is required")
	}
	return nil
}

type Event string

const (
	EventClaim          Event = "claim"
	EventSucceed        Event = "succeed"
	EventFail           Event = "fail"
	EventCancel         Event = "cancel"
	EventRecoveryFailed Event = "recovery_failed"
	EventRequestAbort   Event = "request_abort"
)

// Repository is implemented by a provider-owned durable ledger. Create must
// atomically enforce a unique idempotency key; Update must reject stale Version.
type Repository interface {
	FindByIdempotency(context.Context, string) (Run, error)
	Get(context.Context, string) (Run, error)
	Create(context.Context, Run) error
	Update(context.Context, Run, int64) error
}

// Notifier blocks until provider state may have changed. It is a server-side
// wait seam, not an instruction for agents or clients to poll Get.
type Notifier interface {
	WaitForChange(context.Context, string, int64) error
}

// Executor owns provider work. Run must eventually use CommitTerminal; Abort
// receives only explicit cancellation requests and never caller disconnects.
type Executor interface {
	Run(context.Context, string)
	Abort(context.Context, string) error
}
