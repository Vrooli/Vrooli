package agentops

import (
	"encoding/json"
	"fmt"
)

// ActionName is a registered domain action. The registry is CLOSED: a
// transition policy may select an action ONLY from this set, with typed
// parameters — it can never name a Go function, shell command, service, or file
// path (EXECUTION-MODES.md D3). Domain code owns what each action does and its
// invariants; data only selects which legal action fires.
type ActionName string

const (
	ActionSaveDecisions          ActionName = "save-decisions"
	ActionCommitWorkshopRound    ActionName = "commit-workshop-round"
	ActionStartClarification     ActionName = "start-clarification"
	ActionResolveClarification   ActionName = "resolve-clarification"
	ActionBindPlan               ActionName = "bind-plan"
	ActionQueuePlanExecution     ActionName = "queue-plan-execution"
	ActionStartExecution         ActionName = "start-execution"
	ActionCommitReviewRound      ActionName = "commit-review-round"
	ActionRequestRevision        ActionName = "request-revision"
	ActionRequestEvidence        ActionName = "request-evidence"
	ActionCompleteItem           ActionName = "complete-item"
	ActionFailItem               ActionName = "fail-item"
	ActionCreateFollowup         ActionName = "create-followup"
	ActionOpenReview             ActionName = "open-review"
	ActionEscalateNeedsAttention ActionName = "escalate-needs-attention"
	ActionMarkInitiativeReviewed ActionName = "mark-initiative-reviewed"
	ActionCommitInitiativeReview ActionName = "commit-initiative-review"
	ActionCommitExecutionRound   ActionName = "commit-execution-round"
)

// AllActionNames is the closed domain-action registry, in canonical order.
var AllActionNames = []ActionName{
	ActionSaveDecisions, ActionCommitWorkshopRound, ActionStartClarification,
	ActionResolveClarification, ActionBindPlan, ActionQueuePlanExecution,
	ActionStartExecution, ActionCommitReviewRound, ActionRequestRevision,
	ActionRequestEvidence, ActionCompleteItem, ActionFailItem,
	ActionCreateFollowup, ActionOpenReview, ActionEscalateNeedsAttention,
	ActionMarkInitiativeReviewed, ActionCommitInitiativeReview,
	ActionCommitExecutionRound,
}

// IsRegisteredAction reports whether name is in the closed action registry.
func IsRegisteredAction(name ActionName) bool {
	for _, a := range AllActionNames {
		if a == name {
			return true
		}
	}
	return false
}

// TransitionPolicy is the typed shape of transition-policy.schema.json.
type TransitionPolicy struct {
	Kind        string             `json:"kind"`
	ID          string             `json:"id"`
	Version     string             `json:"version"`
	DomainKind  string             `json:"domain_kind"`
	Transitions []PolicyTransition `json:"transitions"`
}

type PolicyTransition struct {
	FromState string         `json:"from_state"`
	OnOutcome string         `json:"on_outcome,omitempty"`
	Operation OperationID    `json:"operation,omitempty"`
	Action    ActionName     `json:"action"`
	Params    map[string]any `json:"params,omitempty"`
	ToState   string         `json:"to_state"`
}

// ValidateTransitionPolicy validates a policy against the schema and the
// semantic rules JSON Schema cannot express: every action is in the closed
// registry, every from/to state is a known workflow state, and — the security
// property — NO transition can reference anything outside the closed action
// vocabulary. Because the schema forbids any field other than the enumerated
// action/params/state keys (additionalProperties:false throughout) and params
// values are JSON scalars only, a policy provably cannot express arbitrary
// code, shell, service, or file behavior.
func ValidateTransitionPolicy(raw []byte) error {
	if err := ValidateDocument(SchemaTransitionPolicy, raw); err != nil {
		return err
	}
	var p TransitionPolicy
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("decode transition policy: %w", err)
	}
	for i, t := range p.Transitions {
		if !IsRegisteredAction(t.Action) {
			return fmt.Errorf("policy %q transition %d names unregistered action %q", p.ID, i, t.Action)
		}
		if !isValidWorkflowState(WorkflowState(t.FromState)) {
			return fmt.Errorf("policy %q transition %d has unknown from_state %q", p.ID, i, t.FromState)
		}
		if !isValidWorkflowState(WorkflowState(t.ToState)) {
			return fmt.Errorf("policy %q transition %d has unknown to_state %q", p.ID, i, t.ToState)
		}
		// An operation-specific rule must name a registered operation, so a policy
		// can never route on an operation the catalog does not declare.
		if t.Operation != "" && !IsValidOperationID(t.Operation) {
			return fmt.Errorf("policy %q transition %d names unregistered operation %q", p.ID, i, t.Operation)
		}
		if err := validateActionParams(t.Params); err != nil {
			return fmt.Errorf("policy %q transition %d action %q: %w", p.ID, i, t.Action, err)
		}
	}
	return nil
}

// validateActionParams enforces that params carry only scalar data — the
// belt-and-suspenders Go check backing the schema's scalar-only constraint, so
// even a Go-constructed policy cannot smuggle a nested structure (a place code
// or a path could hide) into a params map.
func validateActionParams(params map[string]any) error {
	for key, value := range params {
		switch value.(type) {
		case string, bool, float64, int, int64, json.Number, nil:
			continue
		default:
			return fmt.Errorf("param %q must be a JSON scalar, got %T (nested structures are forbidden so data can never encode behavior)", key, value)
		}
	}
	return nil
}
