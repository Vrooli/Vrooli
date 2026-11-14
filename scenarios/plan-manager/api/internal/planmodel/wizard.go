package planmodel

// =============================================================================
// GUIDED-FLOW STEERING MODEL
// =============================================================================
//
// Guided flows in authoring, execution, and the execution log all return the
// same API-owned steering shape. Workflow policy stays in each domain; these
// types keep the output contract stable across API, CLI, and UI surfaces.
// =============================================================================

// NextActionKind classifies how strongly a guided flow recommends an action.
type NextActionKind string

const (
	NextActionRecommended NextActionKind = "recommended"
	NextActionAlternative NextActionKind = "alternative"
	NextActionOptional    NextActionKind = "optional"
	NextActionRecovery    NextActionKind = "recovery"
)

// NextAction is one API-owned concrete action for the current guided step. Argv
// is canonical; CLI/UI may format it, but they must not decide workflow order.
type NextAction struct {
	ID                 string
	Kind               NextActionKind
	Label              string
	Reason             string
	Argv               []string
	ContentPlaceholder string
	BlockedBy          []string
}

// ChecklistState is the live status of one checklist requirement.
type ChecklistState string

const (
	ChecklistFilled    ChecklistState = "filled"
	ChecklistMissing   ChecklistState = "missing"
	ChecklistViolation ChecklistState = "violation"
)

// ChecklistItem is one requirement in a guided step's full-disclosure
// checklist: the COMPLETE requirement set for the touched scope with live
// status, so an agent never has to submit a field just to learn the next one.
type ChecklistItem struct {
	// Key is the stable requirement key, e.g. "steps", "purpose", "phase:2".
	Key   string
	Label string
	State ChecklistState
	// Detail is a short qualifier — a parse summary for filled items, the
	// violation message for violations, or the accepted escape.
	Detail string
}

// GuidedStep is deterministic just-in-time steering for a guided flow.
type GuidedStep struct {
	StepKind       string
	Title          string
	Summary        string
	Instructions   []string
	RequiredInputs []string
	Examples       []string
	CommonMistakes []string
	NextActions    []NextAction
	// Checklist is the full-disclosure requirement set for the touched scope —
	// a superset of RequiredInputs (which stays for compatibility).
	Checklist []ChecklistItem
}

// OnlyRecommended returns a copy of step with only its recommended action. If a
// step has no recommended action, it falls back to the first action to preserve a
// single-action continue loop without inventing policy.
func OnlyRecommended(step GuidedStep) GuidedStep {
	for _, action := range step.NextActions {
		if action.Kind == NextActionRecommended {
			step.NextActions = []NextAction{action}
			return step
		}
	}
	if len(step.NextActions) > 1 {
		step.NextActions = step.NextActions[:1]
	}
	return step
}
