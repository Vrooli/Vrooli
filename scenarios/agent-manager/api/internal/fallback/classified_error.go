package fallback

import (
	"errors"
	"fmt"
)

// ClassifiedError is the structured outcome of classifying a runner
// failure. Codecs and probes return *ClassifiedError instead of bare
// `error` so the executor can route on Reason without re-parsing strings.
//
// A nil *ClassifiedError is a valid "no failure" sentinel (see
// IsClassifiedFailure). Callers that need to distinguish "missing" from
// "ReasonUnknown" should compare to nil first.
type ClassifiedError struct {
	// Reason is the canonical classification. Required.
	Reason Reason

	// Message is a short, operator-readable summary. Should NOT contain
	// secrets or full stack traces; the underlying Cause carries the raw
	// error for diagnostic logs.
	Message string

	// Cause is the underlying error this was classified from, preserved
	// for log/audit attachment. May be nil when classification was driven
	// by structured signals (HTTP status, JSON event) without a Go error.
	Cause error

	// HTTPStatus is the HTTP status that produced the classification, if
	// the signal was an HTTP response. 0 when not applicable.
	HTTPStatus int

	// ExitCode is the runner process exit code, if available. 0 when not
	// applicable (e.g. classification from a streaming JSON event).
	ExitCode int
}

// Error implements the error interface so *ClassifiedError can flow
// through Go error returns. Format: "<reason>: <message>".
func (e *ClassifiedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return string(e.Reason)
	}
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}

// Unwrap exposes the underlying Cause to errors.Is / errors.As.
func (e *ClassifiedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Recovery returns the canonical RecoveryAction for this Reason. See
// fallback/recovery.go for the full table.
func (e *ClassifiedError) Recovery() RecoveryAction {
	if e == nil {
		return RecoveryAbort
	}
	return Recovery(e.Reason)
}

// IsTransient reports whether the classified Reason is retry-eligible
// without operator intervention. Convenience for callers that don't need
// the full RecoveryAction.
func (e *ClassifiedError) IsTransient() bool {
	if e == nil {
		return false
	}
	return e.Reason.IsTransient()
}

// IsModelUnavailable reports whether the model chain walker should
// advance to the next preset entry on this error. Replaces the old
// runner.ModelErrorUnavailable boolean check.
func (e *ClassifiedError) IsModelUnavailable() bool {
	if e == nil {
		return false
	}
	return e.Reason.IsModelUnavailable()
}

// IsClassifiedFailure reports whether the supplied error is a non-nil
// *ClassifiedError. Used by callsites that accept either a bare error or
// a typed classification.
func IsClassifiedFailure(err error) bool {
	if err == nil {
		return false
	}
	var ce *ClassifiedError
	return errors.As(err, &ce) && ce != nil
}

// AsClassified extracts a *ClassifiedError from an error chain via
// errors.As, returning nil when the chain does not contain one.
func AsClassified(err error) *ClassifiedError {
	if err == nil {
		return nil
	}
	var ce *ClassifiedError
	if errors.As(err, &ce) {
		return ce
	}
	return nil
}

// New constructs a ClassifiedError. Convenience for codec implementations.
func New(reason Reason, message string, cause error) *ClassifiedError {
	return &ClassifiedError{
		Reason:  reason,
		Message: message,
		Cause:   cause,
	}
}
