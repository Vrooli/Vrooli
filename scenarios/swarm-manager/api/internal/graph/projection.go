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
		return p.buildTopology()
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

// buildTopology builds the topology lens projection.
func (p *ProjectionService) buildTopology() (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	// Load backlog items (exclude completed/archived).
	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}
	itemIndex := make(map[string]bool, len(items))
	for _, item := range items {
		if item.Status == backlog.StatusCompleted || item.Status == backlog.StatusArchived {
			continue
		}
		key := backlogItemKey(string(item.Kind), item.Name)
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
			for _, iwr := range inits {
				init := iwr.Initiative
				if init.Status == "archived" {
					continue
				}
				initNodeID := "initiative/" + init.Name
				nodes = append(nodes, Node{
					ID:   initNodeID,
					Type: "Initiative",
					Data: map[string]any{
						"name":   init.Name,
						"title":  init.Title,
						"status": init.Status,
						"rollup": map[string]any{
							"total":       iwr.Rollup.Total,
							"completed":   iwr.Rollup.Completed,
							"in_progress": iwr.Rollup.InProgress,
							"failed":      iwr.Rollup.Failed,
							"pending":     iwr.Rollup.Pending,
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
		scens, err := p.scenario.LoadAll()
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
						"status": string(s.Status),
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

// buildFlow builds the flow lens projection.
func (p *ProjectionService) buildFlow(ctx context.Context) (GraphResponse, error) {
	var nodes []Node
	var edges []Edge

	// Load active backlog items.
	items, err := p.backlog.LoadAll(nil)
	if err != nil {
		return GraphResponse{}, fmt.Errorf("load backlog: %w", err)
	}
	for _, item := range items {
		if !activeBacklogStatuses[item.Status] {
			continue
		}
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
		for _, dep := range item.DependsOn {
			edges = append(edges, Edge{
				ID:     fmt.Sprintf("depends_on:%s->%s", backlogItemKey(string(item.Kind), item.Name), dep),
				Source: nodeID,
				Target: backlogItemNodeIDFromKey(dep),
				Type:   "depends_on",
			})
		}
	}

	// Load active execution records.
	if p.execution != nil {
		records, err := p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			log.Printf("[graph] flow: execution list error: %v", err)
		} else {
			for _, rec := range records {
				if !activeExecutionStatuses[rec.Status] {
					continue
				}
				execNodeID := "execution-record/" + rec.ExecutionID
				nodes = append(nodes, Node{
					ID:   execNodeID,
					Type: "ExecutionRecord",
					Data: map[string]any{
						"execution_id": rec.ExecutionID,
						"backlog_kind": rec.BacklogKind,
						"backlog_name": rec.BacklogName,
						"status":       string(rec.Status),
						"mode":         string(rec.Mode),
						"run_id":       rec.RunID,
					},
				})

				// executes edge: execution -> backlog item.
				targetKey := backlogItemKey(rec.BacklogKind, rec.BacklogName)
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("executes:%s->%s", rec.ExecutionID, targetKey),
					Source: execNodeID,
					Target: backlogItemNodeIDFromKey(targetKey),
					Type:   "executes",
				})

				// follow_up edge: execution -> parent execution.
				if rec.ParentExecutionID != "" {
					edges = append(edges, Edge{
						ID:     fmt.Sprintf("follow_up:%s->%s", rec.ExecutionID, rec.ParentExecutionID),
						Source: execNodeID,
						Target: "execution-record/" + rec.ParentExecutionID,
						Type:   "follow_up",
					})
				}
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

	// Scenario nodes.
	var scens []struct{ name, status string }
	if p.scenario != nil {
		scenList, err := p.scenario.LoadAll()
		if err != nil {
			log.Printf("[graph] operations: scenarios error: %v", err)
		} else {
			for _, s := range scenList {
				scenNodeID := "scenario/" + s.Name
				nodes = append(nodes, Node{
					ID:   scenNodeID,
					Type: "Scenario",
					Data: map[string]any{
						"name":   s.Name,
						"status": string(s.Status),
					},
				})
				scens = append(scens, struct{ name, status string }{s.Name, string(s.Status)})
			}
		}
	}

	// Active execution records.
	type runRef struct {
		runID  string
		execID string
	}
	var runRefs []runRef

	if p.execution != nil {
		records, err := p.execution.List(ctx, execution.ListFilters{})
		if err != nil {
			log.Printf("[graph] operations: execution list error: %v", err)
		} else {
			// Load backlog items for targets edges.
			var items []backlog.BacklogItem
			if p.backlog != nil {
				items, _ = p.backlog.LoadAll(nil)
			}

			for _, rec := range records {
				if !activeExecutionStatuses[rec.Status] {
					continue
				}
				execNodeID := "execution-record/" + rec.ExecutionID
				nodes = append(nodes, Node{
					ID:   execNodeID,
					Type: "ExecutionRecord",
					Data: map[string]any{
						"execution_id": rec.ExecutionID,
						"backlog_kind": rec.BacklogKind,
						"backlog_name": rec.BacklogName,
						"status":       string(rec.Status),
						"mode":         string(rec.Mode),
						"run_id":       rec.RunID,
					},
				})

				// executes edge.
				targetKey := backlogItemKey(rec.BacklogKind, rec.BacklogName)
				edges = append(edges, Edge{
					ID:     fmt.Sprintf("executes:%s->%s", rec.ExecutionID, targetKey),
					Source: execNodeID,
					Target: backlogItemNodeIDFromKey(targetKey),
					Type:   "executes",
				})

				if rec.RunID != "" {
					runRefs = append(runRefs, runRef{runID: rec.RunID, execID: rec.ExecutionID})
				}
			}

			// targets edges from active backlog items to scenarios.
			scenNames := make(map[string]bool, len(scens))
			for _, s := range scens {
				scenNames[s.name] = true
			}
			for _, item := range items {
				if !activeBacklogStatuses[item.Status] && item.Status != backlog.StatusQueued {
					continue
				}
				for _, pattern := range item.AcceptanceAllow {
					for sName := range scenNames {
						if matchesAcceptancePattern(pattern, sName) {
							key := backlogItemKey(string(item.Kind), item.Name)
							edges = append(edges, Edge{
								ID:     fmt.Sprintf("targets:%s->%s", key, sName),
								Source: backlogItemNodeID(string(item.Kind), item.Name),
								Target: "scenario/" + sName,
								Type:   "targets",
							})
						}
					}
				}
			}
		}
	}

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
