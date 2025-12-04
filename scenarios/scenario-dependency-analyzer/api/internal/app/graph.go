package app

import (
	"scenario-dependency-analyzer/internal/seams"
	types "scenario-dependency-analyzer/internal/types"
)

// generateDependencyGraph generates a dependency graph using default seams.
// For testable code, use generateDependencyGraphWithSeams.
func generateDependencyGraph(graphType string) (*types.DependencyGraph, error) {
	return generateDependencyGraphWithSeams(graphType, seams.Default)
}

// generateDependencyGraphWithSeams generates a dependency graph using injected seams.
// This enables deterministic testing by controlling time and ID generation.
func generateDependencyGraphWithSeams(graphType string, deps *seams.Dependencies) (*types.DependencyGraph, error) {
	allDeps, err := loadAllDependencies()
	if err != nil {
		return nil, err
	}

	nodes := []types.GraphNode{}
	edges := []types.GraphEdge{}
	nodeSet := make(map[string]bool)

	for _, dep := range allDeps {
		if graphType == "resource" && dep.DependencyType != "resource" {
			continue
		}
		if graphType == "scenario" && dep.DependencyType == "resource" {
			continue
		}
		if dep.DependencyType == "scenario" && !isKnownScenario(dep.DependencyName) {
			continue
		}

		if !nodeSet[dep.ScenarioName] {
			nodes = append(nodes, types.GraphNode{
				ID:    dep.ScenarioName,
				Label: dep.ScenarioName,
				Type:  "scenario",
				Group: "scenarios",
				Metadata: map[string]interface{}{
					"node_type": "scenario",
				},
			})
			nodeSet[dep.ScenarioName] = true
		}

		if !nodeSet[dep.DependencyName] {
			nodeGroup := "resources"
			nodeType := "resource"
			if dep.DependencyType == "scenario" {
				nodeGroup = "scenarios"
				nodeType = "scenario"
			} else if dep.DependencyType == "shared_workflow" {
				nodeGroup = "workflows"
				nodeType = "workflow"
			}

			nodes = append(nodes, types.GraphNode{
				ID:    dep.DependencyName,
				Label: dep.DependencyName,
				Type:  nodeType,
				Group: nodeGroup,
				Metadata: map[string]interface{}{
					"node_type": dep.DependencyType,
				},
			})
			nodeSet[dep.DependencyName] = true
		}

		weight := 1.0
		if dep.Required {
			weight = 2.0
		}

		edges = append(edges, types.GraphEdge{
			Source:   dep.ScenarioName,
			Target:   dep.DependencyName,
			Label:    dep.DependencyType,
			Type:     dep.DependencyType,
			Required: dep.Required,
			Weight:   weight,
			Metadata: map[string]interface{}{
				"purpose":       dep.Purpose,
				"access_method": dep.AccessMethod,
				"configuration": dep.Configuration,
				"discovered_at": dep.DiscoveredAt,
				"last_verified": dep.LastVerified,
			},
		})
	}

	graph := &types.DependencyGraph{
		ID:    deps.IDs.NewID(),
		Type:  graphType,
		Nodes: nodes,
		Edges: edges,
		Metadata: map[string]interface{}{
			"total_nodes":      len(nodes),
			"total_edges":      len(edges),
			"generated_at":     deps.Clock.Now(),
			"complexity_score": calculateComplexityScore(nodes, edges),
		},
	}

	return graph, nil
}

func calculateComplexityScore(nodes []types.GraphNode, edges []types.GraphEdge) float64 {
	if len(nodes) == 0 {
		return 0.0
	}

	ratio := float64(len(edges)) / float64(len(nodes))
	score := ratio / 5.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}
