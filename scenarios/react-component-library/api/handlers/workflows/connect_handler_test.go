package workflows

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
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

func TestWorkflowErrorMappingKeepsNotFoundDistinctFromInvalidInput(t *testing.T) {
	wrappedNotFound := errors.Join(errors.New("repository lookup failed"), internal.ErrNotFound)
	if got := connect.CodeOf(toError(wrappedNotFound)); got != connect.CodeNotFound {
		t.Fatalf("wrapped not-found mapped to %v, want %v", got, connect.CodeNotFound)
	}
	if got := connect.CodeOf(toError(errors.New("invalid workflow transition"))); got != connect.CodeInvalidArgument {
		t.Fatalf("domain validation error mapped to %v, want %v", got, connect.CodeInvalidArgument)
	}
}

func TestWorkflowStatusProtoMappingCoversEveryDomainState(t *testing.T) {
	cases := []struct {
		status internal.Status
		want   workflowspb.WorkflowStatus
	}{
		{internal.StatusQueued, workflowspb.WorkflowStatus_WORKFLOW_STATUS_QUEUED},
		{internal.StatusRunning, workflowspb.WorkflowStatus_WORKFLOW_STATUS_RUNNING},
		{internal.StatusParked, workflowspb.WorkflowStatus_WORKFLOW_STATUS_PARKED},
		{internal.StatusSucceeded, workflowspb.WorkflowStatus_WORKFLOW_STATUS_SUCCEEDED},
		{internal.StatusFailed, workflowspb.WorkflowStatus_WORKFLOW_STATUS_FAILED},
		{internal.StatusStopped, workflowspb.WorkflowStatus_WORKFLOW_STATUS_STOPPED},
		{internal.StatusUnavailable, workflowspb.WorkflowStatus_WORKFLOW_STATUS_UNAVAILABLE},
		{internal.Status("future-state"), workflowspb.WorkflowStatus_WORKFLOW_STATUS_UNAVAILABLE},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			if got := statusToProto(tc.status); got != tc.want {
				t.Fatalf("statusToProto(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestPromotionReadinessProtoMappingCopiesEvidenceSlices(t *testing.T) {
	in := internal.PromotionReadiness{
		DependencyLibraryIDs: []string{"dep-a"}, OriginFiles: []string{"src/a.tsx"},
		ParityFindings: []string{"missing preview"}, Blockers: []string{"validation pending"},
		RequiredExampleCount: 3, AvailableExampleCount: 2,
	}
	got := readinessToProto(in)
	got.DependencyLibraryIds[0] = "changed"
	got.OriginFiles[0] = "changed"
	got.ParityFindings[0] = "changed"
	got.Blockers[0] = "changed"
	if in.DependencyLibraryIDs[0] != "dep-a" || in.OriginFiles[0] != "src/a.tsx" || in.ParityFindings[0] != "missing preview" || in.Blockers[0] != "validation pending" {
		t.Fatal("readiness proto mapping aliased domain evidence slices")
	}
	if got.RequiredExampleCount != 3 || got.AvailableExampleCount != 2 {
		t.Fatalf("example counts = %d/%d, want 3/2", got.RequiredExampleCount, got.AvailableExampleCount)
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
