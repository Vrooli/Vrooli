package gates

import (
	"context"
	"sort"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/planclient"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// ItemStore is the narrow slice of backlog.Store the backlog-backed gate
// sources need.
type ItemStore interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// lockedStatuses are mid-execution statuses: the item is in flight, so its
// gates are not presented (mirrors the Command Post's locked skip).
var lockedStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusQueued:     true,
	backlog.StatusInProgress: true,
}

// terminalStatuses are finished-execution statuses. Terminal items surface
// as Done outcomes, not gates.
var terminalStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusCompleted: true,
	backlog.StatusFailed:    true,
}

func itemKey(item backlog.BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}

// directDependents builds a reverse depends_on map: item key -> keys of
// items that directly depend on it. Archived dependents are excluded —
// they no longer represent blocked work.
func directDependents(items []backlog.BacklogItem) map[string][]string {
	out := make(map[string][]string)
	for _, item := range items {
		if backlog.IsArchived(item) {
			continue
		}
		key := itemKey(item)
		for _, dep := range item.DependsOn {
			out[dep] = append(out[dep], key)
		}
	}
	for _, deps := range out {
		sort.Strings(deps)
	}
	return out
}

// gateEligible reports whether an item can carry a gate at all.
func gateEligible(item backlog.BacklogItem) bool {
	if backlog.IsArchived(item) {
		return false
	}
	if lockedStatuses[item.Status] || terminalStatuses[item.Status] {
		return false
	}
	return true
}

// WorkshopSource enumerates queueable items that still need an explicit plan
// acceptance. Its historical name and KindWorkshop are retained so clients can
// render old gate records, but it no longer derives readiness from workshop
// scores or a finalization pass.
type WorkshopSource struct {
	Store ItemStore
	Plans planclient.PlanReader
}

// Name identifies the source in degradation logs.
func (s WorkshopSource) Name() string { return "workshop" }

// queueableStatuses mirror the UI's QUEUEABLE_BACKLOG_STATUSES.
var queueableStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusBacklog:     true,
	backlog.StatusResearching: true,
	backlog.StatusReady:       true,
}

// Enumerate implements Source.
func (s WorkshopSource) Enumerate(ctx context.Context) ([]Gate, error) {
	items, err := s.Store.LoadAll(nil)
	if err != nil {
		return nil, err
	}
	dependents := directDependents(items)

	var out []Gate
	for _, item := range items {
		if !gateEligible(item) || !queueableStatuses[item.Status] {
			continue
		}
		if item.Kind == backlog.KindResearch {
			continue // research executes a conclusion, not a canonical plan.
		}
		suggested := "accept-plan"
		if item.PlanRef == nil {
			suggested = "author-plan"
		} else if s.Plans == nil {
			suggested = "validate-plan"
		} else {
			planID := strings.TrimSpace(item.PlanRef.PlanID)
			if planID == "" {
				planID = strings.TrimSpace(item.PlanRef.Slug)
			}
			plan, err := s.Plans.GetPlan(ctx, planID)
			if err != nil || plan == nil || strings.TrimSpace(plan.GetContentHash()) == "" || plan.GetStatus() == sharedv1.PlanStatus_PLAN_STATUS_DRAFT || plan.GetStatus() == sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED {
				suggested = "validate-plan"
			} else if backlog.PlanAcceptanceMatches(item, plan.GetContentHash()) {
				continue
			}
		}

		key := itemKey(item)
		gate := Gate{
			ID:         GateID(KindWorkshop, "backlog", key),
			Kind:       KindWorkshop,
			OwnerType:  "backlog",
			OwnerKind:  string(item.Kind),
			OwnerName:  item.Name,
			OwnerTitle: itemTitle(item),
			Count:      1,
			Blocks:     dependents[key],
			Suggested:  suggested,
		}
		out = append(out, gate)
	}
	return out, nil
}

func itemTitle(item backlog.BacklogItem) string {
	if t := strings.TrimSpace(item.Title); t != "" {
		return t
	}
	return item.Name
}
