package design

import (
	"context"
	"log"

	"brand-manager/internal/design"

	"connectrpc.com/connect"

	designv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design"
)

// Deps wires the seams the Connect design handler needs.
type Deps struct {
	Service design.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC design handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GenerateDesignLanguage(ctx context.Context, req *connect.Request[designv1.GenerateDesignLanguageRequest]) (*connect.Response[designv1.GenerateDesignLanguageResponse], error) {
	result, err := h.deps.Service.GenerateDesignLanguage(ctx, req.Msg.GetBrandId())
	if err != nil {
		return nil, h.translate("design.GenerateDesignLanguage", err)
	}
	return connect.NewResponse(designToProto(result)), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := design.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
