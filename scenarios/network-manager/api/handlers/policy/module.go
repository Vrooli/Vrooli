package policy

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"
	domainpolicy "network-manager/internal/policy"
	domainresolver "network-manager/internal/resolver"

	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy/policy_v1connect"
)

type handler struct {
	service *domainpolicy.Service
}

func Module(db domainpolicy.SQLExecutor) module.Module {
	resolverRepo := domainresolver.NewSQLiteRepository(db)
	service := domainpolicy.NewService(domainpolicy.Config{
		Repo:    domainpolicy.NewSQLiteRepository(db),
		Adapter: domainpolicy.NewAdGuardResolverPolicyAdapter(resolverRepo, domainresolver.NewVaultSecretResolver()),
	})
	path, h := policyconnect.NewPolicyServiceHandler(&handler{service: service})
	return module.Module{Name: "policy", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainpolicy.Schema() }

func (h *handler) PreviewPolicyChange(ctx context.Context, req *connect.Request[policyv1.PreviewPolicyChangeRequest]) (*connect.Response[policyv1.PreviewPolicyChangeResponse], error) {
	change, err := h.service.Preview(ctx, req.Msg.GetTarget(), req.Msg.GetAction(), req.Msg.GetValues())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.PreviewPolicyChangeResponse{Preview: toProtoChange(change)}), nil
}

func (h *handler) ApplyPolicyChange(ctx context.Context, req *connect.Request[policyv1.ApplyPolicyChangeRequest]) (*connect.Response[policyv1.ApplyPolicyChangeResponse], error) {
	change, err := h.service.Apply(ctx, req.Msg.GetPreviewId(), req.Msg.GetApproved())
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.ApplyPolicyChangeResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) RollbackPolicyChange(ctx context.Context, req *connect.Request[policyv1.RollbackPolicyChangeRequest]) (*connect.Response[policyv1.RollbackPolicyChangeResponse], error) {
	change, err := h.service.Rollback(ctx, req.Msg.GetId())
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.RollbackPolicyChangeResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) PauseFiltering(ctx context.Context, req *connect.Request[policyv1.PauseFilteringRequest]) (*connect.Response[policyv1.PauseFilteringResponse], error) {
	change, err := h.service.Pause(ctx, req.Msg.GetTarget(), req.Msg.GetDuration())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.PauseFilteringResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) ResumeFiltering(ctx context.Context, req *connect.Request[policyv1.ResumeFilteringRequest]) (*connect.Response[policyv1.ResumeFilteringResponse], error) {
	change, err := h.service.Resume(ctx, req.Msg.GetTarget())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.ResumeFilteringResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) ListPolicyProfiles(ctx context.Context, req *connect.Request[policyv1.ListPolicyProfilesRequest]) (*connect.Response[policyv1.ListPolicyProfilesResponse], error) {
	profiles, err := h.service.ListProfiles(ctx, req.Msg.GetDeviceGroup())
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.ListPolicyProfilesResponse{Profiles: toProtoProfiles(profiles)}), nil
}

func (h *handler) UpsertPolicyProfile(ctx context.Context, req *connect.Request[policyv1.UpsertPolicyProfileRequest]) (*connect.Response[policyv1.UpsertPolicyProfileResponse], error) {
	profile, err := h.service.UpsertProfile(ctx, toDomainProfile(req.Msg.GetProfile()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.UpsertPolicyProfileResponse{Profile: toProtoProfile(profile)}), nil
}

func (h *handler) EvaluatePolicySchedule(ctx context.Context, req *connect.Request[policyv1.EvaluatePolicyScheduleRequest]) (*connect.Response[policyv1.EvaluatePolicyScheduleResponse], error) {
	var now time.Time
	if req.Msg.GetNow() != "" {
		parsed, err := time.Parse(domainpolicy.TimeFormat, req.Msg.GetNow())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		now = parsed
	}
	evaluation, err := h.service.EvaluateSchedule(ctx, req.Msg.GetProfileId(), req.Msg.GetTarget(), now)
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.EvaluatePolicyScheduleResponse{Evaluation: toProtoEvaluation(evaluation)}), nil
}

func (h *handler) DiagnoseEncryptedDnsBypass(ctx context.Context, req *connect.Request[policyv1.DiagnoseEncryptedDnsBypassRequest]) (*connect.Response[policyv1.DiagnoseEncryptedDnsBypassResponse], error) {
	report := h.service.DiagnoseEncryptedDNSBypass(ctx, req.Msg.GetTarget(), req.Msg.GetAdapterBacked())
	return connect.NewResponse(&policyv1.DiagnoseEncryptedDnsBypassResponse{Report: toProtoGuidanceReport(report)}), nil
}

func (h *handler) GetEndpointDohGuidance(ctx context.Context, req *connect.Request[policyv1.GetEndpointDohGuidanceRequest]) (*connect.Response[policyv1.GetEndpointDohGuidanceResponse], error) {
	report := h.service.EndpointDoHGuidance(ctx, req.Msg.GetPlatform(), req.Msg.GetBrowser(), req.Msg.GetManagementMode())
	return connect.NewResponse(&policyv1.GetEndpointDohGuidanceResponse{Report: toProtoGuidanceReport(report)}), nil
}

