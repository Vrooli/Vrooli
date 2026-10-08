package closure

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Stable error codes. closure_unavailable is shared with apierrors; the
// closure-specific codes are owned here and mapped to HTTP status by Status.
const (
	CodeUnavailable    = "closure_unavailable"
	CodeCycle          = "closure_cycle"
	CodeConflict       = "closure_conflict"
	CodeInvalidRequest = "invalid_request"
)

// Error is a typed closure failure. Details are stable, machine-readable
// fields (the cycle path, the conflicting modes, the missing catalog entry).
type Error struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if len(e.Details) == 0 {
		return e.Code + ": " + e.Message
	}
	keys := make([]string, 0, len(e.Details))
	for key := range e.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, e.Details[key]))
	}
	return e.Code + ": " + e.Message + " (" + strings.Join(parts, ", ") + ")"
}

// Status maps a code to the HTTP status a handler should use.
func (e *Error) Status() int {
	switch e.Code {
	case CodeUnavailable:
		return http.StatusFailedDependency
	case CodeCycle, CodeConflict:
		return http.StatusUnprocessableEntity
	case CodeInvalidRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func newError(code, message string, details map[string]any) *Error {
	return &Error{Code: code, Message: message, Details: details}
}

func unavailable(message string, cause error, details map[string]any) *Error {
	if details == nil {
		details = map[string]any{}
	}
	if cause != nil {
		details["cause"] = cause.Error()
	}
	return newError(CodeUnavailable, message, details)
}

// As extracts a typed closure error from err.
func As(err error) (*Error, bool) {
	var typed *Error
	if errors.As(err, &typed) {
		return typed, true
	}
	return nil, false
}

// Is reports whether err carries the given closure code.
func Is(err error, code string) bool {
	typed, ok := As(err)
	return ok && typed.Code == code
}
