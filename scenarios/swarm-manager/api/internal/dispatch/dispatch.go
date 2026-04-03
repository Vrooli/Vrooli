// Package dispatch defines shared interfaces for graph event dispatching.
//
// Mutating services use these interfaces to notify the graph subsystem of
// state changes. The graph package provides the concrete implementation
// (graph.Dispatch) that broadcasts over WebSocket and invalidates caches.
//
// Three tiers exist, each embedding the tier below:
//
//	Invalidator      — DispatchInvalidate only (backlog, captures, initiatives)
//	NodeDispatcher   — adds DispatchNodeUpdate (execution, agentactivity, scenarios)
//
// The graph package defines its own full EventDispatcher with additional
// methods (DispatchEdgeChange, DispatchInvalidateWithFocus) that are
// internal to the graph subsystem.
package dispatch

// Invalidator notifies the graph subsystem that cached projections for the
// given lenses are stale and should be rebuilt on the next read.
type Invalidator interface {
	DispatchInvalidate(lenses ...string)
}

// NodeDispatcher extends Invalidator with the ability to push individual
// node updates over WebSocket for real-time UI refresh.
type NodeDispatcher interface {
	Invalidator
	DispatchNodeUpdate(nodeType, nodeID string, data any)
}
