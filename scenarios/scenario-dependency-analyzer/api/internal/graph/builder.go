package graph

import (
	"fmt"

	"scenario-dependency-analyzer/internal/seams"
	types "scenario-dependency-analyzer/internal/types"
)

// DependencyStore loads persisted dependency evidence for graph construction.
type DependencyStore interface {
	LoadAllDependencies() ([]types.ScenarioDependency, error)
}

// ScenarioCatalog identifies known scenarios so stale scenario references can be filtered.
type ScenarioCatalog interface {
	KnownScenario(name string) bool
}

// Builder owns dependency graph construction from persisted dependency evidence.
type Builder struct {
	store   DependencyStore
	catalog ScenarioCatalog
	seams   *seams.Dependencies
}

// NewBuilder constructs a dependency graph builder with explicit dependencies.
func NewBuilder(store DependencyStore, catalog ScenarioCatalog, deps *seams.Dependencies) Builder {
	return Builder{
		store:   store,
		catalog: catalog,
		seams:   deps,
	}
}

// Generate builds the graph for the provided type using the configured store and catalog.
func (b Builder) Generate(graphType string) (*types.DependencyGraph, error) {
	if b.store == nil {
		return nil, fmt.Errorf("dependency store not initialized")
	}

	deps := b.seams
	if deps == nil {
		deps = seams.Default
	}

	allDeps, err := b.store.LoadAllDependencies()
	if err != nil {
		return nil, err
	}

	knownScenario := func(name string) bool {
		if b.catalog == nil {
			return true
		}
		return b.catalog.KnownScenario(name)
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
		if dep.DependencyType == "scenario" && !knownScenario(dep.DependencyName) {
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
			"complexity_score": CalculateComplexityScore(nodes, edges),
		},
	}

	return graph, nil
}

// CalculateComplexityScore returns a normalized graph density score.
func CalculateComplexityScore(nodes []types.GraphNode, edges []types.GraphEdge) float64 {
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
