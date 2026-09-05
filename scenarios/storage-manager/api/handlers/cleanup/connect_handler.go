package cleanup

import (
	"context"
	"errors"
	"fmt"

	cleanupcore "storage-manager/internal/cleanup"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/policy"

	"connectrpc.com/connect"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Catalog() []cleanupcore.ProviderMetadata
	CurrentPolicy(context.Context) (orchestrator.Policy, error)
	SetPolicyProfile(context.Context, policy.ProfileName) (orchestrator.Policy, error)
	Plan(context.Context, cleanupcore.ObservationScope) (orchestrator.Plan, error)
	Apply(context.Context, orchestrator.ApplyInput) (orchestrator.ApplyReport, error)
	Audit(context.Context) ([]orchestrator.AuditEvent, error)
	ReportPressure(context.Context, orchestrator.PressureSignal) (orchestrator.PressureOutcome, error)
}

type recoveryService interface {
	StartRecovery(context.Context, string, string, float64, int64, float64, bool) (orchestrator.RecoveryRun, error)
	WaitRecovery(context.Context, string) (orchestrator.RecoveryRun, error)
	ListRecovery(int) []orchestrator.RecoveryRun
}

type durableRecoveryService interface {
	ListRecoveryContext(context.Context, int) ([]orchestrator.RecoveryRun, error)
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

// ReportPressure is the inbound disk-pressure entry point.
//
// An unrecognised band is rejected rather than defaulted. Critical is the band
// that authorises deletion with no operator present, so a value storage-manager
// does not understand must never be interpreted as one.
func (h *connectHandler) ReportPressure(ctx context.Context, req *connect.Request[cleanupv1.ReportPressureRequest]) (*connect.Response[cleanupv1.ReportPressureResponse], error) {
	band, err := bandFromProto(req.Msg.GetBand())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	signal := orchestrator.PressureSignal{
		SourceScenario:       req.Msg.GetSourceScenario(),
		Partition:            req.Msg.GetPartition(),
		UsedPercent:          req.Msg.GetUsedPercent(),
		Band:                 band,
		AvailableBytes:       req.Msg.GetAvailableBytes(),
		FillRateBytesPerHour: req.Msg.GetFillRateBytesPerHour(),
		Trigger:              req.Msg.GetTrigger().String(),
	}
	for _, writer := range req.Msg.GetHotWriters() {
		if writer == nil {
			continue
		}
		signal.HotWriters = append(signal.HotWriters, orchestrator.HotWriter{
			Root:          writer.GetRoot(),
			CurrentBytes:  writer.GetCurrentBytes(),
			BytesPerHour:  writer.GetBytesPerHour(),
			WindowSeconds: writer.GetWindowSeconds(),
		})
	}

	outcome, err := h.service.ReportPressure(ctx, signal)
	if err != nil {
		// Only a malformed signal is the caller's fault. Everything reachable
		// past validation — a failed plan, an unreachable owner scenario — is
		// storage-manager's own problem, and reporting it as InvalidArgument
		// tells a reporting safeguard to stop retrying a request that was
		// perfectly valid. This masked a real failure: a plan that exceeded the
		// write timeout came back to the caller as a 400.
		if errors.Is(err, orchestrator.ErrInvalidPressureSignal) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	reason := outcome.Reason
	if outcome.BugReference != "" {
		if reason != "" {
			reason += "; "
		}
		reason += fmt.Sprintf("warning bug reference: %s", outcome.BugReference)
	}
	return connect.NewResponse(&cleanupv1.ReportPressureResponse{
		Band:                   bandToProto(outcome.Band),
		Action:                 actionToProto(outcome.Action),
		PlanId:                 outcome.PlanID,
		EstimatedBytes:         outcome.EstimatedBytes,
		ReclaimedBytes:         outcome.ReclaimedBytes,
		ProvidersApplied:       append([]string(nil), outcome.ProvidersApplied...),
		ProvidersWithheld:      append([]string(nil), outcome.ProvidersWithheld...),
		Reason:                 reason,
		AutonomousApplyEnabled: outcome.AutonomousApplyEnabled,
		BugReference:           outcome.BugReference,
		RunId:                  outcome.RunID,
	}), nil
}

func (h *connectHandler) StartRecovery(ctx context.Context, req *connect.Request[cleanupv1.RecoveryRunRequest]) (*connect.Response[cleanupv1.RecoveryRunResponse], error) {
	svc, ok := h.service.(recoveryService)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("recovery runs are unavailable"))
	}
	run, err := svc.StartRecovery(ctx, req.Msg.GetTrigger().String(), req.Msg.GetPartition(), req.Msg.GetUsedPercent(), req.Msg.GetAvailableBytes(), req.Msg.GetTargetFreePercent(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(recoveryRunToProto(run)), nil
}

func (h *connectHandler) WaitRecovery(ctx context.Context, req *connect.Request[cleanupv1.RecoveryWaitRequest]) (*connect.Response[cleanupv1.RecoveryRunResponse], error) {
	svc, ok := h.service.(recoveryService)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("recovery runs are unavailable"))
	}
	run, err := svc.WaitRecovery(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(recoveryRunToProto(run)), nil
}

func (h *connectHandler) ListRecovery(ctx context.Context, req *connect.Request[cleanupv1.RecoveryHistoryRequest]) (*connect.Response[cleanupv1.RecoveryHistoryResponse], error) {
	svc, ok := h.service.(recoveryService)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("recovery runs are unavailable"))
	}
	var runs []orchestrator.RecoveryRun
	var err error
	if durable, ok := h.service.(durableRecoveryService); ok {
		runs, err = durable.ListRecoveryContext(ctx, int(req.Msg.GetLimit()))
	} else {
		runs = svc.ListRecovery(int(req.Msg.GetLimit()))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &cleanupv1.RecoveryHistoryResponse{}
	for _, run := range runs {
		resp.Runs = append(resp.Runs, recoveryRunToProto(run))
	}
	return connect.NewResponse(resp), nil
}

