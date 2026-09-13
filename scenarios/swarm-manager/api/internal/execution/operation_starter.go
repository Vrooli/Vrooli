package execution

import (
	"fmt"
	"strings"
)

// executionPlanHandle resolves the plan-execution target handle for an item: the
// canonical execution_spec plan_ref's plan_id (or slug). It mirrors the
// plan_content.go guards so an item without a bound execution plan fails closed
// with the same guidance the render path gives, rather than starting work
// against an empty target.
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
