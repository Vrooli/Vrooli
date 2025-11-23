package contracts

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionPlan represents the compiled workflow ready for orchestration. It
// deliberately omits engine-specific details so multiple engines can reuse the
// same plan without recompilation.
type ExecutionPlan struct {
	SchemaVersion  string                `json:"schema_version"`
	PayloadVersion string                `json:"payload_version"`
	ExecutionID    uuid.UUID             `json:"execution_id"`
	WorkflowID     uuid.UUID             `json:"workflow_id"`
	Instructions   []CompiledInstruction `json:"instructions"`
	Graph          *PlanGraph            `json:"graph,omitempty"` // Optional graph with branching/loop metadata.
	Metadata       map[string]any        `json:"metadata,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
}

// CompiledInstruction captures a single normalized instruction emitted by the
// workflow compiler. Params is intentionally generic and should follow the
// workflow schema; engines must not add provider-specific keys.
type CompiledInstruction struct {
	Index       int               `json:"index"`
	NodeID      string            `json:"node_id"`
	Type        string            `json:"type"`
	Params      map[string]any    `json:"params"`
	PreloadHTML string            `json:"preload_html,omitempty"`
	Context     map[string]any    `json:"context,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // Freeform, engine-agnostic hints (e.g., labels).
}

// PlanGraph preserves branching/loop metadata from the compiled workflow so
// the executor can follow the same control flow as the legacy client.
type PlanGraph struct {
	Steps []PlanStep `json:"steps"`
}

// PlanStep represents a node in the execution graph.
type PlanStep struct {
	Index     int               `json:"index"`
	NodeID    string            `json:"node_id"`
	Type      string            `json:"type"`
	Params    map[string]any    `json:"params,omitempty"`
	Outgoing  []PlanEdge        `json:"outgoing,omitempty"`
	Loop      *PlanGraph        `json:"loop,omitempty"` // Populated for loop nodes only.
	Metadata  map[string]string `json:"metadata,omitempty"`
	Context   map[string]any    `json:"context,omitempty"`
	Preload   string            `json:"preload_html,omitempty"`
	SourcePos map[string]any    `json:"source_position,omitempty"`
}

// PlanEdge represents a connection between two steps with optional condition labels.
type PlanEdge struct {
	ID         string `json:"id"`
	Target     string `json:"target"`
	Condition  string `json:"condition,omitempty"`
	SourcePort string `json:"source_port,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}
