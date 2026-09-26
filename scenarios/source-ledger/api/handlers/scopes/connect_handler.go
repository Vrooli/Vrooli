package scopes

import (
	"context"
	"errors"
	"log"

	internalpolicy "source-ledger/internal/policy"
	internalrecall "source-ledger/internal/recall"

	"connectrpc.com/connect"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
)

type livenessProvider interface {
	Liveness(context.Context) (internalrecall.CompactionLiveness, error)
}

type connectHandler struct {
	registry         *internalpolicy.Registry
	registerProvider func(string) error
	logger           *log.Logger
	liveness         livenessProvider
}

func NewConnectHandler(registry *internalpolicy.Registry, registerProvider func(string) error, logger *log.Logger, liveness ...livenessProvider) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	h := &connectHandler{registry: registry, registerProvider: registerProvider, logger: logger}
	if len(liveness) > 0 {
		h.liveness = liveness[0]
	}
	return h
}

func (h *connectHandler) CreateScope(ctx context.Context, req *connect.Request[scopesv1.CreateScopeRequest]) (*connect.Response[scopesv1.CreateScopeResponse], error) {
	if req.Msg.GetScope() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scope is required"))
	}
	pb := req.Msg.GetScope()
	definition := internalpolicy.ScopeDefinition{
		ID:    pb.GetId(),
		Label: pb.GetLabel(),
		Config: internalpolicy.Config{
			FrontierTarget:  int(pb.GetFrontierTarget()),
			WakeBudget:      int(pb.GetWakeBudget()),
			WakeBudgetChars: int(pb.GetWakeBudgetChars()),
			MaxEntryLines:   int(pb.GetMaxEntryLines()),
			MaxEntryChars:   int(pb.GetMaxEntryChars()),
		},
	}
	for _, facet := range pb.GetFacets() {
		definition.Facets = append(definition.Facets, internalpolicy.FacetDefinition{ID: facet.GetId(), Label: facet.GetLabel(), Guidance: facet.GetGuidance(), RetentionPolicy: facet.GetRetentionPolicy(), CompactionEligible: facet.GetCompactionEligible(), ResidentBudget: int(facet.GetResidentBudget())})
	}
	if err := h.registry.Create(ctx, definition); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.registerProvider != nil {
		if err := h.registerProvider(definition.ID); err != nil {
			h.logger.Printf("scope %q created but search provider registration degraded: %v", definition.ID, err)
		}
	}
	return connect.NewResponse(&scopesv1.CreateScopeResponse{Scope: pb}), nil
}

func (h *connectHandler) ListScopes(ctx context.Context, _ *connect.Request[scopesv1.ListScopesRequest]) (*connect.Response[scopesv1.ListScopesResponse], error) {
	items, err := h.registry.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &scopesv1.ListScopesResponse{}
	for _, item := range items {
		response.Scopes = append(response.Scopes, &scopesv1.Scope{Id: item.ID, Label: item.Label, FrontierTarget: int32(item.Config.FrontierTarget), WakeBudget: int32(item.Config.WakeBudget), MaxEntryLines: int32(item.Config.MaxEntryLines), WakeBudgetChars: int32(item.Config.WakeBudgetChars), MaxEntryChars: int32(item.Config.MaxEntryChars)})
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetPolicy(ctx context.Context, req *connect.Request[scopesv1.GetPolicyRequest]) (*connect.Response[scopesv1.GetPolicyResponse], error) {
	effective, err := h.registry.Resolve(ctx, req.Msg.GetScope())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&scopesv1.GetPolicyResponse{Effective: snapshot(effective), Defaults: snapshot(h.registry.Defaults()), Liveness: h.livenessSnapshot(internalpolicy.WithScope(ctx, req.Msg.GetScope()))}), nil
}

func (h *connectHandler) SetPolicy(ctx context.Context, req *connect.Request[scopesv1.SetPolicyRequest]) (*connect.Response[scopesv1.SetPolicyResponse], error) {
	override := internalpolicy.Override{}
	if req.Msg.FrontierTarget != nil {
		value := int(req.Msg.GetFrontierTarget())
		override.FrontierTarget = &value
	}
	if req.Msg.WakeBudgetLines != nil {
		value := int(req.Msg.GetWakeBudgetLines())
		override.WakeBudget = &value
	}
	if req.Msg.WakeBudgetChars != nil {
		value := int(req.Msg.GetWakeBudgetChars())
		override.WakeBudgetChars = &value
	}
	if req.Msg.MaxEntryLines != nil {
		value := int(req.Msg.GetMaxEntryLines())
		override.MaxEntryLines = &value
	}
	if req.Msg.MaxEntryChars != nil {
		value := int(req.Msg.GetMaxEntryChars())
		override.MaxEntryChars = &value
	}
	effective, err := h.registry.SetOverride(ctx, req.Msg.GetScope(), override)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&scopesv1.SetPolicyResponse{Effective: snapshot(effective), Defaults: snapshot(h.registry.Defaults()), Liveness: h.livenessSnapshot(internalpolicy.WithScope(ctx, req.Msg.GetScope()))}), nil
}

func (h *connectHandler) ResetPolicy(ctx context.Context, req *connect.Request[scopesv1.ResetPolicyRequest]) (*connect.Response[scopesv1.ResetPolicyResponse], error) {
	effective, err := h.registry.ResetOverride(ctx, req.Msg.GetScope())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&scopesv1.ResetPolicyResponse{Effective: snapshot(effective), Defaults: snapshot(h.registry.Defaults()), Liveness: h.livenessSnapshot(internalpolicy.WithScope(ctx, req.Msg.GetScope()))}), nil
}

func (h *connectHandler) livenessSnapshot(ctx context.Context) *scopesv1.CompactionLiveness {
	if h.liveness == nil {
		return nil
	}
	value, err := h.liveness.Liveness(ctx)
	if err != nil {
		h.logger.Printf("read compaction liveness: %v", err)
		return nil
	}
	return &scopesv1.CompactionLiveness{UnsummarizedLeafCount: int32(value.UnsummarizedLeafCount), OldestUnsummarizedLeafAt: value.OldestUnsummarizedLeafAt, LastSummaryAt: value.LastSummaryAt}
}

func snapshot(config internalpolicy.Config) *scopesv1.PolicySnapshot {
	return &scopesv1.PolicySnapshot{
		FrontierTarget: int32(config.FrontierTarget), WakeBudgetLines: int32(config.WakeBudget), WakeBudgetChars: int32(config.WakeBudgetChars), MaxEntryLines: int32(config.MaxEntryLines), MaxEntryChars: int32(config.MaxEntryChars),
		FrontierTargetOrigin: config.Origins.FrontierTarget, WakeBudgetLinesOrigin: config.Origins.WakeBudget, WakeBudgetCharsOrigin: config.Origins.WakeBudgetChars, MaxEntryLinesOrigin: config.Origins.MaxEntryLines, MaxEntryCharsOrigin: config.Origins.MaxEntryChars,
	}
}
