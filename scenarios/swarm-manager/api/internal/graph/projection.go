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

// Project builds a graph for the given lens.
func (p *ProjectionService) Project(ctx context.Context, lens Lens) (GraphResponse, error) {
	switch lens {
	case LensTopology:
		return p.buildTopology(ctx)
	case LensFlow:
		return p.buildFlow(ctx)
	case LensOperations:
		return p.buildOperations(ctx)
	default:
		return GraphResponse{}, fmt.Errorf("unknown lens: %s", lens)
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

// buildTopology builds the topology lens projection.
func (p *ProjectionService) buildTopology(ctx context.Context) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	// Load backlog items (exclude completed/archived).
	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
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
		nodes = append(nodes, Node{
			ID:   nodeID,
			Type: "BacklogItem",
			Data: GraphBacklogNodeData{
				Kind:     string(item.Kind),
				Name:     item.Name,
				Title:    item.Title,
				Status:   string(item.Status),
				Priority: int32(item.Priority),
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

// activeBacklogStatuses are backlog statuses considered "active" for the flow lens.
var activeBacklogStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusQueued:     true,
	backlog.StatusInProgress: true,
}

type executionSelection struct {
	records []execution.Record
	ids     map[string]struct{}
}

type runtimeBacklogSelection struct {
	items []backlog.BacklogItem
	keys  map[string]struct{}
}

type runtimeCaptureSelection struct {
	entries []CaptureEntry
	ids     map[string]struct{}
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

func buildCaptureNode(cap CaptureEntry) Node {
	return Node{
		ID:   "capture/" + cap.ID,
		Type: "Capture",
		Data: GraphCaptureNodeData{
			ID:     cap.ID,
			Text:   cap.Text,
			Status: cap.Status,
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

func selectExecutionRecordsForFlow(records []execution.Record) executionSelection {
	included := make(map[string]struct{}, len(records))
	selected := make([]execution.Record, 0, len(records))
	for _, rec := range records {
		included[rec.ExecutionID] = struct{}{}
		selected = append(selected, rec)
	}
	return executionSelection{
		records: selected,
		ids:     included,
	}
}

func selectRuntimeBacklogItems(
	allItems []backlog.BacklogItem,
	records []execution.Record,
	activities []agentactivity.Record,
	includeDependencies bool,
) runtimeBacklogSelection {
	itemsByKey := make(map[string]backlog.BacklogItem, len(allItems))
	included := make(map[string]struct{})
	queue := make([]string, 0)

	enqueue := func(key string) {
		if _, ok := itemsByKey[key]; !ok {
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
		if activeBacklogStatuses[item.Status] {
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

func selectRuntimeCaptures(allCaptures []CaptureEntry, activities []agentactivity.Record) runtimeCaptureSelection {
	included := make(map[string]struct{})
	for _, activity := range activities {
		if activity.OwnerType == agentactivity.OwnerCapture {
			included[activity.OwnerName] = struct{}{}
		}
	}
	selected := make([]CaptureEntry, 0, len(included))
	for _, cap := range allCaptures {
		if _, ok := included[cap.ID]; ok {
			selected = append(selected, cap)
		}
	}
	return runtimeCaptureSelection{
		entries: selected,
		ids:     included,
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

func appendTargetsEdges(
	edges []Edge,
	items []backlog.BacklogItem,
	scenarioNames map[string]bool,
) []Edge {
	for _, item := range items {
		for _, pattern := range item.AcceptanceAllow {
			for scenarioName := range scenarioNames {
				if matchesAcceptancePattern(pattern, scenarioName) {
					key := backlogItemKey(string(item.Kind), item.Name)
					edges = append(edges, Edge{
						ID:     fmt.Sprintf("targets:%s->%s", key, scenarioName),
						Source: backlogItemNodeID(string(item.Kind), item.Name),
						Target: "scenario/" + scenarioName,
						Type:   "targets",
					})
				}
			}
		}
	}
	return edges
}

func isOperationalScenarioStatus(status string) bool {
	switch status {
	case "running", "error":
		return true
	default:
		return false
	}
}

func selectOperationalScenarios(
	items []backlog.BacklogItem,
	scenarios []ScenarioEntry,
) ([]ScenarioEntry, []Edge) {
	if len(scenarios) == 0 {
		return nil, nil
	}

	selectedNames := make(map[string]struct{}, len(scenarios))
	edgeIDs := make(map[string]struct{})
	edges := make([]Edge, 0)

	for _, item := range items {
		sourceKey := backlogItemKey(string(item.Kind), item.Name)
		sourceID := backlogItemNodeID(string(item.Kind), item.Name)

		for _, pattern := range item.AcceptanceAllow {
			for _, scenario := range scenarios {
				if !matchesAcceptancePattern(pattern, scenario.Name) {
					continue
				}

				selectedNames[scenario.Name] = struct{}{}

				edgeID := fmt.Sprintf("targets:%s->%s", sourceKey, scenario.Name)
				if _, seen := edgeIDs[edgeID]; seen {
					continue
				}
				edgeIDs[edgeID] = struct{}{}
				edges = append(edges, Edge{
					ID:     edgeID,
					Source: sourceID,
					Target: "scenario/" + scenario.Name,
					Type:   "targets",
				})
			}
		}
	}

	selected := make([]ScenarioEntry, 0, len(scenarios))
	for _, scenario := range scenarios {
		_, targeted := selectedNames[scenario.Name]
		if targeted || isOperationalScenarioStatus(scenario.Status) {
			selected = append(selected, scenario)
		}
	}

	return selected, edges
}

// buildFlow builds the flow lens projection.
func (p *ProjectionService) buildFlow(ctx context.Context) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	// Load backlog items and include the closure required by tracked execution/activity history.
	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}
	var captures []CaptureEntry
	if p.capture != nil {
		captures, err = p.capture.ListCaptures()
		if err != nil {
			log.Printf("[graph] flow: capture list error: %v", err)
		}
	}
	var scenarios []ScenarioEntry
	if p.scenario != nil {
		scenarios, err = p.scenario.List(ctx)
		if err != nil {
			log.Printf("[graph] flow: scenario list error: %v", err)
		}
	}
	var records []execution.Record
	if p.execution != nil {
		records, err = p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			log.Printf("[graph] flow: execution list error: %v", err)
		}
	}
	var activities []agentactivity.Record
	if p.activity != nil {
		activities, err = p.activity.List(ctx, agentactivity.ListFilters{})
		if err != nil {
			log.Printf("[graph] flow: activity list error: %v", err)
		}
	}

	selectedExecutions := selectExecutionRecordsForFlow(records)
	selectedBacklog := selectRuntimeBacklogItems(items, selectedExecutions.records, activities, true)
	selectedCaptures := selectRuntimeCaptures(captures, activities)
	ownerNodeIDs := make(map[string]struct{})

	for _, item := range selectedBacklog.items {
		nodeID := backlogItemNodeID(string(item.Kind), item.Name)
		nodes = append(nodes, buildBacklogNode(item))
		ownerNodeIDs[nodeID] = struct{}{}
		for _, dep := range item.DependsOn {
			if _, ok := selectedBacklog.keys[dep]; !ok {
				continue
			}
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("depends_on:%s->%s", backlogItemKey(string(item.Kind), item.Name), dep),
				Source: nodeID,
				Target: backlogItemNodeIDFromKey(dep),
				Type:   "depends_on",
			})
		}
	}
	for _, cap := range selectedCaptures.entries {
		nodeID := "capture/" + cap.ID
		nodes = append(nodes, buildCaptureNode(cap))
		ownerNodeIDs[nodeID] = struct{}{}
	}
	scenarioNames := make(map[string]bool)
	for _, scenario := range scenarios {
		include := false
		for _, activity := range activities {
			if activity.OwnerType == agentactivity.OwnerScenario && activity.OwnerName == scenario.Name {
				include = true
				break
			}
		}
		if !include {
			for _, item := range selectedBacklog.items {
				for _, pattern := range item.AcceptanceAllow {
					if matchesAcceptancePattern(pattern, scenario.Name) {
						include = true
						break
					}
				}
				if include {
					break
				}
			}
		}
		if include {
			scenarioNames[scenario.Name] = true
			nodeID := "scenario/" + scenario.Name
			nodes = append(nodes, buildScenarioNode(scenario.Name, scenario.Status))
			ownerNodeIDs[nodeID] = struct{}{}
		}
	}

	for _, rec := range selectedExecutions.records {
		execNodeID := "execution-record/" + rec.ExecutionID
		nodes = append(nodes, buildExecutionNode(rec))

		targetKey := backlogItemKey(rec.BacklogKind, rec.BacklogName)
		if _, ok := selectedBacklog.keys[targetKey]; ok {
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("executes:%s->%s", rec.ExecutionID, targetKey),
				Source: execNodeID,
				Target: backlogItemNodeIDFromKey(targetKey),
				Type:   "executes",
			})
		}

		if rec.ParentExecutionID != "" {
			if _, ok := selectedExecutions.ids[rec.ParentExecutionID]; ok {
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("follow_up:%s->%s", rec.ExecutionID, rec.ParentExecutionID),
					Source: execNodeID,
					Target: "execution-record/" + rec.ParentExecutionID,
					Type:   "follow_up",
				})
			}
		}
	}

	edges = appendTargetsEdges(edges, selectedBacklog.items, scenarioNames)
	nodes, edges = addActivityNodesAndEdges(nodes, edges, activities, selectedExecutions.ids, ownerNodeIDs)

	return NewGraphResponse(LensFlow, nodes, edges), nil
}

// buildOperations builds the operations lens projection.
func (p *ProjectionService) buildOperations(ctx context.Context) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge
	agentAvailable := true

	// Scenario inventory. Operations intentionally includes only scenarios that
	// are operationally relevant to the active work graph.
	var scens []ScenarioEntry
	if p.scenario != nil {
		scenList, err := p.scenario.List(ctx)
		if err != nil {
			log.Printf("[graph] operations: scenarios error: %v", err)
		} else {
			scens = scenList
		}
	}

	var items []backlog.BacklogItem
	if p.backlog != nil {
		items, _ = p.backlog.LoadAll(nil)
	}
	var captures []CaptureEntry
	if p.capture != nil {
		captures, _ = p.capture.ListCaptures()
	}

	var activeRecords []execution.Record
	if p.execution != nil {
		records, err := p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			log.Printf("[graph] operations: execution list error: %v", err)
		} else {
			for _, rec := range records {
				if !activeExecutionStatuses[rec.Status] {
					continue
				}
				activeRecords = append(activeRecords, rec)
				nodes = append(nodes, buildExecutionNode(rec))
			}
		}
	}
	var activeActivities []agentactivity.Record
	if p.activity != nil {
		agentAvailable = p.activity.IsAvailable(ctx)
		activities, err := p.activity.List(ctx, agentactivity.ListFilters{ActiveOnly: true})
		if err != nil {
			log.Printf("[graph] operations: activity list error: %v", err)
		} else {
			activeActivities = activities
		}
	} else {
		agentAvailable = false
	}

	selectedBacklog := selectRuntimeBacklogItems(items, activeRecords, activeActivities, false)
	for _, item := range selectedBacklog.items {
		nodes = append(nodes, buildBacklogNode(item))
	}
	selectedCaptures := selectRuntimeCaptures(captures, activeActivities)
	for _, cap := range selectedCaptures.entries {
		nodes = append(nodes, buildCaptureNode(cap))
	}
	ownerNodeIDs := make(map[string]struct{})
	for _, item := range selectedBacklog.items {
		ownerNodeIDs[backlogItemNodeID(string(item.Kind), item.Name)] = struct{}{}
	}
	for _, cap := range selectedCaptures.entries {
		ownerNodeIDs["capture/"+cap.ID] = struct{}{}
	}

	selectedScenarios, targetEdges := selectOperationalScenarios(selectedBacklog.items, scens)
	for _, scenario := range selectedScenarios {
		nodes = append(nodes, buildScenarioNode(scenario.Name, scenario.Status))
		ownerNodeIDs["scenario/"+scenario.Name] = struct{}{}
	}
	for _, activity := range activeActivities {
		if activity.OwnerType != agentactivity.OwnerScenario {
			continue
		}
		nodeID := "scenario/" + activity.OwnerName
		if _, ok := ownerNodeIDs[nodeID]; ok {
			continue
		}
		nodes = append(nodes, buildScenarioNode(activity.OwnerName, "unknown"))
		ownerNodeIDs[nodeID] = struct{}{}
	}

	for _, rec := range activeRecords {
		targetKey := backlogItemKey(rec.BacklogKind, rec.BacklogName)
		if _, ok := selectedBacklog.keys[targetKey]; ok {
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("executes:%s->%s", rec.ExecutionID, targetKey),
				Source: "execution-record/" + rec.ExecutionID,
				Target: backlogItemNodeIDFromKey(targetKey),
				Type:   "executes",
			})
		}
	}

	edges = append(edges, targetEdges...)
	nodes, edges = addActivityNodesAndEdges(nodes, edges, activeActivities, selectedExecutionsIDs(activeRecords), ownerNodeIDs)

	resp := NewGraphResponse(LensOperations, nodes, edges)
	resp.Meta.AgentManagerAvailable = &agentAvailable
	return resp, nil
}

func selectedExecutionsIDs(records []execution.Record) map[string]struct{} {
	ids := make(map[string]struct{}, len(records))
	for _, record := range records {
		ids[record.ExecutionID] = struct{}{}
	}
	return ids
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
