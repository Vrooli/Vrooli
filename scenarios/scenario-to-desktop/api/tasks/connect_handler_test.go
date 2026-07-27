package tasks

import (
	"encoding/json"
	"scenario-to-desktop-api/domain"
	"testing"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestTaskRequestFromProtoMapsFixAndAppliesDomainDefaults(t *testing.T) {
	sourceInvestigationID := "investigation-1"
	request, err := taskRequestFromProto(&domainv1.CreateTaskRequest{
		PipelineId: "pipeline-1",
		TaskType:   domainv1.TaskType_TASK_TYPE_FIX,
		Focus:      &domainv1.TaskFocus{Harness: true},
		Permissions: &domainv1.FixPermissions{
			Immediate:  true,
			Prevention: true,
		},
		SourceInvestigationId: &sourceInvestigationID,
		IncludeContexts:       []string{"build-log", "artifact"},
	})
	if err != nil {
		t.Fatalf("taskRequestFromProto() error = %v", err)
	}
	if request.TaskType != domain.TaskTypeFix || !request.Focus.Harness || !request.Permissions.Immediate || !request.Permissions.Prevention {
		t.Fatalf("taskRequestFromProto() mapped request = %#v", request)
	}
	if request.MaxIterations != 5 {
		t.Fatalf("taskRequestFromProto() MaxIterations = %d, want domain default 5", request.MaxIterations)
	}
}

func TestTaskRequestFromProtoRejectsInvalidRequest(t *testing.T) {
	_, err := taskRequestFromProto(&domainv1.CreateTaskRequest{
		PipelineId: "pipeline-1",
		TaskType:   domainv1.TaskType_TASK_TYPE_INVESTIGATE,
	})
	if err == nil {
		t.Fatal("taskRequestFromProto() error = nil, want validation error for missing focus")
	}
}

func TestInvestigationToProtoPreservesStructuredFields(t *testing.T) {
	now := time.Date(2026, 7, 26, 17, 0, 0, 0, time.UTC)
	completed := now.Add(time.Minute)
	findings := "missing signing key"
	agentRunID := "agent-run-1"
	message := "build failed"
	value, err := investigationToProto(&domain.Investigation{
		ID:           "task-1",
		PipelineID:   "pipeline-1",
		Status:       domain.InvestigationStatusFailed,
		Findings:     &findings,
		Progress:     75,
		Details:      json.RawMessage(`{"source":"agent-manager","duration_seconds":12}`),
		AgentRunID:   &agentRunID,
		ErrorMessage: &message,
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  &completed,
	})
	if err != nil {
		t.Fatalf("investigationToProto() error = %v", err)
	}
	if value.GetStatus() != domainv1.InvestigationStatus_INVESTIGATION_STATUS_FAILED || value.GetDetails().GetFields()["source"].GetStringValue() != "agent-manager" {
		t.Fatalf("investigationToProto() = %#v", value)
	}
	if value.GetCompletedAt().AsTime() != completed || value.GetAgentRunId() != agentRunID || value.GetFindings() != findings {
		t.Fatalf("investigationToProto() did not preserve optional fields: %#v", value)
	}
}
