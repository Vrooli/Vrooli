package goals

import (
	"sort"
	"strings"

	"swarm-manager/internal/depgraph"
)

// StatusCompleted is the backlog status that counts an item as done for
// progress. Kept as a literal so the goals package does not import backlog.
const StatusCompleted = "completed"

// ScopeInput carries the pure inputs for scope + gate computation. All refs are
// "<kind>/<name>" for items and initiative names (bare) for initiatives.
type ScopeInput struct {
	// Targets are the goal's raw targets: "<kind>/<name>" or "initiative/<name>".
	Targets []string
	// ItemDeps maps every backlog item ref to its depends_on refs (item refs,
	// or "initiative/<name>" for a mixed item→initiative dep).
	ItemDeps map[string][]string
	// ItemStatus maps every backlog item ref to its backlog status. Membership
	// in this map is the definition of "a known item".
	ItemStatus map[string]string
	// ItemEffort maps every backlog item ref to its effort class (XS..XL, or ""
	// when unsized). Read by the ETA rollup; absent from a ref means unsized.
	ItemEffort map[string]string
	// InitiativeItems maps an initiative name to its member item refs.
	InitiativeItems map[string][]string
	// InitiativeDeps maps an initiative name to the initiatives (or items, for
	// mixed initiative→item deps) it depends on.
	InitiativeDeps map[string][]string
}

// Scope is the computed goal scope.
type Scope struct {
	Targets        []string `json:"targets"`
	Closure        []string `json:"closure"`
	Completed      []string `json:"completed"`
	Ready          []string `json:"ready"`
	Blocked        []string `json:"blocked"`
	Total          int      `json:"total"`
	CompletedCount int      `json:"completed_count"`
	BlockedCount   int      `json:"blocked_count"`
	ProgressPct    float64  `json:"progress_pct"`
}

// initiativeNode returns the graph node key for an initiative.
func initiativeNode(name string) string { return InitiativeTargetPrefix + name }

// ComputeScope resolves a goal's transitive prerequisite closure, progress, and
// gate-aware readiness. It is pure and cycle-safe.
//
// D1: initiative targets expand to member items; the closure walks item
// depends_on; completed items count as done.
// D2: initiative depends_on is a synthetic gate — every item in initiative A is
// blocked until each initiative A depends on is complete (all its items done).
func ComputeScope(in ScopeInput) Scope {
	// Expand targets to item roots (D1: initiative targets → member items).
	var roots []string
	for _, t := range in.Targets {
		if IsInitiativeTarget(t) {
			roots = append(roots, in.InitiativeItems[InitiativeName(t)]...)
		} else {
			roots = append(roots, t)
		}
	}

	// Build the item dependency graph over all known items and walk the
	// transitive closure from the expanded roots.
	itemGraph := depgraph.New()
	for ref := range in.ItemStatus {
		itemGraph.AddNode(ref, in.ItemDeps[ref])
	}
	// Roots that are not known items (e.g. a target typo) still enter the
	// closure as leaves so the scope is not silently narrowed.
	for _, r := range roots {
		if _, ok := in.ItemStatus[r]; !ok {
			itemGraph.AddNode(r, in.ItemDeps[r])
		}
	}
	rawClosure := itemGraph.TransitiveClosure(roots)

	// Restrict the reported closure to known backlog items.
	closure := make([]string, 0, len(rawClosure))
	for _, ref := range rawClosure {
		if _, ok := in.ItemStatus[ref]; ok {
			closure = append(closure, ref)
		}
	}
	sort.Strings(closure)

	// Build the gate-aware readiness graph over items, then fold in the D2
	// initiative gate.
	graphMap := make(map[string][]string, len(in.ItemStatus))
	completed := make(map[string]bool, len(in.ItemStatus))
	for ref, status := range in.ItemStatus {
		graphMap[ref] = in.ItemDeps[ref]
		completed[ref] = status == StatusCompleted
	}
	initSatisfied := ApplyInitiativeGate(graphMap, func(k string) bool { return completed[k] }, in.InitiativeItems, in.InitiativeDeps)
	for k, v := range initSatisfied {
		completed[k] = v
	}

	gateGraph := depgraph.New()
	for key, deps := range graphMap {
		gateGraph.AddNode(key, deps)
	}
	unblocked := make(map[string]bool)
	for _, ref := range gateGraph.UnblockedItems(completed) {
		unblocked[ref] = true
	}

	scope := Scope{Targets: append([]string(nil), in.Targets...), Closure: closure}
	for _, ref := range closure {
		switch {
		case in.ItemStatus[ref] == StatusCompleted:
			scope.Completed = append(scope.Completed, ref)
		case unblocked[ref]:
			scope.Ready = append(scope.Ready, ref)
		default:
			scope.Blocked = append(scope.Blocked, ref)
		}
	}
	scope.Total = len(closure)
	scope.CompletedCount = len(scope.Completed)
	scope.BlockedCount = len(scope.Blocked)
	if scope.Total > 0 {
		scope.ProgressPct = float64(scope.CompletedCount) / float64(scope.Total) * 100.0
	}
	return scope
}

// ApplyInitiativeGate augments an item dependency graph in place with the D2
// initiative gate: every member item inherits its initiative's dependencies,
// and a synthetic "initiative/<name>" node is added per initiative (deps =
// member items + the initiative's dependencies). It returns the initiative-node
// satisfied set (an initiative is satisfied iff all its member items are
// satisfied) so callers can extend their readiness predicate.
//
// itemSatisfied reports whether an item ref is already done (completed/archived
// per the caller's policy). Only initiatives with dependencies add edges to
// member items, so an item graph with no gated initiatives is unchanged apart
// from the added initiative nodes (which callers that render only items ignore).
func ApplyInitiativeGate(graph map[string][]string, itemSatisfied func(string) bool, initiativeItems, initiativeDeps map[string][]string) map[string]bool {
	if itemSatisfied == nil {
		itemSatisfied = func(string) bool { return false }
	}
	itemInitiative := make(map[string]string, len(graph))
	for name, members := range initiativeItems {
		for _, m := range members {
			itemInitiative[m] = name
		}
	}
	// Each member item inherits its initiative's dependencies (the gate).
	for ref, owner := range itemInitiative {
		deps := initiativeDeps[owner]
		if len(deps) == 0 {
			continue
		}
		for _, d := range deps {
			graph[ref] = append(graph[ref], NormalizeInitiativeDepRef(d))
		}
	}
	// Add a synthetic node per initiative and compute its satisfied state.
	initSatisfied := make(map[string]bool, len(initiativeItems))
	for name, members := range initiativeItems {
		node := initiativeNode(name)
		deps := append([]string(nil), members...)
		for _, d := range initiativeDeps[name] {
			deps = append(deps, NormalizeInitiativeDepRef(d))
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

// NormalizeInitiativeDepRef normalizes an initiative dependency ref. A bare
// name is an initiative ("initiative/<name>"); a ref already containing "/" is
// treated as a direct node (an item, for a mixed initiative→item dep).
func NormalizeInitiativeDepRef(ref string) string {
	if strings.Contains(ref, "/") {
		return ref
	}
	return initiativeNode(ref)
}
