package resources

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
	resourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources/resourcesv1connect"
	internalresources "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/resources"
)

type connectHandler struct{ service internalresources.Service }

func NewConnectHandler(service internalresources.Service) resourcesconnect.ResourcesServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListResources(ctx context.Context, _ *connect.Request[resourcesv1.ListResourcesRequest]) (*connect.Response[resourcesv1.ListResourcesResponse], error) {
	value, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list resources: %w", err))
	}
	return connect.NewResponse(value), nil
}

func (h *connectHandler) GetResource(ctx context.Context, req *connect.Request[resourcesv1.GetResourceRequest]) (*connect.Response[resourcesv1.GetResourceResponse], error) {
	resource, err := h.service.Get(ctx, req.Msg.GetName())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&resourcesv1.GetResourceResponse{Resource: resource}), nil
}

func (h *connectHandler) GetResourceHealth(ctx context.Context, _ *connect.Request[resourcesv1.GetResourceHealthRequest]) (*connect.Response[resourcesv1.GetResourceHealthResponse], error) {
	value, err := h.service.Health(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get resource health: %w", err))
	}
	return connect.NewResponse(value), nil
}

func (h *connectHandler) ListDerivedResources(ctx context.Context, _ *connect.Request[resourcesv1.ListDerivedResourcesRequest]) (*connect.Response[resourcesv1.ListDerivedResourcesResponse], error) {
	value, err := h.service.Derived(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list derived resources: %w", err))
	}
	return connect.NewResponse(value), nil
}
