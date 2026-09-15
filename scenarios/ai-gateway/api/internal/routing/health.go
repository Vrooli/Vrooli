package routing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-gateway/internal/providers"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// BreakerState is the persisted circuit-breaker lifecycle state for a
// (provider, role, kind) route. The effective state observed by routing is
// derived from the stored state plus the clock (see Breaker.Effective): a
// stored "open" record whose cooldown has elapsed is surfaced as "half_open"
// without a background writer flipping it.
type BreakerState string

const (
	BreakerClosed   BreakerState = "closed"
	BreakerOpen     BreakerState = "open"
	BreakerHalfOpen BreakerState = "half_open"
)

// FailureClass is the stable, provider-neutral classification of a provider
// execution failure. These codes are durable: they are persisted in provider
// health and (from Phase 3) route evidence, and are surfaced to operators.
type FailureClass string

const (
	FailureNone          FailureClass = ""
	FailureMissingBinary FailureClass = "missing_binary"
	FailureTimeout       FailureClass = "timeout"
	FailureMalformedJSON FailureClass = "malformed_json"
	FailurePolicyError   FailureClass = "policy_error"
	FailureExecution     FailureClass = "execution_error"
	FailureCancellation  FailureClass = "cancellation"
	FailureUnavailable   FailureClass = "unavailable"
	// FailureInsufficientCredits means the account behind the provider ran out
	// of credits (LPBS HTTP 402). It is an account condition, not a transient
	// execution fault, so a bounded retry cannot resolve it and the route must
	// escalate or move to an explicitly funded candidate.
	FailureInsufficientCredits FailureClass = "insufficient_credits"
	// FailureRateLimited means the provider throttled the request (HTTP 429).
	// Unlike execution_error it carries an observed Retry-After window when the
	// provider supplied one.
	FailureRateLimited FailureClass = "rate_limited"
	// FailureProviderOverloaded means the provider was temporarily unavailable
	// (HTTP 502/503/504), a transient condition with an observed recovery
	// window when the provider supplied one.
	FailureProviderOverloaded FailureClass = "provider_overloaded"
	// FailureUnsupportedSampling means a candidate was skipped because its
	// resolved role does not honor the caller's explicit sampling control. It is
	// never recorded against provider health: the provider failed nothing, and
	// tripping a breaker over a policy mismatch would suppress a healthy route.
	FailureUnsupportedSampling FailureClass = "unsupported_sampling"
)

// ProviderFailure is the typed, provider-neutral classification of a provider
// execution failure together with any observed recovery window. Class is the
// stable classification; RetryAfter and ResetAt carry observed provenance only
// (empty means the provider supplied none, and they are never synthesized from
// a default cooldown); Source identifies where the observation came from.
type ProviderFailure struct {
	Class      FailureClass
	HTTPStatus int
	RetryAfter string
	ResetAt    string
	Source     string
}

// HasRecoveryWindow reports whether the failure carries an observed recovery
// time, so a caller parks on it instead of using a generic backoff.
func (f ProviderFailure) HasRecoveryWindow() bool {
	return strings.TrimSpace(f.RetryAfter) != "" || strings.TrimSpace(f.ResetAt) != ""
}

