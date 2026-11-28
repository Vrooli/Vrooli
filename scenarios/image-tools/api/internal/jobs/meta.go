package jobs

// Job result_meta keys written by the output reviewer. The engine
// (internal/ai) writes them and the safety feedback surface reads them, so the
// vocabulary lives in the jobs domain both sides already depend on.
const (
	// MetaSensitive is the flag the API and UI act on (blur, tap to reveal).
	MetaSensitive = "sensitive"
	// MetaSensitiveRefs lists the flagged outputs.
	MetaSensitiveRefs = "sensitive_refs"
	// MetaSensitiveReason says why they were flagged ("classifier", "model").
	MetaSensitiveReason = "sensitive_reason"
	// MetaNSFWScore is the strongest classifier score in the batch.
	MetaNSFWScore = "nsfw_score"
	// MetaSafetyReview is the job-level review state: passed, flagged,
	// refused, partial_refused, shadow_refused or unavailable.
	MetaSafetyReview = "safety_review"
	// MetaSafetyReviewNotes is the reviewer's one-line account.
	MetaSafetyReviewNotes = "safety_review_detail"
	// MetaSafetyAction is the aggregate verdict: allow, flag or refuse.
	MetaSafetyAction = "safety_action"
	// MetaSafetyReason is the comma-separated machine-readable reason codes.
	MetaSafetyReason = "safety_reason"
	// MetaSafetyEvidence is the reviewer's whole JSON verdict.
	MetaSafetyEvidence = "safety_evidence"
	// MetaSafetyReviewedRefs is every ref in the order the reviewer saw it, so a
	// per-output finding maps to its blob after some variations were dropped.
	MetaSafetyReviewedRefs = "safety_reviewed_refs"
	// MetaSafetyAllowedRefs lists the outputs that passed.
	MetaSafetyAllowedRefs = "safety_allowed_refs"
	// MetaSafetyRefusedRefs lists the outputs the reviewer refused.
	MetaSafetyRefusedRefs = "safety_refused_refs"
	// MetaSafetyShadowRefs lists the would-refuse outputs in shadow mode.
	MetaSafetyShadowRefs = "safety_shadow_refs"
	// MetaSafetyQuarantineRefs lists the private quarantine keys that retain the
	// refused outputs for bounded admin review.
	MetaSafetyQuarantineRefs = "safety_quarantine_refs"
	// MetaSafetyQuarantineError records a quarantine write that failed; the
	// refused bytes were deleted as the fail-closed fallback.
	MetaSafetyQuarantineError = "safety_quarantine_error"
)
