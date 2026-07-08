package cleanup

import (
	"context"
	"fmt"

	cleanupcore "cleanup-manager/internal/cleanup"
	"cleanup-manager/internal/orchestrator"
	"cleanup-manager/internal/policy"
	"connectrpc.com/connect"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cleanup-manager/v1/cleanup"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Catalog() []cleanupcore.ProviderMetadata
	CurrentPolicy(context.Context) (orchestrator.Policy, error)
	SetPolicyProfile(context.Context, policy.ProfileName) (orchestrator.Policy, error)
	Plan(context.Context, cleanupcore.ObservationScope) (orchestrator.Plan, error)
	Apply(context.Context, orchestrator.ApplyInput) (orchestrator.ApplyReport, error)
	Audit(context.Context) ([]orchestrator.AuditEvent, error)
}

type connectHandler struct {
	service Service
}

func NewConnectHandler(service Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListProviders(context.Context, *connect.Request[cleanupv1.ListProvidersRequest]) (*connect.Response[cleanupv1.ListProvidersResponse], error) {
	resp := &cleanupv1.ListProvidersResponse{}
	for _, meta := range h.service.Catalog() {
		resp.Providers = append(resp.Providers, providerToProto(meta))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetPolicy(ctx context.Context, _ *connect.Request[cleanupv1.GetPolicyRequest]) (*connect.Response[cleanupv1.GetPolicyResponse], error) {
	pol, err := h.service.CurrentPolicy(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&cleanupv1.GetPolicyResponse{Policy: policyToProto(pol)}), nil
}

func (h *connectHandler) SetPolicyProfile(ctx context.Context, req *connect.Request[cleanupv1.SetPolicyProfileRequest]) (*connect.Response[cleanupv1.SetPolicyProfileResponse], error) {
	name := policy.ProfileName(req.Msg.GetProfile())
	pol, err := h.service.SetPolicyProfile(ctx, name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&cleanupv1.SetPolicyProfileResponse{Policy: policyToProto(pol)}), nil
}

func (h *connectHandler) CreatePlan(ctx context.Context, _ *connect.Request[cleanupv1.CreatePlanRequest]) (*connect.Response[cleanupv1.CreatePlanResponse], error) {
	plan, err := h.service.Plan(ctx, cleanupcore.ObservationScope{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&cleanupv1.CreatePlanResponse{Plan: planToProto(plan)}), nil
}

func (h *connectHandler) ApplyPlan(ctx context.Context, req *connect.Request[cleanupv1.ApplyPlanRequest]) (*connect.Response[cleanupv1.ApplyPlanResponse], error) {
	report, err := h.service.Apply(ctx, orchestrator.ApplyInput{
		PlanID:         req.Msg.GetPlanId(),
		PolicyVersion:  req.Msg.GetPolicyVersion(),
		ApprovalMode:   cleanupcore.ApprovalMode(req.Msg.GetApprovalMode()),
		ApprovalToken:  req.Msg.GetApprovalToken(),
		IdempotencyKey: req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(applyReportToProto(report)), nil
}

func (h *connectHandler) ListAudit(ctx context.Context, _ *connect.Request[cleanupv1.ListAuditRequest]) (*connect.Response[cleanupv1.ListAuditResponse], error) {
	events, err := h.service.Audit(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &cleanupv1.ListAuditResponse{}
	for _, event := range events {
		resp.Events = append(resp.Events, auditToProto(event))
	}
	return connect.NewResponse(resp), nil
}

func providerToProto(meta cleanupcore.ProviderMetadata) *cleanupv1.Provider {
	return &cleanupv1.Provider{
		Id:                  meta.ID,
		Name:                meta.Name,
		Version:             meta.Version,
		OwnerScenario:       meta.OwnerScenario,
		SafetyTier:          string(meta.SafetyTier),
		DefaultMode:         string(meta.DefaultMode),
		DefaultApproval:     string(meta.DefaultApproval),
		SupportedPlatforms:  append([]string(nil), meta.SupportedPlatforms...),
		RequiredPrivileges:  append([]string(nil), meta.RequiredPrivileges...),
		IrreversibleEffects: append([]string(nil), meta.IrreversibleEffects...),
	}
}

func policyToProto(pol orchestrator.Policy) *cleanupv1.Policy {
	out := &cleanupv1.Policy{Version: pol.Version, Profile: string(pol.Profile), CreatedAt: timestamppb.New(pol.CreatedAt)}
	for id, pp := range pol.Providers {
		out.Providers = append(out.Providers, &cleanupv1.ProviderPolicy{
			ProviderId:    id,
			Enabled:       pp.Enabled,
			MinAgeSeconds: int64(pp.MinAge.Seconds()),
			MaxBytes:      pp.MaxBytes,
			ApprovalMode:  string(pp.ApprovalMode),
		})
	}
	return out
}

func planToProto(plan orchestrator.Plan) *cleanupv1.Plan {
	out := &cleanupv1.Plan{
		Id:            plan.ID,
		PolicyVersion: plan.PolicyVersion,
		CreatedAt:     timestamppb.New(plan.CreatedAt),
		TotalBytes:    plan.TotalBytes,
		TotalItems:    int32(plan.TotalItems),
	}
	for _, pp := range plan.Providers {
		provider := &cleanupv1.ProviderPlan{
			ProviderId:      pp.ProviderID,
			ProviderVersion: pp.ProviderVersion,
			EstimatedBytes:  pp.Estimate.EstimatedBytes,
			ItemCount:       int32(pp.Estimate.ItemCount),
			BlockedReason:   firstNonEmpty(pp.Estimate.BlockedReason, pp.Preview.BlockedReason),
			Warnings:        append([]string(nil), pp.Preview.Warnings...),
			ApprovalMode:    string(pp.Policy.ApprovalMode),
		}
		for _, item := range pp.Preview.Items {
			provider.Items = append(provider.Items, &cleanupv1.PreviewItem{
				Id:          item.ID,
				Path:        item.Path,
				Description: item.Description,
				Bytes:       item.Bytes,
				Action:      item.Action,
				SafetyTier:  string(item.SafetyTier),
			})
		}
		out.Providers = append(out.Providers, provider)
	}
	return out
}

func applyReportToProto(report orchestrator.ApplyReport) *cleanupv1.ApplyPlanResponse {
	out := &cleanupv1.ApplyPlanResponse{
		PlanId:         report.PlanID,
		IdempotencyKey: report.IdempotencyKey,
		AlreadyApplied: report.AlreadyApplied,
		ReclaimedBytes: report.ReclaimedBytes,
	}
	for _, result := range report.Results {
		out.Results = append(out.Results, &cleanupv1.ApplyResult{
			ProviderId:     result.ProviderID,
			Applied:        result.Applied,
			AlreadyDone:    result.AlreadyDone,
			ReclaimedBytes: result.ReclaimedBytes,
			SkippedItems:   append([]string(nil), result.SkippedItems...),
			Warnings:       append([]string(nil), result.Warnings...),
		})
	}
	return out
}

func auditToProto(event orchestrator.AuditEvent) *cleanupv1.AuditEvent {
	return &cleanupv1.AuditEvent{
		Id:             event.ID,
		Time:           timestamppb.New(event.Time),
		Type:           event.Type,
		PlanId:         event.PlanID,
		ProviderId:     event.ProviderID,
		IdempotencyKey: event.IdempotencyKey,
		Message:        event.Message,
		Redacted:       event.Redacted,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func requireService(service Service) Service {
	if service == nil {
		panic(fmt.Errorf("cleanup handler requires service"))
	}
	return service
}
