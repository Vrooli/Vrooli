package entitlement

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/browser-automation-studio/services/credits"
	entsvc "github.com/vrooli/browser-automation-studio/services/entitlement"
	entitlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement"
)

// service implements entitlementconnect.EntitlementServiceHandler.
type service struct {
	deps Deps
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func (s *service) GetStatus(
	ctx context.Context,
	req *connect.Request[entitlementv1.GetStatusRequest],
) (*connect.Response[entitlementv1.GetStatusResponse], error) {
	ctx = withConsumerAccessToken(ctx, req.Header().Get("Authorization"))
	user := s.resolveUserIdentity(ctx, req.Msg.GetUser())
	status, err := s.buildStatus(ctx, user)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&entitlementv1.GetStatusResponse{Status: status}), nil
}

func (s *service) GetIdentity(
	ctx context.Context,
	_ *connect.Request[entitlementv1.GetIdentityRequest],
) (*connect.Response[entitlementv1.GetIdentityResponse], error) {
	email := ""
	if s.deps.Settings != nil {
		if saved, err := s.deps.Settings.GetSetting(ctx, userIdentitySettingKey); err == nil {
			email = saved
		}
	}
	return connect.NewResponse(&entitlementv1.GetIdentityResponse{Email: email}), nil
}

func (s *service) SetIdentity(
	ctx context.Context,
	req *connect.Request[entitlementv1.SetIdentityRequest],
) (*connect.Response[entitlementv1.GetStatusResponse], error) {
	ctx = withConsumerAccessToken(ctx, req.Header().Get("Authorization"))
	email := strings.TrimSpace(strings.ToLower(req.Msg.GetEmail()))
	if email != "" && !strings.Contains(email, "@") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidEmail)
	}
	if s.deps.Settings != nil {
		if err := s.deps.Settings.SetSetting(ctx, userIdentitySettingKey, email); err != nil {
			s.deps.Logger.WithError(err).Error("entitlement.SetIdentity persist failed")
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if email != "" {
		s.deps.Provider.InvalidateCache(email)
	}
	status, err := s.buildStatus(entsvc.WithUserIdentity(ctx, email), email)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&entitlementv1.GetStatusResponse{Status: status}), nil
}

func (s *service) ClearIdentity(
	ctx context.Context,
	_ *connect.Request[entitlementv1.ClearIdentityRequest],
) (*connect.Response[entitlementv1.ClearIdentityResponse], error) {
	if s.deps.Settings != nil {
		if err := s.deps.Settings.SetSetting(ctx, userIdentitySettingKey, ""); err != nil {
			s.deps.Logger.WithError(err).Error("entitlement.ClearIdentity persist failed")
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&entitlementv1.ClearIdentityResponse{Status: "cleared"}), nil
}

func (s *service) RefreshStatus(
	ctx context.Context,
	req *connect.Request[entitlementv1.RefreshStatusRequest],
) (*connect.Response[entitlementv1.GetStatusResponse], error) {
	ctx = withConsumerAccessToken(ctx, req.Header().Get("Authorization"))
	user := strings.TrimSpace(req.Msg.GetUser())
	if user == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errUserRequired)
	}
	s.deps.Provider.InvalidateCache(user)
	status, err := s.buildStatus(entsvc.WithUserIdentity(ctx, user), user)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&entitlementv1.GetStatusResponse{Status: status}), nil
}

func withConsumerAccessToken(ctx context.Context, authorization string) context.Context {
	const prefix = "Bearer "
	if strings.HasPrefix(authorization, prefix) {
		return entsvc.WithAccessToken(ctx, strings.TrimSpace(strings.TrimPrefix(authorization, prefix)))
	}
	return ctx
}

// ---------------------------------------------------------------------------
// Usage
// ---------------------------------------------------------------------------

