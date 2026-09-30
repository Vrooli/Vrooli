package gateway

import (
	"context"

	"connectrpc.com/connect"
	gatewayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway"

	internalgateway "ai-gateway/internal/gateway"
)

type Deps struct {
	Validator *internalgateway.Service
}

type connectHandler struct {
	validator *internalgateway.Service
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Validator == nil {
		d.Validator = internalgateway.New()
	}
	return &connectHandler{validator: d.Validator}
}

func (h *connectHandler) ValidateGatewayRequest(_ context.Context, req *connect.Request[gatewayv1.ValidateGatewayRequestRequest]) (*connect.Response[gatewayv1.ValidateGatewayRequestResponse], error) {
	issues := h.validator.Validate(req.Msg.GetRequest())
	return connect.NewResponse(&gatewayv1.ValidateGatewayRequestResponse{
		Valid:            len(issues) == 0,
		Issues:           issues,
		AcceptedProfiles: internalgateway.AcceptedProfiles(),
	}), nil
}
