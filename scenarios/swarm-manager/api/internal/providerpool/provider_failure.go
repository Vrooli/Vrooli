package providerpool

import (
	"fmt"
	"strings"
)

// The AI Gateway classifies provider execution failures with a stable,
// provider-neutral vocabulary (scenarios/ai-gateway/api/internal/routing,
// FailureClass) and persists the observed recovery provenance on
// ProviderHealth. The shared pool consumes that vocabulary at its boundary so
// a forwarded provider failure arrives with the class and window the gateway
// actually observed instead of a caller-invented one.
//
// The gateway and the pool are separate modules, so the vocabulary strings are
// duplicated here rather than imported. This is deliberate: the pool must stay
// independently buildable and must not depend on the gateway's transport.
const (
	GatewayFailureInsufficientCredits = "insufficient_credits"
	GatewayFailureRateLimited         = "rate_limited"
	GatewayFailureProviderOverloaded  = "provider_overloaded"
)

// providerFailureClasses maps the AI Gateway failure vocabulary onto the pool's
// recovery classes. Only classes with a shared-pool recovery condition appear:
//
//	insufficient_credits -> api_credit_exhausted (account exhausted; the pool
//	                        recovers only on a later funded/credit observation)
//	rate_limited         -> rate_limit (observed Retry-After carried through)
//	provider_overloaded  -> provider_overload (transient; observed window)
//
// Per-attempt execution faults (missing_binary, timeout, malformed_json,
// policy_error, execution_error, cancellation, unsupported_sampling,
// unavailable) deliberately have no entry. They describe one failed attempt,
// not a shared pool whose capacity is exhausted, so they must never pause the
// pool for every descendant.
var providerFailureClasses = map[string]string{
	GatewayFailureInsufficientCredits: ClassAPICreditExhausted,
	GatewayFailureRateLimited:         ClassRateLimit,
	GatewayFailureProviderOverloaded:  ClassProviderOverload,
}

// ProviderFailure is the observed AI Gateway provider-failure provenance the
// pool boundary accepts. It mirrors the stable fields the gateway persists on
// ProviderHealth without importing the gateway module. Class is the gateway's
// FailureClass; HTTPStatus/RetryAfter/ResetAt/Source are the observed recovery
// provenance and are carried verbatim (empty means the provider supplied none,
// and the pool never synthesizes a window from its own defaults).
type ProviderFailure struct {
	Pool       string `json:"pool,omitempty"`
	Class      string `json:"class"`
	HTTPStatus int    `json:"http_status,omitempty"`
	RetryAfter string `json:"retry_after,omitempty"`
	ResetAt    string `json:"reset_at,omitempty"`
	Source     string `json:"source,omitempty"`
	Runner     string `json:"runner,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// ObservationFromProviderFailure translates an observed AI Gateway provider
// failure into a pool observation and reports whether the failure is a
// shared-pool recovery condition. It returns ok=false for a per-attempt
// execution fault that must not pause the pool, so a forwarder can report every
// provider failure and let the pool decide. The observed Retry-After, reset
// time, source, HTTP status and pool identity are carried verbatim.
func ObservationFromProviderFailure(failure ProviderFailure) (Observation, bool) {
	class, ok := providerFailureClasses[strings.TrimSpace(failure.Class)]
	if !ok {
		return Observation{}, false
	}
	detail := strings.TrimSpace(failure.Detail)
	if detail == "" {
		detail = fmt.Sprintf("ai-gateway %s", strings.TrimSpace(failure.Class))
		if failure.HTTPStatus != 0 {
			detail = fmt.Sprintf("%s (http %d)", detail, failure.HTTPStatus)
		}
	}
	return Observation{
		Pool:       strings.TrimSpace(failure.Pool),
		Class:      class,
		Runner:     strings.TrimSpace(failure.Runner),
		Provider:   strings.TrimSpace(failure.Provider),
		Model:      strings.TrimSpace(failure.Model),
		RetryAfter: strings.TrimSpace(failure.RetryAfter),
		ResetAt:    strings.TrimSpace(failure.ResetAt),
		Source:     strings.TrimSpace(failure.Source),
		Detail:     detail,
	}, true
}
