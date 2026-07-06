package routing

import (
	"context"
	"database/sql"
	"strings"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"

	"ai-gateway/internal/providers"
	internalrouting "ai-gateway/internal/routing"
)

type Deps struct {
	DB       *sql.DB
	Adapters []providers.Adapter
	Service  *internalrouting.Service
}

type connectHandler struct {
	service *internalrouting.Service
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Service == nil {
		adapters := d.Adapters
		if len(adapters) == 0 {
			adapters = providers.NewDefaultAdapters(nil)
		}
		d.Service = internalrouting.NewSQLService(d.DB, adapters)
	}
	return &connectHandler{service: d.Service}
}

func (h *connectHandler) PreviewRoute(ctx context.Context, req *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error) {
	resp, err := h.service.Preview(ctx, req.Msg.GetRequest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ExecuteRoute(ctx context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	resp, err := h.service.Execute(ctx, req.Msg.GetRequest(), req.Msg.GetInputText())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListRouteEvidence(ctx context.Context, req *connect.Request[routingv1.ListRouteEvidenceRequest]) (*connect.Response[routingv1.ListRouteEvidenceResponse], error) {
	events, err := h.service.ListEvidence(ctx, internalrouting.EvidenceFilter{
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		Limit:    int(req.Msg.GetLimit()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&routingv1.ListRouteEvidenceResponse{Events: events}), nil
}

func (h *connectHandler) GetRouteEvidence(ctx context.Context, req *connect.Request[routingv1.GetRouteEvidenceRequest]) (*connect.Response[routingv1.GetRouteEvidenceResponse], error) {
	event, err := h.service.GetEvidence(ctx, req.Msg.GetEventId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&routingv1.GetRouteEvidenceResponse{Event: event}), nil
}
