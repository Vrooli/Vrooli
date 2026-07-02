package gates

import (
	"context"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// ExecutionLister is the narrow slice of the execution service the review
// source needs.
type ExecutionLister interface {
	List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error)
}

// ReviewSource enumerates KindReview gates from two sources:
//
//   - backlog items in review_pending — the user owns the terminal decision
//     (review-decide) before the item can complete;
//   - execution records in needs_review / needs_fixup — a run is parked on
//     human review.
//
// A run-level gate and an item-level gate can coexist for the same item;
// they are distinct decisions (run outcome vs item terminal status).
type ReviewSource struct {
	Store      ItemStore
	Executions ExecutionLister
}

// Name identifies the source in degradation logs.
func (s ReviewSource) Name() string { return "review" }

// reviewExecutionStatuses are run states parked on human review.
var reviewExecutionStatuses = map[execution.Status]bool{
	execution.StatusNeedsReview: true,
	execution.StatusNeedsFixup:  true,
}

// Enumerate implements Source.
func (s ReviewSource) Enumerate(ctx context.Context) ([]Gate, error) {
	items, err := s.Store.LoadAll(nil)
	if err != nil {
		return nil, err
	}
	dependents := directDependents(items)
	itemsByKey := make(map[string]backlog.BacklogItem, len(items))
	for _, item := range items {
		itemsByKey[itemKey(item)] = item
	}

	var out []Gate
	for _, item := range items {
		if backlog.IsArchived(item) || item.Status != backlog.StatusReviewPending {
			continue
		}
		key := itemKey(item)
		out = append(out, Gate{
			ID:             GateID(KindReview, "backlog", key),
			Kind:           KindReview,
			OwnerType:      "backlog",
			OwnerKind:      string(item.Kind),
			OwnerName:      item.Name,
			OwnerTitle:     itemTitle(item),
			Count:          1,
			Blocks:         dependents[key],
			DecidableSince: item.Updated,
		})
	}

	if s.Executions == nil {
		return out, nil
	}
	records, err := s.Executions.List(ctx, execution.ListFilters{})
	if err != nil {
		return nil, err
	}
	for _, rec := range records {
		if !reviewExecutionStatuses[rec.Status] {
			continue
		}
		key := rec.BacklogKind + "/" + rec.BacklogName
		title := key
		if item, ok := itemsByKey[key]; ok {
			if backlog.IsArchived(item) {
				continue
			}
			title = itemTitle(item)
		}
		out = append(out, Gate{
			ID:             GateID(KindReview, "execution", rec.ExecutionID),
			Kind:           KindReview,
			OwnerType:      "execution",
			OwnerKind:      rec.BacklogKind,
			OwnerName:      rec.BacklogName,
			OwnerTitle:     title,
			Count:          1,
			Blocks:         dependents[key],
			DecidableSince: rec.FinishedAt,
		})
	}
	return out, nil
}
