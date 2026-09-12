// Package apierr provides typed domain errors that carry HTTP status codes.
//
// Services return these errors so handlers can map them to HTTP responses
// through a single MapError call instead of per-handler switch/case chains.
//
// Usage in services:
//
//	return apierr.NotFound("backlog item %q", name)
//	return apierr.Wrap(err, http.StatusConflict, "queue full")
//
// Usage in handlers:
//
//	if err := svc.DoSomething(); err != nil {
//	    apierr.MapError(w, "[handler] action", err)
//	    return
//	}
package apierr

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for programmatic error matching via errors.Is.
var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrConflict           = errors.New("conflict")
	ErrBadRequest         = errors.New("bad request")
	ErrForbidden          = errors.New("forbidden")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrBadGateway         = errors.New("bad gateway")
	ErrCircuitBroken      = errors.New("circuit breaker tripped")
	ErrSessionExpired     = errors.New("session expired")
	ErrAtCapacity         = errors.New("concurrency limit reached")
	ErrNotImplemented     = errors.New("not implemented")
)

// DomainError wraps a sentinel error with an HTTP status code and a
// user-facing message. It implements the error interface and supports
// errors.Is/errors.As unwrapping.
//
// Code, when non-empty, identifies the error class for machine consumption
// (e.g., "plan_stale"); Details, when non-nil, carries arbitrary structured
// payload that the mapper serializes as JSON. Both are optional — handlers
// may continue to use the plain message-only form.
type DomainError struct {
	// Sentinel is the underlying sentinel error (e.g., ErrNotFound).
	Sentinel error
	// Status is the HTTP status code to return.
	Status int
	// Message is the user-facing error message.
	Message string
	// Code is an optional machine-readable error class (e.g., "plan_stale").
	// When set, MapError emits a JSON error body with this code.
	Code string
	// Details is an optional structured payload describing the error.
	// JSON-serializable. When non-nil, MapError emits a JSON error body.
	Details any
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Sentinel
}

// Wrap creates a DomainError wrapping a sentinel with a custom status and message.
func Wrap(sentinel error, status int, msg string) *DomainError {
	return &DomainError{Sentinel: sentinel, Status: status, Message: msg}
}

// Wrapf creates a DomainError with a formatted message.
func Wrapf(sentinel error, status int, format string, args ...any) *DomainError {
	return &DomainError{Sentinel: sentinel, Status: status, Message: fmt.Sprintf(format, args...)}
}

// --- Convenience constructors ---

// NotFound returns a 404 DomainError.
func NotFound(format string, args ...any) *DomainError {
	return Wrapf(ErrNotFound, http.StatusNotFound, format, args...)
}

// AlreadyExists returns a 409 DomainError for duplicate resources.
func AlreadyExists(format string, args ...any) *DomainError {
	return Wrapf(ErrAlreadyExists, http.StatusConflict, format, args...)
}

// Conflict returns a 409 DomainError for general conflicts.
func Conflict(format string, args ...any) *DomainError {
	return Wrapf(ErrConflict, http.StatusConflict, format, args...)
}

// BadRequest returns a 400 DomainError.
func BadRequest(format string, args ...any) *DomainError {
	return Wrapf(ErrBadRequest, http.StatusBadRequest, format, args...)
}

// Forbidden returns a 403 DomainError.
func Forbidden(format string, args ...any) *DomainError {
	return Wrapf(ErrForbidden, http.StatusForbidden, format, args...)
}

// Unavailable returns a 503 DomainError.
func Unavailable(format string, args ...any) *DomainError {
	return Wrapf(ErrServiceUnavailable, http.StatusServiceUnavailable, format, args...)
}

// BadGateway returns a 502 DomainError.
func BadGateway(format string, args ...any) *DomainError {
	return Wrapf(ErrBadGateway, http.StatusBadGateway, format, args...)
}

// Internal returns a 500 DomainError for unexpected failures.
func Internal(format string, args ...any) *DomainError {
	return Wrapf(errors.New("internal error"), http.StatusInternalServerError, format, args...)
}

// NotImplemented returns a 501 DomainError for endpoints whose request shape
// is structurally valid but whose backing functionality is not implemented
// (e.g., a configuration knob like BacklogSyncApplyMode that lands as a
// typed enum but only has v1 implementations for one value).
func NotImplemented(format string, args ...any) *DomainError {
	return Wrapf(ErrNotImplemented, http.StatusNotImplemented, format, args...)
}

// WithCode tags a DomainError with a machine-readable code. Returns the
// same error to allow chaining: apierr.BadRequest("...").WithCode("plan_stale").
func (e *DomainError) WithCode(code string) *DomainError {
	if e == nil {
		return nil
	}
	e.Code = code
	return e
}

// WithDetails attaches a structured payload to a DomainError. The payload
// must be JSON-serializable.
func (e *DomainError) WithDetails(details any) *DomainError {
	if e == nil {
		return nil
	}
	e.Details = details
	return e
}

// PlanStale returns a 409 DomainError carrying the "plan_stale" error code
// and a structured details payload. UI layers detect this code to render
// the re-workshop panel; CLI/script callers can switch on it programmatically.
func PlanStale(message string, details any) *DomainError {
	if message == "" {
		message = "plan references paths that no longer exist; re-workshop required"
	}
	return Wrap(ErrConflict, http.StatusConflict, message).WithCode("plan_stale").WithDetails(details)
}
