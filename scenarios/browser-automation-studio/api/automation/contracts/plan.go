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

	// ExecutionPlanSchemaVersionV2 is the schema version for multi-page aware plans.
	// V2 plans include page definitions and page IDs on instructions.
	ExecutionPlanSchemaVersionV2 = "automation-plan-v2"
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
//
// Multi-Page Support (v2):
// Plans with SchemaVersion = ExecutionPlanSchemaVersionV2 include page definitions
// and page IDs on instructions. This enables playback of workflows that span
// multiple browser tabs/pages.
type ExecutionPlan struct {
	SchemaVersion  string                `json:"schema_version"`
	PayloadVersion string                `json:"payload_version"`
	ExecutionID    uuid.UUID             `json:"execution_id"`
	WorkflowID     uuid.UUID             `json:"workflow_id"`
	Instructions   []CompiledInstruction `json:"instructions"`
	Graph          *PlanGraph            `json:"graph,omitempty"`    // Optional graph with branching/loop metadata.
	Pages          []PageDefinition      `json:"pages,omitempty"`    // V2: Page definitions for multi-page workflows.
	Timeline       []TimelineEvent       `json:"timeline,omitempty"` // V2: Unified timeline including page events.
	Metadata       map[string]any        `json:"metadata,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
}

// PageDefinition describes a page/tab in a multi-page workflow.
// During playback, the executor uses these definitions to match runtime pages
// to recorded pages.
type PageDefinition struct {
	ID          uuid.UUID  `json:"id"`                    // Stable page ID from recording.
	IsInitial   bool       `json:"isInitial"`             // True for the first page in the workflow.
	StartURL    string     `json:"startUrl,omitempty"`    // Expected URL (for initial pages).
	OpenerID    *uuid.UUID `json:"openerId,omitempty"`    // ID of the page that opened this one.
	OpenTrigger string     `json:"openTrigger,omitempty"` // How page was opened: "link_click", "popup", "window.open".
	URLPattern  string     `json:"urlPattern,omitempty"`  // Optional URL regex pattern for matching.
}

// TimelineEvent represents an event in the unified timeline (actions + page events).
// This is used during playback to maintain chronological ordering.
type TimelineEvent struct {
	ID        uuid.UUID         `json:"id"`
	Type      TimelineEventType `json:"type"` // "action" | "page_created" | "page_navigated" | "page_closed"
	PageID    uuid.UUID         `json:"pageId"`
	Timestamp time.Time         `json:"timestamp"`
	// For action events
	InstructionIndex *int `json:"instructionIndex,omitempty"`
	// For page events
	URL      string     `json:"url,omitempty"`
	Title    string     `json:"title,omitempty"`
	OpenerID *uuid.UUID `json:"openerId,omitempty"`
}

// TimelineEventType identifies the type of timeline event.
type TimelineEventType string

const (
	TimelineEventAction        TimelineEventType = "action"
	TimelineEventPageCreated   TimelineEventType = "page_created"
	TimelineEventPageNavigated TimelineEventType = "page_navigated"
	TimelineEventPageClosed    TimelineEventType = "page_closed"
)

// CompiledInstruction captures a single normalized instruction emitted by the
// workflow compiler. The Action field provides type-safe access to step type
// and parameters via proto-generated types.
type CompiledInstruction struct {
	Index       int               `json:"index"`
	NodeID      string            `json:"node_id"`
	PageID      *uuid.UUID        `json:"page_id,omitempty"` // V2: Page this instruction belongs to.
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
	PageID    *uuid.UUID        `json:"page_id,omitempty"` // V2: Page this step belongs to.
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

// =============================================================================
// PLAN MIGRATION HELPERS
// =============================================================================

// MigratePlanV1ToV2 upgrades a v1 plan (no page tracking) to v2 format by
// adding a single implicit page definition and assigning all instructions to it.
// This maintains backward compatibility with workflows recorded before multi-page support.
func MigratePlanV1ToV2(plan *ExecutionPlan) *ExecutionPlan {
	if plan == nil {
		return nil
	}
	// Already v2?
	if plan.SchemaVersion == ExecutionPlanSchemaVersionV2 {
		return plan
	}
	// Create implicit page for all instructions
	implicitPageID := uuid.New()
	implicitPage := PageDefinition{
		ID:        implicitPageID,
		IsInitial: true,
		StartURL:  extractStartURL(plan),
	}
	plan.Pages = []PageDefinition{implicitPage}
	// Assign all instructions to the implicit page
	for i := range plan.Instructions {
		plan.Instructions[i].PageID = &implicitPageID
	}
	// Assign all graph steps to the implicit page
	if plan.Graph != nil {
		for i := range plan.Graph.Steps {
			plan.Graph.Steps[i].PageID = &implicitPageID
		}
	}
	plan.SchemaVersion = ExecutionPlanSchemaVersionV2
	return plan
}

// extractStartURL extracts the start URL from plan metadata or first navigate instruction.
func extractStartURL(plan *ExecutionPlan) string {
	// Try metadata first
	if plan.Metadata != nil {
		if url, ok := plan.Metadata["startUrl"].(string); ok && url != "" {
			return url
		}
	}
	// Look for first navigate instruction
	for _, instr := range plan.Instructions {
		if instr.Action != nil && instr.Action.GetNavigate() != nil {
			return instr.Action.GetNavigate().GetUrl()
		}
	}
	return ""
}

// IsMultiPagePlan returns true if the plan has multiple pages defined.
func IsMultiPagePlan(plan *ExecutionPlan) bool {
	return plan != nil && len(plan.Pages) > 1
}

// GetPlanPage returns the page definition for the given page ID.
func GetPlanPage(plan *ExecutionPlan, pageID uuid.UUID) *PageDefinition {
	if plan == nil {
		return nil
	}
	for i := range plan.Pages {
		if plan.Pages[i].ID == pageID {
			return &plan.Pages[i]
		}
	}
	return nil
}

// GetInitialPage returns the initial page definition from the plan.
func GetInitialPage(plan *ExecutionPlan) *PageDefinition {
	if plan == nil {
		return nil
	}
	for i := range plan.Pages {
		if plan.Pages[i].IsInitial {
			return &plan.Pages[i]
		}
	}
	// Fallback to first page
	if len(plan.Pages) > 0 {
		return &plan.Pages[0]
	}
	return nil
}
