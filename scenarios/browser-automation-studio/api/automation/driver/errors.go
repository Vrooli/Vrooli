// Package driver provides a unified HTTP client for communicating with the playwright-driver.
// This client handles both recording mode and execution mode operations through a single interface.
package driver

import (
	"fmt"
	"strings"
)

// Error represents a structured error from the playwright driver.
// It provides operation context, hints for troubleshooting, and wraps underlying errors.
type Error struct {
	Op      string // operation that failed (health, start_session, run, etc.)
	URL     string // driver URL that was contacted
	Status  int    // HTTP status code if applicable
	Message string // human-readable error message
	Cause   error  // underlying error if any
	Hint    string // troubleshooting suggestion
}

func (e *Error) Error() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("playwright driver %s failed", e.Op))
	if e.URL != "" {
		parts = append(parts, fmt.Sprintf("at %s", e.URL))
	}
	if e.Status != 0 {
		parts = append(parts, fmt.Sprintf("(status %d)", e.Status))
	}
	parts = append(parts, fmt.Sprintf(": %s", e.Message))
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf(" (%v)", e.Cause))
	}
	if e.Hint != "" {
		parts = append(parts, fmt.Sprintf(" [hint: %s]", e.Hint))
	}
	return strings.Join(parts, "")
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// IsConnectionError returns true if the error is due to driver connectivity issues.
func (e *Error) IsConnectionError() bool {
	return e.Cause != nil && (strings.Contains(e.Cause.Error(), "connection refused") ||
		strings.Contains(e.Cause.Error(), "no such host"))
}

// IsSessionLimitError returns true if the error is due to too many concurrent sessions.
func (e *Error) IsSessionLimitError() bool {
	return strings.Contains(e.Message, "Maximum concurrent sessions")
}
