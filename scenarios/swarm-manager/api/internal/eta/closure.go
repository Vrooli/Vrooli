package eta

import (
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/depgraph"
)

// BuildClosureInput maps backlog items into the canonical ETA rollup input.
// The whole item set is treated as the implicit closure; dependency edges
// outside that set are dropped. Done mirrors completed/archived items, and
// Gated marks pending items that are not currently wave 0.
func BuildClosureInput(items []backlog.BacklogItem) GoalClosureInput {
	itemsByKey := make(map[string]backlog.BacklogItem, len(items))
	satisfied := make(map[string]bool, len(items))
	graphMap := make(map[string][]string, len(items))
	for _, item := range items {
		key := itemKey(item)
		itemsByKey[key] = item
		graphMap[key] = item.DependsOn
		if backlog.IsArchived(item) || item.Status == backlog.StatusCompleted {
			satisfied[key] = true
		}
	}

	waves := depgraph.Waves(graphMap, func(k string) bool { return satisfied[k] })
	in := GoalClosureInput{Deps: make(map[string][]string, len(items))}
	for _, item := range items {
		key := itemKey(item)
		done := satisfied[key]
		gated := !done && waves.Waves[key] != 0
		in.Items = append(in.Items, ClosureItem{
			Ref:         key,
			EffortClass: NormalizeEffort(item.Effort),
			Done:        done,
			Gated:       gated,
		})
		var deps []string
		for _, d := range item.DependsOn {
			if _, ok := itemsByKey[d]; ok {
				deps = append(deps, d)
			}
		}
		in.Deps[key] = deps
	}
	return in
}

func itemKey(item backlog.BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}
