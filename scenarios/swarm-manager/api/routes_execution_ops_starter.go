package main

import (
	"context"
	"strings"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opsrunner"
)

// executionOperationStarter adapts the generic operation runner to the execution
// service's OperationStarter seam: it invokes the named operation against the
// target through opsrunner.Runner.Invoke and returns the live run association
// (agent run id + durable operation-execution correlation ids). It is the api-
// package bridge that keeps the execution package free of an agentops/opsrunner
// import.
type executionOperationStarter struct {
	runner *opsrunner.Runner
}

// StartOperation invokes the operation and returns the live run association. A
// live (non-simulated) Invoke returns immediately with a StartHandle; the
// operation runs until its round is delivered to CommitResult by the completion
// bridge.
func (e *executionOperationStarter) StartOperation(ctx context.Context, req execution.OperationStartRequest) (execution.OperationStartResult, error) {
	var callerInputs map[string]any
	for k, v := range req.CallerInputs {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if callerInputs == nil {
			callerInputs = map[string]any{}
		}
		callerInputs[k] = v
	}
	res, err := e.runner.Invoke(ctx, opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetKind(req.TargetKind), ID: req.TargetID},
		Operation:        agentops.OperationID(req.Operation),
		OperationVersion: req.OperationVersion,
		CallerInputs:     callerInputs,
		IdempotencyKey:   req.IdempotencyKey,
		RequestedBy:      req.RequestedBy,
	})
	if err != nil {
		return execution.OperationStartResult{}, err
	}
	out := execution.OperationStartResult{
		WorkflowID:  res.WorkflowInstanceID,
		ExecutionID: res.ExecutionID,
	}
	if res.StartHandle != nil {
		out.RunID = res.StartHandle.RunID
	}
	return out, nil
}

// CancelOperation reaps a running operation execution in the durable workflow
// after the execution service stopped its agent run.
func (e *executionOperationStarter) CancelOperation(ctx context.Context, req execution.OperationCancelRequest) error {
	return e.runner.CancelExecution(ctx, opsrunner.TargetRef{
		Kind: agentops.TargetKind(req.TargetKind),
		ID:   req.TargetID,
	}, req.ExecutionID)
}

// executionPlanContainmentResolver resolves the write-scope containment of the
// backlog item that owns a plan-execution handle, so the plan-execution target
// adapter inherits exactly the acceptance scope the item's legacy execution
// passed. It scans the backlog store for the item whose canonical execution_spec
// plan_ref matches the handle (plan id or slug); execution starts are governed
// and infrequent, so the scan is not a hot path.
type executionPlanContainmentResolver struct {
	store backlog.Store
}

// ContainmentForPlan finds the item owning the plan handle and projects its
// acceptance scope. found=false when no item owns the handle (a plan-first run
// with no backing item), leaving the spawn unconstrained.
func (r *executionPlanContainmentResolver) ContainmentForPlan(handle string) (operatingmode.ContainmentScope, bool, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" || r.store == nil {
		return operatingmode.ContainmentScope{}, false, nil
	}
	items, err := r.store.LoadAll(nil)
	if err != nil {
		return operatingmode.ContainmentScope{}, false, err
	}
	for _, it := range items {
		ref := it.PlanRef
		if ref == nil || strings.TrimSpace(ref.Role) != backlog.PlanRefRoleExecutionSpec {
			continue
		}
		if strings.TrimSpace(ref.PlanID) != handle && strings.TrimSpace(ref.Slug) != handle {
			continue
		}
		return operatingmode.ContainmentScope{
			AcceptanceAllow: it.AcceptanceAllow,
			AcceptanceDeny:  it.AcceptanceDeny,
			Creates:         it.Creates,
		}, true, nil
	}
	return operatingmode.ContainmentScope{}, false, nil
}
