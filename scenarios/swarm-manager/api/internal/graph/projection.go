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

// ProjectionService builds lens-specific graph projections from data sources.
type ProjectionService struct {
	backlog       BacklogLister
	initiative    InitiativeLister
	capture       CaptureLister
	scenario      ScenarioLister
	execution     ExecutionLister
	activity      AgentActivityLister
	operatingMode OperatingModeReader
}

// ProjectionConfig holds constructor dependencies for ProjectionService.
type ProjectionConfig struct {
	Backlog       BacklogLister
	Initiative    InitiativeLister
	Capture       CaptureLister
	Scenario      ScenarioLister
	Execution     ExecutionLister
	Activity      AgentActivityLister
	OperatingMode OperatingModeReader
}

// NewProjectionService creates a ProjectionService.
func NewProjectionService(cfg ProjectionConfig) *ProjectionService {
	return &ProjectionService{
		backlog:       cfg.Backlog,
		initiative:    cfg.Initiative,
		capture:       cfg.Capture,
		scenario:      cfg.Scenario,
		execution:     cfg.Execution,
		activity:      cfg.Activity,
		operatingMode: cfg.OperatingMode,
	}
}

// SetOperatingModeReader wires the active-rounds reader after construction.
// The graph routes are registered before the operating-mode service exists,
// so this setter lets `registerOperatingModeRoutes` attach the reader once
// it's built. Safe to call once; subsequent calls overwrite the reader.
func (p *ProjectionService) SetOperatingModeReader(r OperatingModeReader) {
	if p == nil {
		return
	}
	p.operatingMode = r
}

// loadActiveRounds returns the bulk active-round map. A nil reader (tests
// that don't wire it) or a reader error degrades gracefully — the caller
// gets an empty map and continues building the projection. Reader errors
// are logged so operators can see drift; the graph stays useful without
// active-round info.
func (p *ProjectionService) loadActiveRounds(ctx context.Context) map[string]OperatingModeActiveRound {
	if p.operatingMode == nil {
		return map[string]OperatingModeActiveRound{}
	}
	rounds, err := p.operatingMode.ActiveRoundsByInitiative(ctx)
	if err != nil {
		slog.Warn("graph: active-rounds read failed; omitting active-round chips", "error", err)
		return map[string]OperatingModeActiveRound{}
	}
	if rounds == nil {
		return map[string]OperatingModeActiveRound{}
	}
	return rounds
}

// Project builds a graph for the given lens and optional focus.
func (p *ProjectionService) Project(ctx context.Context, params ProjectionParams) (GraphResponse, error) {
	switch params.Lens {
	case LensTopology:
		return p.buildTopology(ctx)
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
		case backlog.StatusFailed, backlog.StatusNeedsFollowup:
			rollup.Failed++
		case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching,
			backlog.StatusInReview, backlog.StatusReviewPending:
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
	execution.StatusStarting:    2,
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

// backlogItemNodeIDFromKey converts "kind/name" to the full node ID.
func backlogItemNodeIDFromKey(key string) string {
	return "backlog-item/" + key
}
