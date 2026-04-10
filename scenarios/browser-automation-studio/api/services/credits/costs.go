package credits

// OperationType defines the type of billable operation.
// Using typed constants prevents string typos and enables IDE autocomplete.
type OperationType string

const (
	// AI Operations - these use AI models and typically cost credits
	OpAIWorkflowGenerate OperationType = "ai.workflow_generate"
	OpAIWorkflowModify   OperationType = "ai.workflow_modify"
	OpAIElementAnalyze   OperationType = "ai.element_analyze"
	OpAIVisionNavigate   OperationType = "ai.vision_navigate"
	OpAICaptionGenerate  OperationType = "ai.caption_generate"

	// Execution Operations - running workflows
	OpExecutionRun       OperationType = "execution.run"
	OpExecutionScheduled OperationType = "execution.scheduled"

	// Export Operations - generating exports (free for now, revenue from tier limits)
	OpExportVideo OperationType = "export.video"
	OpExportGIF   OperationType = "export.gif"
	OpExportHTML  OperationType = "export.html"
	OpExportJSON  OperationType = "export.json"
)

// OperationCategory groups operations for display and configuration.
type OperationCategory string

const (
	CategoryAI        OperationCategory = "ai"
	CategoryExecution OperationCategory = "execution"
	CategoryExport    OperationCategory = "export"
)

// GetCategory returns the category for an operation type.
func (op OperationType) GetCategory() OperationCategory {
	switch op {
	case OpAIWorkflowGenerate, OpAIWorkflowModify, OpAIElementAnalyze,
		OpAIVisionNavigate, OpAICaptionGenerate:
		return CategoryAI
	case OpExecutionRun, OpExecutionScheduled:
		return CategoryExecution
	case OpExportVideo, OpExportGIF, OpExportHTML, OpExportJSON:
		return CategoryExport
	default:
		return ""
	}
}

// IsValid returns true if this is a known operation type.
func (op OperationType) IsValid() bool {
	return op.GetCategory() != ""
}

// OperationCosts defines the credit cost for each operation type.
// This structure supports configuration via environment variables.
type OperationCosts struct {
	// AI Operations
	AIWorkflowGenerate int `json:"ai_workflow_generate"`
	AIWorkflowModify   int `json:"ai_workflow_modify"`
	AIElementAnalyze   int `json:"ai_element_analyze"`
	AIVisionNavigate   int `json:"ai_vision_navigate"`
	AICaptionGenerate  int `json:"ai_caption_generate"`

	// Execution Operations
	ExecutionRun       int `json:"execution_run"`
	ExecutionScheduled int `json:"execution_scheduled"`

	// Export Operations
	ExportVideo int `json:"export_video"`
	ExportGIF   int `json:"export_gif"`
	ExportHTML  int `json:"export_html"`
	ExportJSON  int `json:"export_json"`
}

// DefaultOperationCosts returns the canonical credit costs for all operations.
// SECURITY: These costs are intentionally hard-coded. Do NOT add environment
// variable overrides - this would allow end-users to bypass credit charges
// by setting all operations to cost 0 credits.
func DefaultOperationCosts() OperationCosts {
	return OperationCosts{
		// AI operations - cost credits
		AIWorkflowGenerate: 1,
		AIWorkflowModify:   1,
		AIElementAnalyze:   1,
		AIVisionNavigate:   2, // Vision is more expensive (image processing)
		AICaptionGenerate:  1,

		// Execution operations - cost credits
		ExecutionRun:       1,
		ExecutionScheduled: 1,

		// Export operations - free for now (revenue comes from tier limits)
		ExportVideo: 0,
		ExportGIF:   0,
		ExportHTML:  0,
		ExportJSON:  0,
	}
}

// GetCost returns the credit cost for a specific operation.
func (c *OperationCosts) GetCost(op OperationType) int {
	switch op {
	case OpAIWorkflowGenerate:
		return c.AIWorkflowGenerate
	case OpAIWorkflowModify:
		return c.AIWorkflowModify
	case OpAIElementAnalyze:
		return c.AIElementAnalyze
	case OpAIVisionNavigate:
		return c.AIVisionNavigate
	case OpAICaptionGenerate:
		return c.AICaptionGenerate
	case OpExecutionRun:
		return c.ExecutionRun
	case OpExecutionScheduled:
		return c.ExecutionScheduled
	case OpExportVideo:
		return c.ExportVideo
	case OpExportGIF:
		return c.ExportGIF
	case OpExportHTML:
		return c.ExportHTML
	case OpExportJSON:
		return c.ExportJSON
	default:
		return 0 // Unknown operations are free (fail open)
	}
}

// AllOperationTypes returns all known operation types.
// Useful for generating documentation or configuration templates.
func AllOperationTypes() []OperationType {
	return []OperationType{
		OpAIWorkflowGenerate,
		OpAIWorkflowModify,
		OpAIElementAnalyze,
		OpAIVisionNavigate,
		OpAICaptionGenerate,
		OpExecutionRun,
		OpExecutionScheduled,
		OpExportVideo,
		OpExportGIF,
		OpExportHTML,
		OpExportJSON,
	}
}
