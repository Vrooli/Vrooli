package eta

import (
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/initiatives"
)

// BuildClosureInput maps backlog items into the canonical ETA rollup input.
// The whole item set is treated as the implicit closure; dependency edges
// outside that set are dropped. Done mirrors completed/archived items plus the
// initiative gate, and Gated marks pending items that are not currently wave 0.
func BuildClosureInput(items []backlog.BacklogItem, inits []initiatives.Initiative) GoalClosureInput {
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

	if len(inits) > 0 {
		initiativeItems := make(map[string][]string, len(inits))
		initiativeDeps := make(map[string][]string, len(inits))
		for _, ini := range inits {
			initiativeItems[ini.Name] = ini.Items
			initiativeDeps[ini.Name] = ini.DependsOn
		}
		initSatisfied := applyInitiativeGate(graphMap, func(k string) bool { return satisfied[k] }, initiativeItems, initiativeDeps)
		for node, done := range initSatisfied {
			satisfied[node] = done
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

func applyInitiativeGate(graph map[string][]string, itemSatisfied func(string) bool, initiativeItems, initiativeDeps map[string][]string) map[string]bool {
	itemInitiative := make(map[string]string, len(graph))
	for name, members := range initiativeItems {
		for _, m := range members {
			itemInitiative[m] = name
		}
	}
	for ref, owner := range itemInitiative {
		for _, d := range initiativeDeps[owner] {
			graph[ref] = append(graph[ref], normalizeInitiativeDepRef(d))
		}
	}

	initSatisfied := make(map[string]bool, len(initiativeItems))
	for name, members := range initiativeItems {
		node := initiativeNode(name)
		deps := append([]string(nil), members...)
		for _, d := range initiativeDeps[name] {
			deps = append(deps, normalizeInitiativeDepRef(d))
		}
		graph[node] = deps
		complete := true
		for _, m := range members {
			if !itemSatisfied(m) {
				complete = false
				break
			}
		}
		initSatisfied[node] = complete
	}
	return initSatisfied
}

func normalizeInitiativeDepRef(ref string) string {
	if strings.Contains(ref, "/") {
		return ref
	}
	return initiativeNode(ref)
}

func initiativeNode(name string) string {
	return "initiative/" + name
}
