package toolexecution

import "testing"

func TestGetToolExecutor(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected ExecutorType
	}{
		// Pipeline tools
		{name: "run_pipeline", toolName: "run_pipeline", expected: ExecutorPipeline},
		{name: "check_pipeline_status", toolName: "check_pipeline_status", expected: ExecutorPipeline},
		{name: "cancel_pipeline", toolName: "cancel_pipeline", expected: ExecutorPipeline},
		{name: "resume_pipeline", toolName: "resume_pipeline", expected: ExecutorPipeline},
		{name: "list_pipelines", toolName: "list_pipelines", expected: ExecutorPipeline},

		// Legacy tools
		{name: "generate_desktop_wrapper", toolName: "generate_desktop_wrapper", expected: ExecutorLegacy},
		{name: "build_for_platform", toolName: "build_for_platform", expected: ExecutorLegacy},
		{name: "cancel_build", toolName: "cancel_build", expected: ExecutorLegacy},
		{name: "list_builds", toolName: "list_builds", expected: ExecutorLegacy},

		// Signing tools
		{name: "configure_signing", toolName: "configure_signing", expected: ExecutorSigning},
		{name: "sign_application", toolName: "sign_application", expected: ExecutorSigning},
		{name: "verify_signature", toolName: "verify_signature", expected: ExecutorSigning},
		{name: "get_signing_status", toolName: "get_signing_status", expected: ExecutorSigning},
		{name: "discover_certificates", toolName: "discover_certificates", expected: ExecutorSigning},

		// Distribution tools
		{name: "upload_artifact", toolName: "upload_artifact", expected: ExecutorDistribution},
		{name: "publish_release", toolName: "publish_release", expected: ExecutorDistribution},
		{name: "list_artifacts", toolName: "list_artifacts", expected: ExecutorDistribution},
		{name: "list_distribution_targets", toolName: "list_distribution_targets", expected: ExecutorDistribution},
		{name: "validate_distribution_target", toolName: "validate_distribution_target", expected: ExecutorDistribution},
		{name: "check_distribution_status", toolName: "check_distribution_status", expected: ExecutorDistribution},

		// Inspection tools
		{name: "check_build_status", toolName: "check_build_status", expected: ExecutorInspection},
		{name: "get_pipeline_status", toolName: "get_pipeline_status", expected: ExecutorInspection},
		{name: "list_generated_wrappers", toolName: "list_generated_wrappers", expected: ExecutorInspection},
		{name: "validate_configuration", toolName: "validate_configuration", expected: ExecutorInspection},
		{name: "get_system_prerequisites", toolName: "get_system_prerequisites", expected: ExecutorInspection},

		// Unknown tools
		{name: "unknown_tool", toolName: "unknown_tool", expected: ExecutorUnknown},
		{name: "empty string", toolName: "", expected: ExecutorUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetToolExecutor(tt.toolName)
			if result != tt.expected {
				t.Errorf("GetToolExecutor(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestGetToolCategory(t *testing.T) {
	tests := []struct {
		toolName string
		expected ToolCategory
	}{
		{"run_pipeline", CategoryPipelineLifecycle},
		{"check_pipeline_status", CategoryPipelineLifecycle},
		{"list_pipelines", CategoryPipelineLifecycle},
		{"get_pipeline_status", CategoryPipelineLifecycle},
		{"generate_desktop_wrapper", CategoryBuildControl},
		{"build_for_platform", CategoryBuildControl},
		{"configure_signing", CategorySigning},
		{"sign_application", CategorySigning},
		{"upload_artifact", CategoryDistribution},
		{"publish_release", CategoryDistribution},
		{"check_build_status", CategoryInspection},
		{"validate_configuration", CategoryInspection},
		{"unknown_tool", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := GetToolCategory(tt.toolName)
			if result != tt.expected {
				t.Errorf("GetToolCategory(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestIsToolKnown(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"run_pipeline", true},
		{"check_pipeline_status", true},
		{"configure_signing", true},
		{"upload_artifact", true},
		{"unknown_tool", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := IsToolKnown(tt.toolName)
			if result != tt.expected {
				t.Errorf("IsToolKnown(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestIsToolDeprecated(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		// Deprecated tools
		{"generate_desktop_wrapper", true},
		{"build_for_platform", true},
		{"cancel_build", true},
		{"list_builds", true},
		{"get_pipeline_status", true},

		// Non-deprecated tools
		{"run_pipeline", false},
		{"check_pipeline_status", false},
		{"configure_signing", false},
		{"unknown_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := IsToolDeprecated(tt.toolName)
			if result != tt.expected {
				t.Errorf("IsToolDeprecated(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestGetRecommendedTool(t *testing.T) {
	tests := []struct {
		toolName string
		expected string
	}{
		{"generate_desktop_wrapper", "run_pipeline"},
		{"build_for_platform", "run_pipeline"},
		{"cancel_build", "cancel_pipeline"},
		{"list_builds", "list_pipelines"},
		{"get_pipeline_status", "check_pipeline_status"},
		{"run_pipeline", ""},
		{"unknown_tool", ""},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := GetRecommendedTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("GetRecommendedTool(%q) = %q, want %q", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestIsReadOnlyTool(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		// Read-only tools
		{"check_pipeline_status", true},
		{"list_pipelines", true},
		{"check_build_status", true},
		{"list_builds", true},
		{"list_generated_wrappers", true},
		{"validate_configuration", true},
		{"get_system_prerequisites", true},
		{"get_signing_status", true},
		{"discover_certificates", true},
		{"list_artifacts", true},
		{"verify_signature", true},

		// Mutating tools
		{"run_pipeline", false},
		{"cancel_pipeline", false},
		{"configure_signing", false},
		{"sign_application", false},
		{"upload_artifact", false},
		{"publish_release", false},
		{"generate_desktop_wrapper", false},
		{"build_for_platform", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := IsReadOnlyTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("IsReadOnlyTool(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestExecutorTypeString(t *testing.T) {
	tests := []struct {
		executor ExecutorType
		expected string
	}{
		{ExecutorPipeline, "pipeline"},
		{ExecutorSigning, "signing"},
		{ExecutorDistribution, "distribution"},
		{ExecutorInspection, "inspection"},
		{ExecutorLegacy, "legacy"},
		{ExecutorUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.executor.String()
			if result != tt.expected {
				t.Errorf("ExecutorType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}
