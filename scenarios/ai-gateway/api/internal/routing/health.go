package routing

import (
	"context"
	"errors"
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
	// FailureUnsupportedSampling means a candidate was skipped because its
	// resolved role does not honor the caller's explicit sampling control. It is
	// never recorded against provider health: the provider failed nothing, and
	// tripping a breaker over a policy mismatch would suppress a healthy route.
	FailureUnsupportedSampling FailureClass = "unsupported_sampling"
)

// ClassifyProviderError maps a provider adapter error into a stable failure
// class. It reads the resource CommandError.Code where present and falls back
// to context cancellation/deadline semantics so callers never have to inspect
// provider-specific strings.
func ClassifyProviderError(err error) FailureClass {
	if err == nil {
		return FailureNone
	}
	if errors.Is(err, context.Canceled) {
		return FailureCancellation
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureTimeout
	}
	var cmdErr *providers.CommandError
	if errors.As(err, &cmdErr) {
		switch cmdErr.Code {
		case "missing_binary":
			return FailureMissingBinary
		case "timeout":
			return FailureTimeout
		case "malformed_json":
			return FailureMalformedJSON
		case "exit_error", "command_failed":
			return FailureExecution
		case "unsupported_command", "unsupported_provider", "unsupported_kind", "invalid_request":
			return FailurePolicyError
		case "unavailable", "empty_inventory":
			return FailureUnavailable
		}
	}
	return FailureExecution
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
type ProviderHealth struct {
	Provider            string
	Role                string
	Kind                sharedv1.RequestKind
	State               BreakerState
	ConsecutiveFailures int
	LastFailureClass    FailureClass
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
