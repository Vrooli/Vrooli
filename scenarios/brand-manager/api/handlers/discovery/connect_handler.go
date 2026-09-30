package discovery

import (
	"context"
	"log"

	"brand-manager/internal/discovery"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery"
)

// Deps wires the seams the Connect discovery handler needs.
type Deps struct {
	Service discovery.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC discovery handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) DiscoverScenario(ctx context.Context, req *connect.Request[discoveryv1.DiscoverScenarioRequest]) (*connect.Response[discoveryv1.DiscoveryResult], error) {
	result, err := h.deps.Service.Discover(ctx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, h.translate("discovery.DiscoverScenario", err)
	}
	return connect.NewResponse(resultToProto(result)), nil
}

func (h *connectHandler) ImportBrand(ctx context.Context, req *connect.Request[discoveryv1.ImportBrandRequest]) (*connect.Response[discoveryv1.ImportBrandResponse], error) {
	result, err := h.deps.Service.Import(ctx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, h.translate("discovery.ImportBrand", err)
	}
	return connect.NewResponse(importResultToProto(result)), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := discovery.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
