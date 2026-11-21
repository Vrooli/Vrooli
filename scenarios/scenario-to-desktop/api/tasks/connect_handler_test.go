package tasks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"scenario-to-desktop-api/domain"

	"connectrpc.com/connect"
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

func TestConnectServiceExposesTaskReadAndStopContracts(t *testing.T) {
	service, store, _, agent, hub := newTestService()
	connectService := NewConnectService(service)
	now := time.Date(2026, 7, 27, 13, 0, 0, 0, time.UTC)
	runID := "run-1"
	store.investigations["task-1"] = &domain.Investigation{
		ID: "task-1", PipelineID: "pipeline-1", Status: domain.InvestigationStatusRunning,
		Progress: 50, CreatedAt: now, UpdatedAt: now, AgentRunID: &runID,
	}
	stopped := false
	store.SetCancel("task-1", func() { stopped = true })
	agent.url = "http://127.0.0.1:19001"

	status, err := connectService.GetAgentManagerStatus(context.Background(), connect.NewRequest(&domainv1.AgentManagerStatusRequest{}))
	if err != nil || !status.Msg.GetAvailable() || status.Msg.GetUrl() != agent.url {
		t.Fatalf("GetAgentManagerStatus = %#v, %v", status.Msg, err)
	}
	task, err := connectService.GetTask(context.Background(), connect.NewRequest(&domainv1.GetTaskRequest{PipelineId: "pipeline-1", TaskId: "task-1"}))
	if err != nil || task.Msg.GetTask().GetId() != "task-1" || task.Msg.GetTask().GetProgress() != 50 {
		t.Fatalf("GetTask = %#v, %v", task.Msg, err)
	}
	list, err := connectService.ListTasks(context.Background(), connect.NewRequest(&domainv1.ListTasksRequest{PipelineId: "pipeline-1"}))
	if err != nil || len(list.Msg.GetTasks()) != 1 || list.Msg.GetTasks()[0].GetId() != "task-1" {
		t.Fatalf("ListTasks = %#v, %v", list.Msg, err)
	}
	stop, err := connectService.StopTask(context.Background(), connect.NewRequest(&domainv1.StopTaskRequest{PipelineId: "pipeline-1", TaskId: "task-1"}))
	if err != nil || !stop.Msg.GetSuccess() || !stopped || store.investigations["task-1"].Status != domain.InvestigationStatusCancelled || len(hub.events) != 1 {
		t.Fatalf("StopTask = %#v, %v; task=%#v events=%#v", stop.Msg, err, store.investigations["task-1"], hub.events)
	}
	_, err = connectService.GetTask(context.Background(), connect.NewRequest(&domainv1.GetTaskRequest{PipelineId: "pipeline-1", TaskId: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing task code = %v", connect.CodeOf(err))
	}
}

func TestConnectServiceReportsUnavailableAgentAndMissingService(t *testing.T) {
	_, _, _, agent, _ := newTestService()
	service, _, _, _, _ := newTestService()
	connectService := NewConnectService(service)
	agent.available = false
	// Use the service's actual executor so this check exercises its status path.
	service.agentSvc = agent
	response, err := connectService.GetAgentManagerStatus(context.Background(), connect.NewRequest(&domainv1.AgentManagerStatusRequest{}))
	if err != nil || response.Msg.GetAvailable() || response.Msg.GetReason() == "" {
		t.Fatalf("unavailable agent response = %#v, %v", response.Msg, err)
	}
	_, err = (&ConnectService{}).ListTasks(context.Background(), connect.NewRequest(&domainv1.ListTasksRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfigured service code = %v", connect.CodeOf(err))
	}
}
