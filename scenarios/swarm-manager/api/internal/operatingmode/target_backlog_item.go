package operatingmode

import (
	"context"
	"fmt"
	"strings"
)

// Stable prompt aliases for backlog-item target inputs. Their logical ids and
// capability sources are authored in each mode input_contract (Phase 4).
const (
	ReadItemTitle       = "ITEM_TITLE"
	ReadItemDescription = "ITEM_DESCRIPTION"
	ReadItemStatus      = "ITEM_STATUS"
	ReadItemSpec        = "ITEM_SPEC"
	ReadItemPlanRef     = "ITEM_PLAN_REF"
)

// backlogItemTargetAdapter is the target adapter for a single swarm-manager
// backlog item. It keeps the shared engine target-agnostic: the engine consumes
// only the generic RunContext, and this adapter supplies the item-specific
// identity, ownership key, and typed capability projection. Item id is the
// "kind/name" ref (e.g. "fix/flaky-test").
type backlogItemTargetAdapter struct{}

func (backlogItemTargetAdapter) Kind() TargetKind { return TargetBacklogItem }

func (backlogItemTargetAdapter) Resolve(_ context.Context, s *Service, _ Definition, _ PhaseDefinition, ref string) (TargetInstance, error) {
	itemRef := strings.TrimSpace(ref)
	if itemRef == "" {
		return TargetInstance{}, fmt.Errorf("mode targets a backlog item: a kind/name item ref is required")
	}
	// Prefer the rich reader (spec document + plan_ref) when wired; degrade to
	// the coarse snapshot otherwise rather than resolving a half-built target.
	if s != nil && s.backlogTargets != nil {
		bt, err := s.backlogTargets.LoadBacklogItemTarget(itemRef)
		if err != nil {
			return TargetInstance{}, err
		}
		bt.Ref = itemRef
		return TargetInstance{
			Kind: TargetBacklogItem, ID: itemRef,
			Title: bt.Title, Description: bt.Description, Item: bt,
		}, nil
	}
	kind, name, ok := strings.Cut(itemRef, "/")
	if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(name) == "" {
		return TargetInstance{}, fmt.Errorf("backlog-item ref %q must be kind/name", itemRef)
	}
	if s == nil || s.backlog == nil {
		return TargetInstance{}, fmt.Errorf("backlog-item target %q cannot be resolved: no backlog reader is wired", itemRef)
	}
	snap, err := s.backlog.LoadBacklogItem(kind, name)
	if err != nil {
		return TargetInstance{}, err
	}
	item := BacklogItemTarget{Ref: itemRef, Title: snap.Title, Status: snap.Status}
	return TargetInstance{Kind: TargetBacklogItem, ID: itemRef, Title: snap.Title, Item: item}, nil
}

func (backlogItemTargetAdapter) Values(t TargetInstance) map[string]any {
	return map[string]any{
		"target.item_title":       t.Item.Title,
		"target.item_description": t.Item.Description,
		"target.item_status":      t.Item.Status,
		"target.item_spec":        t.Item.SpecDocument,
		"target.item_plan_ref":    backlogPlanRefValue(t.Item.PlanRef),
	}
}

// OwnershipKey gives backlog-item targets a distinct lock namespace so an item
// run never collides with an initiative of the same name and never requires an
// initiative.
func (backlogItemTargetAdapter) OwnershipKey(id string) string {
	return "item--" + sanitizeOwnershipToken(id)
}

func backlogPlanRefValue(ref *PlanRef) any {
	if ref == nil {
		return nil
	}
	return ref
}
