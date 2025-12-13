package app

import (
	"fmt"

	"scenario-dependency-analyzer/internal/seams"
	types "scenario-dependency-analyzer/internal/types"
)

type dependencyStore interface {
	LoadAllDependencies() ([]types.ScenarioDependency, error)
}

type scenarioCatalog interface {
	KnownScenario(name string) bool
}

// graphBuilder owns dependency graph construction given explicit dependencies,
// keeping the dependency store and scenario catalog explicit rather than hidden behind globals.
type graphBuilder struct {
	store   dependencyStore
	catalog scenarioCatalog
	seams   *seams.Dependencies
}

// Generate builds the graph for the provided type using the configured store/catalog.
func (g graphBuilder) Generate(graphType string) (*types.DependencyGraph, error) {
	if g.store == nil {
		return nil, fmt.Errorf("dependency store not initialized")
	}

	deps := g.seams
	if deps == nil {
		deps = seams.Default
	}

	allDeps, err := g.store.LoadAllDependencies()
	if err != nil {
		return nil, err
	}

	knownScenario := func(name string) bool {
		if g.catalog == nil {
			return true
		}
		return g.catalog.KnownScenario(name)
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
			"complexity_score": calculateComplexityScore(nodes, edges),
		},
	}

	return graph, nil
}

// generateDependencyGraph generates a dependency graph using the active analyzer runtime (or fallback globals).
// For deterministic testing, prefer generateDependencyGraphWithSeams.
func generateDependencyGraph(graphType string) (*types.DependencyGraph, error) {
	if analyzer := analyzerInstance(); analyzer != nil {
		return analyzer.GenerateGraph(graphType)
	}

	depSvc := defaultDependencyService()
	builder := graphBuilder{
		store:   depSvc.store,
		catalog: depSvc.detector,
		seams:   seams.Default,
	}
	return builder.Generate(graphType)
}

// generateDependencyGraphWithSeams generates a dependency graph using injected seams.
// This enables deterministic testing by controlling time and ID generation.
func generateDependencyGraphWithSeams(graphType string, deps *seams.Dependencies) (*types.DependencyGraph, error) {
	if analyzer := analyzerInstance(); analyzer != nil {
		return analyzer.generateGraphWithSeams(graphType, deps)
	}

	depSvc := defaultDependencyService()
	builder := graphBuilder{
		store:   depSvc.store,
		catalog: depSvc.detector,
		seams:   deps,
	}
	return builder.Generate(graphType)
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
