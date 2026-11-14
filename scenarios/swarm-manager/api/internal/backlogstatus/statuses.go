// Package backlogstatus is the single source of truth for backlog item
// lifecycle statuses: the vocabulary, each status's classification, and the
// predicates over them. It has no dependencies on other internal packages so
// both `backlog` and `execution` can import it without creating a cycle
// (backlog imports execution to queue runs, and execution needs to write
// backlog statuses).
//
// # Adding a status
//
// Add one row to `definitions` below and run `make gen-status` (from the
// scenario root) to regenerate the UI's mirror. That is the whole procedure —
// every predicate, the valid-value set, and the TypeScript union all derive
// from the table. `TestEveryStatusIsClassified` fails if a row is incomplete,
// so a new status cannot slip through unclassified and land in some consumer's
// `default` branch.
//
// The table is deliberately the only place a status is enumerated. Predicates
// used to be hand-written switches, which meant adding one status required
// five separate edits *in this file alone* and gave no signal when a consumer
// elsewhere was missed.
package backlogstatus

// Status string values. Kept as untyped constants so callers in either package
// can interpolate them into typed or untyped strings freely.
const (
	Suggested     = "suggested"
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
	Dropped       = "dropped"
)

// Phase partitions the lifecycle. Every status belongs to exactly one phase,
// and the phases together cover the whole vocabulary — so a consumer that
// switches on phase cannot silently miss a newly added status the way a switch
// over individual statuses can.
type Phase string

const (
	// PhaseIntake covers items that have not been accepted as work yet.
	PhaseIntake Phase = "intake"
	// PhasePlanning covers items being shaped: refinable, and queueable.
	PhasePlanning Phase = "planning"
	// PhaseInFlight covers statuses owned by the execution system.
	PhaseInFlight Phase = "in_flight"
	// PhaseReview covers evidence gathering and the operator's pending verdict.
	PhaseReview Phase = "review"
	// PhaseTerminal covers settled items — the item is not coming back without
	// an explicit revival.
	PhaseTerminal Phase = "terminal"
)

// Definition is the full classification of one status. Every field is
// deliberately explicit rather than derived: adding a status should force a
// decision on each axis, not inherit a default that happens to compile.
type Definition struct {
	// Value is the wire/storage string.
	Value string
	// Label is the human-readable name for CLI and UI display.
	Label string
	// Phase is the lifecycle partition this status belongs to.
	Phase Phase
	// Resolved reports that nothing depending on this item is still waiting.
	// Dependency gates and readiness math key on this, NOT on "completed" —
	// see IsResolved.
	Resolved bool
	// UserSettable reports whether an operator may set this status directly
	// via PATCH. See IsUserSettable for the policy rationale.
	UserSettable bool
	// Doc explains what the status means and, where it is not obvious, why it
	// is classified the way it is. Surfaced in generated output.
	Doc string
}

// definitions is the SSOT table. Order is the lifecycle order and is preserved
// by All(), the generated TypeScript, and any UI that renders the vocabulary.
var definitions = []Definition{
	{
		Value: Suggested, Label: "Suggested", Phase: PhaseIntake,
		Resolved: false, UserSettable: false,
		Doc: "Proposed by the auto-filer and not yet accepted. Not user-settable: " +
			"operators accept a suggestion by moving it into the backlog, rather than " +
			"authoring the suggested state by hand.",
	},
	{
		Value: Backlog, Label: "Backlog", Phase: PhasePlanning,
		Resolved: false, UserSettable: true,
		Doc: "Accepted work that has not been shaped yet.",
	},
	{
		Value: Researching, Label: "Researching", Phase: PhasePlanning,
		Resolved: false, UserSettable: true,
		Doc: "Being investigated before it can be specified.",
	},
	{
		Value: Ready, Label: "Ready", Phase: PhasePlanning,
		Resolved: false, UserSettable: true,
		Doc: "Shaped and queueable.",
	},
	{
		Value: Queued, Label: "Queued", Phase: PhaseInFlight,
		Resolved: false, UserSettable: false,
		Doc: "Accepted into the execution queue. Owned by the execution system; " +
			"an operator PATCH must not fabricate it.",
	},
	{
		Value: InProgress, Label: "In Progress", Phase: PhaseInFlight,
		Resolved: false, UserSettable: false,
		Doc: "A run is active. Owned by the execution system.",
	},
	{
		Value: InReview, Label: "In Review", Phase: PhaseReview,
		Resolved: false, UserSettable: false,
		Doc: "A review round is actively gathering evidence. Set by the review " +
			"system. Invariant: a code path that cannot start or continue a round " +
			"must route to review_pending rather than leave an item here — the " +
			"review sweeper and the recover-review endpoint drain any item stranded " +
			"in_review with no live round, so this can never become a dead end.",
	},
	{
		Value: ReviewPending, Label: "Review Pending", Phase: PhaseReview,
		Resolved: false, UserSettable: false,
		Doc: "Review finished; awaiting the operator's verdict. Exit via " +
			"review-decide so the decision carries an audit trail.",
	},
	{
		Value: Completed, Label: "Completed", Phase: PhaseTerminal,
		Resolved: true, UserSettable: true,
		Doc: "The work was achieved. The only status that counts toward goal progress.",
	},
	{
		Value: Failed, Label: "Failed", Phase: PhaseTerminal,
		Resolved: false, UserSettable: true,
		Doc: "The work was attempted and did not land. NOT resolved: failed work " +
			"may still be retried, so its dependents are genuinely still blocked.",
	},
	{
		Value: NeedsFollowup, Label: "Needs Follow-up", Phase: PhaseTerminal,
		Resolved: false, UserSettable: true,
		Doc: "Delivered, but more work is needed. A live attention state, not an " +
			"archive dead end. NOT resolved: dependents are still waiting on the " +
			"remainder. Do not conflate with execution.StatusNeedsFixup, a run-level " +
			"state on a different enum.",
	},
	{
		Value: Dropped, Label: "Dropped", Phase: PhaseTerminal,
		Resolved: true, UserSettable: true,
		Doc: "Closed by operator decision: not going to be done, or no longer " +
			"relevant. Carries no verdict about the work, so unlike the other " +
			"terminals it needs no run or review round behind it and may be set " +
			"straight from a planning status. Resolved, because an item nobody will " +
			"ever finish must stop blocking its dependents.",
	},
}

