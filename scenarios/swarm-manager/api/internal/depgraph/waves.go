package depgraph

import (
	"sort"
	"strings"
)

// CycleWave is the wave index assigned to nodes trapped in (or strictly
// downstream of) a dependency cycle. Such nodes can never become runnable
// through normal drain, so they carry a diagnostic marker instead of a
// fabricated ordinal.
const CycleWave = -1

// WaveResult is the outcome of frontier peeling over a dependency graph.
type WaveResult struct {
	// Waves maps node key -> wave index. Wave 0 nodes are runnable now
	// (every dependency satisfied); wave n nodes become runnable after n
	// peeling rounds. Cycle-trapped nodes carry CycleWave. Nodes excluded
	// by the isSatisfied predicate are absent.
	Waves map[string]int
	// MaxWave is the highest non-cycle wave index assigned, or -1 when no
	// node received a wave.
	MaxWave int
	// Cycles holds one human-readable path ("a -> b -> a") per distinct
	// cycle reachable from a trapped node, sorted for determinism.
	Cycles []string
}

// Waves computes an ordinal wave index for every node via iterative
// frontier peeling over the depends-on adjacency map (key -> upstream keys).
//
// isSatisfied reports whether a node is out of play — already completed,
// archived, or otherwise not requiring work. Satisfied nodes receive no
// wave and count as met dependencies for their dependents. Dependencies
// that are not graph keys are fail-open (presumed satisfied), matching
// ComputeBlocking.
//
// Wave membership is honest ordinality — "how many dependency layers from
// runnable" — never clock time. Concurrency affects drain timing, not
// membership.
func Waves(graph map[string][]string, isSatisfied func(string) bool) WaveResult {
	if isSatisfied == nil {
		isSatisfied = func(string) bool { return false }
	}

	result := WaveResult{
		Waves:   make(map[string]int),
		MaxWave: -1,
	}

	// Track unresolved nodes and how many unmet deps each still has.
	pendingDeps := make(map[string][]string, len(graph))
	for key, deps := range graph {
		if isSatisfied(key) {
			continue
		}
		var unmet []string
		for _, dep := range deps {
			if _, exists := graph[dep]; !exists {
				continue // unknown dep: fail-open
			}
			if isSatisfied(dep) {
				continue
			}
			unmet = append(unmet, dep)
		}
		pendingDeps[key] = unmet
	}

	peeled := make(map[string]bool, len(pendingDeps))
	for wave := 0; len(peeled) < len(pendingDeps); wave++ {
		var frontier []string
		for key, deps := range pendingDeps {
			if peeled[key] {
				continue
			}
			ready := true
			for _, dep := range deps {
				if _, pending := pendingDeps[dep]; pending && !peeled[dep] {
					ready = false
					break
				}
			}
			if ready {
				frontier = append(frontier, key)
			}
		}
		if len(frontier) == 0 {
			break // remaining nodes are cycle-trapped
		}
		for _, key := range frontier {
			peeled[key] = true
			result.Waves[key] = wave
		}
		result.MaxWave = wave
	}

	// Anything never peeled sits in or downstream of a cycle.
	cycleSeen := make(map[string]bool)
	var trapped []string
	for key := range pendingDeps {
		if !peeled[key] {
			trapped = append(trapped, key)
		}
	}
	sort.Strings(trapped)
	for _, key := range trapped {
		result.Waves[key] = CycleWave
		path := DetectCycleFrom(graph, key)
		if path == "" {
			continue
		}
		// The same cycle discovered from different start nodes yields
		// rotated paths; dedupe on the sorted member set.
		if members := cycleMemberKey(path); !cycleSeen[members] {
			cycleSeen[members] = true
			result.Cycles = append(result.Cycles, path)
		}
	}
	sort.Strings(result.Cycles)

	return result
}

// cycleMemberKey reduces a cycle path like "b -> a -> b" to a canonical
// sorted-member key ("a|b") so rotations of the same cycle compare equal.
func cycleMemberKey(path string) string {
	parts := strings.Split(path, " -> ")
	members := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		members[p] = struct{}{}
	}
	keys := make([]string, 0, len(members))
	for k := range members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}
