package toolregistry

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

func TestPipelineToolProvider_Name(t *testing.T) {
	provider := NewPipelineToolProvider()
	if provider.Name() != "scenario-to-desktop-pipeline" {
		t.Errorf("expected name 'scenario-to-desktop-pipeline', got %q", provider.Name())
	}
}

func TestPipelineToolProvider_Categories(t *testing.T) {
	provider := NewPipelineToolProvider()
	categories := provider.Categories(context.Background())

	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]
	if cat.Id != "pipeline" {
		t.Errorf("expected category ID 'pipeline', got %q", cat.Id)
	}
	if cat.Name != "Pipeline Orchestration" {
		t.Errorf("expected category name 'Pipeline Orchestration', got %q", cat.Name)
	}
}

func TestPipelineToolProvider_Tools(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	expectedTools := map[string]bool{
		"run_pipeline":          false,
		"check_pipeline_status": false,
		"cancel_pipeline":       false,
		"resume_pipeline":       false,
		"list_pipelines":        false,
	}

	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	for _, tool := range tools {
		if _, ok := expectedTools[tool.Name]; !ok {
			t.Errorf("unexpected tool: %s", tool.Name)
		} else {
			expectedTools[tool.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool not found: %s", name)
		}
	}
}

func TestRunPipelineTool(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	var runPipeline *toolspb.ToolDefinition
	for _, tool := range tools {
		if tool.Name == "run_pipeline" {
			runPipeline = tool
			break
		}
	}

	if runPipeline == nil {
		t.Fatalf("run_pipeline tool not found")
	}

	t.Run("has correct category", func(t *testing.T) {
		if runPipeline.Category != "pipeline" {
			t.Errorf("expected category 'pipeline', got %q", runPipeline.Category)
		}
	})

	t.Run("has required scenario_name parameter", func(t *testing.T) {
		if runPipeline.Parameters == nil {
			t.Fatalf("parameters is nil")
		}
		if len(runPipeline.Parameters.Required) == 0 {
			t.Fatalf("no required parameters")
		}
		found := false
		for _, req := range runPipeline.Parameters.Required {
			if req == "scenario_name" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("scenario_name should be required")
		}
	})

	t.Run("has expected parameters", func(t *testing.T) {
		expectedParams := []string{
			"scenario_name", "platforms", "deployment_mode", "stop_after_stage",
			"skip_preflight", "skip_smoke_test", "proxy_url", "template_type",
			"distribute", "distribution_targets", "sign", "clean", "version",
		}
		for _, param := range expectedParams {
			if _, ok := runPipeline.Parameters.Properties[param]; !ok {
				t.Errorf("expected parameter %q not found", param)
			}
		}
	})

	t.Run("is marked as long-running", func(t *testing.T) {
		if runPipeline.Metadata == nil {
			t.Fatalf("metadata is nil")
		}
		if !runPipeline.Metadata.LongRunning {
			t.Errorf("expected LongRunning to be true")
		}
	})

	t.Run("has async behavior configured", func(t *testing.T) {
		if runPipeline.Metadata.AsyncBehavior == nil {
			t.Fatalf("async behavior is nil")
		}
		if runPipeline.Metadata.AsyncBehavior.StatusPolling == nil {
			t.Fatalf("status polling is nil")
		}
		if runPipeline.Metadata.AsyncBehavior.StatusPolling.StatusTool != "check_pipeline_status" {
			t.Errorf("expected status tool 'check_pipeline_status', got %q",
				runPipeline.Metadata.AsyncBehavior.StatusPolling.StatusTool)
		}
	})

	t.Run("has cancellation configured", func(t *testing.T) {
		if runPipeline.Metadata.AsyncBehavior.Cancellation == nil {
			t.Fatalf("cancellation is nil")
		}
		if runPipeline.Metadata.AsyncBehavior.Cancellation.CancelTool != "cancel_pipeline" {
			t.Errorf("expected cancel tool 'cancel_pipeline', got %q",
				runPipeline.Metadata.AsyncBehavior.Cancellation.CancelTool)
		}
	})

	t.Run("has examples", func(t *testing.T) {
		if len(runPipeline.Metadata.Examples) == 0 {
			t.Errorf("expected at least one example")
		}
	})
}

func TestCheckPipelineStatusTool(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	var statusTool *toolspb.ToolDefinition
	for _, tool := range tools {
		if tool.Name == "check_pipeline_status" {
			statusTool = tool
			break
		}
	}

	if statusTool == nil {
		t.Fatalf("check_pipeline_status tool not found")
	}

	t.Run("has required pipeline_id parameter", func(t *testing.T) {
		found := false
		for _, req := range statusTool.Parameters.Required {
			if req == "pipeline_id" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pipeline_id should be required")
		}
	})

	t.Run("is not long-running", func(t *testing.T) {
		if statusTool.Metadata.LongRunning {
			t.Errorf("check_pipeline_status should not be long-running")
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		if !statusTool.Metadata.Idempotent {
			t.Errorf("check_pipeline_status should be idempotent")
		}
	})

	t.Run("does not modify state", func(t *testing.T) {
		if statusTool.Metadata.ModifiesState {
			t.Errorf("check_pipeline_status should not modify state")
		}
	})

	t.Run("has high rate limit for polling", func(t *testing.T) {
		if statusTool.Metadata.RateLimitPerMinute < 60 {
			t.Errorf("expected high rate limit for polling, got %d", statusTool.Metadata.RateLimitPerMinute)
		}
	})
}

func TestCancelPipelineTool(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	var cancelTool *toolspb.ToolDefinition
	for _, tool := range tools {
		if tool.Name == "cancel_pipeline" {
			cancelTool = tool
			break
		}
	}

	if cancelTool == nil {
		t.Fatalf("cancel_pipeline tool not found")
	}

	t.Run("has required pipeline_id parameter", func(t *testing.T) {
		found := false
		for _, req := range cancelTool.Parameters.Required {
			if req == "pipeline_id" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pipeline_id should be required")
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		if !cancelTool.Metadata.Idempotent {
			t.Errorf("cancel_pipeline should be idempotent")
		}
	})

	t.Run("modifies state", func(t *testing.T) {
		if !cancelTool.Metadata.ModifiesState {
			t.Errorf("cancel_pipeline should modify state")
		}
	})
}

func TestResumePipelineTool(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	var resumeTool *toolspb.ToolDefinition
	for _, tool := range tools {
		if tool.Name == "resume_pipeline" {
			resumeTool = tool
			break
		}
	}

	if resumeTool == nil {
		t.Fatalf("resume_pipeline tool not found")
	}

	t.Run("has required pipeline_id parameter", func(t *testing.T) {
		found := false
		for _, req := range resumeTool.Parameters.Required {
			if req == "pipeline_id" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pipeline_id should be required")
		}
	})

	t.Run("has optional stop_after_stage parameter", func(t *testing.T) {
		if _, ok := resumeTool.Parameters.Properties["stop_after_stage"]; !ok {
			t.Errorf("stop_after_stage parameter not found")
		}
	})

	t.Run("is long-running", func(t *testing.T) {
		if !resumeTool.Metadata.LongRunning {
			t.Errorf("resume_pipeline should be long-running")
		}
	})

	t.Run("is not idempotent", func(t *testing.T) {
		if resumeTool.Metadata.Idempotent {
			t.Errorf("resume_pipeline should not be idempotent")
		}
	})

	t.Run("has async behavior configured", func(t *testing.T) {
		if resumeTool.Metadata.AsyncBehavior == nil {
			t.Fatalf("async behavior is nil")
		}
		if resumeTool.Metadata.AsyncBehavior.StatusPolling.StatusTool != "check_pipeline_status" {
			t.Errorf("expected status tool 'check_pipeline_status'")
		}
	})
}

func TestListPipelinesTool(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	var listTool *toolspb.ToolDefinition
	for _, tool := range tools {
		if tool.Name == "list_pipelines" {
			listTool = tool
			break
		}
	}

	if listTool == nil {
		t.Fatalf("list_pipelines tool not found")
	}

	t.Run("has no required parameters", func(t *testing.T) {
		if len(listTool.Parameters.Required) > 0 {
			t.Errorf("list_pipelines should have no required parameters")
		}
	})

	t.Run("has filter parameters", func(t *testing.T) {
		expectedFilters := []string{"status", "scenario_name", "limit"}
		for _, filter := range expectedFilters {
			if _, ok := listTool.Parameters.Properties[filter]; !ok {
				t.Errorf("expected filter parameter %q not found", filter)
			}
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		if !listTool.Metadata.Idempotent {
			t.Errorf("list_pipelines should be idempotent")
		}
	})

	t.Run("does not modify state", func(t *testing.T) {
		if listTool.Metadata.ModifiesState {
			t.Errorf("list_pipelines should not modify state")
		}
	})

	t.Run("has examples", func(t *testing.T) {
		if len(listTool.Metadata.Examples) == 0 {
			t.Errorf("expected at least one example")
		}
	})
}

// TestPipelineToolsConsistency verifies that all pipeline tools follow conventions.
func TestPipelineToolsConsistency(t *testing.T) {
	provider := NewPipelineToolProvider()
	tools := provider.Tools(context.Background())

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// All tools should have metadata
			if tool.Metadata == nil {
				t.Errorf("tool %s has no metadata", tool.Name)
				return
			}

			// All tools should be enabled by default
			if !tool.Metadata.EnabledByDefault {
				t.Errorf("tool %s should be enabled by default", tool.Name)
			}

			// All tools should not require approval
			if tool.Metadata.RequiresApproval {
				t.Errorf("tool %s should not require approval", tool.Name)
			}

			// All tools should have a timeout
			if tool.Metadata.TimeoutSeconds == 0 {
				t.Errorf("tool %s should have a timeout", tool.Name)
			}

			// All tools should have a rate limit
			if tool.Metadata.RateLimitPerMinute == 0 {
				t.Errorf("tool %s should have a rate limit", tool.Name)
			}

			// All tools should have tags
			if len(tool.Metadata.Tags) == 0 {
				t.Errorf("tool %s should have tags", tool.Name)
			}

			// All tools should belong to the pipeline category
			if tool.Category != "pipeline" {
				t.Errorf("tool %s should have category 'pipeline', got %q", tool.Name, tool.Category)
			}

			// All tools should have at least one example
			if len(tool.Metadata.Examples) == 0 {
				t.Errorf("tool %s should have at least one example", tool.Name)
			}

			// All tools should have a non-empty description
			if tool.Description == "" {
				t.Errorf("tool %s should have a description", tool.Name)
			}
		})
	}
}
