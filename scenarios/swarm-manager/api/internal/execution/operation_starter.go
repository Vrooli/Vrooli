package execution

import (
	"context"
	"fmt"
	"strings"
)

const (
	targetKindPlanExecution = "plan-execution"
)

// OperationStarter launches a target-bound execution operation through the
// generic operation runner and returns the live run association. It is the
// cutover seam: execution start/retry/followup/fixup route their agent launch
// through this instead of a direct Agent Manager spawn, so every autonomous run is
// a declarative operation execution correlated by a durable operation workflow
// while the execution record keeps tracking the returned agent run id for
// status and finalization (transitional — slice C consolidates on the workflow;
// see plan-manager note d789cb50).
type OperationStarter interface {
	// CancelOperation reaps a running operation execution after its agent run was
	// stopped, so the durable workflow record does not linger "running" and the
	// refresh driver stops polling the stopped run. Best-effort and idempotent: an
	// unknown or already-terminal execution is a no-op.
	CancelOperation(ctx context.Context, req OperationCancelRequest) error
}

// OperationCancelRequest identifies the operation execution to reap by its target
// and execution id (both recorded on the execution Record at start).
type OperationCancelRequest struct {
	TargetKind  string
	TargetID    string
	ExecutionID string
}

// SetOperationStarter injects the operation runner seam after construction. The
// wiring layer (server bootstrap) calls this once both the execution service and
// the operation runner exist. When unset, the start paths that require it fail
// closed with a typed Unavailable error rather than falling back to a legacy
// direct spawn.
func (s *Service) SetOperationStarter(starter OperationStarter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operationStarter = starter
}

// executionPlanHandle resolves the plan-execution target handle for an item: the
// canonical execution_spec plan_ref's plan_id (or slug). It mirrors the
// plan_content.go guards so an item without a bound execution plan fails closed
// with the same guidance the render path gives, rather than starting an
// operation against an empty target.
func executionPlanHandle(item backlogItem) (string, error) {
	ref := item.PlanRef
	if ref == nil {
		return "", fmt.Errorf("backlog item %s/%s has no plan_ref; finalize the item through plan-manager before queueing", item.Kind, item.Name)
	}
	if strings.TrimSpace(ref.Provider) != planRefProviderPlanManager {
		return "", fmt.Errorf("backlog item %s/%s plan_ref.provider must be %q", item.Kind, item.Name, planRefProviderPlanManager)
	}
	if strings.TrimSpace(ref.Role) != planRefRoleExecutionSpec {
		return "", fmt.Errorf("backlog item %s/%s plan_ref.role must be %q", item.Kind, item.Name, planRefRoleExecutionSpec)
	}
	handle := strings.TrimSpace(ref.PlanID)
	if handle == "" {
		handle = strings.TrimSpace(ref.Slug)
	}
	if handle == "" {
		return "", fmt.Errorf("backlog item %s/%s plan_ref requires plan_id or slug", item.Kind, item.Name)
	}
	return handle, nil
}
