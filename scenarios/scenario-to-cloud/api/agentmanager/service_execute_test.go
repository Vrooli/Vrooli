package agentmanager

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInvestigationTaskCarriesScopeAndContext(t *testing.T) {
	attachments := []*domainpb.ContextAttachment{{Label: "deployment report"}}
	task := investigationTask(ExecuteRequest{
		InvestigationID:    "inv-123",
		Prompt:             "inspect the deployed service",
		WorkingDir:         "/tmp/deployment",
		ContextAttachments: attachments,
	})

	if task.Title != "Deployment Investigation inv-123" || task.Description != "inspect the deployed service" {
		t.Fatalf("unexpected task metadata: %+v", task)
	}
	if task.ScopePath != "/tmp/deployment" || task.ProjectRoot != "/tmp/deployment" {
		t.Fatalf("task should use the deployment working directory as scope and project root: %+v", task)
	}
	if task.CreatedBy != "scenario-to-cloud" || len(task.ContextAttachments) != 1 || task.ContextAttachments[0] != attachments[0] {
		t.Fatalf("task should preserve provenance and context attachments: %+v", task)
	}
}

func TestInvestigationRunRequestAppliesOverridesAndManualReview(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "stc-investigation", Enabled: true})
	role := "code.review"
	model := "test-model"
	req := ExecuteRequest{InvestigationID: "inv-123", AdditionalTag: "nightly", RoleRef: &role, Model: model}
	runReq := investigationRunRequest(svc, req, "task-42")

	if runReq.TaskId != "task-42" || runReq.Tag == nil || *runReq.Tag != "scenario-to-cloud-inv-123|nightly" {
		t.Fatalf("run identity is not preserved: %+v", runReq)
	}
	if runReq.ProfileRef == nil || runReq.ProfileRef.ProfileKey != "stc-investigation" {
		t.Fatalf("default profile reference was not attached: %+v", runReq.ProfileRef)
	}
	if runReq.RunMode == nil || runReq.GetRunMode() != domainpb.RunMode_RUN_MODE_IN_PLACE || !runReq.Force {
		t.Fatalf("investigation run must remain in-place and force-enabled: %+v", runReq)
	}
	if runReq.InlineConfig == nil || runReq.InlineConfig.RoleRef == nil || *runReq.InlineConfig.RoleRef != role || runReq.InlineConfig.Model == nil || *runReq.InlineConfig.Model != model {
		t.Fatalf("inline role/model overrides were not preserved: %+v", runReq.InlineConfig)
	}
	if runReq.InlineConfig.SandboxConfig == nil || !runReq.InlineConfig.SandboxConfig.ManualReview {
		t.Fatalf("investigation changes must require manual review: %+v", runReq.InlineConfig)
	}
}

func TestInvestigationRunRequestCreatesManualReviewConfigWithoutOverrides(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "stc-investigation", Enabled: true})
	runReq := investigationRunRequest(svc, ExecuteRequest{InvestigationID: "inv-124"}, "task-43")
	if runReq.InlineConfig == nil || runReq.InlineConfig.SandboxConfig == nil || !runReq.InlineConfig.SandboxConfig.ManualReview {
		t.Fatalf("manual review config must exist even without inline overrides: %+v", runReq.InlineConfig)
	}
	if runReq.InlineConfig.RoleRef != nil || runReq.InlineConfig.Model != nil {
		t.Fatalf("empty role/model overrides should remain absent: %+v", runReq.InlineConfig)
	}
}

func TestInvestigationResultMapsSummaryErrorsAndDuration(t *testing.T) {
	start := time.Date(2026, time.November, 16, 10, 0, 0, 0, time.UTC)
	end := start.Add(17*time.Second + 900*time.Millisecond)
	run := &domainpb.Run{
		Id:        "run-1",
		Status:    domainpb.RunStatus_RUN_STATUS_COMPLETE,
		ErrorMsg:  "provider rate limit timeout",
		StartedAt: timestamppb.New(start),
		EndedAt:   timestamppb.New(end),
		Summary: &domainpb.RunSummary{
			Description:  "completed investigation",
			TokensUsed:   1234,
			CostEstimate: 0.42,
		},
	}
	result := investigationResult(run)
	if !result.Success || result.RunID != "run-1" || result.Output != "completed investigation" {
		t.Fatalf("run identity, status, and summary were not mapped: %+v", result)
	}
	if result.TokensUsed != 1234 || result.CostEstimate != 0.42 || result.DurationSeconds != 17 {
		t.Fatalf("run metrics were not mapped: %+v", result)
	}
	if result.ErrorMessage != run.ErrorMsg || !result.RateLimited || !result.Timeout {
		t.Fatalf("error classifications were not mapped: %+v", result)
	}
}

func TestInvestigationResultHandlesMissingOptionalFields(t *testing.T) {
	result := investigationResult(&domainpb.Run{Id: "run-2", Status: domainpb.RunStatus_RUN_STATUS_FAILED})
	if result.Success || result.RunID != "run-2" || result.Output != "" || result.DurationSeconds != 0 {
		t.Fatalf("missing optional fields should map to empty result values: %+v", result)
	}
}
