package graph

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

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
	runState   RunStateGetter
}

// ProjectionConfig holds constructor dependencies for ProjectionService.
type ProjectionConfig struct {
	Backlog    BacklogLister
	Initiative InitiativeLister
	Capture    CaptureLister
	Scenario   ScenarioLister
	Execution  ExecutionLister
	RunState   RunStateGetter
}

// NewProjectionService creates a ProjectionService.
func NewProjectionService(cfg ProjectionConfig) *ProjectionService {
	return &ProjectionService{
		backlog:    cfg.Backlog,
		initiative: cfg.Initiative,
		capture:    cfg.Capture,
		scenario:   cfg.Scenario,
		execution:  cfg.Execution,
		runState:   cfg.RunState,
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
			Data: map[string]any{
				"kind":     string(item.Kind),
				"name":     item.Name,
				"title":    item.Title,
				"status":   string(item.Status),
				"priority": item.Priority,
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
					Data: map[string]any{
						"name":   init.Name,
						"title":  init.Title,
						"status": init.Status,
						"rollup": map[string]any{
							"total":       rollup.Total,
							"completed":   rollup.Completed,
							"in_progress": rollup.InProgress,
							"failed":      rollup.Failed,
							"pending":     rollup.Pending,
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
					Data: map[string]any{
						"id":     cap.ID,
						"text":   cap.Text,
						"status": cap.Status,
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
	if p.scenario != nil {
		scens, err := p.scenario.List(ctx)
		if err != nil {
			log.Printf("[graph] topology: scenarios error: %v", err)
		} else {
			for _, s := range scens {
				scenNodeID := "scenario/" + s.Name
				nodes = append(nodes, Node{
					ID:   scenNodeID,
					Type: "Scenario",
					Data: map[string]any{
						"name":   s.Name,
						"status": s.Status,
					},
				})
			}

			// targets edges: backlog items -> scenarios via acceptance_allow prefix match.
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
						}
					}
				}
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

func buildBacklogNode(item backlog.BacklogItem) Node {
	return Node{
		ID:   backlogItemNodeID(string(item.Kind), item.Name),
		Type: "BacklogItem",
		Data: map[string]any{
			"kind":     string(item.Kind),
			"name":     item.Name,
			"title":    item.Title,
			"status":   string(item.Status),
			"priority": item.Priority,
		},
	}
}

func buildExecutionNode(rec execution.Record) Node {
	return Node{
		ID:   "execution-record/" + rec.ExecutionID,
		Type: "ExecutionRecord",
		Data: map[string]any{
			"execution_id": rec.ExecutionID,
			"backlog_kind": rec.BacklogKind,
			"backlog_name": rec.BacklogName,
			"status":       string(rec.Status),
			"mode":         string(rec.Mode),
			"run_id":       rec.RunID,
		},
	}
}

func selectExecutionRecordsForFlow(records []execution.Record) executionSelection {
	recordsByID := make(map[string]execution.Record, len(records))
	included := make(map[string]struct{})

	for _, rec := range records {
		recordsByID[rec.ExecutionID] = rec
		if activeExecutionStatuses[rec.Status] {
			included[rec.ExecutionID] = struct{}{}
		}
	}

	for _, rec := range records {
		if _, ok := included[rec.ExecutionID]; !ok || rec.ParentExecutionID == "" {
			continue
		}
		if _, ok := recordsByID[rec.ParentExecutionID]; ok {
			included[rec.ParentExecutionID] = struct{}{}
		}
	}

	selected := make([]execution.Record, 0, len(included))
	for _, rec := range records {
		if _, ok := included[rec.ExecutionID]; ok {
			selected = append(selected, rec)
		}
	}

	return executionSelection{
		records: selected,
		ids:     included,
	}
}

func selectRuntimeBacklogItems(
	allItems []backlog.BacklogItem,
	records []execution.Record,
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

	// Load backlog items and include the closure required by active executions.
	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}
	var records []execution.Record
	if p.execution != nil {
		records, err = p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			log.Printf("[graph] flow: execution list error: %v", err)
		}
	}

	selectedExecutions := selectExecutionRecordsForFlow(records)
	selectedBacklog := selectRuntimeBacklogItems(items, selectedExecutions.records, true)

	for _, item := range selectedBacklog.items {
		nodeID := backlogItemNodeID(string(item.Kind), item.Name)
		nodes = append(nodes, buildBacklogNode(item))
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

	return NewGraphResponse(LensFlow, nodes, edges), nil
}

// buildOperations builds the operations lens projection.
func (p *ProjectionService) buildOperations(ctx context.Context) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge
	agentAvailable := true

	// Scenario inventory. Operations intentionally includes only scenarios that
	// are operationally relevant to the active work graph.
	var scens []struct{ name, status string }
	if p.scenario != nil {
		scenList, err := p.scenario.List(ctx)
		if err != nil {
			log.Printf("[graph] operations: scenarios error: %v", err)
		} else {
			for _, s := range scenList {
				scens = append(scens, struct{ name, status string }{s.Name, s.Status})
			}
		}
	}

	var items []backlog.BacklogItem
	if p.backlog != nil {
		items, _ = p.backlog.LoadAll(nil)
	}

	// Active execution records.
	type runRef struct {
		runID  string
		execID string
	}
	var runRefs []runRef
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
				if rec.RunID != "" {
					runRefs = append(runRefs, runRef{runID: rec.RunID, execID: rec.ExecutionID})
				}
			}
		}
	}

	selectedBacklog := selectRuntimeBacklogItems(items, activeRecords, false)
	for _, item := range selectedBacklog.items {
		nodes = append(nodes, buildBacklogNode(item))
	}

	selectedScenarios, targetEdges := selectOperationalScenarios(selectedBacklog.items, func() []ScenarioEntry {
		result := make([]ScenarioEntry, 0, len(scens))
		for _, s := range scens {
			result = append(result, ScenarioEntry{Name: s.name, Status: s.status})
		}
		return result
	}())
	for _, scenario := range selectedScenarios {
		nodes = append(nodes, Node{
			ID:   "scenario/" + scenario.Name,
			Type: "Scenario",
			Data: map[string]any{
				"name":   scenario.Name,
				"status": scenario.Status,
			},
		})
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

	// Run nodes from agent-manager.
	if p.runState != nil && len(runRefs) > 0 {
		if !p.runState.IsAvailable(ctx) {
			agentAvailable = false
		} else {
			var mu sync.Mutex
			var wg sync.WaitGroup
			sem := make(chan struct{}, 5) // bounded concurrency
			for _, ref := range runRefs {
				wg.Add(1)
				go func(r runRef) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					state, err := p.runState.GetRunState(ctx, r.runID)
					if err != nil {
						log.Printf("[graph] operations: run state %s error: %v", r.runID, err)
						return
					}
					runNodeID := "run/" + r.runID
					node := Node{
						ID:   runNodeID,
						Type: "Run",
						Data: map[string]any{
							"run_id":  state.RunID,
							"task_id": state.TaskID,
							"status":  state.Status,
						},
					}
					edge := Edge{
						ID:     fmt.Sprintf("spawned_run:%s->%s", r.execID, r.runID),
						Source: "execution-record/" + r.execID,
						Target: runNodeID,
						Type:   "spawned_run",
					}
					mu.Lock()
					nodes = append(nodes, node)
					edges = append(edges, edge)
					mu.Unlock()
				}(ref)
			}
			wg.Wait()
		}
	} else if p.runState == nil && len(runRefs) > 0 {
		agentAvailable = false
	}

	resp := NewGraphResponse(LensOperations, nodes, edges)
	resp.Meta.AgentManagerAvailable = &agentAvailable
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
