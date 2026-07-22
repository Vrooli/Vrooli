package backlog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/planrepair"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// BindRepairedPlan is the domain-owned final authority step for plan repair.
// It rejects any changed backlog/workshop frontier before moving plan_ref.
func (h *Handler) BindRepairedPlan(ctx context.Context, record planrepair.Record, plan *sharedv1.Plan) error {
	kind, err := ParseBacklogKind(record.EntityKind)
	if err != nil {
		return err
	}
	item, err := h.store.LoadItem(kind, record.EntityName)
	if err != nil {
		return err
	}
	if record.EntityVersion != immutableBacklogSnapshotVersion(item) {
		return fmt.Errorf("backlog item changed after repair start")
	}
	if item.PlanRef == nil || (item.PlanRef.PlanID != record.PlanReference && item.PlanRef.Slug != record.PlanReference) {
		return fmt.Errorf("backlog plan reference changed after repair start")
	}
	if h.planClient == nil {
		return fmt.Errorf("plan manager client is not configured")
	}
	rendered, err := h.planClient.RenderMarkdown(ctx, item.PlanRef.PlanID, true)
	if err != nil {
		return fmt.Errorf("render current plan frontier: %w", err)
	}
	if repairFrontier(item.PlanRef.PlanID, rendered.Markdown) != record.FrontierDigest {
		return fmt.Errorf("plan frontier changed after repair start")
	}
	if strings.TrimSpace(plan.GetId()) == "" {
		return fmt.Errorf("canonical repaired plan id is required")
	}
	item.PlanRef = &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: plan.GetId(), Slug: plan.GetSlug(), Role: PlanRefRoleExecutionSpec}
	item.PlanAcceptance = nil
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	return h.store.SaveItem(item)
}
