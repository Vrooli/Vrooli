package agentops

import (
	"encoding/json"
	"fmt"
)

// WorkflowState is the coordination-layer lifecycle state of a domain workflow
// instance. It never restates the domain entity's own status.
type WorkflowState string

const (
	WorkflowOpen             WorkflowState = "open"
	WorkflowRunning          WorkflowState = "running"
	WorkflowAwaitingDecision WorkflowState = "awaiting-decision"
	WorkflowBlocked          WorkflowState = "blocked"
	WorkflowTerminalComplete WorkflowState = "terminal-complete"
	WorkflowTerminalAbandon  WorkflowState = "terminal-abandoned"
	WorkflowTerminalFailed   WorkflowState = "terminal-failed"
)

// AllWorkflowStates is the canonical ordered state vocabulary.
var AllWorkflowStates = []WorkflowState{
	WorkflowOpen, WorkflowRunning, WorkflowAwaitingDecision, WorkflowBlocked,
	WorkflowTerminalComplete, WorkflowTerminalAbandon, WorkflowTerminalFailed,
}

func isValidWorkflowState(s WorkflowState) bool {
	return IsValidWorkflowState(s)
}

// IsValidWorkflowState reports whether s is a registered workflow state.
func IsValidWorkflowState(s WorkflowState) bool {
	for _, v := range AllWorkflowStates {
		if v == s {
			return true
		}
	}
	return false
}

// WorkflowInstance is the typed shape of workflow-instance.schema.json.
type WorkflowInstance struct {
	Kind            string                     `json:"kind"`
	SchemaVersion   string                     `json:"schema_version"`
	InstanceID      string                     `json:"instance_id"`
	Domain          WorkflowDomain             `json:"domain"`
	Strategy        *WorkflowStrategyRef       `json:"strategy,omitempty"`
	State           WorkflowState              `json:"state"`
	Operations      []OperationExecutionRecord `json:"operations,omitempty"`
	Decisions       []HumanDecision            `json:"decisions,omitempty"`
	Timers          []ScheduledIntent          `json:"timers,omitempty"`
	LegalActions    []ActionName               `json:"legal_actions,omitempty"`
	IdempotencyKeys []string                   `json:"idempotency_keys,omitempty"`
	Version         int                        `json:"version"`
}

type WorkflowDomain struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type WorkflowStrategyRef struct {
	Name string `json:"name"`
}

type OperationExecutionRecord struct {
	Operation        OperationID `json:"operation"`
	ExecutionID      string      `json:"execution_id"`
	IdempotencyKey   string      `json:"idempotency_key"`
	ProvenanceDigest string      `json:"provenance_digest"`
	State            string      `json:"state"`
	// RunID is the live agent-run id the operation's round is dispatched under,
	// set on the non-blocking live start. It correlates a delivered round back to
	// this operation record so CommitResult can be keyed from the round's run id.
	// Empty on the synchronous simulation path and until a run association exists.
	RunID   string `json:"run_id,omitempty"`
	Outcome string `json:"outcome,omitempty"`
}

type HumanDecision struct {
	Decision  string `json:"decision"`
	Actor     string `json:"actor,omitempty"`
	AtVersion int    `json:"at_version"`
	Note      string `json:"note,omitempty"`
}

// ScheduledIntent is a durable timer on a workflow. It fires EITHER a
// closed-vocabulary domain Action (a coordination no-op that runs the domain
// handler) OR — when Operation is set — an operation START through the runner's
// Invoke (e.g. "advance to the next workshop round"). Exactly one of Action /
// Operation carries the intent; the scheduler firer branches on Operation being
// non-empty. Modeling "start the next round" as the operation-start it is keeps
// the closed domain-action vocabulary unpolluted by a fake schedule-next-round
// action.
type ScheduledIntent struct {
	Intent string     `json:"intent"`
	Action ActionName `json:"action,omitempty"`
	// Operation, when set, routes the fire to the runner's Invoke of that
	// operation instead of dispatching Action. It must be a registered operation
	// id (validated semantically); Action is empty in that case.
	Operation OperationID `json:"operation,omitempty"`
	NotBefore string      `json:"not_before,omitempty"`
}

// ValidateWorkflowInstance validates a workflow-instance document against the
// schema and the semantic rules JSON Schema cannot express: the state is known,
// every correlated operation is registered, every scheduled intent's action and
// every listed legal action are registered domain actions, a strategy is only
// present on an initiative domain, and correlated idempotency keys are unique
// (so a replayed dispatch is a no-op, not a duplicate).
func ValidateWorkflowInstance(raw []byte) error {
	if err := ValidateDocument(SchemaWorkflowInstance, raw); err != nil {
		return err
	}
	var w WorkflowInstance
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("decode workflow instance: %w", err)
	}
	if !isValidWorkflowState(w.State) {
		return fmt.Errorf("workflow instance %q has unknown state %q", w.InstanceID, w.State)
	}
	if w.Strategy != nil && w.Domain.Kind != "initiative" {
		return fmt.Errorf("workflow instance %q: member-item strategy is only valid on an initiative domain (got %q)", w.InstanceID, w.Domain.Kind)
	}
	keys := map[string]bool{}
	for _, op := range w.Operations {
		if !IsValidOperationID(op.Operation) {
			return fmt.Errorf("workflow instance %q correlates unregistered operation %q", w.InstanceID, op.Operation)
		}
		if op.IdempotencyKey != "" {
			if keys[op.IdempotencyKey] {
				return fmt.Errorf("workflow instance %q reuses idempotency key %q across operations", w.InstanceID, op.IdempotencyKey)
			}
			keys[op.IdempotencyKey] = true
		}
	}
	for _, a := range w.LegalActions {
		if !IsRegisteredAction(a) {
			return fmt.Errorf("workflow instance %q lists unregistered legal action %q", w.InstanceID, a)
		}
	}
	for _, t := range w.Timers {
		if err := validateScheduledIntent(w.InstanceID, t); err != nil {
			return err
		}
	}
	return nil
}

// validateScheduledIntent enforces the "action XOR operation" rule JSON Schema
// cannot fully express: an intent fires either a registered domain action or a
// registered operation, never both and never neither.
func validateScheduledIntent(instanceID string, t ScheduledIntent) error {
	hasOp := t.Operation != ""
	hasAction := t.Action != ""
	switch {
	case hasOp && hasAction:
		return fmt.Errorf("workflow instance %q schedules an intent with both an action %q and an operation %q", instanceID, t.Action, t.Operation)
	case hasOp:
		if !IsValidOperationID(t.Operation) {
			return fmt.Errorf("workflow instance %q schedules unregistered operation %q", instanceID, t.Operation)
		}
	case hasAction:
		if !IsRegisteredAction(t.Action) {
			return fmt.Errorf("workflow instance %q schedules unregistered action %q", instanceID, t.Action)
		}
	default:
		return fmt.Errorf("workflow instance %q schedules an intent with neither an action nor an operation", instanceID)
	}
	return nil
}
