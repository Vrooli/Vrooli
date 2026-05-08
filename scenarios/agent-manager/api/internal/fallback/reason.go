// Package fallback owns the canonical taxonomy for "why did the runner or
// model reject this attempt?" — the Reason enum, the structured
// ClassifiedError shape, the per-Reason RecoveryAction map, and the
// Classifier interface that codecs implement to emit Reasons from their
// own structured signals (HTTP status, exit code, JSON event fields).
//
// This package is the single source of truth for fallback classification.
// Codecs may inspect their native error shapes; they MUST translate to a
// fallback.Reason here, not invent ad-hoc strings.
//
// Contract:
//   - Reason values are stable strings (used in typed event payloads and
//     the persisted health audit). Renaming a Reason is a schema break.
//   - RecoveryAction values map exhaustively from Reason via Recovery();
//     adding a Reason without updating the map is a compile-time error
//     (TestReasonRecoveryActionExhaustive).
//   - Classifier implementations live alongside their codecs (claude,
//     codex, opencode); this package owns only the interface, the data
//     types, and the residual TextClassifier safety net.
//
// DOC: scenarios/agent-manager/docs/internal/EVENT_TAXONOMY.md
// DOC: scenarios/agent-manager/docs/internal/ERROR_SEMANTICS.md
package fallback

// Reason categorises a runner/model rejection. Values are stable strings;
// they appear in persisted run_events payloads and the health audit
// tables, so renaming or removing a value is a schema break.
type Reason string

const (
	// ReasonRateLimit — the provider responded with a quota/throttling
	// indicator (HTTP 429, "rate limit exceeded", X-RateLimit-Remaining=0,
	// claude-cli rate-limit-reset event). Recovery: RetryBackoff.
	ReasonRateLimit Reason = "rate_limit"

	// ReasonAuthFailure — credentials are missing, invalid, or revoked
	// (HTTP 401/403, "invalid api key", "authentication failed"). Recovery:
	// EscalateOperator (no automatic retry will help).
	ReasonAuthFailure Reason = "auth_failure"

	// ReasonQuotaExhausted — the account is out of credit/quota beyond
	// what a wait will solve ("quota exceeded", "billing", HTTP 402).
	// Recovery: EscalateOperator.
	ReasonQuotaExhausted Reason = "quota_exhausted"

	// ReasonModelDeprecated — the requested model was retired or moved
	// ("deprecated", "retired", "no longer available", "sunset").
	// Recovery: FallbackToNext.
	ReasonModelDeprecated Reason = "model_deprecated"

	// ReasonModelUnknown — the runner does not recognise the model name
	// ("unknown model", "model not found", "invalid model"). Recovery:
	// FallbackToNext.
	ReasonModelUnknown Reason = "model_unknown"

	// ReasonModelUnavailable — generic "the runner rejected the model" with
	// no more-specific signal. Prefer ReasonModelUnknown / ReasonModelDeprecated
	// when classification has stronger evidence. Recovery: FallbackToNext.
	ReasonModelUnavailable Reason = "model_unavailable"

	// ReasonNetworkTransient — connectivity fault that may recover on retry
	// ("connection reset", "connection refused", "timed out", "temporarily
	// unavailable", HTTP 502/503/504). Recovery: RetryBackoff.
	ReasonNetworkTransient Reason = "network_transient"

	// ReasonContextLengthExceeded — request exceeds the model's context
	// window ("context length", "max tokens", "too long"). Recovery:
	// FallbackToNext (typically to a model with larger context).
	ReasonContextLengthExceeded Reason = "context_length_exceeded"

	// ReasonBinaryMissing — the runner's executable is not present on the
	// host ("command not found", "no such file", "ENOENT" on the binary).
	// Recovery: FallbackToNext (different runner) or EscalateOperator.
	ReasonBinaryMissing Reason = "binary_missing"

	// ReasonProbeTimeout — the model-availability probe timed out before
	// completing. Distinct from a network blip during execution; this is
	// the health-probe path. Recovery: RetryBackoff.
	ReasonProbeTimeout Reason = "probe_timeout"

	// ReasonInvalidFlag — the runner rejected an invocation flag (CLI
	// version skew, removed/renamed flag). Recovery: EscalateOperator.
	ReasonInvalidFlag Reason = "invalid_flag"

	// ReasonSessionExpired — codex/claude session token expired but the
	// session record is still recoverable; the runner can be re-attached
	// after re-auth. Recovery: EscalateOperator.
	ReasonSessionExpired Reason = "session_expired"

	// ReasonSessionStateLost — codex rollout file or session state was
	// truncated/lost mid-run; recovery requires starting a fresh session
	// (cannot be re-attached). Recovery: Abort (the run cannot continue
	// from where it left off).
	ReasonSessionStateLost Reason = "session_state_lost"

	// ReasonUnknown — classifier could not match any of the above; the
	// raw error text is preserved on ClassifiedError.Cause for operator
	// inspection. Recovery: Abort (we cannot reason about how to recover).
	ReasonUnknown Reason = "unknown"
)

// AllReasons returns every Reason in stable order. Used by exhaustiveness
// tests and by docs/event-taxonomy generation.
func AllReasons() []Reason {
	return []Reason{
		ReasonRateLimit,
		ReasonAuthFailure,
		ReasonQuotaExhausted,
		ReasonModelDeprecated,
		ReasonModelUnknown,
		ReasonModelUnavailable,
		ReasonNetworkTransient,
		ReasonContextLengthExceeded,
		ReasonBinaryMissing,
		ReasonProbeTimeout,
		ReasonInvalidFlag,
		ReasonSessionExpired,
		ReasonSessionStateLost,
		ReasonUnknown,
	}
}

// IsModelUnavailable reports whether this Reason indicates the runner
// rejected the requested model (and the chain walker should try the next
// preset entry). Replaces the old runner.ModelErrorUnavailable boolean.
func (r Reason) IsModelUnavailable() bool {
	switch r {
	case ReasonModelDeprecated, ReasonModelUnknown, ReasonModelUnavailable, ReasonContextLengthExceeded:
		return true
	}
	return false
}

// IsTransient reports whether this Reason represents a condition that
// might recover on retry without operator intervention. Replaces the old
// runner.ModelErrorTransient signal.
func (r Reason) IsTransient() bool {
	switch r {
	case ReasonRateLimit, ReasonNetworkTransient, ReasonProbeTimeout:
		return true
	}
	return false
}

// String is the stable wire form (== the Reason constant value).
func (r Reason) String() string { return string(r) }
