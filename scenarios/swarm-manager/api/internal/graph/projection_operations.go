package graph

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// activeExecutionStatuses are statuses considered "active" for flow/operations lenses.
var activeExecutionStatuses = map[execution.Status]bool{
	execution.StatusPending:     true,
	execution.StatusStarting:    true,
	execution.StatusRunning:     true,
	execution.StatusNeedsReview: true,
	execution.StatusValidating:  true,
	execution.StatusNeedsFixup:  true,
}

// actionableBacklogStatuses are backlog statuses where a user can take a next action
// (workshop, execute, review, retry, archive, etc.) for the operations lens.
// Excludes archived and completed which are terminal/non-actionable.
var actionableBacklogStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusBacklog:     true,
	backlog.StatusResearching: true,
	backlog.StatusReady:       true,
	backlog.StatusQueued:      true,
	backlog.StatusInProgress:  true,
	backlog.StatusFailed:      true,
}

type runtimeBacklogSelection struct {
	items []backlog.BacklogItem
	keys  map[string]struct{}
}

func selectRuntimeBacklogItems(
	allItems []backlog.BacklogItem,
	records []execution.Record,
	activities []agentactivity.Record,
	includeDependencies bool,
	activeStatuses map[backlog.BacklogStatus]bool,
) runtimeBacklogSelection {
	itemsByKey := make(map[string]backlog.BacklogItem, len(allItems))
	included := make(map[string]struct{})
	queue := make([]string, 0)

	enqueue := func(key string) {
		item, ok := itemsByKey[key]
		if !ok {
			return
		}
		if !activeStatuses[item.Status] {
			return
		}
		if _, ok := included[key]; ok {
			return
		}
		included[key] = struct{}{}
		queue = append(queue, key)
	}

	for _, item := range allItems {
		key := backlogItemKey(string(item.Kind), item.Name)
		itemsByKey[key] = item
		if activeStatuses[item.Status] {
			enqueue(key)
		}
	}

	for _, rec := range records {
		enqueue(backlogItemKey(rec.BacklogKind, rec.BacklogName))
	}
	for _, activity := range activities {
		if activity.OwnerType == agentactivity.OwnerBacklog {
			enqueue(backlogItemKey(activity.OwnerKind, activity.OwnerName))
		}
	}

	if includeDependencies {
		for len(queue) > 0 {
			key := queue[0]
			queue = queue[1:]

			item, ok := itemsByKey[key]
			if !ok {
				continue
			}
			for _, dep := range item.DependsOn {
				enqueue(dep)
			}
		}
	}

	selected := make([]backlog.BacklogItem, 0, len(included))
	for _, item := range allItems {
		key := backlogItemKey(string(item.Kind), item.Name)
		if _, ok := included[key]; ok {
			selected = append(selected, item)
		}
	}

	return runtimeBacklogSelection{
		items: selected,
		keys:  included,
	}
}

func addActivityNodesAndEdges(
	nodes []Node,
	edges []Edge,
	activities []agentactivity.Record,
	executionIDs map[string]struct{},
	ownerNodeIDs map[string]struct{},
) ([]Node, []Edge) {
	runNodes := make(map[string]GraphRunNodeData)
	for _, activity := range activities {
		activityNode := buildActivityNode(activity)
		nodes = append(nodes, activityNode)

		targetID := ownerNodeID(activity)
		if _, ok := ownerNodeIDs[targetID]; targetID != "" && ok {
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("activity_for:%s->%s", activity.ActivityID, targetID),
				Source: activityNode.ID,
				Target: targetID,
				Type:   "activity_for",
			})
		}
		if activity.ExecutionID != "" {
			if _, ok := executionIDs[activity.ExecutionID]; ok {
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("records_activity:%s->%s", activity.ExecutionID, activity.ActivityID),
					Source: "execution-record/" + activity.ExecutionID,
					Target: activityNode.ID,
					Type:   "records_activity",
				})
			}
		}
		if strings.TrimSpace(activity.RunID) == "" {
			continue
		}

		runData, ok := runNodes[activity.RunID]
		if !ok || runData.Status == string(agentactivity.StatusComplete) || runData.Status == string(agentactivity.StatusCancelled) {
			runNodes[activity.RunID] = GraphRunNodeData{
				RunID:  activity.RunID,
				TaskID: activity.TaskID,
				Status: string(activity.Status),
			}
		}
		edgeType := "spawned_run"
		if activity.InteractionType == agentactivity.InteractionContinue {
			edgeType = "continued_run"
		}
		edges = append(edges, Edge{
			ID:     fmt.Sprintf("%s:%s->%s", edgeType, activity.ActivityID, activity.RunID),
			Source: activityNode.ID,
			Target: "run/" + activity.RunID,
			Type:   edgeType,
		})
	}
	for _, runNode := range runNodes {
		nodes = append(nodes, buildRunNode(runNode.RunID, runNode.TaskID, runNode.Status))
	}
	return nodes, edges
}

