package execution

import (
	"fmt"
	"strings"
)

// ErrExecutionNotFound is returned when no execution matches an id.
type ErrExecutionNotFound struct{ ID string }

func (e ErrExecutionNotFound) Error() string { return fmt.Sprintf("execution %q not found", e.ID) }

// ErrInvalidExecution is returned when a request fails structural validation at
// the service boundary (e.g. an empty plan id on Start).
type ErrInvalidExecution struct{ Reason string }

func (e ErrInvalidExecution) Error() string {
	return fmt.Sprintf("invalid execution request: %s", e.Reason)
}

// ErrActiveExecutionConflict prevents accidental duplicate durable runners.
type ErrActiveExecutionConflict struct {
	PlanID       string
	ExecutionIDs []string
}

func (e ErrActiveExecutionConflict) Error() string {
	return fmt.Sprintf("plan %q already has active execution(s): %s", e.PlanID, strings.Join(e.ExecutionIDs, ", "))
}

// ErrValidationRequired is returned when a caller attempts to mark a phase done
// without a recent passing validation result or an explicit override reason.
type ErrValidationRequired struct {
	PhaseID string
	Reason  string
}

func (e ErrValidationRequired) Error() string {
	return fmt.Sprintf("phase %q requires validation before done: %s", e.PhaseID, e.Reason)
}
