package workflows

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	internal "react-component-library/internal/workflows"
)

type connectHandler struct {
	svc    internal.Service
	logger *log.Logger
}

func NewConnectHandler(s internal.Service, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{svc: s, logger: l}
}
func (h *connectHandler) StartWorkflow(c context.Context, r *connect.Request[workflowsv1.StartWorkflowRequest]) (*connect.Response[workflowsv1.StartWorkflowResponse], error) {
	w, q, e := h.svc.Start(c, startInput(r.Msg))
	if e != nil {
		return nil, toError(e)
	}
	return connect.NewResponse(&workflowsv1.StartWorkflowResponse{Workflow: toProto(w), QueueDepth: int32(q)}), nil
}
func (h *connectHandler) ListWorkflows(c context.Context, r *connect.Request[workflowsv1.ListWorkflowsRequest]) (*connect.Response[workflowsv1.ListWorkflowsResponse], error) {
	xs, e := h.svc.List(c, r.Msg.AssetId, r.Msg.TargetScenario, r.Msg.ActiveOnly, int(r.Msg.Limit))
	if e != nil {
		return nil, toError(e)
	}
	out := make([]*workflowsv1.Workflow, 0, len(xs))
	for _, w := range xs {
		out = append(out, toProto(w))
	}
	return connect.NewResponse(&workflowsv1.ListWorkflowsResponse{Workflows: out}), nil
}
func (h *connectHandler) GetWorkflow(c context.Context, r *connect.Request[workflowsv1.GetWorkflowRequest]) (*connect.Response[workflowsv1.GetWorkflowResponse], error) {
	w, e := h.svc.Get(c, r.Msg.Id)
	if e != nil {
		return nil, toError(e)
	}
	return connect.NewResponse(&workflowsv1.GetWorkflowResponse{Workflow: toProto(w)}), nil
}
func (h *connectHandler) RefreshWorkflow(c context.Context, r *connect.Request[workflowsv1.RefreshWorkflowRequest]) (*connect.Response[workflowsv1.RefreshWorkflowResponse], error) {
	w, e := h.svc.Refresh(c, r.Msg.Id)
	if e != nil {
		return nil, toError(e)
	}
	return connect.NewResponse(&workflowsv1.RefreshWorkflowResponse{Workflow: toProto(w)}), nil
}
func (h *connectHandler) StopWorkflow(c context.Context, r *connect.Request[workflowsv1.StopWorkflowRequest]) (*connect.Response[workflowsv1.StopWorkflowResponse], error) {
	w, e := h.svc.Stop(c, r.Msg.Id)
	if e != nil {
		return nil, toError(e)
	}
	return connect.NewResponse(&workflowsv1.StopWorkflowResponse{Workflow: toProto(w)}), nil
}
func (h *connectHandler) RetryWorkflow(c context.Context, r *connect.Request[workflowsv1.RetryWorkflowRequest]) (*connect.Response[workflowsv1.RetryWorkflowResponse], error) {
	w, q, e := h.svc.Retry(c, r.Msg.Id, r.Msg.IdempotencyKey)
	if e != nil {
		return nil, toError(e)
	}
	return connect.NewResponse(&workflowsv1.RetryWorkflowResponse{Workflow: toProto(w), QueueDepth: int32(q)}), nil
}
func (h *connectHandler) GetPromotionReadiness(c context.Context, r *connect.Request[workflowsv1.GetPromotionReadinessRequest]) (*connect.Response[workflowsv1.GetPromotionReadinessResponse], error) {
	out, err := h.svc.PromotionReadiness(c, internal.PromotionReadinessInput{AssetID: r.Msg.AssetId, OriginScenario: r.Msg.OriginScenario, Version: r.Msg.Version})
	if err != nil {
		return nil, toError(err)
	}
	return connect.NewResponse(&workflowsv1.GetPromotionReadinessResponse{Readiness: readinessToProto(out)}), nil
}
func readinessToProto(in internal.PromotionReadiness) *workflowsv1.PromotionReadiness {
	return &workflowsv1.PromotionReadiness{AssetId: in.AssetID, LibraryId: in.LibraryID, SelectedVersion: in.SelectedVersion, OriginScenario: in.OriginScenario, DependencyLibraryIds: append([]string(nil), in.DependencyLibraryIDs...), OriginFiles: append([]string(nil), in.OriginFiles...), RequiredExampleCount: int32(in.RequiredExampleCount), AvailableExampleCount: int32(in.AvailableExampleCount), ParityReportPresent: in.ParityReportPresent, ParityWaived: in.ParityWaived, ParityFindings: append([]string(nil), in.ParityFindings...), OriginReplacementPresent: in.OriginReplacementPresent, OriginReplacementClean: in.OriginReplacementClean, Blockers: append([]string(nil), in.Blockers...), Ready: in.Ready, NextValidationCommand: in.NextValidationCommand}
}
func startInput(m *workflowsv1.StartWorkflowRequest) internal.StartInput {
	return internal.StartInput{Kind: kindFromProto(m.Kind), AssetID: m.AssetId, SourceScenario: m.SourceScenario, TargetScenario: m.TargetScenario, SourcePath: m.SourcePath, RequestedVersion: m.RequestedVersion, IdempotencyKey: m.IdempotencyKey, ConfirmOverwrite: m.ConfirmOverwrite, OverrideValidation: m.OverrideValidation}
}
func toProto(w internal.Workflow) *workflowsv1.Workflow {
	return &workflowsv1.Workflow{Id: w.ID, Kind: kindToProto(w.Kind), AssetId: w.AssetID, SourceScenario: w.SourceScenario, TargetScenario: w.TargetScenario, SourcePath: w.SourcePath, RequestedVersion: w.RequestedVersion, AgentManagerTaskId: w.AgentManagerTaskID, AgentManagerRunId: w.AgentManagerRunID, IdempotencyKey: w.IdempotencyKey, Status: statusToProto(w.Status), LastEventSequence: w.LastEventSequence, Summary: w.Summary, Error: w.Error, CreatedAt: timestamppb.New(w.CreatedAt), UpdatedAt: timestamppb.New(w.UpdatedAt), CompletedAt: timestamppb.New(w.CompletedAt), CanStop: w.Status.Active(), CanRetry: !w.Status.Active()}
}
func kindFromProto(k workflowsv1.WorkflowKind) internal.Kind {
	if k == workflowsv1.WorkflowKind_WORKFLOW_KIND_EXTRACT {
		return internal.KindExtract
	}
	if k == workflowsv1.WorkflowKind_WORKFLOW_KIND_ADOPT {
		return internal.KindAdopt
	}
	return ""
}
func kindToProto(k internal.Kind) workflowsv1.WorkflowKind {
	if k == internal.KindAdopt {
		return workflowsv1.WorkflowKind_WORKFLOW_KIND_ADOPT
	}
	return workflowsv1.WorkflowKind_WORKFLOW_KIND_EXTRACT
}
func statusToProto(s internal.Status) workflowsv1.WorkflowStatus {
	switch s {
	case internal.StatusQueued:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_QUEUED
	case internal.StatusRunning:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING
	case internal.StatusParked:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_PARKED
	case internal.StatusSucceeded:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_SUCCEEDED
	case internal.StatusFailed:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_FAILED
	case internal.StatusStopped:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_STOPPED
	default:
		return workflowsv1.WorkflowStatus_WORKFLOW_STATUS_UNAVAILABLE
	}
}
func toError(e error) error {
	if errors.Is(e, internal.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, e)
	}
	return connect.NewError(connect.CodeInvalidArgument, e)
}
