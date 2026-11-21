package apply

import (
	"context"
	"log"

	"brand-manager/internal/apply"

	"connectrpc.com/connect"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply"
)

// Deps wires the seams the Connect apply handler needs.
type Deps struct {
	Service apply.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC apply handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) PreviewApply(ctx context.Context, req *connect.Request[applyv1.PreviewApplyRequest]) (*connect.Response[applyv1.ApplyResponse], error) {
	result, err := h.deps.Service.Preview(ctx, apply.Request{
		BrandID:  req.Msg.GetBrandId(),
		Scenario: req.Msg.GetScenarioName(),
		Elements: req.Msg.GetElements(),
	})
	if err != nil {
		return nil, h.translate("apply.PreviewApply", err)
	}
	return connect.NewResponse(resultToProto(result)), nil
}

func (h *connectHandler) ApplyBrand(ctx context.Context, req *connect.Request[applyv1.ApplyBrandRequest]) (*connect.Response[applyv1.ApplyResponse], error) {
	result, err := h.deps.Service.Apply(ctx, apply.Request{
		BrandID:  req.Msg.GetBrandId(),
		Scenario: req.Msg.GetScenarioName(),
		Elements: req.Msg.GetElements(),
	})
	if err != nil {
		return nil, h.translate("apply.ApplyBrand", err)
	}
	return connect.NewResponse(resultToProto(result)), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := apply.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
