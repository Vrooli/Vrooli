package tasks

import "fmt"

// TransitionErrorCode classifies lifecycle transition failures for easier HTTP mapping.
type TransitionErrorCode string

const (
	// TransitionErrorCodeConflict indicates the requested move conflicts with the current task state (cooldown, locks, etc.).
	TransitionErrorCodeConflict TransitionErrorCode = "conflict"
	// TransitionErrorCodeUnsupported indicates the requested transition is not part of the canonical lifecycle.
	TransitionErrorCodeUnsupported TransitionErrorCode = "unsupported"
)

// TransitionError wraps the underlying error so callers can determine the severity/type without parsing strings.
type TransitionError struct {
	Code TransitionErrorCode
	Err  error
}

func (e *TransitionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func (e *TransitionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newTransitionError(code TransitionErrorCode, format string, args ...any) error {
	return &TransitionError{
		Code: code,
		Err:  fmt.Errorf(format, args...),
	}
}
