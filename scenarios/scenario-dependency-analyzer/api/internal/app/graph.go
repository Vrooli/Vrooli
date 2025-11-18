package app

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	types "scenario-dependency-analyzer/internal/types"
)

func generateDependencyGraph(graphType string) (*types.DependencyGraph, error) {
	nodes := []types.GraphNode{}
	edges := []types.GraphEdge{}

	rows, err := db.Query(`
        SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
        FROM scenario_dependencies
        ORDER BY scenario_name, dependency_type, dependency_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodeSet := make(map[string]bool)

	for rows.Next() {
		var scenarioName, depType, depName, purpose, accessMethod string
		var required bool
		var configJSON []byte
		var discoveredAt, lastVerified time.Time

		if err := rows.Scan(&scenarioName, &depType, &depName, &required, &purpose, &accessMethod, &configJSON, &discoveredAt, &lastVerified); err != nil {
			continue
		}

		var configuration map[string]interface{}
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &configuration); err != nil {
				configuration = map[string]interface{}{}
			}
		}

		if graphType == "resource" && depType != "resource" {
			continue
		}
		if graphType == "scenario" && depType == "resource" {
			continue
		}
		if depType == "scenario" && !isKnownScenario(depName) {
			continue
		}

		if !nodeSet[scenarioName] {
			nodes = append(nodes, types.GraphNode{
				ID:    scenarioName,
				Label: scenarioName,
				Type:  "scenario",
				Group: "scenarios",
				Metadata: map[string]interface{}{
					"node_type": "scenario",
				},
			})
			nodeSet[scenarioName] = true
		}

		if !nodeSet[depName] {
			nodeGroup := "resources"
			nodeType := "resource"
			if depType == "scenario" {
				nodeGroup = "scenarios"
				nodeType = "scenario"
			} else if depType == "shared_workflow" {
				nodeGroup = "workflows"
				nodeType = "workflow"
			}

			nodes = append(nodes, types.GraphNode{
				ID:    depName,
				Label: depName,
				Type:  nodeType,
				Group: nodeGroup,
				Metadata: map[string]interface{}{
					"node_type": depType,
				},
			})
			nodeSet[depName] = true
		}

		weight := 1.0
		if required {
			weight = 2.0
		}

		edges = append(edges, types.GraphEdge{
			Source:   scenarioName,
			Target:   depName,
			Label:    depType,
			Type:     depType,
			Required: required,
			Weight:   weight,
			Metadata: map[string]interface{}{
				"purpose":       purpose,
				"access_method": accessMethod,
				"configuration": configuration,
				"discovered_at": discoveredAt,
				"last_verified": lastVerified,
			},
		})
	}

	graph := &types.DependencyGraph{
		ID:    uuid.New().String(),
		Type:  graphType,
		Nodes: nodes,
		Edges: edges,
		Metadata: map[string]interface{}{
			"total_nodes":      len(nodes),
			"total_edges":      len(edges),
			"generated_at":     time.Now(),
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
