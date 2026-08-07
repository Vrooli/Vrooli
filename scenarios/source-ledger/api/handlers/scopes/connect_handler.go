package scopes

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	internalpolicy "source-ledger/internal/policy"
)

type connectHandler struct {
	registry         *internalpolicy.Registry
	registerProvider func(string) error
	logger           *log.Logger
}

func NewConnectHandler(registry *internalpolicy.Registry, registerProvider func(string) error, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{registry: registry, registerProvider: registerProvider, logger: logger}
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
			FrontierTarget: int(pb.GetFrontierTarget()),
			WakeBudget:     int(pb.GetWakeBudget()),
			MaxEntryLines:  int(pb.GetMaxEntryLines()),
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
		response.Scopes = append(response.Scopes, &scopesv1.Scope{Id: item.ID, Label: item.Label, FrontierTarget: int32(item.Config.FrontierTarget), WakeBudget: int32(item.Config.WakeBudget), MaxEntryLines: int32(item.Config.MaxEntryLines)})
	}
	return connect.NewResponse(response), nil
}
