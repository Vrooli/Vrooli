// Package providererror owns the provider-neutral failure vocabulary shared by
// AI provider resources and the AI Gateway resource-command runner.
package providererror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Typed provider-limit codes. They are the neutral seam between a native
// provider failure and the routing classifier so an account-credit exhaustion,
// a rate limit, or an overload is not flattened into a generic failure.
const (
	CodeInsufficientCredits = "insufficient_credits"
	CodeRateLimited         = "rate_limited"
	CodeProviderOverloaded  = "provider_overloaded"
	CodeProviderFailed      = "provider_failed"
	CodeStreamFailed        = "stream_failed"
	CodeUnreachable         = "unreachable"
	CodeCommandFailed       = "command_failed"
	CodeUnconfigured        = "unconfigured"
	CodeContextOverflow     = "context_overflow"
)

// Marker is the stable line protocol an owner adapter writes to stderr.
const Marker = "VROOLI_PROVIDER_ERROR "

// Error is an owner adapter's typed classification of a provider-limit or
// transport failure. It preserves the observed HTTP status and Retry-After
// exactly as the provider sent them; empty means the provider sent none.
type Error struct {
	Code       string
	HTTPStatus int
	RetryAfter string
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	detail := strings.TrimSpace(e.Message)
	if detail == "" && e.Err != nil {
		detail = e.Err.Error()
	}
	if detail == "" {
		detail = e.Code
	}
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("provider failure (%s, HTTP %d): %s", e.Code, e.HTTPStatus, detail)
	}
	return fmt.Sprintf("provider failure (%s): %s", e.Code, detail)
}

func (e *Error) Unwrap() error { return e.Err }

// As reports whether err is, or wraps, a typed provider error.
func As(err error) (*Error, bool) {
	var providerErr *Error
	if errors.As(err, &providerErr) {
		return providerErr, true
	}
	return nil, false
}

// ClassifyHTTPStatus maps an HTTP status to the typed vocabulary.
func ClassifyHTTPStatus(status int) string {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return CodeUnconfigured
	case status == http.StatusPaymentRequired:
		return CodeInsufficientCredits
	case status == http.StatusTooManyRequests:
		return CodeRateLimited
	case status == http.StatusServiceUnavailable, status == http.StatusBadGateway, status == http.StatusGatewayTimeout:
		return CodeProviderOverloaded
	default:
		return CodeProviderFailed
	}
}

// NewHTTP classifies an HTTP failure response.
func NewHTTP(status int, retryAfter, detail string) *Error {
	return &Error{
		Code:       ClassifyHTTPStatus(status),
		HTTPStatus: status,
		RetryAfter: strings.TrimSpace(retryAfter),
		Message:    strings.TrimSpace(detail),
	}
}

// Envelope is the JSON payload carried after Marker.
type Envelope struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"http_status,omitempty"`
	RetryAfter string `json:"retry_after,omitempty"`
	ResetAt    string `json:"reset_at,omitempty"`
	Message    string `json:"message,omitempty"`
}

// MarkerLine renders the single stderr line carrying a typed failure.
func (e *Error) MarkerLine() ([]byte, error) {
	if e == nil {
		return nil, errors.New("nil provider error")
	}
	raw, err := json.Marshal(Envelope{
		Code:       e.Code,
		HTTPStatus: e.HTTPStatus,
		RetryAfter: strings.TrimSpace(e.RetryAfter),
		Message:    strings.TrimSpace(e.Message),
	})
	if err != nil {
		return nil, err
	}
	return append([]byte(Marker), raw...), nil
}

// ParseMarker recovers the last typed envelope from stderr. It is best-effort:
// a missing or malformed marker returns false.
func ParseMarker(stderr string) (Envelope, bool) {
	var found Envelope
	ok := false
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, Marker) {
			continue
		}
		var envelope Envelope
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, Marker))), &envelope); err != nil {
			continue
		}
		if normalizeCode(envelope.Code) == "" {
			continue
		}
		found = envelope
		ok = true
	}
	return found, ok
}

// NormalizeCode returns a known code or the empty string.
func NormalizeCode(code string) string { return normalizeCode(code) }

func normalizeCode(code string) string {
	code = strings.TrimSpace(code)
	switch code {
	case CodeInsufficientCredits, CodeRateLimited, CodeProviderOverloaded,
		CodeProviderFailed, CodeStreamFailed, CodeUnreachable, CodeCommandFailed,
		CodeUnconfigured, CodeContextOverflow:
		return code
	default:
		return ""
	}
}
