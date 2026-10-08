package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Provider-limit failure codes. They mirror the provider-neutral vocabulary the
// AI Gateway's routing classifier understands so an account-credit exhaustion,
// a rate limit or a transient overload is never flattened into a generic
// execution failure.
const (
	CodeInsufficientCredits = "insufficient_credits"
	CodeRateLimited         = "rate_limited"
	CodeProviderOverloaded  = "provider_overloaded"
	CodeProviderFailed      = "provider_failed"
	CodeStreamFailed        = "stream_failed"
	CodeUnreachable         = "unreachable"
)

// ProviderError is the OpenRouter owner adapter's typed classification of a
// provider-limit or transport failure. It preserves the observed HTTP status
// and Retry-After exactly as the provider sent them so a caller can park on the
// provider's own recovery window rather than a generic backoff. Empty
// RetryAfter means the provider sent none and must stay unknown.
type ProviderError struct {
	Code       string
	HTTPStatus int
	RetryAfter string
	Message    string
	Err        error
}

func (e *ProviderError) Error() string {
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
		return fmt.Sprintf("OpenRouter provider failure (%s, HTTP %d): %s", e.Code, e.HTTPStatus, detail)
	}
	return fmt.Sprintf("OpenRouter provider failure (%s): %s", e.Code, detail)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// AsProviderError reports whether err is, or wraps, a typed provider error.
func AsProviderError(err error) (*ProviderError, bool) {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr, true
	}
	return nil, false
}

func classifyHTTPStatus(status int) string {
	switch {
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

// newHTTPProviderError classifies an HTTP failure response, preserving the
// observed Retry-After. detail is the provider-supplied message when one was
// available.
func newHTTPProviderError(status int, retryAfter, detail string) *ProviderError {
	return &ProviderError{
		Code:       classifyHTTPStatus(status),
		HTTPStatus: status,
		RetryAfter: strings.TrimSpace(retryAfter),
		Message:    strings.TrimSpace(detail),
	}
}

// ProviderErrorMarker is the stable, provider-neutral line protocol an owner
// adapter writes to stderr so a subprocess caller can recover the typed failure
// across the resource-command boundary. The JSON shape must stay in sync with
// the `ai-gateway` resource-command runner that parses it.
const ProviderErrorMarker = "VROOLI_PROVIDER_ERROR "

// MarkerLine renders the single stderr line that carries the typed failure and
// its observed recovery provenance across the subprocess boundary.
func (e *ProviderError) MarkerLine() ([]byte, error) {
	if e == nil {
		return nil, errors.New("nil provider error")
	}
	payload := struct {
		Code       string `json:"code"`
		HTTPStatus int    `json:"http_status"`
		RetryAfter string `json:"retry_after,omitempty"`
		Message    string `json:"message,omitempty"`
	}{
		Code:       e.Code,
		HTTPStatus: e.HTTPStatus,
		RetryAfter: strings.TrimSpace(e.RetryAfter),
		Message:    strings.TrimSpace(e.Message),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return append([]byte(ProviderErrorMarker), raw...), nil
}