func (s *service) GetUsage(
	ctx context.Context,
	req *connect.Request[entitlementv1.GetUsageRequest],
) (*connect.Response[entitlementv1.UsageSummary], error) {
	if s.deps.Credits == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errCreditsUnavailable)
	}
	user := s.resolveUserIdentityRaw(ctx, req.Msg.GetUser())
	summary, err := s.deps.Credits.GetUsage(ctx, user)
	if err != nil {
		s.deps.Logger.WithError(err).Error("entitlement.GetUsage failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(usageSummaryToProto(summary)), nil
}

func (s *service) GetUsageHistory(
	ctx context.Context,
	req *connect.Request[entitlementv1.GetUsageHistoryRequest],
) (*connect.Response[entitlementv1.GetUsageHistoryResponse], error) {
	if s.deps.Credits == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errCreditsUnavailable)
	}
	user := s.resolveUserIdentity(ctx, req.Msg.GetUser())
	months := int(req.Msg.GetMonths())
	if months <= 0 {
		months = defaultHistoryMonths
	}
	offset := int(req.Msg.GetOffset())
	if offset < 0 {
		offset = 0
	}
	periods, hasMore, err := s.deps.Credits.GetUsageHistory(ctx, user, months, offset)
	if err != nil {
		s.deps.Logger.WithError(err).Error("entitlement.GetUsageHistory failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*entitlementv1.UsageSummary, 0, len(periods))
	for i := range periods {
		out = append(out, usageSummaryToProto(&periods[i]))
	}
	return connect.NewResponse(&entitlementv1.GetUsageHistoryResponse{
		UserIdentity: user,
		Periods:      out,
		HasMore:      hasMore,
	}), nil
}

func (s *service) GetOperationLog(
	ctx context.Context,
	req *connect.Request[entitlementv1.GetOperationLogRequest],
) (*connect.Response[entitlementv1.OperationLogPage], error) {
	if s.deps.Credits == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errCreditsUnavailable)
	}
	user := s.resolveUserIdentity(ctx, req.Msg.GetUser())
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultOperationLogLimit
	}
	offset := int(req.Msg.GetOffset())
	if offset < 0 {
		offset = 0
	}
	page, err := s.deps.Credits.GetOperationLog(ctx, user, req.Msg.GetMonth(), req.Msg.GetCategory(), limit, offset)
	if err != nil {
		s.deps.Logger.WithError(err).Error("entitlement.GetOperationLog failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(operationLogPageToProto(page)), nil
}

func (s *service) GetOverride(
	_ context.Context,
	_ *connect.Request[entitlementv1.GetOverrideRequest],
) (*connect.Response[entitlementv1.GetOverrideResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errLocalControlsRemoved)
}

func (s *service) SetOverride(
	_ context.Context,
	_ *connect.Request[entitlementv1.SetOverrideRequest],
) (*connect.Response[entitlementv1.SetOverrideResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errLocalControlsRemoved)
}

func (s *service) ClearOverride(
	_ context.Context,
	_ *connect.Request[entitlementv1.ClearOverrideRequest],
) (*connect.Response[entitlementv1.ClearOverrideResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errLocalControlsRemoved)
}

func (s *service) GetApiSource(
	_ context.Context,
	_ *connect.Request[entitlementv1.GetApiSourceRequest],
) (*connect.Response[entitlementv1.ApiSourceConfig], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errApiSourceRemoved)
}

func (s *service) SetApiSource(
	_ context.Context,
	_ *connect.Request[entitlementv1.SetApiSourceRequest],
) (*connect.Response[entitlementv1.ApiSourceConfig], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errApiSourceRemoved)
}

