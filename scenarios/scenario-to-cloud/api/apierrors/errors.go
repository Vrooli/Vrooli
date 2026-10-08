// Package apierrors is the typed error model shared by the scenario-to-cloud
// API, CLI and UI. Every failure carries a stable code, a message that never
// contains secrets, a retryability flag and an optional next action. The wire
// form is {"error":{...}} with snake_case keys and matches
// vrooli.scenario_to_cloud.v1.errors.Error.
package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Error is one typed failure. HTTPStatus is derived from Code unless set
// explicitly; it is never serialised.
type Error struct {
	Code       string
	Message    string
	Retryable  bool
	NextAction *NextAction
	Details    map[string]any
	HTTPStatus int
}

// NextAction points a caller at the owner and artifact that can move them past
// the error.
type NextAction struct {
	Owner     string `json:"owner,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Reference string `json:"reference,omitempty"`
	Label     string `json:"label,omitempty"`
}

// wireError is the JSON projection of Error.
type wireError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Retryable  bool           `json:"retryable"`
	NextAction *NextAction    `json:"next_action,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

// Envelope is the wire body: {"error": {...}}.
type Envelope struct {
	Error wireError `json:"error"`
}

// New builds an Error for a stable code.
func New(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf builds an Error with a formatted message.
func Newf(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Internal wraps an unexpected failure. The underlying error text is kept in
// details.cause for operators; callers must not put secrets in it.
func Internal(message string, cause error) *Error {
	e := &Error{Code: CodeInternal, Message: message}
	if cause != nil {
		e.Details = map[string]any{"cause": cause.Error()}
	}
	return e
}

// Error implements error.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

// Status returns the HTTP status for the error.
func (e *Error) Status() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}
	return StatusFor(e.Code)
}

// WithRetryable marks the error retryable.
func (e *Error) WithRetryable(retryable bool) *Error {
	e.Retryable = retryable
	return e
}

// WithNextAction attaches a next action.
func (e *Error) WithNextAction(action NextAction) *Error {
	e.NextAction = &action
	return e
}

// WithDetail adds one structured detail.
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	e.Details[key] = value
	return e
}

// MarshalJSON writes the wire projection of the error itself (not the
// envelope); Write and Envelope add the outer object.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.wire())
}

// UnmarshalJSON reads the wire projection.
func (e *Error) UnmarshalJSON(data []byte) error {
	var w wireError
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	*e = w.toError()
	return nil
}

func (e *Error) wire() wireError {
	return wireError{Code: e.Code, Message: e.Message, Retryable: e.Retryable, NextAction: e.NextAction, Details: e.Details}
}

func (w wireError) toError() Error {
	return Error{Code: w.Code, Message: w.Message, Retryable: w.Retryable, NextAction: w.NextAction, Details: w.Details}
}

// As extracts a typed Error from any error chain. Untyped errors become an
// internal error so callers always receive a stable code.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed
	}
	return Internal("Unexpected failure", err)
}

// Is reports whether err carries the given stable code.
func Is(err error, code string) bool {
	typed := As(err)
	return typed != nil && typed.Code == code
}

// Write encodes err as the typed envelope with the status derived from its
// code. A nil error writes nothing.
func Write(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	typed := As(err)
	body, encodeErr := json.Marshal(Envelope{Error: typed.wire()})
	if encodeErr != nil {
		body = []byte(`{"error":{"code":"internal","message":"Failed to encode error response","retryable":false}}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(typed.Status())
	_, _ = w.Write(append(body, '\n'))
}

// FromHTTP decodes a non-2xx response body into a typed Error. A body that is
// not the typed envelope still yields a stable code so clients never branch on
// free text: the status is kept and the raw body is preserved in details.
func FromHTTP(status int, body []byte) *Error {
	trimmed := strings.TrimSpace(string(body))
	var env Envelope
	if trimmed != "" && json.Unmarshal(body, &env) == nil && env.Error.Code != "" {
		typed := env.Error.toError()
		typed.HTTPStatus = status
		return &typed
	}
	typed := &Error{Code: codeForStatus(status), Message: http.StatusText(status), HTTPStatus: status}
	if trimmed != "" {
		typed.Details = map[string]any{"body": trimmed}
	}
	return typed
}

// FromResponse reads and decodes a non-2xx HTTP response.
func FromResponse(resp *http.Response) *Error {
	if resp == nil {
		return Internal("No response", nil)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return FromHTTP(resp.StatusCode, body)
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeInvalidRequest
	case http.StatusUnauthorized:
		return CodeUnauthenticated
	case http.StatusForbidden:
		return CodeForbiddenScope
	case http.StatusNotFound:
		return CodeDeploymentNotFound
	case http.StatusConflict:
		return CodeOperationConflict
	default:
		return CodeInternal
	}
}
