package fallback

// RecoveryAction is the canonical "what should the system do next?" for a
// classified failure. The map from Reason → RecoveryAction is the single
// source of truth for fallback policy; it lives here, not scattered across
// callsites.
type RecoveryAction string

const (
	// RecoveryRetryImmediate — retry now, no backoff. Reserved for cases
	// where we have direct evidence the retry will succeed (e.g. recovered
	// session token); not currently emitted by any classifier.
	RecoveryRetryImmediate RecoveryAction = "retry_immediate"

	// RecoveryRetryBackoff — retry after exponential backoff. Used for
	// transient network/rate-limit conditions.
	RecoveryRetryBackoff RecoveryAction = "retry_backoff"

	// RecoveryFallbackToNext — advance to the next entry in the fallback
	// chain (model preset chain or runner fallback chain). Used when the
	// current target is structurally unable to serve this request.
	RecoveryFallbackToNext RecoveryAction = "fallback_to_next"

	// RecoveryAbort — terminate the run; no automatic recovery is safe.
	// Used for session-state-lost and ReasonUnknown.
	RecoveryAbort RecoveryAction = "abort"

	// RecoveryEscalateOperator — surface to a human; the system cannot
	// self-heal (auth failures, quota, invalid CLI flags).
	RecoveryEscalateOperator RecoveryAction = "escalate_operator"
)

// reasonRecovery is the exhaustive Reason → RecoveryAction table. Every
// Reason in AllReasons() must appear here; TestReasonRecoveryActionExhaustive
// pins this contract at build time.
//
// When adding a new Reason: add the constant in reason.go, append it to
// AllReasons(), then add the recovery row here. CI enforces all three.
var reasonRecovery = map[Reason]RecoveryAction{
	ReasonRateLimit:             RecoveryRetryBackoff,
	ReasonAuthFailure:           RecoveryEscalateOperator,
	ReasonQuotaExhausted:        RecoveryEscalateOperator,
	ReasonModelDeprecated:       RecoveryFallbackToNext,
	ReasonModelUnknown:          RecoveryFallbackToNext,
	ReasonModelUnavailable:      RecoveryFallbackToNext,
	ReasonNetworkTransient:      RecoveryRetryBackoff,
	ReasonContextLengthExceeded: RecoveryFallbackToNext,
	ReasonBinaryMissing:         RecoveryFallbackToNext,
	ReasonProbeTimeout:          RecoveryRetryBackoff,
	ReasonInvalidFlag:           RecoveryEscalateOperator,
	ReasonSessionExpired:        RecoveryEscalateOperator,
	ReasonSessionStateLost:      RecoveryAbort,
	ReasonUnknown:               RecoveryAbort,
}

// Recovery returns the canonical RecoveryAction for a Reason. Unknown
// inputs (constructed outside this package via casting) return
// RecoveryAbort to avoid silent misclassification.
func Recovery(r Reason) RecoveryAction {
	if action, ok := reasonRecovery[r]; ok {
		return action
	}
	return RecoveryAbort
}
