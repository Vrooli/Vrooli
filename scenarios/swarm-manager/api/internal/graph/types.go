// Package graph provides graph projection logic for the swarm-manager API.
// It assembles nodes and edges from multiple data sources (backlog, initiatives,
// captures, scenarios, executions, agent runs) and projects them through
// lens-specific filters for the graph workspace UI.
package graph

import "time"

// Lens represents a graph projection lens.
type Lens string

const (
	LensTopology   Lens = "topology"
	LensFlow       Lens = "flow"
	LensOperations Lens = "operations"
)

// ValidateLens returns true if l is a known lens value.
func ValidateLens(l Lens) bool {
	switch l {
	case LensTopology, LensFlow, LensOperations:
		return true
	default:
		return false
	}
}

// Node is a React Flow-compatible graph node.
type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Data     any      `json:"data"`
	Position Position `json:"position"`
}

// Position is a 2D coordinate for node placement. Always {0,0} server-side;
// client-side Dagre computes actual layout.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Edge is a React Flow-compatible graph edge.
type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// Meta provides metadata about the graph response.
type Meta struct {
	Lens                  Lens   `json:"lens"`
	NodeCount             int    `json:"node_count"`
	EdgeCount             int    `json:"edge_count"`
	GeneratedAt           string `json:"generated_at"`
	AgentManagerAvailable *bool  `json:"agent_manager_available,omitempty"`
}

// GraphResponse is the top-level response for the graph endpoint.
type GraphResponse struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	Meta  Meta   `json:"meta"`
}

// NewGraphResponse builds a GraphResponse with computed meta.
func NewGraphResponse(lens Lens, nodes []Node, edges []Edge) GraphResponse {
	if nodes == nil {
		nodes = []Node{}
	}
	if edges == nil {
		edges = []Edge{}
	}
	return GraphResponse{
		Nodes: nodes,
		Edges: edges,
		Meta: Meta{
			Lens:        lens,
			NodeCount:   len(nodes),
			EdgeCount:   len(edges),
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// WSMessage is a WebSocket message envelope.
type WSMessage struct {
	Type      string `json:"type"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// WebSocket message types.
const (
	WSFullSync   = "full-sync"
	WSNodeUpdate = "node-update"
	WSNodeAdd    = "node-add"
	WSNodeRemove = "node-remove"
	WSEdgeAdd    = "edge-add"
	WSEdgeRemove = "edge-remove"
	WSHeartbeat  = "heartbeat"
)
