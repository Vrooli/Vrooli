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
	nodes, edges = appendTopologyBacklogNodes(nodes, edges, items, execRecords, itemIndex, itemByKey)

	// Initiative nodes and member_of edges.
	nodes = p.appendTopologyInitiativeNodes(ctx, nodes, itemByKey)
	edges = appendTopologyMemberOfEdges(edges, items)

	// Capture nodes and classified_as edges.
	nodes, edges = p.appendTopologyCaptureNodes(nodes, edges, itemIndex)

	// Scenario nodes and targets edges.
	nodes, edges = p.appendTopologyScenarioNodes(ctx, nodes, edges, items)

	return NewGraphResponse(LensTopology, nodes, edges), nil
}

// appendTopologyBacklogNodes appends backlog item nodes and their depends_on
// edges. It also populates itemIndex (active item keys) and itemByKey (all
// items by key) for use by later topology sections.
func appendTopologyBacklogNodes(
	nodes []Node,
	edges []Edge,
	items []backlog.BacklogItem,
	execRecords []execution.Record,
	itemIndex map[string]bool,
	itemByKey map[string]backlog.BacklogItem,
) ([]Node, []Edge) {
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
	return nodes, edges
}

// appendTopologyInitiativeNodes appends non-archived initiative nodes.
func (p *ProjectionService) appendTopologyInitiativeNodes(
	ctx context.Context,
	nodes []Node,
	itemByKey map[string]backlog.BacklogItem,
) []Node {
	if p.initiative == nil {
		return nodes
	}
	inits, err := p.initiative.List()
	if err != nil {
		slog.Error("topology: initiatives error", "error", err)
		return nodes
	}
	activeRounds := p.loadActiveRounds(ctx)
	for _, init := range inits {
		if init.ArchivedAt != nil {
			continue
		}

		rollup := computeInitiativeRollup(init.Items, itemByKey)
		initNodeID := "initiative/" + init.Name
		data := GraphInitiativeNodeData{
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
		}
		if round, ok := activeRounds[init.Name]; ok {
			data.OperatingMode = round.Mode
			data.ActiveRound = &GraphInitiativeActiveRound{
				Mode:   round.Mode,
				Phase:  round.Phase,
				Round:  round.Round,
				Status: round.Status,
			}
		}
		nodes = append(nodes, Node{
			ID:   initNodeID,
			Type: "Initiative",
			Data: data,
		})
	}
	return nodes
}

// appendTopologyMemberOfEdges appends member_of edges from active backlog items
// to their initiatives.
func appendTopologyMemberOfEdges(edges []Edge, items []backlog.BacklogItem) []Edge {
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
	return edges
}

// appendTopologyCaptureNodes appends capture nodes and classified_as edges to
// active backlog items.
func (p *ProjectionService) appendTopologyCaptureNodes(
	nodes []Node,
	edges []Edge,
	itemIndex map[string]bool,
) ([]Node, []Edge) {
	if p.capture == nil {
		return nodes, edges
	}
	caps, err := p.capture.ListCaptures()
	if err != nil {
		slog.Error("topology: captures error", "error", err)
		return nodes, edges
	}
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
	return nodes, edges
}

// appendTopologyScenarioNodes appends scenario nodes and targets edges.
// Only scenarios targeted by at least one active backlog item are included —
// disconnected scenarios clutter the topology with no structural value and
// cause Dagre to produce linear layouts.
func (p *ProjectionService) appendTopologyScenarioNodes(
	ctx context.Context,
	nodes []Node,
	edges []Edge,
	items []backlog.BacklogItem,
) ([]Node, []Edge) {
	if p.scenario == nil {
		return nodes, edges
	}
	scens, err := p.scenario.List(ctx)
	if err != nil {
		slog.Error("topology: scenarios error", "error", err)
		return nodes, edges
	}

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
	return nodes, edges
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
