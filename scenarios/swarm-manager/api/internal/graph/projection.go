package graph

import (
	"context"
	"fmt"
	"log"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// ProjectionService builds lens-specific graph projections from data sources.
type ProjectionService struct {
	backlog    BacklogLister
	initiative InitiativeLister
	capture    CaptureLister
	scenario   ScenarioLister
	execution  ExecutionLister
	activity   AgentActivityLister
}

// ProjectionConfig holds constructor dependencies for ProjectionService.
type ProjectionConfig struct {
	Backlog    BacklogLister
	Initiative InitiativeLister
	Capture    CaptureLister
	Scenario   ScenarioLister
	Execution  ExecutionLister
	Activity   AgentActivityLister
}

// NewProjectionService creates a ProjectionService.
func NewProjectionService(cfg ProjectionConfig) *ProjectionService {
	return &ProjectionService{
		backlog:    cfg.Backlog,
		initiative: cfg.Initiative,
		capture:    cfg.Capture,
		scenario:   cfg.Scenario,
		execution:  cfg.Execution,
		activity:   cfg.Activity,
	}
}

// Project builds a graph for the given lens and optional focus.
func (p *ProjectionService) Project(ctx context.Context, params ProjectionParams) (GraphResponse, error) {
	switch params.Lens {
	case LensTopology:
		return p.buildTopology(ctx)
	case LensFlow:
		return p.buildFlow(ctx, params.FocusNodeID)
	case LensOperations:
		return p.buildOperations(ctx, params.FocusNodeID)
	default:
		return GraphResponse{}, fmt.Errorf("unknown lens: %s", params.Lens)
	}
}

// backlogItemNodeID returns the canonical node ID for a backlog item.
func backlogItemNodeID(kind, name string) string {
	return "backlog-item/" + kind + "/" + name
}

// backlogItemKey returns the "kind/name" key used for dependency references.
func backlogItemKey(kind, name string) string {
	return kind + "/" + name
}

type initiativeRollup struct {
	Total      int
	Completed  int
	InProgress int
	Failed     int
	Pending    int
}

func computeInitiativeRollup(items []string, itemByKey map[string]backlog.BacklogItem) initiativeRollup {
	rollup := initiativeRollup{
		Total: len(items),
	}

	for _, ref := range items {
		item, ok := itemByKey[ref]
		if !ok {
			rollup.Pending++
			continue
		}

		switch item.Status {
		case backlog.StatusCompleted:
			rollup.Completed++
		case backlog.StatusFailed:
			rollup.Failed++
		case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching:
			rollup.InProgress++
		default:
			rollup.Pending++
		}
	}

	return rollup
}

// executionStatusPriority defines the display priority for active execution statuses.
// Higher values are shown when multiple executions are active.
var executionStatusPriority = map[execution.Status]int{
	execution.StatusPending:     1,
	execution.StatusScheduled:   2,
	execution.StatusStarting:    3,
	execution.StatusNeedsFixup:  4,
	execution.StatusValidating:  5,
	execution.StatusNeedsReview: 6,
	execution.StatusRunning:     7,
}

// computeActiveExecutionSummary returns the most relevant active execution
// status and the count of active executions for a given backlog item key.
func computeActiveExecutionSummary(backlogKey string, records []execution.Record) (status string, count int) {
	bestPriority := 0
	for _, rec := range records {
		recKey := backlogItemKey(rec.BacklogKind, rec.BacklogName)
		if recKey != backlogKey {
			continue
		}
		if !activeExecutionStatuses[rec.Status] {
			continue
		}
		count++
		if p := executionStatusPriority[rec.Status]; p > bestPriority {
			bestPriority = p
			status = string(rec.Status)
		}
	}
	return status, count
}

// parseFocusNodeID extracts the node type and identifying parts from a canonical node ID.
// Returns ("backlog", [kind, name]) or ("initiative", [name]) or ("scenario", [name]).
func parseFocusNodeID(nodeID string) (nodeType string, parts []string, err error) {
	if strings.HasPrefix(nodeID, "backlog-item/") {
		rest := strings.TrimPrefix(nodeID, "backlog-item/")
		slashIdx := strings.Index(rest, "/")
		if slashIdx < 0 || slashIdx == len(rest)-1 {
			return "", nil, fmt.Errorf("invalid backlog focus: %s", nodeID)
		}
		return "backlog", []string{rest[:slashIdx], rest[slashIdx+1:]}, nil
	}
	if strings.HasPrefix(nodeID, "initiative/") {
		name := strings.TrimPrefix(nodeID, "initiative/")
		if name == "" {
			return "", nil, fmt.Errorf("empty initiative name in focus: %s", nodeID)
		}
		return "initiative", []string{name}, nil
	}
	if strings.HasPrefix(nodeID, "scenario/") {
		name := strings.TrimPrefix(nodeID, "scenario/")
		if name == "" {
			return "", nil, fmt.Errorf("empty scenario name in focus: %s", nodeID)
		}
		return "scenario", []string{name}, nil
	}
	return "", nil, fmt.Errorf("unsupported focus node type: %s", nodeID)
}

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
		if item.Status == backlog.StatusCompleted || item.Status == backlog.StatusArchived {
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
			log.Printf("[graph] topology: initiatives error: %v", err)
		} else {
			for _, init := range inits {
				if init.Status == "archived" {
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
		if item.Status == backlog.StatusCompleted || item.Status == backlog.StatusArchived {
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
			log.Printf("[graph] topology: captures error: %v", err)
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
			log.Printf("[graph] topology: scenarios error: %v", err)
		} else {
			scenByName := make(map[string]ScenarioEntry, len(scens))
			for _, s := range scens {
				scenByName[s.Name] = s
			}

			// Build targets edges first to discover which scenarios are referenced.
			referencedScenarios := make(map[string]struct{})
			for _, item := range items {
				if item.Status == backlog.StatusCompleted || item.Status == backlog.StatusArchived {
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

// activeExecutionStatuses are statuses considered "active" for flow/operations lenses.
var activeExecutionStatuses = map[execution.Status]bool{
	execution.StatusPending:     true,
	execution.StatusScheduled:   true,
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

func buildBacklogNode(item backlog.BacklogItem) Node {
	return Node{
		ID:   backlogItemNodeID(string(item.Kind), item.Name),
		Type: "BacklogItem",
		Data: GraphBacklogNodeData{
			Kind:     string(item.Kind),
			Name:     item.Name,
			Title:    item.Title,
			Status:   string(item.Status),
			Priority: int32(item.Priority),
		},
	}
}

func buildScenarioNode(name, status string) Node {
	return Node{
		ID:   "scenario/" + name,
		Type: "Scenario",
		Data: GraphScenarioNodeData{
			Name:   name,
			Status: status,
		},
	}
}

func buildExecutionNode(rec execution.Record) Node {
	return Node{
		ID:   "execution-record/" + rec.ExecutionID,
		Type: "ExecutionRecord",
		Data: GraphExecutionNodeData{
			ExecutionID: rec.ExecutionID,
			BacklogKind: rec.BacklogKind,
			BacklogName: rec.BacklogName,
			Status:      string(rec.Status),
			Mode:        string(rec.Mode),
			RunID:       rec.RunID,
		},
	}
}

func buildActivityNode(rec agentactivity.Record) Node {
	return Node{
		ID:   activityNodeID(rec.ActivityID),
		Type: "AgentActivity",
		Data: GraphAgentActivityNodeData{
			ActivityID:      rec.ActivityID,
			OwnerType:       string(rec.OwnerType),
			OwnerKind:       rec.OwnerKind,
			OwnerName:       rec.OwnerName,
			OwnerTitle:      rec.OwnerTitle,
			ExecutionID:     rec.ExecutionID,
			Purpose:         string(rec.Purpose),
			InteractionType: string(rec.InteractionType),
			Status:          string(rec.Status),
			RunID:           rec.RunID,
			TaskID:          rec.TaskID,
			RequestedAt:     rec.RequestedAt,
		},
	}
}

func buildRunNode(runID, taskID, status string) Node {
	return Node{
		ID:   "run/" + runID,
		Type: "Run",
		Data: GraphRunNodeData{
			RunID:  runID,
			TaskID: taskID,
			Status: status,
		},
	}
}

func activityNodeID(activityID string) string {
	return "agent-activity/" + activityID
}

func ownerNodeID(record agentactivity.Record) string {
	switch record.OwnerType {
	case agentactivity.OwnerBacklog:
		return backlogItemNodeID(record.OwnerKind, record.OwnerName)
	case agentactivity.OwnerCapture:
		return "capture/" + record.OwnerName
	case agentactivity.OwnerScenario:
		return "scenario/" + record.OwnerName
	default:
		return ""
	}
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

// buildFlow builds the flow lens projection, focused on a specific node.
func (p *ProjectionService) buildFlow(ctx context.Context, focusNodeID string) (GraphResponse, error) {
	if focusNodeID == "" {
		resp := NewGraphResponse(LensFlow, nil, nil)
		resp.Meta.Hint = "Select a node in the topology view to see its execution history"
		return resp, nil
	}

	nodeType, parts, err := parseFocusNodeID(focusNodeID)
	if err != nil {
		return GraphResponse{}, err
	}

	var resp GraphResponse
	switch nodeType {
	case "backlog":
		resp, err = p.buildFlowForBacklogItem(ctx, parts[0], parts[1])
	case "initiative":
		resp, err = p.buildFlowForInitiative(ctx, parts[0])
	case "scenario":
		resp, err = p.buildFlowForScenario(ctx, parts[0])
	default:
		return GraphResponse{}, fmt.Errorf("unsupported focus node type: %s", nodeType)
	}
	if err != nil {
		return GraphResponse{}, err
	}
	resp.Meta.FocusNodeID = focusNodeID
	resp.Meta.FocusNodeType = nodeType
	return resp, nil
}

// buildFlowForBacklogItem builds focused flow for a single backlog item:
// the item + all its ExecutionRecords + their AgentActivities + Runs.
func (p *ProjectionService) buildFlowForBacklogItem(ctx context.Context, kind, name string) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}
	targetKey := backlogItemKey(kind, name)
	var focusItem *backlog.BacklogItem
	for i := range items {
		if backlogItemKey(string(items[i].Kind), items[i].Name) == targetKey {
			focusItem = &items[i]
			break
		}
	}
	if focusItem == nil {
		resp := NewGraphResponse(LensFlow, nil, nil)
		resp.Meta.Hint = "Backlog item not found"
		return resp, nil
	}

	nodes = append(nodes, buildBacklogNode(*focusItem))
	ownerNodeIDs := map[string]struct{}{
		backlogItemNodeID(kind, name): {},
	}

	// Load all execution records for this item.
	var records []execution.Record
	if p.execution != nil {
		allRecords, err := p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			log.Printf("[graph] flow/backlog: execution list error: %v", err)
		} else {
			for _, rec := range allRecords {
				if backlogItemKey(rec.BacklogKind, rec.BacklogName) == targetKey {
					records = append(records, rec)
				}
			}
		}
	}

	execIDs := make(map[string]struct{}, len(records))
	for _, rec := range records {
		nodes = append(nodes, buildExecutionNode(rec))
		execIDs[rec.ExecutionID] = struct{}{}
		edges = append(edges, Edge{
			ID:     fmt.Sprintf("executes:%s->%s", rec.ExecutionID, targetKey),
			Source: "execution-record/" + rec.ExecutionID,
			Target: backlogItemNodeID(kind, name),
			Type:   "executes",
		})
		if rec.ParentExecutionID != "" {
			if _, ok := execIDs[rec.ParentExecutionID]; ok {
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("follow_up:%s->%s", rec.ExecutionID, rec.ParentExecutionID),
					Source: "execution-record/" + rec.ExecutionID,
					Target: "execution-record/" + rec.ParentExecutionID,
					Type:   "follow_up",
				})
			}
		}
	}

	// Load agent activities for this item.
	var activities []agentactivity.Record
	if p.activity != nil {
		allActivities, err := p.activity.List(ctx, agentactivity.ListFilters{})
		if err != nil {
			log.Printf("[graph] flow/backlog: activity list error: %v", err)
		} else {
			for _, a := range allActivities {
				if a.OwnerType == agentactivity.OwnerBacklog &&
					backlogItemKey(a.OwnerKind, a.OwnerName) == targetKey {
					activities = append(activities, a)
				}
			}
		}
	}

	nodes, edges = addActivityNodesAndEdges(nodes, edges, activities, execIDs, ownerNodeIDs)
	return NewGraphResponse(LensFlow, nodes, edges), nil
}

// buildFlowForInitiative builds focused flow for an initiative:
// the initiative + its member backlog items with execution summaries.
func (p *ProjectionService) buildFlowForInitiative(ctx context.Context, name string) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	if p.initiative == nil {
		resp := NewGraphResponse(LensFlow, nil, nil)
		resp.Meta.Hint = "Initiative not found"
		return resp, nil
	}
	inits, err := p.initiative.List()
	if err != nil {
		return GraphResponse{}, fmt.Errorf("list initiatives: %w", err)
	}
	var focusInit *InitiativeEntry
	for i := range inits {
		if inits[i].Name == name {
			focusInit = &inits[i]
			break
		}
	}
	if focusInit == nil {
		resp := NewGraphResponse(LensFlow, nil, nil)
		resp.Meta.Hint = "Initiative not found"
		return resp, nil
	}

	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}
	itemByKey := make(map[string]backlog.BacklogItem, len(items))
	for _, item := range items {
		itemByKey[backlogItemKey(string(item.Kind), item.Name)] = item
	}

	var execRecords []execution.Record
	if p.execution != nil {
		execRecords, _ = p.execution.List(ctx, execution.ListFilters{})
	}

	rollup := computeInitiativeRollup(focusInit.Items, itemByKey)
	nodes = append(nodes, Node{
		ID:   "initiative/" + focusInit.Name,
		Type: "Initiative",
		Data: GraphInitiativeNodeData{
			Name:   focusInit.Name,
			Title:  focusInit.Title,
			Status: focusInit.Status,
			Rollup: GraphInitiativeRollup{
				Total:      int32(rollup.Total),
				Completed:  int32(rollup.Completed),
				InProgress: int32(rollup.InProgress),
				Failed:     int32(rollup.Failed),
				Pending:    int32(rollup.Pending),
			},
		},
	})

	for _, memberKey := range focusInit.Items {
		item, ok := itemByKey[memberKey]
		if !ok {
			continue
		}
		activeStatus, activeCount := computeActiveExecutionSummary(memberKey, execRecords)
		nodes = append(nodes, Node{
			ID:   backlogItemNodeIDFromKey(memberKey),
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
		edges = append(edges, Edge{
			ID:     fmt.Sprintf("member_of:%s->%s", memberKey, focusInit.Name),
			Source: backlogItemNodeIDFromKey(memberKey),
			Target: "initiative/" + focusInit.Name,
			Type:   "member_of",
		})
	}

	return NewGraphResponse(LensFlow, nodes, edges), nil
}

// buildFlowForScenario builds focused flow for a scenario:
// the scenario + all backlog items targeting it with execution summaries.
func (p *ProjectionService) buildFlowForScenario(ctx context.Context, name string) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	var scens []ScenarioEntry
	if p.scenario != nil {
		scens, _ = p.scenario.List(ctx)
	}
	var focusScen *ScenarioEntry
	for i := range scens {
		if scens[i].Name == name {
			focusScen = &scens[i]
			break
		}
	}
	if focusScen == nil {
		resp := NewGraphResponse(LensFlow, nil, nil)
		resp.Meta.Hint = "Scenario not found"
		return resp, nil
	}

	nodes = append(nodes, buildScenarioNode(focusScen.Name, focusScen.Status))

	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}

	var execRecords []execution.Record
	if p.execution != nil {
		execRecords, _ = p.execution.List(ctx, execution.ListFilters{})
	}

	for _, item := range items {
		if item.Status == backlog.StatusCompleted || item.Status == backlog.StatusArchived {
			continue
		}
		for _, pattern := range item.AcceptanceAllow {
			if matchesAcceptancePattern(pattern, name) {
				key := backlogItemKey(string(item.Kind), item.Name)
				activeStatus, activeCount := computeActiveExecutionSummary(key, execRecords)
				nodeID := backlogItemNodeID(string(item.Kind), item.Name)
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
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("targets:%s->%s", key, name),
					Source: nodeID,
					Target: "scenario/" + name,
					Type:   "targets",
				})
				break // Only one targets edge per item
			}
		}
	}

	return NewGraphResponse(LensFlow, nodes, edges), nil
}

// buildOperations builds the operations lens projection.
// The operations lens shows only actionable entities: backlog items that need
// attention, their active executions, and initiatives (for grouping context).
// Scenarios, agent activities, runs, and captures are excluded.
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
			log.Printf("[graph] operations: execution list error: %v", err)
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
			log.Printf("[graph] operations: initiatives error: %v", err)
		} else {
			for _, init := range inits {
				if init.Status == "archived" {
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

	resp := NewGraphResponse(LensOperations, nodes, edges)
	resp.Meta.AgentManagerAvailable = &agentAvailable
	if focusNodeID != "" {
		resp.Meta.FocusNodeID = focusNodeID
		nodeType, _, _ := parseFocusNodeID(focusNodeID)
		resp.Meta.FocusNodeType = nodeType
	}
	return resp, nil
}

// backlogItemNodeIDFromKey converts "kind/name" to the full node ID.
func backlogItemNodeIDFromKey(key string) string {
	return "backlog-item/" + key
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
