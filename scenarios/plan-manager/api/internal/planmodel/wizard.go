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
