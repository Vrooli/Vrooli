package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"scenario-to-desktop-api/domain"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConnectService struct {
	domainconnect.UnimplementedTaskServiceHandler
	service *Service
}

var _ domainconnect.TaskServiceHandler = (*ConnectService)(nil)

func NewConnectService(service *Service) *ConnectService { return &ConnectService{service: service} }
func (s *ConnectService) require() error {
	if s == nil || s.service == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("task service is not configured"))
	}
	return nil
}

func (s *ConnectService) GetAgentManagerStatus(ctx context.Context, _ *connect.Request[domainv1.AgentManagerStatusRequest]) (*connect.Response[domainv1.AgentManagerStatusResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	result := &domainv1.AgentManagerStatusResponse{Available: s.service.IsAgentAvailable(ctx)}
	if result.Available {
		url, err := s.service.GetAgentManagerURL(ctx)
		if err == nil {
			result.Url = &url
		}
	} else {
		reason := "agent-manager service not reachable"
		result.Reason = &reason
	}
	return connect.NewResponse(result), nil
}

func (s *ConnectService) CreateTask(ctx context.Context, r *connect.Request[domainv1.CreateTaskRequest]) (*connect.Response[domainv1.CreateTaskResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	request, err := taskRequestFromProto(r.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	task, err := s.service.TriggerTask(ctx, *request)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value, err := investigationToProto(task)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.CreateTaskResponse{Task: value}), nil
}

func (s *ConnectService) GetTask(ctx context.Context, r *connect.Request[domainv1.GetTaskRequest]) (*connect.Response[domainv1.GetTaskResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	task, err := s.service.GetTask(ctx, r.Msg.GetPipelineId(), r.Msg.GetTaskId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}
	value, err := investigationToProto(task)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.GetTaskResponse{Task: value}), nil
}

func (s *ConnectService) ListTasks(ctx context.Context, r *connect.Request[domainv1.ListTasksRequest]) (*connect.Response[domainv1.ListTasksResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	limit := int(r.Msg.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	tasks, err := s.service.ListTasks(ctx, r.Msg.GetPipelineId(), limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &domainv1.ListTasksResponse{}
	for _, task := range tasks {
		out.Tasks = append(out.Tasks, summaryToProto(task.ToSummary()))
	}
	return connect.NewResponse(out), nil
}

func (s *ConnectService) StopTask(ctx context.Context, r *connect.Request[domainv1.StopTaskRequest]) (*connect.Response[domainv1.StopTaskResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	if err := s.service.StopTask(ctx, r.Msg.GetPipelineId(), r.Msg.GetTaskId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&domainv1.StopTaskResponse{Success: true, Message: "Task stopped"}), nil
}

func taskRequestFromProto(v *domainv1.CreateTaskRequest) (*domain.CreateTaskRequest, error) {
	if v == nil {
		return nil, fmt.Errorf("request is required")
	}
	kind := map[domainv1.TaskType]domain.TaskType{domainv1.TaskType_TASK_TYPE_INVESTIGATE: domain.TaskTypeInvestigate, domainv1.TaskType_TASK_TYPE_FIX: domain.TaskTypeFix}[v.GetTaskType()]
	effort := map[domainv1.InvestigationEffort]domain.InvestigationEffort{domainv1.InvestigationEffort_INVESTIGATION_EFFORT_CHECKS: domain.EffortChecks, domainv1.InvestigationEffort_INVESTIGATION_EFFORT_LOGS: domain.EffortLogs, domainv1.InvestigationEffort_INVESTIGATION_EFFORT_TRACE: domain.EffortTrace}[v.GetEffort()]
	result := &domain.CreateTaskRequest{PipelineID: v.GetPipelineId(), TaskType: kind, Focus: domain.TaskFocus{Harness: v.GetFocus().GetHarness(), Subject: v.GetFocus().GetSubject()}, Note: v.GetNote(), Effort: effort, Permissions: domain.FixPermissions{Immediate: v.GetPermissions().GetImmediate(), Permanent: v.GetPermissions().GetPermanent(), Prevention: v.GetPermissions().GetPrevention()}, SourceInvestigationID: v.GetSourceInvestigationId(), MaxIterations: int(v.GetMaxIterations()), IncludeContexts: v.GetIncludeContexts()}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}

func investigationToProto(v *domain.Investigation) (*domainv1.Investigation, error) {
	details, err := detailsStruct(v.Details)
	if err != nil {
		return nil, err
	}
	out := &domainv1.Investigation{Id: v.ID, PipelineId: v.PipelineID, Status: statusToProto(v.Status), Progress: int32(v.Progress), Details: details, CreatedAt: timestamppb.New(v.CreatedAt), UpdatedAt: timestamppb.New(v.UpdatedAt)}
	if v.Findings != nil {
		out.Findings = v.Findings
	}
	if v.AgentRunID != nil {
		out.AgentRunId = v.AgentRunID
	}
	if v.ErrorMessage != nil {
		out.ErrorMessage = v.ErrorMessage
	}
	if v.CompletedAt != nil {
		out.CompletedAt = timestamppb.New(*v.CompletedAt)
	}
	return out, nil
}

func detailsStruct(raw json.RawMessage) (*structpb.Struct, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return structpb.NewStruct(value)
}

func summaryToProto(v domain.InvestigationSummary) *domainv1.InvestigationSummary {
	out := &domainv1.InvestigationSummary{Id: v.ID, PipelineId: v.PipelineID, Status: statusToProto(v.Status), Progress: int32(v.Progress), HasFindings: v.HasFindings, CreatedAt: timestamppb.New(v.CreatedAt)}
	if v.ErrorMessage != nil {
		out.ErrorMessage = v.ErrorMessage
	}
	if v.SourceInvestigationID != nil {
		out.SourceInvestigationId = v.SourceInvestigationID
	}
	if v.CompletedAt != nil {
		out.CompletedAt = timestamppb.New(*v.CompletedAt)
	}
	return out
}

func statusToProto(v domain.InvestigationStatus) domainv1.InvestigationStatus {
	switch v {
	case domain.InvestigationStatusPending:
		return domainv1.InvestigationStatus_INVESTIGATION_STATUS_PENDING
	case domain.InvestigationStatusRunning:
		return domainv1.InvestigationStatus_INVESTIGATION_STATUS_RUNNING
	case domain.InvestigationStatusCompleted:
		return domainv1.InvestigationStatus_INVESTIGATION_STATUS_COMPLETED
	case domain.InvestigationStatusFailed:
		return domainv1.InvestigationStatus_INVESTIGATION_STATUS_FAILED
	case domain.InvestigationStatusCancelled:
		return domainv1.InvestigationStatus_INVESTIGATION_STATUS_CANCELLED
	default:
		return domainv1.InvestigationStatus_INVESTIGATION_STATUS_UNSPECIFIED
	}
}
