package plans

import (
	"context"
	"log"

	"data-backup-manager/internal/plans"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans"
)

// Deps wires the seams the Connect plans handler needs.
type Deps struct {
	Service plans.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the plans Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) CreatePlan(ctx context.Context, req *connect.Request[plansv1.CreatePlanRequest]) (*connect.Response[plansv1.CreatePlanResponse], error) {
	in := plans.CreateInput{
		Name:                    req.Msg.Name,
		TargetIDs:               req.Msg.TargetIds,
		DestinationIDs:          req.Msg.DestinationIds,
		Schedule:                req.Msg.Schedule,
		Enabled:                 req.Msg.Enabled,
		AllowIncompleteCoverage: req.Msg.AllowIncompleteCoverage,
		ProtectionTier:          protectionTierFromProto(req.Msg.ProtectionTier),
		RecoveryDrillSchedule:   req.Msg.RecoveryDrillSchedule,
		TargetSelector:          plans.TargetSelector(req.Msg.TargetSelector),
		ExcludedTargetIDs:       req.Msg.ExcludedTargetIds,
	}
	if req.Msg.Retention != nil {
		in.KeepLatest = req.Msg.Retention.KeepLatest
		in.KeepHourly, in.KeepDaily, in.KeepWeekly = req.Msg.Retention.KeepHourly, req.Msg.Retention.KeepDaily, req.Msg.Retention.KeepWeekly
		in.KeepMonthly, in.KeepAnnual = req.Msg.Retention.KeepMonthly, req.Msg.Retention.KeepAnnual
	}
	p, err := h.deps.Service.Create(ctx, in)
	if err != nil {
		return nil, h.translate("CreatePlan", err)
	}
	return connect.NewResponse(&plansv1.CreatePlanResponse{Plan: domainToProto(p)}), nil
}

func (h *connectHandler) GetPlan(ctx context.Context, req *connect.Request[plansv1.GetPlanRequest]) (*connect.Response[plansv1.GetPlanResponse], error) {
	p, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetPlan", err)
	}
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: domainToProto(p)}), nil
}

func (h *connectHandler) ListPlans(ctx context.Context, req *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	var list []plans.Plan
	var next string
	var err error
	if pager, ok := h.deps.Service.(plans.PageService); ok {
		list, next, err = pager.ListPage(ctx, int(req.Msg.PageSize), req.Msg.PageToken)
	} else {
		list, err = h.deps.Service.List(ctx, int(req.Msg.PageSize))
	}
	if err != nil {
		return nil, h.translate("ListPlans", err)
	}
	resp := &plansv1.ListPlansResponse{Plans: make([]*plansv1.Plan, 0, len(list)), NextPageToken: next}
	for _, p := range list {
		resp.Plans = append(resp.Plans, domainToProto(p))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) UpdatePlan(ctx context.Context, req *connect.Request[plansv1.UpdatePlanRequest]) (*connect.Response[plansv1.UpdatePlanResponse], error) {
	in := plans.UpdateInput{
		ID:                      req.Msg.Id,
		Name:                    req.Msg.Name,
		TargetIDs:               req.Msg.TargetIds,
		DestinationIDs:          req.Msg.DestinationIds,
		Schedule:                req.Msg.Schedule,
		Enabled:                 req.Msg.Enabled,
		AllowIncompleteCoverage: req.Msg.AllowIncompleteCoverage,
		ProtectionTier:          protectionTierFromProto(req.Msg.ProtectionTier),
		RecoveryDrillSchedule:   req.Msg.RecoveryDrillSchedule,
		TargetSelector:          plans.TargetSelector(req.Msg.TargetSelector),
		ExcludedTargetIDs:       req.Msg.ExcludedTargetIds,
	}
	if req.Msg.Retention != nil {
		in.KeepLatest = req.Msg.Retention.KeepLatest
		in.KeepHourly, in.KeepDaily, in.KeepWeekly = req.Msg.Retention.KeepHourly, req.Msg.Retention.KeepDaily, req.Msg.Retention.KeepWeekly
		in.KeepMonthly, in.KeepAnnual = req.Msg.Retention.KeepMonthly, req.Msg.Retention.KeepAnnual
	}
	p, err := h.deps.Service.Update(ctx, in)
	if err != nil {
		return nil, h.translate("UpdatePlan", err)
	}
	return connect.NewResponse(&plansv1.UpdatePlanResponse{Plan: domainToProto(p)}), nil
}

func (h *connectHandler) DeletePlan(ctx context.Context, req *connect.Request[plansv1.DeletePlanRequest]) (*connect.Response[plansv1.DeletePlanResponse], error) {
	removed, err := h.deps.Service.Delete(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("DeletePlan", err)
	}
	return connect.NewResponse(&plansv1.DeletePlanResponse{Removed: removed}), nil
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := plans.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("plans.%s: %v", op, err)
	}
	return connectErr
}

// domainToProto converts the internal Plan to its wire shape.
func domainToProto(p plans.Plan) *plansv1.Plan {
	pp := &plansv1.Plan{
		Id:                                p.ID,
		Name:                              p.Name,
		TargetIds:                         p.TargetIDs,
		DestinationIds:                    p.DestinationIDs,
		Schedule:                          p.Schedule,
		Enabled:                           p.Enabled,
		ProtectionTier:                    protectionTierToProto(p.ProtectionTier),
		RecoveryDrillSchedule:             p.RecoveryDrillSchedule,
		DestinationsPhysicallyIndependent: p.DestinationsPhysicallyIndependent,
		SharedRiskWarnings:                p.SharedRiskWarnings,
		TargetSelector:                    string(p.TargetSelector),
		ExcludedTargetIds:                 p.ExcludedTargetIDs,
		Retention: &plansv1.RetentionPolicy{
			KeepLatest: p.KeepLatest,
			KeepHourly: p.KeepHourly, KeepDaily: p.KeepDaily, KeepWeekly: p.KeepWeekly, KeepMonthly: p.KeepMonthly, KeepAnnual: p.KeepAnnual,
		},
	}
	if !p.CreatedAt.IsZero() {
		pp.CreatedAt = timestamppb.New(p.CreatedAt)
	}
	if !p.UpdatedAt.IsZero() {
		pp.UpdatedAt = timestamppb.New(p.UpdatedAt)
	}
	return pp
}

func protectionTierFromProto(t plansv1.ProtectionTier) plans.ProtectionTier {
	switch t {
	case plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_PRIMARY:
		return plans.TierCriticalPrimary
	case plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_SECONDARY:
		return plans.TierCriticalSecondary
	default:
		return plans.TierFullPrimary
	}
}

func protectionTierToProto(t plans.ProtectionTier) plansv1.ProtectionTier {
	switch t {
	case plans.TierCriticalPrimary:
		return plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_PRIMARY
	case plans.TierCriticalSecondary:
		return plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_SECONDARY
	default:
		return plansv1.ProtectionTier_PROTECTION_TIER_FULL_PRIMARY
	}
}
