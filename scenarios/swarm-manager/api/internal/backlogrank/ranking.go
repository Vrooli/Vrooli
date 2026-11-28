package backlogrank

import (
	"time"
)

const (
	UnblockWeight = 0.5
	UnblockCap    = 3.0
)

var SortResolvedStatuses = map[string]struct{}{
	"completed": {},
}

type Item struct {
	Kind      string
	Name      string
	Status    string
	DependsOn []string
	Archived  bool
	Priority  int
	UpdatedAt time.Time
}

func Key(kind, name string) string {
	return kind + "/" + name
}

func ItemKey(item Item) string {
	return Key(item.Kind, item.Name)
}

func ComputeDepthMap(items []Item) map[string]int {
	itemsByKey := make(map[string]Item, len(items))
	depths := make(map[string]int, len(items))
	for _, item := range items {
		key := ItemKey(item)
		itemsByKey[key] = item
		depths[key] = 0
	}

	incompleteDeps := make(map[string][]string)
	for _, item := range items {
		var deps []string
		for _, dep := range item.DependsOn {
			depItem, ok := itemsByKey[dep]
			if !ok || isResolved(depItem) {
				continue
			}
			deps = append(deps, dep)
		}
		if len(deps) > 0 {
			incompleteDeps[ItemKey(item)] = deps
		}
	}

	maxIterations := len(items)
	for i := 0; i < maxIterations; i++ {
		changed := false
		for key, deps := range incompleteDeps {
			maxDepDepth := 0
			for _, dep := range deps {
				if depDepth := depths[dep]; depDepth > maxDepDepth {
					maxDepDepth = depDepth
				}
			}
			newDepth := maxDepDepth + 1
			if newDepth != depths[key] {
				depths[key] = newDepth
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return depths
}

func ComputeUnblockingMap(items []Item) map[string]int {
	itemsByKey := make(map[string]Item, len(items))
	for _, item := range items {
		itemsByKey[ItemKey(item)] = item
	}

	reverseDeps := make(map[string][]string)
	for _, item := range items {
		if isResolved(item) {
			continue
		}
		key := ItemKey(item)
		for _, dep := range item.DependsOn {
			if _, ok := itemsByKey[dep]; !ok {
				continue
			}
			reverseDeps[dep] = append(reverseDeps[dep], key)
		}
	}

	cache := make(map[string]map[string]struct{}, len(items))
	var getTransitive func(string) map[string]struct{}
	getTransitive = func(key string) map[string]struct{} {
		if cached, ok := cache[key]; ok {
			return cached
		}
		result := make(map[string]struct{})
		cache[key] = result
		for _, dep := range reverseDeps[key] {
			result[dep] = struct{}{}
			for transitive := range getTransitive(dep) {
				result[transitive] = struct{}{}
			}
		}
		return result
	}

	unblocking := make(map[string]int, len(items))
	for _, item := range items {
		key := ItemKey(item)
		unblocking[key] = len(getTransitive(key))
	}
	return unblocking
}

func EffectivePriority(manualPriority int, transitiveDependentCount int) float64 {
	boost := float64(transitiveDependentCount) * UnblockWeight
	if boost > UnblockCap {
		boost = UnblockCap
	}
	return float64(manualPriority) - boost
}

func Less(a, b Item, depthMap, unblockingMap map[string]int) bool {
	keyA := ItemKey(a)
	keyB := ItemKey(b)

	depthA := depthMap[keyA]
	depthB := depthMap[keyB]
	if depthA != depthB {
		return depthA < depthB
	}

	effA := EffectivePriority(a.Priority, unblockingMap[keyA])
	effB := EffectivePriority(b.Priority, unblockingMap[keyB])
	if effA != effB {
		return effA < effB
	}

	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		return a.UpdatedAt.After(b.UpdatedAt)
	}

	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.Name < b.Name
}

func isResolved(item Item) bool {
	if item.Archived {
		return true
	}
	_, ok := SortResolvedStatuses[item.Status]
	return ok
}
