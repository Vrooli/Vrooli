package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"agent-manager/internal/investigation"
	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *AgentManagerConnectHandler) StartInvestigation(ctx context.Context, req *connect.Request[api.StartInvestigationRequest]) (*connect.Response[api.StartInvestigationResponse], error) {
	if h.h == nil || h.h.investigations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("investigation lifecycle is unavailable"))
	}
	if req == nil || req.Msg == nil || req.Msg.GetRequest() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("investigation request is required"))
	}
	item, reused, err := h.h.admitInvestigation(ctx, investigationRequestFromProto(req.Msg.GetRequest()))
	if err != nil {
		return nil, investigationConnectError(err)
	}
	return connect.NewResponse(&api.StartInvestigationResponse{Investigation: investigationRecordToProto(item), Reused: reused}), nil
}

func (h *AgentManagerConnectHandler) GetInvestigation(ctx context.Context, req *connect.Request[api.GetInvestigationRequest]) (*connect.Response[api.InvestigationRecord], error) {
	if h.h == nil || h.h.investigations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("investigation lifecycle is unavailable"))
	}
	if req == nil || req.Msg == nil || req.Msg.GetInvestigationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("investigation id is required"))
	}
	item, err := h.h.investigations.Get(ctx, req.Msg.GetInvestigationId())
	if err != nil {
		return nil, investigationConnectError(err)
	}
	return connect.NewResponse(investigationRecordToProto(item)), nil
}

func (h *AgentManagerConnectHandler) ListInvestigations(ctx context.Context, req *connect.Request[api.ListInvestigationsRequest]) (*connect.Response[api.ListInvestigationsResponse], error) {
	if h.h == nil || h.h.investigations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("investigation lifecycle is unavailable"))
	}
	status, limit := "", 50
	if req != nil && req.Msg != nil {
		status, limit = req.Msg.GetOperationStatus(), int(req.Msg.GetLimit())
	}
	items, err := h.h.investigations.List(ctx, status, limit)
	if err != nil {
		return nil, investigationConnectError(err)
	}
	response := &api.ListInvestigationsResponse{Investigations: make([]*api.InvestigationRecord, 0, len(items))}
	for _, item := range items {
		response.Investigations = append(response.Investigations, investigationRecordToProto(item))
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) WaitInvestigation(ctx context.Context, req *connect.Request[api.WaitInvestigationRequest]) (*connect.Response[api.WaitInvestigationResponse], error) {
	if h.h == nil || h.h.investigations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("investigation lifecycle is unavailable"))
	}
	if req == nil || req.Msg == nil || req.Msg.GetInvestigationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("investigation id is required"))
	}
	timeout := 30 * time.Second
	if req.Msg.GetTimeoutSeconds() > 0 {
		timeout = time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	}
	item, terminal, err := h.h.investigations.Wait(ctx, req.Msg.GetInvestigationId(), timeout)
	if err != nil {
		return nil, investigationConnectError(err)
	}
	return connect.NewResponse(&api.WaitInvestigationResponse{Investigation: investigationRecordToProto(item), Terminal: terminal}), nil
}

func (h *AgentManagerConnectHandler) CancelInvestigation(ctx context.Context, req *connect.Request[api.CancelInvestigationRequest]) (*connect.Response[api.InvestigationRecord], error) {
	if h.h == nil || h.h.investigations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("investigation lifecycle is unavailable"))
	}
	if req == nil || req.Msg == nil || req.Msg.GetInvestigationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("investigation id is required"))
	}
	item, err := h.h.cancelInvestigation(ctx, req.Msg.GetInvestigationId())
	if err != nil {
		return nil, investigationConnectError(err)
	}
	return connect.NewResponse(investigationRecordToProto(item)), nil
}

func investigationRequestFromProto(request *api.InvestigationRequest) investigation.Request {
	result := investigation.Request{SchemaVersion: request.GetSchemaVersion(), RequestKey: request.GetRequestKey(), CallerAuthority: request.GetCallerAuthority(), Question: request.GetQuestion()}
	if subject := request.GetSubject(); subject != nil {
		result.Subject = investigation.Subject{Owner: subject.GetOwner(), Kind: subject.GetKind(), Ref: subject.GetRef(), Revision: subject.GetRevision(), RunIDs: subject.GetRunIds()}
	}
	if method := request.GetMethodRef(); method != nil {
		result.MethodRef = &investigation.MethodReference{SkillID: method.GetSkillId(), Revision: method.GetRevision()}
	}
	for _, ref := range request.GetDomainEvidence() {
		result.DomainEvidence = append(result.DomainEvidence, investigation.EvidenceReference{Owner: ref.GetOwner(), Kind: ref.GetKind(), Ref: ref.GetRef(), Revision: ref.GetRevision(), SchemaVersion: ref.GetSchemaVersion(), SubjectRunIDs: ref.GetSubjectRunIds()})
	}
	if policy := request.GetEvidencePolicy(); policy != nil {
		result.EvidencePolicy = investigation.EvidencePolicy{Mode: policy.GetMode(), RequiredPlanes: policy.GetRequiredPlanes(), OptionalPlanes: policy.GetOptionalPlanes(), MaxEvents: int(policy.GetMaxEvents()), MaxEvidenceBytes: int(policy.GetMaxEvidenceBytes()), MaxReconciliations: int(policy.GetMaxReconciliations())}
	}
	if budget := request.GetBudget(); budget != nil {
		result.Budget = investigation.Budget{MaxDelegatedRuns: int(budget.GetMaxDelegatedRuns()), MaxTurns: int(budget.GetMaxTurns()), WallSeconds: int(budget.GetWallSeconds()), MaxChargeMicroUSD: budget.GetMaxChargeMicroUsd()}
	}
	if policy := request.GetRecommendationPolicy(); policy != nil {
		result.RecommendationPolicy = investigation.RecommendationPolicy{AllowedKinds: policy.GetAllowedKinds(), AllowSubjectMutation: policy.GetAllowSubjectMutation()}
	}
	if provenance := request.GetProvenance(); provenance != nil {
		result.Provenance = investigation.Provenance{Kind: provenance.GetKind(), TriggerOccurrenceRef: provenance.GetTriggerOccurrenceRef()}
	}
	return result
}

