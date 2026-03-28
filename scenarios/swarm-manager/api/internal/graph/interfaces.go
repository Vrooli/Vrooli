package graph

import (
	"context"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/scenarios"
)

// BacklogLister loads backlog items from the store.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// InitiativeLister lists initiatives with computed rollup status.
type InitiativeLister interface {
	List() ([]initiatives.InitiativeWithRollup, error)
}

// CaptureEntry represents a capture with its classification data.
type CaptureEntry struct {
	ID     string
	Text   string
	Status string
	Items  []CaptureClassificationItem
}

// CaptureClassificationItem is a classified backlog suggestion from a capture.
type CaptureClassificationItem struct {
	Kind  string
	Title string
}

// CaptureLister lists captures with their classification data.
type CaptureLister interface {
	ListCaptures() ([]CaptureEntry, error)
}

// ScenarioLister loads all scenarios.
type ScenarioLister interface {
	LoadAll() ([]scenarios.Scenario, error)
}

// ExecutionLister lists execution records with optional filters.
type ExecutionLister interface {
	List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error)
}

// RunStateGetter retrieves live run state from agent-manager.
type RunStateGetter interface {
	IsAvailable(ctx context.Context) bool
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
}

// Broadcaster sends real-time events to connected WebSocket clients.
type Broadcaster interface {
	BroadcastUpdate(event string, payload any)
}

// EventDispatcher emits graph change events to the broadcaster.
type EventDispatcher interface {
	DispatchNodeUpdate(nodeType, nodeID string, data any)
	DispatchEdgeChange(action string, edge Edge)
}
