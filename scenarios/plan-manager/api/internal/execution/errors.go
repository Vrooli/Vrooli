package execution

import "fmt"

// ErrExecutionNotFound is returned when no execution matches an id.
type ErrExecutionNotFound struct{ ID string }

func (e ErrExecutionNotFound) Error() string { return fmt.Sprintf("execution %q not found", e.ID) }

// ErrFindingNotFound is returned when no finding matches an id.
type ErrFindingNotFound struct{ ID string }

func (e ErrFindingNotFound) Error() string { return fmt.Sprintf("finding %q not found", e.ID) }

// ErrInvalidExecution is returned when a request fails structural validation at
// the service boundary (e.g. an empty plan id on Start).
type ErrInvalidExecution struct{ Reason string }

func (e ErrInvalidExecution) Error() string {
	return fmt.Sprintf("invalid execution request: %s", e.Reason)
}
