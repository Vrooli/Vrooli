package execution

import (
	"context"
	"fmt"
	"strings"
)

// Operation identities and pins the execution service launches its autonomous
// runs as. These mirror the agentops operation vocabulary + the pinned
// system-default binding version; kept as local constants so the execution
// package does not import the agentops catalog (the adapter in the api package
// bridges to it).
const (
	operationExecutionRun      = "execution-run"
	operationExecutionRetry    = "execution-retry"
	operationExecutionFollowup = "execution-followup"
	operationExecutionFixup    = "execution-fixup"
	operationResearchConclude  = "research-conclude"
	operationSpecSync          = "spec-sync"
	operationVersionPinned     = "1.0.0"
	targetKindPlanExecution    = "plan-execution"
	targetKindBacklogItem      = "backlog-item"
	targetKindScenario         = "scenario"
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
	StartOperation(ctx context.Context, req OperationStartRequest) (OperationStartResult, error)
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

// OperationStartRequest names the operation, target, and caller context for one
// operation start. CallerInputs are keyed by the operation contract's declared
// caller-input names (e.g. OPERATOR_NOTE, RETRY_NOTE); empty/absent values are
// omitted by the caller so an optional input never appears.
type OperationStartRequest struct {
	Operation        string
	OperationVersion string
	TargetKind       string
	TargetID         string
	CallerInputs     map[string]string
	IdempotencyKey   string
	RequestedBy      string
}

// OperationStartResult is the live run association a non-blocking operation
// start returns: the agent run id (stored on the execution record so polling /
// finalization / cancel keep working) plus the durable operation-execution
// correlation ids.
type OperationStartResult struct {
	RunID       string
	WorkflowID  string
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
