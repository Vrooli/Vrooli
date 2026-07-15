package opsrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"swarm-manager/internal/agentops"
)

// Typed errors for the async operation lifecycle.
var (
	// ErrUnknownExecution is returned by CommitResult when no running operation
	// record matches the execution id (a result delivered for an execution the
	// workflow never started, or one already reaped).
	ErrUnknownExecution = errors.New("no running operation execution matches the id")
	// ErrUndeclaredOutcome is returned when a delivered outcome name is not one
	// the operation contract declares.
	ErrUndeclaredOutcome = errors.New("outcome is not declared by the operation contract")
	// ErrInvalidResult is returned when a delivered result omits a required
	// contract result field or violates a declared enum. It is fail-closed: no
	// workflow state is mutated, so the domain artifacts the round already wrote
	// are preserved and an operator (or a needs-attention commit) can recover.
	ErrInvalidResult = errors.New("delivered result violates the operation contract")
)

// CommitRequest delivers the terminal result of a live operation execution to
// the runner. It is produced by the domain's result-arrival seam (the operating-
// mode engine's round-refresh / resolution ladder) once the agent's round is in
// hand — never by an agent calling back directly.
type CommitRequest struct {
	Target      TargetRef
	ExecutionID string
	// Outcome is the contract outcome name the delivered round classifies to. The
	// classifier at the result-arrival seam derives it from the round handoff; the
	// runner validates it is a declared outcome and maps its disposition. A round
	// the classifier could not honestly resolve delivers the abstain outcome.
	Outcome string
	// DeliveredResult is the structured result payload the round produced,
	// validated against the operation contract's result schema before it is
	// recorded (unless the outcome abstains).
	DeliveredResult json.RawMessage
	// RequestedBy is a provenance label for who delivered the result.
	RequestedBy string
}

// CommitResult finalizes a running live operation execution: it validates the
// delivered outcome+result against the pinned operation contract, records the
// terminal outcome on the operation record and execution snapshot (CAS), and
// fires the policy transition through the dispatcher. It is idempotent — a
// second commit for the same execution is a no-op replay — and fail-closed: an
// undeclared outcome or an invalid result mutates no state, so the round's
// domain artifacts survive for recovery.
func (r *Runner) CommitResult(ctx context.Context, req CommitRequest) (OperationResult, error) {
	workflow, found, err := r.repo.Load(req.Target.Kind, req.Target.ID)
	if err != nil {
		return OperationResult{}, err
	}
	if !found {
		return OperationResult{}, fmt.Errorf("%w: no workflow for %s/%s", ErrUnknownExecution, req.Target.Kind, req.Target.ID)
	}

	opRecord, ok := findOperationByExecutionID(workflow, req.ExecutionID)
	if !ok {
		return OperationResult{}, fmt.Errorf("%w: execution %q", ErrUnknownExecution, req.ExecutionID)
	}

	snap, snapFound, err := r.executions.Load(req.Target.Kind, req.Target.ID, req.ExecutionID)
	if err != nil {
		return OperationResult{}, err
	}
	if !snapFound {
		return OperationResult{}, fmt.Errorf("%w: no snapshot for execution %q", ErrUnknownExecution, req.ExecutionID)
	}

	// Idempotent replay: a terminal record means the result was already
	// committed. Return it without recording or transitioning a second time.
	if opRecord.State != "running" {
		return OperationResult{
			WorkflowInstanceID: workflow.InstanceID, ExecutionID: req.ExecutionID,
			Provenance: snap.Provenance, ProvenanceDigest: opRecord.ProvenanceDigest,
			Outcome: opRecord.Outcome, Result: snap.Result,
			WorkflowState: workflow.State, Replayed: true,
		}, nil
	}

	// Resolve the pinned contract and validate the delivered outcome + result.
	lc, ok := r.catalog.Contract(opRecord.Operation, snap.Provenance.OperationVersion)
	if !ok {
		return OperationResult{}, fmt.Errorf("%w: %s@%s", ErrUnknownOperation, opRecord.Operation, snap.Provenance.OperationVersion)
	}
	disposition, ok := outcomeDisposition(lc.Contract, req.Outcome)
	if !ok {
		return OperationResult{}, fmt.Errorf("%w: %q for %s", ErrUndeclaredOutcome, req.Outcome, opRecord.Operation)
	}
	// A non-abstaining outcome must carry a contract-valid result. An abstaining
	// (needs-attention) outcome may carry a partial/absent result — that is the
	// whole point of the abstain path when the round's output was unparseable.
	if disposition != "abstain" {
		if err := validateDeliveredResult(lc.Contract, req.DeliveredResult); err != nil {
			return OperationResult{}, err
		}
	}

	outcome := ExecutionOutcome{
		Outcome:     req.Outcome,
		Disposition: Disposition(disposition),
		Result:      req.DeliveredResult,
	}

	// Record the terminal outcome on the operation record + snapshot (CAS).
	workflow, err = r.recordOutcome(workflow, req.ExecutionID, outcome)
	if err != nil {
		return OperationResult{}, err
	}
	snap.Outcome = outcome.Outcome
	snap.Result = outcome.Result
	if err := r.executions.SaveExecution(req.Target.Kind, req.Target.ID, req.ExecutionID, snap); err != nil {
		return OperationResult{}, fmt.Errorf("update execution snapshot: %w", err)
	}

	result := OperationResult{
		WorkflowInstanceID: workflow.InstanceID, ExecutionID: req.ExecutionID,
		Provenance: snap.Provenance, ProvenanceDigest: opRecord.ProvenanceDigest,
		Outcome: outcome.Outcome, Disposition: outcome.Disposition, Result: outcome.Result,
		WorkflowState: workflow.State,
	}
	if snap.PolicyID != "" {
		acted, err := r.applyTransition(ctx, req.Target, workflow, snap.PolicyID, outcome.Outcome, opRecord.Operation, req.ExecutionID, outcome.Result)
		if err != nil {
			return OperationResult{}, err
		}
		result.Action = acted.action
		result.WorkflowState = acted.state
	}
	return result, nil
}

