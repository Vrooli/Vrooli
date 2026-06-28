package generation

import (
	"context"
	"log"

	"brand-manager/internal/generation"

	"connectrpc.com/connect"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
)

// Deps wires the seams the Connect generation handler needs.
type Deps struct {
	Service generation.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC generation handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetProviderStatus(ctx context.Context, _ *connect.Request[generationv1.GetProviderStatusRequest]) (*connect.Response[generationv1.GetProviderStatusResponse], error) {
	available, statuses := h.deps.Service.ProviderStatus(ctx)
	return connect.NewResponse(&generationv1.GetProviderStatusResponse{
		Available: available,
		Providers: statusesToProto(statuses),
	}), nil
}

func (h *connectHandler) GenerateBrandElements(ctx context.Context, req *connect.Request[generationv1.GenerateBrandElementsRequest]) (*connect.Response[generationv1.GenerateBrandElementsResponse], error) {
	result, err := h.deps.Service.GenerateElements(ctx, req.Msg.GetBrandId(), req.Msg.GetElements(), req.Msg.GetModel())
	if err != nil {
		return nil, h.translate("generation.GenerateBrandElements", err)
	}
	return connect.NewResponse(elementsResultToProto(result)), nil
}

func (h *connectHandler) GenerateBrandImage(ctx context.Context, req *connect.Request[generationv1.GenerateBrandImageRequest]) (*connect.Response[generationv1.GenerateBrandImageResponse], error) {
	result, err := h.deps.Service.GenerateImage(ctx, req.Msg.GetBrandId(), req.Msg.GetType(), req.Msg.GetModel())
	if err != nil {
		return nil, h.translate("generation.GenerateBrandImage", err)
	}
	return connect.NewResponse(imageResultToProto(result)), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := generation.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
