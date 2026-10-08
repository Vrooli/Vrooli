package depgraph

import "sort"

// TransitiveClosure returns the roots plus every node reachable by following
// dependency edges (depends-on / prerequisites) from any root — the transitive
// prerequisite closure. The result is de-duplicated, sorted, and cycle-safe
// (a node is visited at most once). Referenced-but-absent deps are included as
// leaves so callers can see prerequisites whose node was never registered.
//
// This is the goal-scope primitive: given a goal's targets, it resolves the
// full set of work that must complete for the goal to be done. Dependents (the
// reverse direction) remains available via Dependents for direct edges.
func (g *Graph) TransitiveClosure(roots []string) []string {
	seen := make(map[string]bool, len(roots))
	queue := make([]string, 0, len(roots))
	for _, r := range roots {
		if !seen[r] {
			seen[r] = true
			queue = append(queue, r)
		}
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, dep := range g.nodes[node] {
			if !seen[dep] {
				seen[dep] = true
				queue = append(queue, dep)
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TransitiveDependents returns every node that transitively depends on any of
// the given roots (the reverse of TransitiveClosure), excluding the roots
// unless a root depends on another. De-duplicated, sorted, cycle-safe.
func (g *Graph) TransitiveDependents(roots []string) []string {
	// Build a reverse adjacency once.
	reverse := make(map[string][]string, len(g.nodes))
	for node, deps := range g.nodes {
		for _, dep := range deps {
			reverse[dep] = append(reverse[dep], node)
		}
	}
	seen := make(map[string]bool, len(roots))
	var queue []string
	for _, r := range roots {
		for _, dependent := range reverse[r] {
			if !seen[dependent] {
				seen[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, dependent := range reverse[node] {
			if !seen[dependent] {
				seen[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
