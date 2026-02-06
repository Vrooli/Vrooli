// Package toolexecution provides tool routing decision helpers.
//
// This file extracts tool categorization and routing logic from executor.go
// into named, testable functions. This makes tool dispatch decisions explicit
// and easier to reason about.
package toolexecution

// ExecutorType identifies which domain executor should handle a tool.
type ExecutorType int

const (
	// ExecutorUnknown indicates an unrecognized tool.
	ExecutorUnknown ExecutorType = iota
	// ExecutorPipeline handles pipeline lifecycle tools.
	ExecutorPipeline
	// ExecutorSigning handles code signing tools.
	ExecutorSigning
	// ExecutorInspection handles status checking and validation tools.
	ExecutorInspection
	// ExecutorLegacy handles deprecated build/generation tools.
	ExecutorLegacy
)

// String returns the executor type name for logging and debugging.
func (e ExecutorType) String() string {
	switch e {
	case ExecutorPipeline:
		return "pipeline"
	case ExecutorSigning:
		return "signing"
	case ExecutorInspection:
		return "inspection"
	case ExecutorLegacy:
		return "legacy"
	default:
		return "unknown"
	}
}

// ToolCategory describes the functional category of a tool.
// Unlike ExecutorType which determines routing, ToolCategory describes
// the tool's purpose for documentation and analytics.
type ToolCategory string

const (
	// CategoryPipelineLifecycle for tools that manage pipeline execution.
	CategoryPipelineLifecycle ToolCategory = "pipeline_lifecycle"
	// CategoryBuildControl for tools that control build processes (legacy).
	CategoryBuildControl ToolCategory = "build_control"
	// CategorySigning for code signing operations.
	CategorySigning ToolCategory = "signing"
	// CategoryInspection for read-only status and validation.
	CategoryInspection ToolCategory = "inspection"
	// CategoryUnknown for unrecognized tools.
	CategoryUnknown ToolCategory = "unknown"
)

// GetToolExecutor returns the executor type for a given tool name.
// This centralizes the routing decision that was previously inline in Execute().
func GetToolExecutor(toolName string) ExecutorType {
	switch toolName {
	// Pipeline tools (preferred path for all builds)
	case "run_pipeline", "check_pipeline_status", "cancel_pipeline",
		"resume_pipeline", "list_pipelines":
		return ExecutorPipeline

	// Legacy build/generation tools (deprecated, use run_pipeline instead)
	case "generate_desktop_wrapper", "build_for_platform",
		"cancel_build", "list_builds":
		return ExecutorLegacy

	// Signing tools
	case "configure_signing", "sign_application", "verify_signature",
		"get_signing_status", "discover_certificates":
		return ExecutorSigning

	// Inspection tools
	case "check_build_status", "get_pipeline_status",
		"list_generated_wrappers", "validate_configuration",
		"get_system_prerequisites":
		return ExecutorInspection

	default:
		return ExecutorUnknown
	}
}

// GetToolCategory returns the functional category for a tool.
// This is useful for logging, analytics, and documentation.
func GetToolCategory(toolName string) ToolCategory {
	switch toolName {
	// Pipeline lifecycle
	case "run_pipeline", "check_pipeline_status", "cancel_pipeline",
		"resume_pipeline", "list_pipelines", "get_pipeline_status":
		return CategoryPipelineLifecycle

	// Build control (legacy)
	case "generate_desktop_wrapper", "build_for_platform",
		"cancel_build", "list_builds":
		return CategoryBuildControl

	// Signing
	case "configure_signing", "sign_application", "verify_signature",
		"get_signing_status", "discover_certificates":
		return CategorySigning

	// Inspection (read-only)
	case "check_build_status", "list_generated_wrappers",
		"validate_configuration", "get_system_prerequisites":
		return CategoryInspection

	default:
		return CategoryUnknown
	}
}

// IsToolKnown returns true if the tool name is recognized.
func IsToolKnown(toolName string) bool {
	return GetToolExecutor(toolName) != ExecutorUnknown
}

// IsToolDeprecated returns true if the tool is deprecated.
// Deprecated tools still work but agents should use alternatives.
func IsToolDeprecated(toolName string) bool {
	switch toolName {
	case "generate_desktop_wrapper", "build_for_platform",
		"cancel_build", "list_builds", "get_pipeline_status":
		return true
	default:
		return false
	}
}

// GetRecommendedTool returns the recommended replacement for deprecated tools.
// Returns empty string if the tool is not deprecated.
func GetRecommendedTool(toolName string) string {
	switch toolName {
	case "generate_desktop_wrapper", "build_for_platform":
		return "run_pipeline"
	case "cancel_build":
		return "cancel_pipeline"
	case "list_builds":
		return "list_pipelines"
	case "get_pipeline_status":
		return "check_pipeline_status"
	default:
		return ""
	}
}

// IsReadOnlyTool returns true if the tool only reads state (no mutations).
// Useful for permission checks and caching decisions.
func IsReadOnlyTool(toolName string) bool {
	switch toolName {
	case "check_pipeline_status", "list_pipelines", "check_build_status",
		"get_pipeline_status", "list_builds", "list_generated_wrappers",
		"validate_configuration", "get_system_prerequisites",
		"get_signing_status", "discover_certificates",
		"verify_signature":
		return true
	default:
		return false
	}
}