// CancelExecution reaps a running operation execution after its live agent run
// was stopped cooperatively (the domain issues the StopRun; the round refresh
// then observes a canceled round, which is not a deliverable outcome so
// CommitResult never fires). Without this the operation record would linger
// "running" and the refresh driver would keep polling a stopped run forever. It
// marks the record canceled via compare-and-swap and is idempotent: a record
// already terminal (or a target/execution the workflow never tracked) is a no-op.
func (r *Runner) CancelExecution(ctx context.Context, target TargetRef, executionID string) error {
	workflow, found, err := r.repo.Load(target.Kind, target.ID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	op, ok := findOperationByExecutionID(workflow, executionID)
	if !ok || op.State != "running" {
		return nil
	}
	_, err = r.recordOutcome(workflow, executionID, ExecutionOutcome{Outcome: "canceled", Disposition: "canceled"})
	return err
}

// findOperationByExecutionID returns the operation record for an execution id.
func findOperationByExecutionID(w agentops.WorkflowInstance, executionID string) (agentops.OperationExecutionRecord, bool) {
	for _, op := range w.Operations {
		if op.ExecutionID == executionID {
			return op, true
		}
	}
	return agentops.OperationExecutionRecord{}, false
}

// outcomeDisposition returns the disposition a contract declares for an outcome
// name, and whether the outcome is declared at all.
func outcomeDisposition(contract agentops.OperationContract, outcome string) (string, bool) {
	for _, o := range contract.Outcomes {
		if o.Name == outcome {
			return o.Disposition, true
		}
	}
	return "", false
}

// validateDeliveredResult enforces the operation contract's result schema on a
// delivered payload: every required field is present and every declared enum is
// honored. It is deliberately generic — it reads the contract, never a named
// operation — so the runner stays free of per-operation branches.
func validateDeliveredResult(contract agentops.OperationContract, raw json.RawMessage) error {
	fields := contract.Result.Fields
	if len(fields) == 0 {
		return nil
	}
	var payload map[string]json.RawMessage
	if len(raw) == 0 {
		payload = map[string]json.RawMessage{}
	} else if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("%w: result is not a JSON object: %v", ErrInvalidResult, err)
	}
	for _, f := range fields {
		val, present := payload[f.Name]
		if f.Required && !present {
			return fmt.Errorf("%w: missing required field %q", ErrInvalidResult, f.Name)
		}
		if !present || len(f.Enum) == 0 {
			continue
		}
		var s string
		if err := json.Unmarshal(val, &s); err != nil {
			return fmt.Errorf("%w: field %q is not a string for enum check", ErrInvalidResult, f.Name)
		}
		if !enumContains(f.Enum, s) {
			return fmt.Errorf("%w: field %q value %q is not an allowed value", ErrInvalidResult, f.Name, s)
		}
	}
	return nil
}

func enumContains(enum []any, s string) bool {
	for _, e := range enum {
		if es, ok := e.(string); ok && es == s {
			return true
		}
	}
	return false
}
