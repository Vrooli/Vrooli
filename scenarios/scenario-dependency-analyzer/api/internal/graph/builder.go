package graph

import (
	"fmt"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/seams"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// DependencyStore loads the persisted unified, evidence-tagged graph store used
// for graph construction and centrality.
type DependencyStore interface {
	LoadGraphEdges() ([]types.UnifiedGraphEdge, error)
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

	allEdges, err := b.store.LoadGraphEdges()
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

	for _, edge := range allEdges {
		isResource := edge.Kind == "resource"
		if graphType == "resource" && !isResource {
			continue
		}
		if graphType == "scenario" && isResource {
			continue
		}
		if !isResource && !knownScenario(edge.To) {
			continue
		}

		if !nodeSet[edge.From] {
			nodes = append(nodes, types.GraphNode{
				ID:    edge.From,
				Label: edge.From,
				Type:  "scenario",
				Group: "scenarios",
				Metadata: map[string]interface{}{
					"node_type": "scenario",
				},
			})
			nodeSet[edge.From] = true
		}

		if !nodeSet[edge.To] {
			nodeGroup := "scenarios"
			nodeType := "scenario"
			if isResource {
				nodeGroup = "resources"
				nodeType = "resource"
			}

			nodes = append(nodes, types.GraphNode{
				ID:    edge.To,
				Label: edge.To,
				Type:  nodeType,
				Group: nodeGroup,
				Metadata: map[string]interface{}{
					"node_type": edge.Kind,
				},
			})
			nodeSet[edge.To] = true
		}

		weight := 1.0
		if edge.Required {
			weight = 2.0
		}

		edges = append(edges, types.GraphEdge{
			Source:   edge.From,
			Target:   edge.To,
			Label:    edge.Source,
			Type:     edgeType(isResource),
			Required: edge.Required,
			Weight:   weight,
			Metadata: map[string]interface{}{
				"evidence_source": edge.Source,
				"confidence":      edge.Confidence,
				"evidence":        edge.Evidence,
				"stale":           edge.Stale,
				"last_verified":   edge.LastVerified,
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

func edgeType(isResource bool) string {
	if isResource {
		return "resource"
	}
	return "scenario"
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
