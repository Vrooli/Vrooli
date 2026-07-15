package opsrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"swarm-manager/internal/agentops"
)

// ActionContext is what a domain action handler receives. Handlers own the real
// domain mutation and its invariants (complete an item, bind a plan); the
// dispatcher only decides — from data — which legal action fires and commits the
// coordination-state change atomically around the handler.
type ActionContext struct {
	Target    TargetRef
	Workflow  agentops.WorkflowInstance
	Action    agentops.ActionName
	Params    map[string]any
	Outcome   string
	Operation agentops.OperationID
	// ExecutionID is the operation execution that produced this transition, so a
	// handler can attach operation-execution provenance to the domain artifact it
	// writes.
	ExecutionID string
	// Result is the validated operation result the completing execution produced
	// (the enriched declared output the driver/CommitResult already validated
	// against the operation contract). A handler that materializes a domain
	// artifact from the run's output — e.g. commit-workshop-round writing the
	// round file — reads it here. Nil for an abstaining outcome that carried no
	// result, or a transition fired from a state with no producing execution.
	Result json.RawMessage
}

// ActionHandler performs the domain mutation for one registered action. A
// handler must be idempotent at the domain layer where it can be, because the
// dispatcher already deduplicates by idempotency key at the coordination layer.
type ActionHandler func(ctx context.Context, ac ActionContext) error

// ActionRegistry maps registered domain-action names to handlers. It is CLOSED:
// only names in agentops.AllActionNames can be registered, and dispatch of an
// unregistered name is rejected — data can never invoke code outside this set.
type ActionRegistry struct {
	mu       sync.RWMutex
	handlers map[agentops.ActionName]ActionHandler
}

// NewActionRegistry returns a registry with a no-op handler pre-registered for
// every action in the closed registry, so the coordination layer is exercisable
// end to end before the real domain handlers (Phases 5-6) are wired. Callers
// override specific actions with Register.
func NewActionRegistry() *ActionRegistry {
	r := &ActionRegistry{handlers: map[agentops.ActionName]ActionHandler{}}
	for _, name := range agentops.AllActionNames {
		r.handlers[name] = func(context.Context, ActionContext) error { return nil }
	}
	return r
}