func investigationRecordToProto(item *investigation.Lifecycle) *api.InvestigationRecord {
	if item == nil {
		return nil
	}
	return &api.InvestigationRecord{InvestigationId: item.ID, Request: investigationRequestToProto(item.Request), RequestDigest: item.RequestDigest, OperationStatus: item.OperationStatus, ResultJson: resultJSON(item), SourceCutJson: item.SourceCutJSON, WorkflowRef: item.WorkflowRef, CancelRequested: item.CancelRequested, CreatedAt: timestamppb.New(item.CreatedAt), UpdatedAt: timestamppb.New(item.UpdatedAt), CompletedAt: optionalTimestamp(item.CompletedAt), CancelledAt: optionalTimestamp(item.CancelledAt)}
}

func investigationRequestToProto(request investigation.Request) *api.InvestigationRequest {
	result := &api.InvestigationRequest{SchemaVersion: request.SchemaVersion, RequestKey: request.RequestKey, CallerAuthority: request.CallerAuthority, Subject: &api.InvestigationSubject{Owner: request.Subject.Owner, Kind: request.Subject.Kind, Ref: request.Subject.Ref, Revision: request.Subject.Revision, RunIds: request.Subject.RunIDs}, Question: request.Question, EvidencePolicy: &api.InvestigationEvidencePolicy{Mode: request.EvidencePolicy.Mode, RequiredPlanes: request.EvidencePolicy.RequiredPlanes, OptionalPlanes: request.EvidencePolicy.OptionalPlanes, MaxEvents: int32(request.EvidencePolicy.MaxEvents), MaxEvidenceBytes: int32(request.EvidencePolicy.MaxEvidenceBytes), MaxReconciliations: int32(request.EvidencePolicy.MaxReconciliations)}, Budget: &api.InvestigationBudget{MaxDelegatedRuns: int32(request.Budget.MaxDelegatedRuns), MaxTurns: int32(request.Budget.MaxTurns), WallSeconds: int32(request.Budget.WallSeconds), MaxChargeMicroUsd: request.Budget.MaxChargeMicroUSD}, RecommendationPolicy: &api.InvestigationRecommendationPolicy{AllowedKinds: request.RecommendationPolicy.AllowedKinds, AllowSubjectMutation: request.RecommendationPolicy.AllowSubjectMutation}, Provenance: &api.InvestigationProvenance{Kind: request.Provenance.Kind, TriggerOccurrenceRef: request.Provenance.TriggerOccurrenceRef}}
	if request.MethodRef != nil {
		result.MethodRef = &api.InvestigationMethodReference{SkillId: request.MethodRef.SkillID, Revision: request.MethodRef.Revision}
	}
	for _, ref := range request.DomainEvidence {
		result.DomainEvidence = append(result.DomainEvidence, &api.InvestigationEvidenceReference{Owner: ref.Owner, Kind: ref.Kind, Ref: ref.Ref, Revision: ref.Revision, SchemaVersion: ref.SchemaVersion, SubjectRunIds: ref.SubjectRunIDs})
	}
	return result
}

func resultJSON(item *investigation.Lifecycle) string {
	if item == nil || item.Result == nil {
		return ""
	}
	raw, _ := json.Marshal(item.Result)
	return string(raw)
}

func optionalTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func investigationConnectError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, investigation.ErrInvalidRequest), errors.Is(err, investigation.ErrInvalidResult):
		code = connect.CodeInvalidArgument
	case errors.Is(err, investigation.ErrRequestKeyConflict), errors.Is(err, investigation.ErrInvalidTransition):
		code = connect.CodeAborted
	case errors.Is(err, investigation.ErrNotFound):
		code = connect.CodeNotFound
	}
	return connect.NewError(code, err)
}