func recoveryRunToProto(run orchestrator.RecoveryRun) *cleanupv1.RecoveryRunResponse {
	return &cleanupv1.RecoveryRunResponse{
		RunId: run.ID, Status: run.Status, Trigger: run.Trigger, Partition: run.Partition,
		Action: string(run.Action), EstimatedBytes: run.EstimatedBytes, ReclaimedBytes: run.ReclaimedBytes,
		PlanId: run.PlanID, Reason: run.Reason, StartedAt: timestamppb.New(run.StartedAt), CompletedAt: timestamppb.New(run.CompletedAt),
		TargetFreeBytes: run.TargetFreeBytes, StoppedBecause: run.StoppedBecause,
	}
}

// bandFromProto maps the wire enum onto the domain band, refusing UNSPECIFIED
// and any value this build does not know.
func bandFromProto(band cleanupv1.PressureBand) (orchestrator.PressureBand, error) {
	switch band {
	case cleanupv1.PressureBand_PRESSURE_BAND_WARNING:
		return orchestrator.BandWarning, nil
	case cleanupv1.PressureBand_PRESSURE_BAND_HIGH:
		return orchestrator.BandHigh, nil
	case cleanupv1.PressureBand_PRESSURE_BAND_CRITICAL:
		return orchestrator.BandCritical, nil
	default:
		return "", fmt.Errorf("unknown pressure band %q", band.String())
	}
}

func bandToProto(band orchestrator.PressureBand) cleanupv1.PressureBand {
	switch band {
	case orchestrator.BandWarning:
		return cleanupv1.PressureBand_PRESSURE_BAND_WARNING
	case orchestrator.BandHigh:
		return cleanupv1.PressureBand_PRESSURE_BAND_HIGH
	case orchestrator.BandCritical:
		return cleanupv1.PressureBand_PRESSURE_BAND_CRITICAL
	default:
		return cleanupv1.PressureBand_PRESSURE_BAND_UNSPECIFIED
	}
}

func actionToProto(action orchestrator.PressureAction) cleanupv1.PressureAction {
	switch action {
	case orchestrator.ActionObserved:
		return cleanupv1.PressureAction_PRESSURE_ACTION_OBSERVED
	case orchestrator.ActionPreviewed:
		return cleanupv1.PressureAction_PRESSURE_ACTION_PREVIEWED
	case orchestrator.ActionApplied:
		return cleanupv1.PressureAction_PRESSURE_ACTION_APPLIED
	case orchestrator.ActionDeduplicated:
		return cleanupv1.PressureAction_PRESSURE_ACTION_DEDUPLICATED
	case orchestrator.ActionSuppressed:
		return cleanupv1.PressureAction_PRESSURE_ACTION_SUPPRESSED
	default:
		return cleanupv1.PressureAction_PRESSURE_ACTION_UNSPECIFIED
	}
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
		Id:                plan.ID,
		PolicyVersion:     plan.PolicyVersion,
		CreatedAt:         timestamppb.New(plan.CreatedAt),
		TotalBytes:        plan.TotalBytes,
		TotalItems:        int32(plan.TotalItems),
		CensusId:          plan.CensusID,
		CensusStatus:      plan.CensusStatus,
		CensusStartedAt:   timestamppb.New(plan.CensusStartedAt),
		CensusCompletedAt: timestamppb.New(plan.CensusCompletedAt),
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
