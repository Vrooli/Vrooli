// Package depgraph provides generic dependency-graph utilities shared by
// backlog items and initiatives (and any future entity with a DAG). It is
// intentionally kind-agnostic: callers adapt their domain type to the Node
// interface and supply the statuses that count as "blocking" for their
// workflow.
package depgraph

import "strings"

// Node is the minimal shape depgraph needs: a stable key, its upstream
// dependency keys, and a status string used for blocking evaluation.
type Node interface {
	Key() string
	Deps() []string
	Status() string
}

// BlockingInfo summarizes which upstream nodes are still blocking a node.
type BlockingInfo struct {
	Blocked      bool
	BlockingKeys []string
	AllForceable bool
}

// ComputeBlocking evaluates each node's dependencies against `blockingStatuses`
// and returns a map keyed by node.Key() for nodes that are blocked. Missing
// dependencies are treated as fail-open (presumed completed).
//
// `forceable` controls the AllForceable flag set on returned BlockingInfo —
// for backlog-style workflows this is always true (force flag can override),
// for initiatives callers pass false if they want hard blocks.
func ComputeBlocking(nodes []Node, blockingStatuses map[string]bool, forceable bool) map[string]BlockingInfo {
	byKey := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		byKey[n.Key()] = n
	}
	result := make(map[string]BlockingInfo)
	for _, n := range nodes {
		deps := n.Deps()
		if len(deps) == 0 {
			continue
		}
		var blocking []string
		for _, ref := range deps {
			dep, found := byKey[ref]
			if !found {
				continue
			}
			if blockingStatuses[dep.Status()] {
				blocking = append(blocking, ref)
			}
		}
		if len(blocking) == 0 {
			continue
		}
		result[n.Key()] = BlockingInfo{
			Blocked:      true,
			BlockingKeys: blocking,
			AllForceable: forceable,
		}
	}
	return result
}

// DetectCycleFrom runs a DFS from `start` over the provided adjacency map
// (key -> upstream keys) and returns a human-readable cycle path like
// "a -> b -> a" if one is reachable, or "" otherwise.
//
// Callers typically build the graph by copying existing nodes' deps and
// overlaying a proposed change, then invoke DetectCycleFrom with the changed
// node's key to validate the update.
func DetectCycleFrom(graph map[string][]string, start string) string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(graph))
	var stack []string
	var found []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		stack = append(stack, node)
		for _, next := range graph[node] {
			if color[next] == gray {
				idx := 0
				for i, n := range stack {
					if n == next {
						idx = i
						break
					}
				}
				found = append([]string{}, stack[idx:]...)
				found = append(found, next)
				return true
			}
			if color[next] == white {
				if dfs(next) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[node] = black
		return false
	}

	if dfs(start) {
		return strings.Join(found, " -> ")
	}
	return ""
}

// BuildGraph materializes a plain adjacency map from a node slice, which
// callers can mutate before passing to DetectCycleFrom (e.g. to overlay a
// proposed dependency change before committing it).
func BuildGraph(nodes []Node) map[string][]string {
	g := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		g[n.Key()] = append([]string(nil), n.Deps()...)
	}
	return g
}