func (s *service) ClearApiSource(
	_ context.Context,
	_ *connect.Request[entitlementv1.ClearApiSourceRequest],
) (*connect.Response[entitlementv1.ClearApiSourceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errApiSourceRemoved)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

const (
	userIdentitySettingKey   = "user_identity"
	defaultHistoryMonths     = 6
	defaultOperationLogLimit = 20
)

// resolveUserIdentity matches the legacy fallback chain:
// context → query-arg → stored setting → "anonymous".
func (s *service) resolveUserIdentity(ctx context.Context, requested string) string {
	if id := entsvc.UserIdentityFromContext(ctx); id != "" {
		return id
	}
	if t := strings.TrimSpace(requested); t != "" {
		return t
	}
	if s.deps.Settings != nil {
		if saved, err := s.deps.Settings.GetSetting(ctx, userIdentitySettingKey); err == nil && saved != "" {
			return saved
		}
	}
	return "anonymous"
}

// resolveUserIdentityRaw is the variant used by GetUsage, which historically
// did not fall back to stored settings nor to "anonymous" (it just passed an
// empty string through to the credit service when nothing was found).
func (s *service) resolveUserIdentityRaw(ctx context.Context, requested string) string {
	if id := entsvc.UserIdentityFromContext(ctx); id != "" {
		return id
	}
	return strings.TrimSpace(requested)
}

// buildStatus assembles the full EntitlementStatus payload, replicating the
// historical legacy/REST shape so wire compatibility for the UI is exact.
func (s *service) buildStatus(ctx context.Context, user string) (*entitlementv1.EntitlementStatus, error) {
	if user == "" {
		user = "anonymous"
	}

	var ent *entsvc.Entitlement
	var err error
	ent, err = s.deps.Provider.GetEntitlement(ctx, user)
	if err != nil {
		ent = &entsvc.Entitlement{
			UserIdentity: user,
			Status:       entsvc.StatusInactive,
			Tier:         "free",
		}
	}

	var usage *credits.UsageSummary
	if s.deps.Credits != nil && user != "" {
		u, getErr := s.deps.Credits.GetUsage(ctx, user)
		if getErr == nil {
			usage = u
		}
	}

	usedCount := 0
	monthlyLimit := 0
	if limit, found := ent.LimitValue("ai_credits"); found {
		monthlyLimit = int(limit)
	}
	remaining := monthlyLimit
	aiCreditsUsed := 0
	aiRequestsCount := 0
	aiResetDate := ""

	if usage != nil {
		usedCount = usage.TotalCreditsUsed
		monthlyLimit = usage.CreditsLimit
		remaining = usage.CreditsRemaining
		aiCreditsUsed = usage.TotalCreditsUsed
		aiRequestsCount = usage.TotalOperations
		if !usage.ResetDate.IsZero() {
			aiResetDate = usage.ResetDate.Format("2006-01-02")
		}
	}

	status := &entitlementv1.EntitlementStatus{
		UserIdentity:        user,
		Status:              string(ent.Status),
		Tier:                string(ent.Tier),
		IsActive:            ent.IsActive(),
		Features:            append([]string(nil), ent.Features...),
		FeatureAccess:       s.buildFeatureAccess(ent),
		MonthlyLimit:        int32(monthlyLimit),
		MonthlyUsed:         int32(usedCount),
		MonthlyRemaining:    int32(remaining),
		RequiresWatermark:   s.resolveRequiresWatermark(ctx, user),
		CanUseAi:            s.resolveCanUseAI(ctx, user),
		CanUseRecording:     s.resolveCanUseRecording(ctx, user),
		EntitlementsEnabled: true,
		AiCreditsUsed:       int32(aiCreditsUsed),
		AiCreditsLimit:      int32(monthlyLimit),
		AiCreditsRemaining:  int32(remaining),
		AiRequestsCount:     int32(aiRequestsCount),
		AiResetDate:         aiResetDate,
	}
	return status, nil
}

func (s *service) resolveRequiresWatermark(ctx context.Context, user string) bool {
	return s.deps.Provider.RequiresWatermark(ctx, user)
}

func (s *service) resolveCanUseAI(ctx context.Context, user string) bool {
	return s.deps.Provider.CanUseAI(ctx, user)
}

func (s *service) resolveCanUseRecording(ctx context.Context, user string) bool {
	return s.deps.Provider.CanUseRecording(ctx, user)
}

func (s *service) buildFeatureAccess(ent *entsvc.Entitlement) []*entitlementv1.FeatureAccess {
	canUseAI := ent.HasFeature(entsvc.FeatureAI)
	canUseRecording := ent.HasFeature(entsvc.FeatureRecording)
	requiresWatermark := !ent.HasFeature(entsvc.FeatureWatermarkFree)

	return []*entitlementv1.FeatureAccess{
		{
			Id:           "ai",
			Label:        "AI-Powered Features",
			Description:  "Use AI to generate and edit workflows automatically",
			RequiredTier: "signed lease",
			HasAccess:    canUseAI,
		},
		{
			Id:           "recording",
			Label:        "Live Recording",
			Description:  "Record browser interactions to create workflows",
			RequiredTier: "signed lease",
			HasAccess:    canUseRecording,
		},
		{
			Id:           "watermark-free",
			Label:        "Watermark-Free Exports",
			Description:  "Export videos and replays without watermarks",
			RequiredTier: "signed lease",
			HasAccess:    !requiresWatermark,
		},
	}
}

// ---------------------------------------------------------------------------
// proto conversion helpers
// ---------------------------------------------------------------------------

func usageSummaryToProto(u *credits.UsageSummary) *entitlementv1.UsageSummary {
	if u == nil {
		return &entitlementv1.UsageSummary{}
	}
	byOp := make(map[string]int32, len(u.ByOperation))
	for k, v := range u.ByOperation {
		byOp[string(k)] = int32(v)
	}
	opCounts := make(map[string]int32, len(u.OperationCounts))
	for k, v := range u.OperationCounts {
		opCounts[string(k)] = int32(v)
	}
	out := &entitlementv1.UsageSummary{
		UserIdentity:     u.UserIdentity,
		BillingMonth:     u.BillingMonth,
		TotalCreditsUsed: int32(u.TotalCreditsUsed),
		TotalOperations:  int32(u.TotalOperations),
		ByOperation:      byOp,
		OperationCounts:  opCounts,
		CreditsLimit:     int32(u.CreditsLimit),
		CreditsRemaining: int32(u.CreditsRemaining),
	}
	if !u.PeriodStart.IsZero() {
		out.PeriodStart = timestamppb.New(u.PeriodStart)
	}
	if !u.PeriodEnd.IsZero() {
		out.PeriodEnd = timestamppb.New(u.PeriodEnd)
	}
	if !u.ResetDate.IsZero() {
		out.ResetDate = timestamppb.New(u.ResetDate)
	}
	return out
}

func operationLogPageToProto(p *credits.OperationLogPage) *entitlementv1.OperationLogPage {
	if p == nil {
		return &entitlementv1.OperationLogPage{}
	}
	entries := make([]*entitlementv1.OperationLogEntry, 0, len(p.Operations))
	for i := range p.Operations {
		entries = append(entries, operationLogEntryToProto(&p.Operations[i]))
	}
	return &entitlementv1.OperationLogPage{
		UserIdentity: p.UserIdentity,
		BillingMonth: p.BillingMonth,
		Operations:   entries,
		Total:        int32(p.Total),
		Limit:        int32(p.Limit),
		Offset:       int32(p.Offset),
		HasMore:      p.HasMore,
	}
}

func operationLogEntryToProto(e *credits.OperationLogEntry) *entitlementv1.OperationLogEntry {
	out := &entitlementv1.OperationLogEntry{
		Id:             e.ID,
		OperationType:  string(e.OperationType),
		CreditsCharged: int32(e.CreditsCharged),
		Success:        e.Success,
		ErrorMessage:   e.ErrorMessage,
	}
	if !e.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(e.CreatedAt)
	}
	if len(e.Metadata) > 0 {
		if s, err := structpb.NewStruct(e.Metadata); err == nil {
			out.Metadata = s
		}
	}
	return out
}