// Register installs a handler for a registered action, replacing the default.
// Registering an unregistered action name is a programming error and panics —
// the closed vocabulary is compiled, not data.
func (r *ActionRegistry) Register(name agentops.ActionName, h ActionHandler) {
	if !agentops.IsRegisteredAction(name) {
		panic(fmt.Sprintf("opsrunner: cannot register unregistered action %q", name))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = h
}

func (r *ActionRegistry) handler(name agentops.ActionName) (ActionHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// ErrNoTransition is returned by EvaluateTransition when the policy has no rule
// for (state, outcome). It is not an error condition for the caller — it means
// the workflow simply waits — so the runner treats it as "no action fired".
var ErrNoTransition = errors.New("no transition matches the workflow state and outcome")

// SelectedTransition is the single policy rule chosen for a (state, outcome).
type SelectedTransition struct {
	Action  agentops.ActionName
	Params  map[string]any
	ToState agentops.WorkflowState
}

// EvaluateTransition selects the transition a policy fires for the current
// workflow state, the completing operation, and its outcome. It is deterministic
// and fails closed: two rules matching at the SAME specificity is an authoring
// error, not a pick-one situation.
//
// A rule may pin an operation and/or an outcome; a rule with neither is the
// state's unconditional fallback. Specificity is scored so the MOST specific
// matching rule wins: an operation-specific rule beats a generic (operation-less)
// one, and within the same operation-specificity an outcome-specific rule beats a
// wildcard-outcome one. The workflow instance exists to correlate operations, so
// routing on which operation completed is honest structure — e.g. one "running +
// completed" state can send workshop-round to open-review and workshop-finalize to
// bind-plan without conflating operation identity into the outcome vocabulary.
func EvaluateTransition(policy agentops.TransitionPolicy, state agentops.WorkflowState, operation agentops.OperationID, outcome string) (SelectedTransition, error) {
	var chosen *agentops.PolicyTransition
	bestScore := -1
	ambiguous := false
	for i := range policy.Transitions {
		t := policy.Transitions[i]
		if agentops.WorkflowState(t.FromState) != state {
			continue
		}
		// An operation-pinned rule matches only its operation; an outcome-pinned
		// rule matches only its outcome. An empty field matches anything.
		if t.Operation != "" && t.Operation != operation {
			continue
		}
		if t.OnOutcome != "" && t.OnOutcome != outcome {
			continue
		}
		score := 0
		if t.Operation != "" {
			score += 2
		}
		if t.OnOutcome != "" {
			score++
		}
		switch {
		case score > bestScore:
			bestScore = score
			chosen = &policy.Transitions[i]
			ambiguous = false
		case score == bestScore:
			ambiguous = true
		}
	}
	if chosen == nil {
		return SelectedTransition{}, ErrNoTransition
	}
	if ambiguous {
		return SelectedTransition{}, fmt.Errorf("policy %q is ambiguous: two rules match state %q operation %q outcome %q at the same specificity", policy.ID, state, operation, outcome)
	}
	return SelectedTransition{Action: chosen.Action, Params: chosen.Params, ToState: agentops.WorkflowState(chosen.ToState)}, nil
}

// Dispatcher commits a selected transition: it validates the action is
// registered and legal from the current state, deduplicates by idempotency key,
// invokes the domain handler, and atomically advances the workflow's
// coordination state through the repository's compare-and-swap. A crash between
// the handler and the commit is recovered by the idempotency key: re-dispatching
// the same key is a no-op.
type Dispatcher struct {
	registry *ActionRegistry
	repo     *WorkflowRepo
}

// NewDispatcher constructs a dispatcher.
func NewDispatcher(registry *ActionRegistry, repo *WorkflowRepo) *Dispatcher {
	return &Dispatcher{registry: registry, repo: repo}
}

// DispatchDelivery carries the producing execution's identity and validated
// result into a dispatch, so the domain handler can materialize an artifact from
// the run's output and stamp its operation-execution provenance. It is the zero
// value for a transition fired without a producing execution.
type DispatchDelivery struct {
	ExecutionID string
	Result      json.RawMessage
}

// DispatchResult reports what a dispatch did.
type DispatchResult struct {
	Applied  bool
	Replayed bool
	State    agentops.WorkflowState
}

// Dispatch validates and commits a transition against the loaded workflow. The
// caller passes the workflow it read (prevVersion) so the commit is a
// compare-and-swap; a concurrent mutation makes Commit return ErrWorkflowConflict
// and Dispatch surfaces it rather than clobbering. idempotencyKey deduplicates:
// if the workflow already consumed it, Dispatch is a no-op reporting Replayed.
func (d *Dispatcher) Dispatch(ctx context.Context, target TargetRef, w agentops.WorkflowInstance, sel SelectedTransition, outcome string, operation agentops.OperationID, idempotencyKey string, delivery DispatchDelivery) (DispatchResult, error) {
	if idempotencyKey == "" {
		return DispatchResult{}, errors.New("dispatch requires an idempotency key")
	}
	if HasIdempotencyKey(w, idempotencyKey) {
		return DispatchResult{Replayed: true, State: w.State}, nil
	}
	if !agentops.IsRegisteredAction(sel.Action) {
		return DispatchResult{}, fmt.Errorf("refusing to dispatch unregistered action %q", sel.Action)
	}
	handler, ok := d.registry.handler(sel.Action)
	if !ok {
		return DispatchResult{}, fmt.Errorf("no handler registered for action %q", sel.Action)
	}
	if !agentops.IsValidWorkflowState(sel.ToState) {
		return DispatchResult{}, fmt.Errorf("transition targets unknown workflow state %q", sel.ToState)
	}

	// Invoke the domain handler BEFORE committing the coordination state. If the
	// handler fails, no state change is persisted and the idempotency key is not
	// consumed, so a retry re-runs cleanly.
	ac := ActionContext{
		Target: target, Workflow: w, Action: sel.Action, Params: sel.Params,
		Outcome: outcome, Operation: operation,
		ExecutionID: delivery.ExecutionID, Result: delivery.Result,
	}
	if err := handler(ctx, ac); err != nil {
		return DispatchResult{}, fmt.Errorf("action %q handler failed: %w", sel.Action, err)
	}

	prevVersion := w.Version
	next := cloneWorkflow(w)
	next.State = sel.ToState
	next.Version = prevVersion + 1
	next.IdempotencyKeys = appendUnique(next.IdempotencyKeys, idempotencyKey)
	next.LegalActions = legalActionsFrom(next.State)
	if err := d.repo.Commit(prevVersion, next); err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Applied: true, State: next.State}, nil
}

// legalActionsFrom returns the registered actions an operator could next take
// from a state (a UI snapshot; the policy remains the SSOT). Terminal states
// have none.
func legalActionsFrom(state agentops.WorkflowState) []agentops.ActionName {
	switch state {
	case agentops.WorkflowTerminalComplete, agentops.WorkflowTerminalAbandon, agentops.WorkflowTerminalFailed:
		return nil
	default:
		return nil
	}
}

func appendUnique(xs []string, x string) []string {
	for _, existing := range xs {
		if existing == x {
			return xs
		}
	}
	return append(xs, x)
}

// cloneWorkflow deep-copies the mutable slices of a workflow so a caller's
// in-memory copy is never aliased by a committed one.
func cloneWorkflow(w agentops.WorkflowInstance) agentops.WorkflowInstance {
	out := w
	out.Operations = append([]agentops.OperationExecutionRecord(nil), w.Operations...)
	out.Decisions = append([]agentops.HumanDecision(nil), w.Decisions...)
	out.Timers = append([]agentops.ScheduledIntent(nil), w.Timers...)
	out.LegalActions = append([]agentops.ActionName(nil), w.LegalActions...)
	out.IdempotencyKeys = append([]string(nil), w.IdempotencyKeys...)
	if w.Strategy != nil {
		s := *w.Strategy
		out.Strategy = &s
	}
	return out
}