// byValue indexes definitions for O(1) lookup.
var byValue = func() map[string]Definition {
	m := make(map[string]Definition, len(definitions))
	for _, d := range definitions {
		m[d.Value] = d
	}
	return m
}()

// Definitions returns the full classification table in lifecycle order.
func Definitions() []Definition {
	out := make([]Definition, len(definitions))
	copy(out, definitions)
	return out
}

// Lookup returns the definition for s, and whether s is a known status.
func Lookup(s string) (Definition, bool) {
	d, ok := byValue[s]
	return d, ok
}

// All returns every valid status value in lifecycle order.
func All() []string {
	out := make([]string, 0, len(definitions))
	for _, d := range definitions {
		out = append(out, d.Value)
	}
	return out
}

// InPhase returns every status in the given phase, in lifecycle order.
func InPhase(p Phase) []string {
	var out []string
	for _, d := range definitions {
		if d.Phase == p {
			out = append(out, d.Value)
		}
	}
	return out
}

// IsValid reports whether s is one of the known status strings.
func IsValid(s string) bool {
	_, ok := byValue[s]
	return ok
}

// IsUserSettable reports whether users may set this status directly via the
// PATCH /api/v1/backlog endpoint.
//
// Whitelist rationale:
//   - Planning states (backlog, researching, ready) — normal editing.
//   - Terminal states (completed, failed, needs_followup, dropped) — the manual
//     override escape hatch. Users can override a run that auto-landed in
//     failed / needs_followup, or mark an item completed without a run.
//     `dropped` is settable straight from a planning state and needs no run at
//     all: deciding not to do something is an operator judgment, not a run
//     outcome. When the item is currently in a review-gated status (in_review /
//     review_pending), update_patch.go rejects the PATCH and forces the
//     review-decide flow — this whitelist is the first line; review-state
//     gating is the second.
//
// Forbidden unconditionally via PATCH:
//   - queued, in_progress — execution owns these.
//   - in_review — the review agent sets this.
//   - review_pending — only set when an agent finishes review; exit via
//     review-decide to preserve the audit trail.
//
// New statuses default to NOT user-settable — the table requires a deliberate
// choice to opt in.
func IsUserSettable(s string) bool {
	return byValue[s].UserSettable
}

// IsTerminal reports whether s is a settled status. Reaching completed /
// failed / needs_followup goes through the review-decide endpoint so the
// verdict is audited; `dropped` is the exception — it carries no verdict about
// the work, so it is reachable by a direct PATCH from any non-review status.
func IsTerminal(s string) bool {
	return byValue[s].Phase == PhaseTerminal
}

// IsResolved reports whether s means "this item will not be worked again", and
// therefore that anything depending on it is no longer waiting.
//
// This is the predicate the dependency gate must use — NOT `== Completed`.
// Using completion alone strands every dependent of an abandoned item in
// `blocked` forever, because an item nobody will ever finish never satisfies
// its dependents. `dropped` resolves a dependency without claiming the work was
// achieved; goal progress reporting keeps the two apart (see goals.ComputeScope,
// which counts only Completed toward progress and removes Dropped from the
// denominator).
//
// Failure states are deliberately NOT resolved: a failed or follow-up item is
// still live work whose dependents genuinely are blocked.
func IsResolved(s string) bool {
	return byValue[s].Resolved
}

// IsReview reports whether s is an active review-phase status (agent gathering
// evidence or awaiting user verdict).
func IsReview(s string) bool {
	return byValue[s].Phase == PhaseReview
}

// IsInFlight reports whether the execution system owns the item's status.
// Operators cannot write these directly.
func IsInFlight(s string) bool {
	return byValue[s].Phase == PhaseInFlight
}

// IsPlanning reports whether the item is being shaped and is queueable.
func IsPlanning(s string) bool {
	return byValue[s].Phase == PhasePlanning
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
