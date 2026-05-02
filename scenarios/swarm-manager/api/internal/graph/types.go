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
	LensOperations Lens = "operations"
)

// ValidateLens returns true if l is a known lens value.
func ValidateLens(l Lens) bool {
	switch l {
	case LensTopology, LensOperations:
		return true
	default:
		return false
	}
}

// AllLenses returns the supported graph lenses in stable order.
func AllLenses() []Lens {
	return []Lens{LensTopology, LensOperations}
}

// GraphBacklogNodeData describes a backlog item node payload.
type GraphBacklogNodeData struct {
	Kind                  string `json:"kind"`
	Name                  string `json:"name"`
	Title                 string `json:"title"`
	Status                string `json:"status"`
	Priority              int32  `json:"priority"`
	ActiveExecutionStatus string `json:"active_execution_status,omitempty"`
	ActiveExecutionCount  int32  `json:"active_execution_count"`
}

// GraphInitiativeRollup describes initiative member status counts.
type GraphInitiativeRollup struct {
	Total      int32 `json:"total"`
	Completed  int32 `json:"completed"`
	InProgress int32 `json:"in_progress"`
	Failed     int32 `json:"failed"`
	Pending    int32 `json:"pending"`
}

// GraphInitiativeNodeData describes an initiative node payload.
type GraphInitiativeNodeData struct {
	Name          string                      `json:"name"`
	Title         string                      `json:"title"`
	Status        string                      `json:"status"`
	OperatingMode string                      `json:"operating_mode,omitempty"`
	ActiveRound   *GraphInitiativeActiveRound `json:"active_round,omitempty"`
	Rollup        GraphInitiativeRollup       `json:"rollup"`
}

// GraphInitiativeActiveRound surfaces the first non-terminal round of an
// initiative's operating mode so the workspace graph can render a phase chip
// + pulse without re-deriving state from per-initiative workspace endpoints.
type GraphInitiativeActiveRound struct {
	Mode   string `json:"mode"`
	Phase  string `json:"phase"`
	Round  int    `json:"round"`
	Status string `json:"status"`
}

// GraphCaptureNodeData describes a capture node payload.
type GraphCaptureNodeData struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Status string `json:"status"`
}

// GraphScenarioNodeData describes a scenario node payload.
type GraphScenarioNodeData struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GraphExecutionNodeData describes an execution node payload.
type GraphExecutionNodeData struct {
	ExecutionID string `json:"execution_id"`
	BacklogKind string `json:"backlog_kind"`
	BacklogName string `json:"backlog_name"`
	Status      string `json:"status"`
	Mode        string `json:"mode"`
	RunID       string `json:"run_id,omitempty"`
}

// GraphAgentActivityNodeData describes a tracked activity node payload.
type GraphAgentActivityNodeData struct {
	ActivityID      string `json:"activity_id"`
	OwnerType       string `json:"owner_type"`
	OwnerKind       string `json:"owner_kind,omitempty"`
	OwnerName       string `json:"owner_name"`
	OwnerTitle      string `json:"owner_title,omitempty"`
	ExecutionID     string `json:"execution_id,omitempty"`
	Purpose         string `json:"purpose"`
	InteractionType string `json:"interaction_type"`
	Status          string `json:"status"`
	RunID           string `json:"run_id,omitempty"`
	TaskID          string `json:"task_id,omitempty"`
	RequestedAt     string `json:"requested_at"`
}

// GraphRunNodeData describes an agent-manager run node payload.
type GraphRunNodeData struct {
	RunID  string `json:"run_id"`
	TaskID string `json:"task_id,omitempty"`
	Status string `json:"status"`
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
	FocusNodeID           string `json:"focus_node_id,omitempty"`
	FocusNodeType         string `json:"focus_node_type,omitempty"`
	Hint                  string `json:"hint,omitempty"`
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

// NodeDataToProtoKind identifies the proto oneof variant for node data.
func NodeDataToProtoKind(data any) string {
	switch data.(type) {
	case GraphBacklogNodeData, *GraphBacklogNodeData:
		return "backlog"
	case GraphInitiativeNodeData, *GraphInitiativeNodeData:
		return "initiative"
	case GraphCaptureNodeData, *GraphCaptureNodeData:
		return "capture"
	case GraphScenarioNodeData, *GraphScenarioNodeData:
		return "scenario"
	case GraphExecutionNodeData, *GraphExecutionNodeData:
		return "execution"
	case GraphAgentActivityNodeData, *GraphAgentActivityNodeData:
		return "activity"
	case GraphRunNodeData, *GraphRunNodeData:
		return "run"
	default:
		return ""
	}
}

// WSMessage is a WebSocket message envelope.
type WSMessage struct {
	Type      string `json:"type"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// ProjectionParams holds all parameters for a projection request.
type ProjectionParams struct {
	Lens        Lens
	FocusNodeID string // Optional for operations lens.
}

// InvalidationPayload identifies which graph lenses should refresh.
type InvalidationPayload struct {
	Lenses      []Lens `json:"lenses"`
	FocusNodeID string `json:"focus_node_id,omitempty"`
}

// WebSocket message types.
const (
	WSFullSync   = "full-sync"
	WSNodeUpdate = "node-update"
	WSNodeAdd    = "node-add"
	WSNodeRemove = "node-remove"
	WSEdgeAdd    = "edge-add"
	WSEdgeRemove = "edge-remove"
	WSInvalidate = "invalidate"
	WSHeartbeat  = "heartbeat"
)
