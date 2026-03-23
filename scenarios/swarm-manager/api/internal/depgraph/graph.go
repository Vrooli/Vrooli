// Package depgraph provides a directed dependency graph with cycle detection
// and topological sorting. It is used to validate and reason about
// dependencies between backlog items.
package depgraph

import (
	"fmt"
	"sort"
)

// Graph represents a directed dependency graph where each node maps to its
// direct dependencies (predecessors that must complete before this node).
type Graph struct {
	nodes map[string][]string // key -> list of dependency keys
}

// New creates an empty Graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string][]string),
	}
}

// AddNode registers a node with its dependencies. If the node already exists,
// its dependency list is replaced. Dependencies referencing unknown nodes are
// still recorded (they may be added later).
func (g *Graph) AddNode(key string, deps []string) {
	g.nodes[key] = deps
}

// DetectCycle performs a DFS-based cycle detection. If a cycle is found, it
// returns the cycle path (a slice of node keys forming the cycle) and true.
// If no cycle exists, it returns nil and false.
func (g *Graph) DetectCycle() ([]string, bool) {
	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[string]int, len(g.nodes))
	parent := make(map[string]string, len(g.nodes))

	// Sort keys for deterministic cycle detection.
	keys := g.sortedKeys()

	var cyclePath []string
	found := false

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, dep := range g.nodes[node] {
			if _, exists := g.nodes[dep]; !exists {
				continue // skip references to nodes not in graph
			}
			if color[dep] == gray {
				// Found a cycle: reconstruct the path.
				cyclePath = []string{dep, node}
				cur := node
				for cur != dep {
					cur = parent[cur]
					cyclePath = append(cyclePath, cur)
				}
				// Reverse to get forward order.
				for i, j := 0, len(cyclePath)-1; i < j; i, j = i+1, j-1 {
					cyclePath[i], cyclePath[j] = cyclePath[j], cyclePath[i]
				}
				return true
			}
			if color[dep] == white {
				parent[dep] = node
				if dfs(dep) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	for _, key := range keys {
		if color[key] == white {
			if dfs(key) {
				found = true
				break
			}
		}
	}

	return cyclePath, found
}

// TopologicalSort returns nodes in dependency-first order using Kahn's
// algorithm. Returns an error if the graph contains a cycle.
func (g *Graph) TopologicalSort() ([]string, error) {
	// Compute in-degree (number of dependencies within the graph).
	inDegree := make(map[string]int, len(g.nodes))
	for key := range g.nodes {
		if _, ok := inDegree[key]; !ok {
			inDegree[key] = 0
		}
		for _, dep := range g.nodes[key] {
			if _, exists := g.nodes[dep]; exists {
				inDegree[dep]++ // dep has an extra dependent
			}
		}
	}

	// Note: in Kahn's algorithm for dependency ordering, we process nodes
	// whose dependencies are all satisfied. Here "in-degree" actually tracks
	// how many times a node appears as a dependency. We want nodes with all
	// their dependencies resolved first.

	// Recompute: for each node, count how many of its deps are in the graph.
	depCount := make(map[string]int, len(g.nodes))
	for key, deps := range g.nodes {
		count := 0
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; exists {
				count++
			}
		}
		depCount[key] = count
	}

	// Seed queue with nodes that have zero unresolved dependencies.
	var queue []string
	for key, count := range depCount {
		if count == 0 {
			queue = append(queue, key)
		}
	}
	sort.Strings(queue) // deterministic ordering

	var result []string
	for len(queue) > 0 {
		// Pop first element.
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// For each node that depends on this one, decrement its dep count.
		for other, deps := range g.nodes {
			for _, dep := range deps {
				if dep == node {
					depCount[other]--
					if depCount[other] == 0 {
						queue = append(queue, other)
						sort.Strings(queue) // maintain deterministic order
					}
				}
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("dependency cycle detected: processed %d of %d nodes", len(result), len(g.nodes))
	}
	return result, nil
}

// Dependents returns all nodes that directly depend on the given key.
func (g *Graph) Dependents(key string) []string {
	var result []string
	for node, deps := range g.nodes {
		for _, dep := range deps {
			if dep == key {
				result = append(result, node)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}

// UnblockedItems returns nodes whose dependencies are all in the completed set.
// Only considers dependencies that exist as nodes in the graph.
func (g *Graph) UnblockedItems(completed map[string]bool) []string {
	var result []string
	for key, deps := range g.nodes {
		if completed[key] {
			continue // already completed
		}
		blocked := false
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; !exists {
				continue // external dep, skip
			}
			if !completed[dep] {
				blocked = true
				break
			}
		}
		if !blocked {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

// BlockedItems returns nodes that have at least one incomplete dependency.
func (g *Graph) BlockedItems(completed map[string]bool) []string {
	var result []string
	for key, deps := range g.nodes {
		if completed[key] {
			continue
		}
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; !exists {
				continue
			}
			if !completed[dep] {
				result = append(result, key)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}

// Edges returns all dependency edges as [from, to] pairs where "from" depends
// on "to". Only includes edges where both endpoints are graph nodes.
func (g *Graph) Edges() [][2]string {
	var result [][2]string
	for node, deps := range g.nodes {
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; exists {
				result = append(result, [2]string{node, dep})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i][0] != result[j][0] {
			return result[i][0] < result[j][0]
		}
		return result[i][1] < result[j][1]
	})
	return result
}

// sortedKeys returns graph node keys in sorted order for deterministic iteration.
func (g *Graph) sortedKeys() []string {
	keys := make([]string, 0, len(g.nodes))
	for k := range g.nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
