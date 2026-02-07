package storage

import "fmt"

// ErrorKind identifies structured storage failures.
type ErrorKind string

const (
	// ErrInvalidInput indicates caller-provided invalid parameters.
	ErrInvalidInput ErrorKind = "invalid_input"
	// ErrResolve indicates failure resolving platform/profile roots.
	ErrResolve ErrorKind = "resolve_error"
	// ErrIO indicates filesystem operation failures.
	ErrIO ErrorKind = "io_error"
)

// Error is a structured storage error.
type Error struct {
	Kind    ErrorKind
	Message string
	Details string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := fmt.Sprintf("api-core storage %s: %s", e.Kind, e.Message)
	if e.Details != "" {
		msg += " (" + e.Details + ")"
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
