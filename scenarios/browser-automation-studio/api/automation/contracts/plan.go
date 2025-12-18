package contracts

import (
	"time"

	"github.com/google/uuid"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

const (
	// ExecutionPlanSchemaVersion tracks the shape of ExecutionPlan and PlanGraph payloads.
	// Bump when plan or graph fields change so executors and engines can assert compatibility.
	ExecutionPlanSchemaVersion = "automation-plan-v1"
)

// =============================================================================
// NATIVE GO PLAN TYPES
// =============================================================================
// These types use time.Time and uuid.UUID which require conversion to/from proto.
// For proto interoperability, use ProtoExecutionPlan, ProtoCompiledInstruction,
// etc. defined in contracts.go with conversion helpers.
//
// TYPE DISTINCTION: PlanStep vs CompiledInstruction
// -------------------------------------------------
// ExecutionPlan contains TWO representations of workflow steps:
//
//  1. Instructions ([]CompiledInstruction) - FLAT list for sequential execution
//     - Ordered by step index for linear execution
//     - Used by engines that execute steps one at a time
//     - Simpler structure, no graph metadata
//
//  2. Graph (*PlanGraph with []PlanStep) - DAG for complex control flow
//     - Preserves branching, loops, and conditional edges
//     - Contains Outgoing edges for graph traversal
//     - Contains nested Loop *PlanGraph for loop bodies
//     - Used by executor for control flow decisions
//
// Both representations describe the same steps but serve different purposes:
// - The flat Instructions list is for step-by-step execution
// - The Graph is for control flow decisions (which step comes next)
//
// Conversion helpers planStepToInstruction() and planStepToInstructionStep()
// in flow_utils.go convert between these representations.
//
// NOTE: The proto types use string for UUIDs and google.protobuf.Timestamp for
// time fields. The native Go types below provide a more idiomatic Go interface.

// ExecutionPlan represents the compiled workflow ready for orchestration. It
// deliberately omits engine-specific details so multiple engines can reuse the
// same plan without recompilation.
//
// For proto serialization, use ProtoExecutionPlan from contracts.go.
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
// workflow compiler. The Action field provides type-safe access to step type
// and parameters via proto-generated types.
type CompiledInstruction struct {
	Index       int               `json:"index"`
	NodeID      string            `json:"node_id"`
	PreloadHTML string            `json:"preload_html,omitempty"`
	Context     map[string]any    `json:"context,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // Freeform, engine-agnostic hints (e.g., labels).
	// Action is the typed action definition with full type safety.
	Action *basactions.ActionDefinition `json:"action,omitempty"`
}

// PlanGraph preserves branching/loop metadata from the compiled workflow so
// the executor can follow the same control flow as the legacy client.
type PlanGraph struct {
	Steps []PlanStep `json:"steps"`
}

// PlanStep represents a node in the execution graph. The Action field provides
// type-safe access to step type and parameters via proto-generated types.
type PlanStep struct {
	Index     int               `json:"index"`
	NodeID    string            `json:"node_id"`
	Outgoing  []PlanEdge        `json:"outgoing,omitempty"`
	Loop      *PlanGraph        `json:"loop,omitempty"` // Populated for loop nodes only.
	Metadata  map[string]string `json:"metadata,omitempty"`
	Context   map[string]any    `json:"context,omitempty"`
	Preload   string            `json:"preload_html,omitempty"`
	SourcePos map[string]any    `json:"source_position,omitempty"`
	// Action is the typed action definition with full type safety.
	Action *basactions.ActionDefinition `json:"action,omitempty"`
}

// PlanEdge represents a connection between two steps with optional condition labels.
type PlanEdge struct {
	ID         string `json:"id"`
	Target     string `json:"target"`
	Condition  string `json:"condition,omitempty"`
	SourcePort string `json:"source_port,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}