func policyError(err error) error {
	switch {
	case errors.Is(err, domainpolicy.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}

func toDomainProfile(profile *policyv1.PolicyProfile) domainpolicy.Profile {
	if profile == nil {
		return domainpolicy.Profile{}
	}
	return domainpolicy.Profile{
		ID:                profile.GetId(),
		Name:              profile.GetName(),
		DeviceGroup:       profile.GetDeviceGroup(),
		FilteringStrength: profile.GetFilteringStrength(),
		Schedule:          profile.GetSchedule(),
		OverrideBehavior:  profile.GetOverrideBehavior(),
		Status:            profile.GetStatus(),
	}
}

func toProtoChange(change domainpolicy.Change) *policyv1.PolicyChange {
	return &policyv1.PolicyChange{
		Id:                change.ID,
		Target:            change.Target,
		Action:            change.Action,
		Status:            change.Status,
		Effects:           change.Effects,
		RollbackSupported: change.RollbackSupported,
	}
}

func toProtoProfiles(profiles []domainpolicy.Profile) []*policyv1.PolicyProfile {
	out := make([]*policyv1.PolicyProfile, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, toProtoProfile(profile))
	}
	return out
}

func toProtoProfile(profile domainpolicy.Profile) *policyv1.PolicyProfile {
	return &policyv1.PolicyProfile{
		Id:                profile.ID,
		Name:              profile.Name,
		DeviceGroup:       profile.DeviceGroup,
		FilteringStrength: profile.FilteringStrength,
		Schedule:          profile.Schedule,
		OverrideBehavior:  profile.OverrideBehavior,
		Status:            profile.Status,
		Effects:           profile.Effects,
		UpdatedAt:         profile.UpdatedAt.Format(domainpolicy.TimeFormat),
	}
}

func toProtoEvaluation(evaluation domainpolicy.ScheduleEvaluation) *policyv1.PolicyScheduleEvaluation {
	nextChangeAt := ""
	if !evaluation.NextChangeAt.IsZero() {
		nextChangeAt = evaluation.NextChangeAt.Format(domainpolicy.TimeFormat)
	}
	return &policyv1.PolicyScheduleEvaluation{
		ProfileId:    evaluation.ProfileID,
		ProfileName:  evaluation.ProfileName,
		Target:       evaluation.Target,
		Active:       evaluation.Active,
		Status:       evaluation.Status,
		Effects:      evaluation.Effects,
		NextChangeAt: nextChangeAt,
	}
}

func toProtoGuidanceReport(report domainpolicy.GuidanceReport) *policyv1.PolicyGuidanceReport {
	checks := make([]*policyv1.GuidanceCheck, 0, len(report.Checks))
	for _, check := range report.Checks {
		checks = append(checks, &policyv1.GuidanceCheck{
			Id:              check.ID,
			Title:           check.Title,
			Status:          check.Status,
			Evidence:        check.Evidence,
			Recommendations: check.Recommendations,
		})
	}
	return &policyv1.PolicyGuidanceReport{
		Id:             report.ID,
		Target:         report.Target,
		Profile:        report.Profile,
		Status:         report.Status,
		Checks:         checks,
		ManualSteps:    report.ManualSteps,
		AdapterActions: report.AdapterActions,
		Guardrails:     report.Guardrails,
		GeneratedAt:    report.GeneratedAt.Format(domainpolicy.TimeFormat),
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("policy_preview", policyconnect.PolicyServicePreviewPolicyChangeProcedure, "Preview filtering policy change"),
	connectEndpoint("policy_apply", policyconnect.PolicyServiceApplyPolicyChangeProcedure, "Apply approved filtering policy change"),
	connectEndpoint("policy_rollback", policyconnect.PolicyServiceRollbackPolicyChangeProcedure, "Rollback filtering policy change"),
	connectEndpoint("policy_pause", policyconnect.PolicyServicePauseFilteringProcedure, "Pause filtering"),
	connectEndpoint("policy_resume", policyconnect.PolicyServiceResumeFilteringProcedure, "Resume filtering"),
	connectEndpoint("policy_profiles_list", policyconnect.PolicyServiceListPolicyProfilesProcedure, "List household policy profiles"),
	connectEndpoint("policy_profiles_upsert", policyconnect.PolicyServiceUpsertPolicyProfileProcedure, "Create or update a household policy profile"),
	connectEndpoint("policy_schedule_evaluate", policyconnect.PolicyServiceEvaluatePolicyScheduleProcedure, "Evaluate a policy schedule window"),
	connectEndpoint("policy_encrypted_dns_bypass", policyconnect.PolicyServiceDiagnoseEncryptedDnsBypassProcedure, "Diagnose IPv6 and encrypted-DNS bypass guidance"),
	connectEndpoint("policy_endpoint_doh_guidance", policyconnect.PolicyServiceGetEndpointDohGuidanceProcedure, "Generate endpoint DoH guidance"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "policy", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"change": "PolicyChange"}}}
}
