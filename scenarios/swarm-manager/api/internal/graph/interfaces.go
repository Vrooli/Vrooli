package graph

import (
	"context"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// BacklogLister loads backlog items from the store.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// InitiativeEntry is the graph-specific initiative view needed by projections.
type InitiativeEntry struct {
	Name   string
	Title  string
	Status string
	Items  []string
}

// InitiativeLister lists initiatives needed by graph projections.
type InitiativeLister interface {
	List() ([]InitiativeEntry, error)
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

// ScenarioEntry is the graph-specific scenario view needed by projections.
type ScenarioEntry struct {
	Name   string
	Status string
}

// ScenarioLister loads the graph scenario inventory.
type ScenarioLister interface {
	List(ctx context.Context) ([]ScenarioEntry, error)
}

// ExecutionLister lists execution records with optional filters.
type ExecutionLister interface {
	List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error)
}

// AgentActivityLister retrieves tracked agent activity records.
type AgentActivityLister interface {
	List(ctx context.Context, filters agentactivity.ListFilters) ([]agentactivity.Record, error)
	IsAvailable(ctx context.Context) bool
}

// Broadcaster sends real-time events to connected WebSocket clients.
type Broadcaster interface {
	BroadcastUpdate(event string, payload any)
}

// EventDispatcher emits graph change events to the broadcaster.
type EventDispatcher interface {
	DispatchNodeUpdate(nodeType, nodeID string, data any)
	DispatchEdgeChange(action string, edge Edge)
	DispatchInvalidate(lenses ...string)
	DispatchInvalidateWithFocus(focusNodeID string)
}

// Projector builds or serves graph projections for a lens.
type Projector interface {
	Project(ctx context.Context, params ProjectionParams) (GraphResponse, error)
}

// CacheInvalidator clears cached graph projections for one or more lenses.
type CacheInvalidator interface {
	Invalidate(lenses ...Lens)
	InvalidateFocus(focusNodeID string)
}