// buildOperations builds the operations lens projection.
// The operations lens shows actionable entities: backlog items that need
// attention, their active executions, initiatives (for grouping context),
// and active agent activities tied to those entities.
func (p *ProjectionService) buildOperations(ctx context.Context, focusNodeID string) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge
	agentAvailable := true

	// Load backlog items.
	var items []backlog.BacklogItem
	if p.backlog != nil {
		items, _ = p.backlog.LoadAll(nil)
	}

	// Load all execution records.
	var allRecords []execution.Record
	if p.execution != nil {
		records, err := p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			slog.Error("operations: execution list error", "error", err)
		} else {
			for _, rec := range records {
				if activeExecutionStatuses[rec.Status] {
					allRecords = append(allRecords, rec)
				}
			}
		}
	}

	// Check agent-manager availability (useful for frontend even without activity nodes).
	if p.activity != nil {
		agentAvailable = p.activity.IsAvailable(ctx)
	} else {
		agentAvailable = false
	}

	// Select actionable backlog items. The status gate in enqueue ensures
	// cross-referenced items (from executions) are also filtered by status.
	selectedBacklog := selectRuntimeBacklogItems(items, allRecords, nil, false, actionableBacklogStatuses)
	for _, item := range selectedBacklog.items {
		nodes = append(nodes, buildBacklogNode(item))
	}

	// Build execution nodes only for executions whose parent backlog is actionable.
	for _, rec := range allRecords {
		targetKey := backlogItemKey(rec.BacklogKind, rec.BacklogName)
		if _, ok := selectedBacklog.keys[targetKey]; !ok {
			continue
		}
		nodes = append(nodes, buildExecutionNode(rec))
		edges = append(edges, Edge{
			ID:     fmt.Sprintf("executes:%s->%s", rec.ExecutionID, targetKey),
			Source: "execution-record/" + rec.ExecutionID,
			Target: backlogItemNodeIDFromKey(targetKey),
			Type:   "executes",
		})
	}

	// Initiative nodes and member_of edges (placeholder for future actions).
	itemByKey := make(map[string]backlog.BacklogItem, len(items))
	for _, item := range items {
		itemByKey[backlogItemKey(string(item.Kind), item.Name)] = item
	}
	if p.initiative != nil {
		inits, err := p.initiative.List()
		if err != nil {
			slog.Error("operations: initiatives error", "error", err)
		} else {
			for _, init := range inits {
				if init.ArchivedAt != nil {
					continue
				}
				rollup := computeInitiativeRollup(init.Items, itemByKey)
				nodes = append(nodes, Node{
					ID:   "initiative/" + init.Name,
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
	for _, item := range selectedBacklog.items {
		if item.Initiative == "" {
			continue
		}
		key := backlogItemKey(string(item.Kind), item.Name)
		edges = append(edges, Edge{
			ID:     fmt.Sprintf("member_of:%s->%s", key, item.Initiative),
			Source: backlogItemNodeID(string(item.Kind), item.Name),
			Target: "initiative/" + item.Initiative,
			Type:   "member_of",
		})
	}

	// Load active agent activities whose owner is already in the graph.
	if p.activity != nil {
		allActivities, err := p.activity.List(ctx, agentactivity.ListFilters{ActiveOnly: true})
		if err != nil {
			slog.Error("operations: activity list error", "error", err)
		} else {
			// Build owner and execution ID sets from nodes already in the graph.
			ownerNodeIDs := make(map[string]struct{}, len(nodes))
			executionIDs := make(map[string]struct{})
			for _, n := range nodes {
				ownerNodeIDs[n.ID] = struct{}{}
				if n.Type == "ExecutionRecord" {
					executionIDs[n.ID[len("execution-record/"):]] = struct{}{}
				}
			}

			// Filter to activities whose owner is in the graph.
			var ownedActivities []agentactivity.Record
			for _, a := range allActivities {
				targetID := ownerNodeID(a)
				if _, ok := ownerNodeIDs[targetID]; targetID != "" && ok {
					ownedActivities = append(ownedActivities, a)
				}
			}

			nodes, edges = addActivityNodesAndEdges(nodes, edges, ownedActivities, executionIDs, ownerNodeIDs)
		}
	}

	resp := NewGraphResponse(LensOperations, nodes, edges)
	resp.Meta.AgentManagerAvailable = &agentAvailable
	if focusNodeID != "" {
		resp.Meta.FocusNodeID = focusNodeID
		nodeType, _, _ := parseFocusNodeID(focusNodeID)
		resp.Meta.FocusNodeType = nodeType
	}
	return resp, nil
}
