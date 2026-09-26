package orchestration

import (
	"context"
	"fmt"

	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"

	"github.com/google/uuid"
)

// WithMaintenanceGate installs the single serving owner's durable admission
// fence. Production installs it before any recovery worker or HTTP route starts.
func WithMaintenanceGate(gate *maintenance.Gate) Option {
	return func(o *Orchestrator) {
		if gate != nil {
			o.maintenanceGate = gate
		}
	}
}

type admittedWorkflowKey struct{}
type admittedWorkflow struct {
	owner       *Orchestrator
	executionID uuid.UUID
}

func (o *Orchestrator) admitMaintenance(ctx context.Context) (func(), error) {
	if o.hasMaintenanceAdmission(ctx) {
		return func() {}, nil
	}
	if o.maintenanceGate == nil {
		return func() {}, nil
	}
	if admitted, ok := ctx.Value(admittedWorkflowKey{}).(admittedWorkflow); ok && admitted.owner == o {
		x, err := o.workflowExecutions.Get(ctx, admitted.executionID)
		if err != nil {
			return nil, err
		}
		if x != nil && !x.Status.Terminal() {
			return func() {}, nil
		}
		return nil, domain.NewStateError("WorkflowExecution", "terminal", "dispatch child", "admitted workflow is no longer active")
	}
	release, err := o.maintenanceGate.Admit(ctx)
	if err != nil {
		return nil, domain.RefuseBeforeEffects(fmt.Errorf("%w: %w", domain.NewStateError("admission", "maintenance", "admit", "owner admission is unavailable; inspect maintenance status"), err))
	}
	return release, nil
}

// Only the interpreter's private launchers can establish this context. Public
// Force, run environment, workload labels and caller-supplied parent IDs cannot
// bypass admission. Verify the durable parent and attempt before marking it.
func (o *Orchestrator) admittedWorkflowContext(ctx context.Context, executionID, attemptID uuid.UUID) (context.Context, error) {
	if o.maintenanceGate == nil {
		return ctx, nil
	}
	if o.workflowExecutions == nil {
		return nil, fmt.Errorf("admitted workflow repository unavailable")
	}
	x, err := o.workflowExecutions.Get(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if x == nil || x.Status.Terminal() {
		return nil, domain.NewStateError("WorkflowExecution", "not_active", "dispatch child", "durable admitted parent is required")
	}
	attempts, err := o.workflowExecutions.ListAttempts(ctx, executionID)
	if err != nil {
		return nil, err
	}
	for _, attempt := range attempts {
		if attempt.ID == attemptID && attempt.ExecutionID == executionID {
			return context.WithValue(ctx, admittedWorkflowKey{}, admittedWorkflow{owner: o, executionID: executionID}), nil
		}
	}
	return nil, domain.NewStateError("WorkflowNodeAttempt", "missing", "dispatch child", "durable parent attempt is required")
}
