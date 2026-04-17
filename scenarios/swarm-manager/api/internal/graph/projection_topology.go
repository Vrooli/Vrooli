package graph

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// buildTopology builds the topology lens projection.
func (p *ProjectionService) buildTopology(ctx context.Context) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	// Load backlog items (exclude completed/archived).
	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}

	// Load execution records for cross-lens status annotation on backlog nodes.
	var execRecords []execution.Record
	if p.execution != nil {
		execRecords, _ = p.execution.List(ctx, execution.ListFilters{})
	}

	itemIndex := make(map[string]bool, len(items))
	itemByKey := make(map[string]backlog.BacklogItem, len(items))
	for _, item := range items {
		key := backlogItemKey(string(item.Kind), item.Name)
		itemByKey[key] = item
		if item.Status == backlog.StatusCompleted || item.ArchivedAt != nil {
			continue
		}
		itemIndex[key] = true
		nodeID := backlogItemNodeID(string(item.Kind), item.Name)
		activeStatus, activeCount := computeActiveExecutionSummary(key, execRecords)
		nodes = append(nodes, Node{
			ID:   nodeID,
			Type: "BacklogItem",
			Data: GraphBacklogNodeData{
				Kind:                  string(item.Kind),
				Name:                  item.Name,
				Title:                 item.Title,
				Status:                string(item.Status),
				Priority:              int32(item.Priority),
				ActiveExecutionStatus: activeStatus,
				ActiveExecutionCount:  int32(activeCount),
			},
		})

		// depends_on edges.
		for _, dep := range item.DependsOn {
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("depends_on:%s->%s", key, dep),
				Source: nodeID,
				Target: backlogItemNodeIDFromKey(dep),
				Type:   "depends_on",
			})
		}
	}

	// Initiative nodes and member_of edges.
	if p.initiative != nil {
		inits, err := p.initiative.List()
		if err != nil {
			slog.Error("topology: initiatives error", "error", err)
		} else {
			for _, init := range inits {
				if init.ArchivedAt != nil {
					continue
				}

				rollup := computeInitiativeRollup(init.Items, itemByKey)
				initNodeID := "initiative/" + init.Name
				nodes = append(nodes, Node{
					ID:   initNodeID,
					Type: "Initiative",
					Data: GraphInitiativeNodeData{
						Name:   init.Name,
						Title:  init.Title,
						Status: init.Status,
						Rollup: GraphInitiativeRollup{
							Total:      int32(rollup.Total),
							Completed:  int32(rollup.Completed),
							InProgress: int32(rollup.InProgress),
							Failed:     int32(rollup.Failed),
							Pending:    int32(rollup.Pending),
						},
					},
				})
			}
		}
	}

	// member_of edges from backlog items to initiatives.
	for _, item := range items {
		if item.Status == backlog.StatusCompleted || item.ArchivedAt != nil {
			continue
		}
		if item.Initiative != "" {
			key := backlogItemKey(string(item.Kind), item.Name)
			nodeID := backlogItemNodeID(string(item.Kind), item.Name)
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("member_of:%s->%s", key, item.Initiative),
				Source: nodeID,
				Target: "initiative/" + item.Initiative,
				Type:   "member_of",
			})
		}
	}

	// Capture nodes and classified_as edges.
	if p.capture != nil {
		caps, err := p.capture.ListCaptures()
		if err != nil {
			slog.Error("topology: captures error", "error", err)
		} else {
			for _, cap := range caps {
				if len(cap.Items) == 0 {
					continue
				}
				capNodeID := "capture/" + cap.ID
				nodes = append(nodes, Node{
					ID:   capNodeID,
					Type: "Capture",
					Data: GraphCaptureNodeData{
						ID:     cap.ID,
						Text:   cap.Text,
						Status: cap.Status,
					},
				})
				for _, ci := range cap.Items {
					targetKey := backlogItemKey(ci.Kind, ci.Title)
					if itemIndex[targetKey] {
						edges = append(edges, Edge{
							ID:     fmt.Sprintf("classified_as:%s->%s", cap.ID, targetKey),
							Source: capNodeID,
							Target: backlogItemNodeIDFromKey(targetKey),
							Type:   "classified_as",
						})
					}
				}
			}
		}
	}

	// Scenario nodes and targets edges.
	// Only include scenarios that are targeted by at least one active backlog
	// item — disconnected scenarios clutter the topology with no structural
	// value and cause Dagre to produce linear layouts.
	if p.scenario != nil {
		scens, err := p.scenario.List(ctx)
		if err != nil {
			slog.Error("topology: scenarios error", "error", err)
		} else {
			scenByName := make(map[string]ScenarioEntry, len(scens))
			for _, s := range scens {
				scenByName[s.Name] = s
			}

			// Build targets edges first to discover which scenarios are referenced.
			referencedScenarios := make(map[string]struct{})
			for _, item := range items {
				if item.Status == backlog.StatusCompleted || item.ArchivedAt != nil {
					continue
				}
				for _, pattern := range item.AcceptanceAllow {
					for _, s := range scens {
						if matchesAcceptancePattern(pattern, s.Name) {
							key := backlogItemKey(string(item.Kind), item.Name)
							edges = append(edges, Edge{
								ID:     fmt.Sprintf("targets:%s->%s", key, s.Name),
								Source: backlogItemNodeID(string(item.Kind), item.Name),
								Target: "scenario/" + s.Name,
								Type:   "targets",
							})
							referencedScenarios[s.Name] = struct{}{}
						}
					}
				}
			}

			// Only emit scenario nodes that are targeted by backlog items.
			for name := range referencedScenarios {
				s := scenByName[name]
				nodes = append(nodes, Node{
					ID:   "scenario/" + s.Name,
					Type: "Scenario",
					Data: GraphScenarioNodeData(s),
				})
			}
		}
	}

	return NewGraphResponse(LensTopology, nodes, edges), nil
}

// matchesAcceptancePattern checks if a scenario name matches an acceptance_allow glob pattern.
// Uses simple prefix matching: "scenarios/foo" matches scenario "foo",
// "scenarios/foo/**" also matches "foo".
func matchesAcceptancePattern(pattern, scenarioName string) bool {
	// Strip common prefixes.
	p := pattern
	p = strings.TrimPrefix(p, "scenarios/")
	// Strip glob suffixes.
	p = strings.TrimSuffix(p, "/**")
	p = strings.TrimSuffix(p, "/*")
	// Exact match or prefix match.
	return p == scenarioName || strings.HasPrefix(scenarioName, p+"/")
}
