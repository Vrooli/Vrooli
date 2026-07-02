package graph

import (
	"context"
	"fmt"
	"log/slog"

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
	operatingMode OperatingModeReader
}

// ProjectionConfig holds constructor dependencies for ProjectionService.
type ProjectionConfig struct {
	Backlog       BacklogLister
	Initiative    InitiativeLister
	Capture       CaptureLister
	Scenario      ScenarioLister
	Execution     ExecutionLister
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

// Project builds a graph for the given lens. Topology is the only
// projection lens; the Focus lens is a client-side filter over topology
// data, and the Plan lens has its own endpoint (GET /api/v1/plan).
func (p *ProjectionService) Project(ctx context.Context, params ProjectionParams) (GraphResponse, error) {
	switch params.Lens {
	case LensTopology:
		return p.buildTopology(ctx)
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

// activeExecutionStatuses are execution statuses considered "active" when
// annotating backlog nodes with in-flight run summaries.
var activeExecutionStatuses = map[execution.Status]bool{
	execution.StatusPending:     true,
	execution.StatusStarting:    true,
	execution.StatusRunning:     true,
	execution.StatusNeedsReview: true,
	execution.StatusValidating:  true,
	execution.StatusNeedsFixup:  true,
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

// backlogItemNodeIDFromKey converts "kind/name" to the full node ID.
func backlogItemNodeIDFromKey(key string) string {
	return "backlog-item/" + key
}
