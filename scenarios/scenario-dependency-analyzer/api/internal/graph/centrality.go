package graph

import (
	"sort"
	"strings"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type reverseEdge struct {
	source   string
	required bool
	weight   float64
}

// CalculateCentrality summarizes reverse-dependency importance for scenario graph nodes.
func CalculateCentrality(graph *types.DependencyGraph, coreSeeds []string, scenario string) *types.GraphCentralityReport {
	scenario = strings.TrimSpace(scenario)
	if graph == nil {
		return &types.GraphCentralityReport{
			GraphType: "combined",
			Scenario:  scenario,
			Nodes:     []types.GraphCentralityMetric{},
		}
	}

	scenarios := scenarioNodes(graph)
	reverse := reverseAdjacency(graph)
	undirected := undirectedAdjacency(graph)
	seedSet := stringSet(coreSeeds)

	names := sortedScenarioNames(scenarios)
	if scenario != "" {
		if _, ok := scenarios[scenario]; !ok {
			names = []string{}
		} else {
			names = []string{scenario}
		}
	}

	nodes := make([]types.GraphCentralityMetric, 0, len(names))
	for _, name := range names {
		nodes = append(nodes, centralityForScenario(name, reverse, undirected, seedSet, coreSeeds))
	}

	return &types.GraphCentralityReport{
		GraphType: graph.Type,
		Scenario:  scenario,
		Nodes:     nodes,
		Metadata: map[string]interface{}{
			"total_scenarios": len(scenarios),
			"core_seed_count": len(coreSeeds),
		},
	}
}

func scenarioNodes(graph *types.DependencyGraph) map[string]struct{} {
	out := map[string]struct{}{}
	for _, node := range graph.Nodes {
		if node.Type == "scenario" || node.Group == "scenarios" {
			name := strings.TrimSpace(node.ID)
			if name != "" {
				out[name] = struct{}{}
			}
		}
	}
	for _, edge := range graph.Edges {
		if strings.TrimSpace(edge.Type) != "resource" {
			if source := strings.TrimSpace(edge.Source); source != "" {
				out[source] = struct{}{}
			}
			if target := strings.TrimSpace(edge.Target); target != "" {
				out[target] = struct{}{}
			}
		}
	}
	return out
}

func reverseAdjacency(graph *types.DependencyGraph) map[string][]reverseEdge {
	out := map[string][]reverseEdge{}
	for _, edge := range graph.Edges {
		if strings.TrimSpace(edge.Type) == "resource" {
			continue
		}
		source := strings.TrimSpace(edge.Source)
		target := strings.TrimSpace(edge.Target)
		if source == "" || target == "" {
			continue
		}
		weight := edge.Weight
		if weight <= 0 {
			weight = 1
		}
		if edge.Required && weight < 2 {
			weight = 2
		}
		out[target] = append(out[target], reverseEdge{
			source:   source,
			required: edge.Required,
			weight:   weight,
		})
	}
	return out
}

func undirectedAdjacency(graph *types.DependencyGraph) map[string][]string {
	out := map[string][]string{}
	for _, edge := range graph.Edges {
		if strings.TrimSpace(edge.Type) == "resource" {
			continue
		}
		source := strings.TrimSpace(edge.Source)
		target := strings.TrimSpace(edge.Target)
		if source == "" || target == "" {
			continue
		}
		out[source] = append(out[source], target)
		out[target] = append(out[target], source)
	}
	for key := range out {
		sort.Strings(out[key])
	}
	return out
}

func centralityForScenario(name string, reverse map[string][]reverseEdge, undirected map[string][]string, seedSet map[string]struct{}, orderedSeeds []string) types.GraphCentralityMetric {
	direct := map[string]struct{}{}
	for _, edge := range reverse[name] {
		if edge.source == name {
			continue
		}
		direct[edge.source] = struct{}{}
	}

	transitive, requiredCount, weightedScore := transitiveReverseDependents(name, reverse)
	distance, nearestSeed := nearestCoreSeed(name, undirected, seedSet, orderedSeeds)

	return types.GraphCentralityMetric{
		Scenario:                         name,
		DirectReverseDependencyCount:     len(direct),
		TransitiveReverseDependencyCount: len(transitive),
		RequiredReverseDependencyCount:   requiredCount,
		RequiredEdgeWeightedScore:        weightedScore,
		DistanceToCoreSeed:               distance,
		NearestCoreSeed:                  nearestSeed,
		DirectDependents:                 sortedKeysFromSet(direct),
		TransitiveDependents:             sortedKeysFromSet(transitive),
	}
}

func transitiveReverseDependents(root string, reverse map[string][]reverseEdge) (map[string]struct{}, int, float64) {
	type item struct {
		name       string
		pathWeight float64
		required   bool
	}

	seen := map[string]struct{}{}
	requiredSeen := map[string]struct{}{}
	var weighted float64
	queue := []item{}
	for _, edge := range reverse[root] {
		queue = append(queue, item{name: edge.source, pathWeight: edge.weight, required: edge.required})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.name == root {
			continue
		}
		if _, ok := seen[current.name]; ok {
			continue
		}
		seen[current.name] = struct{}{}
		weighted += current.pathWeight
		if current.required {
			requiredSeen[current.name] = struct{}{}
		}
		for _, edge := range reverse[current.name] {
			queue = append(queue, item{
				name:       edge.source,
				pathWeight: current.pathWeight + edge.weight,
				required:   current.required || edge.required,
			})
		}
	}

	return seen, len(requiredSeen), weighted
}

func nearestCoreSeed(root string, undirected map[string][]string, seedSet map[string]struct{}, orderedSeeds []string) (int, string) {
	if _, ok := seedSet[root]; ok {
		return 0, root
	}
	visited := map[string]struct{}{root: {}}
	queue := []struct {
		name  string
		depth int
	}{{name: root, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range undirected[current.name] {
			if _, ok := visited[next]; ok {
				continue
			}
			visited[next] = struct{}{}
			if _, ok := seedSet[next]; ok {
				return current.depth + 1, next
			}
			queue = append(queue, struct {
				name  string
				depth int
			}{name: next, depth: current.depth + 1})
		}
	}

	for _, seed := range orderedSeeds {
		if strings.TrimSpace(seed) != "" {
			return -1, ""
		}
	}
	return -1, ""
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func sortedScenarioNames(values map[string]struct{}) []string {
	return sortedKeysFromSet(values)
}

func sortedKeysFromSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
