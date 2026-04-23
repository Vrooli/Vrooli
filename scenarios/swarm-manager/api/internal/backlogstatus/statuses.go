// Package backlogstatus holds the canonical string values for backlog item
// lifecycle statuses, plus predicates over them. It has no dependencies on
// other internal packages so both `backlog` and `execution` can import it
// without creating a cycle (backlog imports execution to queue runs, and
// execution needs to write backlog statuses).
package backlogstatus

// Status string values. Kept as untyped constants so callers in either
// package can interpolate them into typed or untyped strings freely.
const (
	Backlog       = "backlog"
	Researching   = "researching"
	Ready         = "ready"
	Queued        = "queued"
	InProgress    = "in_progress"
	InReview      = "in_review"
	ReviewPending = "review_pending"
	Completed     = "completed"
	Failed        = "failed"
	NeedsFollowup = "needs_followup"
)

// All returns every valid status value in a stable order.
func All() []string {
	return []string{
		Backlog, Researching, Ready, Queued, InProgress,
		InReview, ReviewPending, Completed, Failed, NeedsFollowup,
	}
}

// IsValid reports whether s is one of the known status strings.
func IsValid(s string) bool {
	switch s {
	case Backlog, Researching, Ready, Queued, InProgress,
		InReview, ReviewPending, Completed, Failed, NeedsFollowup:
		return true
	}
	return false
}

// IsUserSettable reports whether users may set this status directly via the
// PATCH /api/v1/backlog endpoint.
//
// Whitelist rationale:
//   - Planning states (backlog, researching, ready) — normal editing.
//   - Terminal states (completed, failed, needs_followup) — the manual
//     override escape hatch. Users can override a run that auto-landed in
//     failed / needs_followup, or mark an item completed without a run
//     (e.g., research items that concluded in discussion). When the item
//     is currently in a review-gated status (in_review / review_pending),
//     update_patch.go rejects the PATCH and forces the review-decide flow
//     — this whitelist is the first line; review-state gating is the
//     second.
//
// Forbidden unconditionally via PATCH:
//   - queued, in_progress — execution owns these.
//   - in_review — the review agent sets this.
//   - review_pending — only set when an agent finishes review; exit via
//     review-decide to preserve the audit trail.
//
// New statuses added to the enum default to NOT user-settable — a deliberate
// choice must be made to include them here.
func IsUserSettable(s string) bool {
	switch s {
	case Backlog, Researching, Ready, Completed, Failed, NeedsFollowup:
		return true
	}
	return false
}

// IsTerminal reports whether s is a user-decided terminal status. Only the
// review-decide endpoint should transition an item into one of these.
func IsTerminal(s string) bool {
	switch s {
	case Completed, Failed, NeedsFollowup:
		return true
	}
	return false
}

// IsReview reports whether s is an active review-phase status (agent
// gathering evidence or awaiting user verdict).
func IsReview(s string) bool {
	switch s {
	case InReview, ReviewPending:
		return true
	}
	return false
}

// IsValidTransition is a centralized safety net over the backlog state
// machine. It validates that both endpoints of a proposed transition are
// recognized statuses — it intentionally does NOT encode authorization
// ("who may do this transition") or policy ("should we allow this
// revival"). Those concerns live in the callers:
//
//   - update_patch.go — user PATCH authorization (IsUserSettable +
//     review-gate guard).
//   - review_decide.go — review_pending → terminal with an audit trail.
//   - execution finalization/polling — writes only into `in_review` or
//     `review_pending`, never terminal directly.
//
// By keeping this predicate narrow (known → known, empty → known) we avoid
// accidentally blocking legitimate user overrides like the failed →
// completed manual-accept escape hatch. Callers opt in to stricter rules.
//
// Rules:
//   - An empty `from` is a new item: any valid `to` is allowed.
//   - Unknown statuses (either side) are rejected.
//   - Otherwise transitions between known statuses are allowed.
func IsValidTransition(from, to string) bool {
	if !IsValid(to) {
		return false
	}
	if from == "" {
		return true
	}
	return IsValid(from)
}
