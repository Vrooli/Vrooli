package heartbeat

import (
	"errors"
	"strings"
)

// Dispatch-outcome classification for a run-creation attempt.
//
// A heartbeat dispatch creates a task and then a run at the owner. The unsafe
// case is a run-creation request whose outcome is unknown: the owner may have
// accepted the run even though the caller never saw the response. Releasing the
// member's slot on that error lets the next scheduled tick admit a duplicate
// heartbeat, recreating the stuck-running ambiguity the phase exists to remove.
//
// The seam distinguishes two outcomes:
//
//   - DispatchRejectedError: the owner gave a definitive response (validation,
//     conflict, auth, malformed request) or the request provably never left the
//     process. No run was accepted, so the slot may be released.
//   - DispatchUncertainError: the request may have reached the owner (transport
//     failure, exhausted retries, unreadable success body). The slot must be
//     retained as a visible, recoverable obligation until reconciled.
type DispatchRejectedError struct {
	Cause error
}

func (e *DispatchRejectedError) Error() string {
	if e == nil || e.Cause == nil {
		return "dispatch rejected"
	}
	return "dispatch rejected: " + e.Cause.Error()
}

func (e *DispatchRejectedError) Unwrap() error { return e.Cause }

// DispatchUncertainError marks a dispatch whose outcome the caller cannot prove.
type DispatchUncertainError struct {
	Cause error
}

func (e *DispatchUncertainError) Error() string {
	if e == nil || e.Cause == nil {
		return "dispatch outcome uncertain"
	}
	return "dispatch outcome uncertain: " + e.Cause.Error()
}

func (e *DispatchUncertainError) Unwrap() error { return e.Cause }

// NewDispatchRejectedError wraps a definitive owner refusal.
func NewDispatchRejectedError(cause error) error { return &DispatchRejectedError{Cause: cause} }

// NewDispatchUncertainError wraps a request whose acceptance cannot be proven.
func NewDispatchUncertainError(cause error) error { return &DispatchUncertainError{Cause: cause} }

// IsDispatchRejected reports whether err is a definitive owner rejection.
func IsDispatchRejected(err error) bool {
	var rejected *DispatchRejectedError
	return errors.As(err, &rejected)
}

// IsDispatchUncertain reports whether err leaves the dispatch outcome unknown.
//
// It is deliberately conservative at the boundaries: a typed uncertain error is
// always uncertain, a typed rejection is always deterministic, and an
// unclassified error is treated as uncertain only when it carries a transport
// marker. Unclassified local errors (for example a request-building failure)
// stay deterministic so they do not strand a member's slot.
func IsDispatchUncertain(err error) bool {
	if err == nil {
		return false
	}
	var uncertain *DispatchUncertainError
	if errors.As(err, &uncertain) {
		return true
	}
	if IsDispatchRejected(err) {
		return false
	}
	return looksLikeTransportUncertainty(err.Error())
}

func looksLikeTransportUncertainty(message string) bool {
	lower := strings.ToLower(message)
	for _, marker := range []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"timed out",
		"timeout",
		"deadline exceeded",
		"unexpected eof",
		"unreachable",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
