package integrations

import (
	"context"

	"connectrpc.com/connect"

	"portal/internal/integrations/registry"

	integrationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations"
)

type Handler struct {
	registry *registry.Service
}

func NewHandler(registryService *registry.Service) *Handler {
	return &Handler{registry: registryService}
}

func (h *Handler) Status(ctx context.Context, _ *connect.Request[integrationsv1.StatusRequest]) (*connect.Response[integrationsv1.StatusResponse], error) {
	if h.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errRegistryUnavailable)
	}
	status, err := h.registry.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(registry.ToProtoStatus(status)), nil
}

func (h *Handler) UpdateOverride(ctx context.Context, req *connect.Request[integrationsv1.UpdateOverrideRequest]) (*connect.Response[integrationsv1.UpdateOverrideResponse], error) {
	if h.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errRegistryUnavailable)
	}
	status, err := h.registry.SetOverride(ctx, registry.OverrideFromProto(req.Msg.GetOverride()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&integrationsv1.UpdateOverrideResponse{
		Status: registry.ToProtoStatus(status),
	}), nil
}
