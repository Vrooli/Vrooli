package workflows

import (
	"testing"
	"time"

	internal "react-component-library/internal/workflows"

	workflowspb "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
)

func TestWorkflowProtoMappingPreservesControlState(t *testing.T) {
	now := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	got := toProto(internal.Workflow{
		ID: "wf-1", Kind: internal.KindAdopt, Status: internal.StatusRunning,
		AssetID: "asset-1", TargetScenario: "target", AgentManagerRunID: "run-1", AgentManagerExecutionID: "execution-1",
		CreatedAt: now, UpdatedAt: now,
	})
	if got.Id != "wf-1" || got.Kind != workflowspb.WorkflowKind_WORKFLOW_KIND_ADOPT || got.Status != workflowspb.WorkflowStatus_WORKFLOW_STATUS_RUNNING {
		t.Fatalf("unexpected workflow mapping: %#v", got)
	}
	if !got.CanStop || got.CanRetry {
		t.Fatalf("running workflow controls = stop:%v retry:%v, want true/false", got.CanStop, got.CanRetry)
	}
	if got.AgentManagerExecutionId != "execution-1" {
		t.Fatalf("execution reference=%q, want execution-1", got.AgentManagerExecutionId)
	}

	terminal := toProto(internal.Workflow{Status: internal.StatusFailed})
	if terminal.CanStop || !terminal.CanRetry {
		t.Fatalf("terminal workflow controls = stop:%v retry:%v, want false/true", terminal.CanStop, terminal.CanRetry)
	}
}

func TestStartInputMapsExplicitWorkflowControls(t *testing.T) {
	got := startInput(&workflowspb.StartWorkflowRequest{
		Kind: workflowspb.WorkflowKind_WORKFLOW_KIND_EXTRACT, SourceScenario: "web-console", SourcePath: "ui/src/Card.tsx",
		IdempotencyKey: "request-1", ConfirmOverwrite: true, OverrideValidation: true,
	})
	if got.Kind != internal.KindExtract || got.SourceScenario != "web-console" || got.SourcePath != "ui/src/Card.tsx" || !got.ConfirmOverwrite || !got.OverrideValidation {
		t.Fatalf("unexpected start mapping: %#v", got)
	}
}