// ClassifyProviderFailure maps a provider adapter error into a typed failure
// class with its observed recovery provenance. It reads the resource
// CommandError.Code where present and falls back to context
// cancellation/deadline semantics so callers never have to inspect
// provider-specific strings.
func ClassifyProviderFailure(err error) ProviderFailure {
	if err == nil {
		return ProviderFailure{}
	}
	if errors.Is(err, context.Canceled) {
		return ProviderFailure{Class: FailureCancellation, Source: "context"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ProviderFailure{Class: FailureTimeout, Source: "context"}
	}
	observed := ProviderFailure{}
	if errors.Is(err, providers.ErrInsufficientCredits) {
		observed.Class = FailureInsufficientCredits
	}
	var cmdErr *providers.CommandError
	if errors.As(err, &cmdErr) {
		observed.HTTPStatus = cmdErr.HTTPStatus
		observed.RetryAfter = strings.TrimSpace(cmdErr.RetryAfter)
		observed.ResetAt = strings.TrimSpace(cmdErr.ResetAt)
		if observed.Class == "" {
			switch cmdErr.Code {
			case "missing_binary":
				observed.Class = FailureMissingBinary
			case "timeout":
				observed.Class = FailureTimeout
			case "malformed_json":
				observed.Class = FailureMalformedJSON
			case "exit_error", "command_failed", "provider_failed":
				observed.Class = FailureExecution
			case "unsupported_command", "unsupported_provider", "unsupported_kind", "invalid_request":
				observed.Class = FailurePolicyError
			case "unavailable", "empty_inventory":
				observed.Class = FailureUnavailable
			case "insufficient_credits":
				observed.Class = FailureInsufficientCredits
			case "rate_limited":
				observed.Class = FailureRateLimited
			case "provider_overloaded", "stream_failed":
				observed.Class = FailureProviderOverloaded
			case "unreachable":
				observed.Class = FailureUnavailable
			}
		}
	}
	if observed.Source == "" {
		switch {
		case observed.HTTPStatus != 0:
			observed.Source = fmt.Sprintf("http:%d", observed.HTTPStatus)
		case observed.Class != "":
			observed.Source = "provider-command"
		}
	}
	if observed.Class == "" {
		observed.Class = FailureExecution
	}
	return observed
}

// ClassifyProviderError returns only the stable failure class. Prefer
// ClassifyProviderFailure when the observed Retry-After/reset provenance is
// needed for recovery.
func ClassifyProviderError(err error) FailureClass {
	return ClassifyProviderFailure(err).Class
}

// HealthKey identifies one circuit breaker. Breakers are isolated per provider,
// role, and request kind so one provider/role's failures never suppress a
// healthy fallback.
type HealthKey struct {
	Provider string
	Role     string
	Kind     sharedv1.RequestKind
}

func normalizeHealthKey(k HealthKey) HealthKey {
	return HealthKey{
		Provider: strings.TrimSpace(strings.ToLower(k.Provider)),
		Role:     strings.TrimSpace(k.Role),
		Kind:     k.Kind,
	}
}

// ProviderHealth is the persisted breaker record for one HealthKey.
//
// The HTTPStatus/RetryAfter/ResetAt/FailureSource fields preserve the observed
// provenance of the most recent failure alongside its class. ResetAt and
// RetryAfter are recorded only when the provider supplied them; they are never
// synthesized from the breaker's own cooldown, so a reader can tell an observed
// recovery window from a policy default. A zero HTTPStatus and empty strings
// mean the failure carried no transport provenance (context cancellation, a
// local command fault).
type ProviderHealth struct {
	Provider            string
	Role                string
	Kind                sharedv1.RequestKind
	State               BreakerState
	ConsecutiveFailures int
	LastFailureClass    FailureClass
	HTTPStatus          int
	RetryAfter          string
	ResetAt             string
	FailureSource       string
	LastSuccessAt       time.Time
	LastFailureAt       time.Time
	CooldownUntil       time.Time
	OpenedAt            time.Time
	Generation          int64
	UpdatedAt           time.Time
}

// BreakerPolicy carries the deterministic thresholds that govern transitions.
type BreakerPolicy struct {
	// FailureThreshold is the number of consecutive failures that opens a
	// closed breaker.
	FailureThreshold int
	// Cooldown is how long a breaker stays fully open before a half-open probe
	// is allowed.
	Cooldown time.Duration
}

// DefaultBreakerPolicy is conservative: a small burst of consecutive failures
// opens the breaker, and a short cooldown allows recovery probes.
func DefaultBreakerPolicy() BreakerPolicy {
	return BreakerPolicy{FailureThreshold: 3, Cooldown: 30 * time.Second}
}

func (p BreakerPolicy) normalized() BreakerPolicy {
	if p.FailureThreshold <= 0 {
		p.FailureThreshold = DefaultBreakerPolicy().FailureThreshold
	}
	if p.Cooldown <= 0 {
		p.Cooldown = DefaultBreakerPolicy().Cooldown
	}
	return p
}

// Breaker holds the pure transition logic. It has no storage or clock of its
// own; callers pass `now` so behavior is deterministic and testable.
type Breaker struct {
	policy BreakerPolicy
}

func NewBreaker(policy BreakerPolicy) Breaker {
	return Breaker{policy: policy.normalized()}
}

// Effective returns the breaker state routing should act on at time `now`. A
// stored open record whose cooldown has elapsed is reported as half_open so a
// single bounded probe can attempt recovery.
func (b Breaker) Effective(h ProviderHealth, now time.Time) BreakerState {
	switch h.State {
	case BreakerOpen:
		if !h.CooldownUntil.IsZero() && !now.Before(h.CooldownUntil) {
			return BreakerHalfOpen
		}
		return BreakerOpen
	case BreakerHalfOpen:
		return BreakerHalfOpen
	default:
		return BreakerClosed
	}
}

// OnSuccess records a successful execution, closing the breaker.
func (b Breaker) OnSuccess(h ProviderHealth, now time.Time) ProviderHealth {
	h.State = BreakerClosed
	h.ConsecutiveFailures = 0
	h.LastSuccessAt = now
	h.CooldownUntil = time.Time{}
	h.OpenedAt = time.Time{}
	h.UpdatedAt = now
	return h
}

// OnFailure records a typed failure and opens (or reopens) the breaker when the
// consecutive-failure threshold is crossed or a half-open probe fails.
func (b Breaker) OnFailure(h ProviderHealth, class FailureClass, now time.Time) ProviderHealth {
	prevEffective := b.Effective(h, now)
	h.ConsecutiveFailures++
	h.LastFailureClass = class
	h.LastFailureAt = now
	h.UpdatedAt = now
	if prevEffective == BreakerHalfOpen || h.ConsecutiveFailures >= b.policy.FailureThreshold {
		h.State = BreakerOpen
		h.OpenedAt = now
		h.CooldownUntil = now.Add(b.policy.Cooldown)
		h.Generation++
	}
	return h
}

// OnProviderFailure records a typed failure together with the observed
// provenance from its ProviderFailure. It is the full-fidelity form of
// OnFailure: the breaker transition is identical, and the observed HTTP status,
// Retry-After, reset time and source are stored verbatim so a reader can park
// on the provider's own recovery window instead of the breaker's default
// cooldown. The provenance is previous-failure state, so it is cleared when a
// later observation supplies none rather than lingering with a stale window.
func (b Breaker) OnProviderFailure(h ProviderHealth, failure ProviderFailure, now time.Time) ProviderHealth {
	h = b.OnFailure(h, failure.Class, now)
	h.HTTPStatus = failure.HTTPStatus
	h.RetryAfter = strings.TrimSpace(failure.RetryAfter)
	h.ResetAt = strings.TrimSpace(failure.ResetAt)
	h.FailureSource = strings.TrimSpace(failure.Source)
	return h
}
