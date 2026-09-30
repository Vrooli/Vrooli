package interactive

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	interactiveconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive/interactive_v1connect"
	"vrooli-bridge/internal/auth"
	internalinteractive "vrooli-bridge/internal/interactive"
	"vrooli-bridge/internal/module"
)

type Deps struct {
	Admission     *internalinteractive.Admission
	AuthorizeNode func(context.Context, string) bool
	SignalBroker  *internalinteractive.SignalBroker
	RevokePusher  internalinteractive.RevokePusher
	Logger        *log.Logger
}
type connectHandler struct{ deps Deps }

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Admission == nil {
		d.Admission = internalinteractive.NewAdmission()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) OpenChannel(ctx context.Context, req *connect.Request[interactivev1.OpenChannelRequest]) (*connect.Response[interactivev1.OpenChannelResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, mapError(internalinteractive.ErrInvalid)
	}
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.AuthorizeNode == nil || !h.deps.AuthorizeNode(ctx, req.Msg.GetNodeId()) {
		return nil, connect.NewError(connect.CodePermissionDenied, internalinteractive.ErrUnauthorized)
	}
	h.deps.Admission.SetNode(req.Msg.GetNodeId(), true)
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	if actor == "" {
		actor = "owner"
	}
	out, err := h.deps.Admission.Open(actor, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Signal(ctx context.Context, req *connect.Request[interactivev1.SignalRequest]) (*connect.Response[interactivev1.SignalResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, mapError(internalinteractive.ErrInvalid)
	}
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	out, err := h.deps.Admission.Signal(req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	defer h.deps.Admission.ReleaseSignal(req.Msg.GetGrant().GetChannelId())
	if h.deps.SignalBroker != nil {
		out, err = h.deps.SignalBroker.Submit(ctx, req.Msg)
		if err != nil {
			return nil, mapError(err)
		}
	}
	return connect.NewResponse(out), nil
}
func (h *connectHandler) CloseChannel(ctx context.Context, req *connect.Request[interactivev1.CloseChannelRequest]) (*connect.Response[interactivev1.CloseChannelResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetGrant() == nil || req.Msg.GetGrant().GetChannelId() == "" {
		return nil, mapError(internalinteractive.ErrInvalid)
	}
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	grant, closed := h.deps.Admission.RevokeGrant(req.Msg.GetGrant().GetChannelId())
	if closed && h.deps.RevokePusher != nil {
		if err := h.deps.RevokePusher.PushInteractiveRevoke(ctx, grant, "operator_close"); err != nil {
			h.deps.Logger.Printf("interactive close revoke push failed: %v", err)
		}
	}
	return connect.NewResponse(&interactivev1.CloseChannelResponse{Closed: closed}), nil
}
func (h *connectHandler) RevokeChannel(ctx context.Context, req *connect.Request[interactivev1.RevokeChannelRequest]) (*connect.Response[interactivev1.RevokeChannelResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetChannelId() == "" {
		return nil, mapError(internalinteractive.ErrInvalid)
	}
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	grant, revoked := h.deps.Admission.RevokeGrant(req.Msg.GetChannelId())
	if revoked && h.deps.RevokePusher != nil {
		if err := h.deps.RevokePusher.PushInteractiveRevoke(ctx, grant, "operator_revoke"); err != nil {
			h.deps.Logger.Printf("interactive revoke push failed: %v", err)
		}
	}
	return connect.NewResponse(&interactivev1.RevokeChannelResponse{Revoked: revoked}), nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, internalinteractive.ErrUnauthorized), errors.Is(err, internalinteractive.ErrViewerWrite):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, internalinteractive.ErrStale):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, internalinteractive.ErrRevoked):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, internalinteractive.ErrSignalLimit):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, internalinteractive.ErrProtocol):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalinteractive.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalinteractive.ErrSignalTransportUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, internalinteractive.ErrSignalResponseTimeout):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, internalinteractive.ErrSignalResponseMismatch):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func Module(admission *internalinteractive.Admission, authorizeNode func(context.Context, string) bool, logger *log.Logger, brokers ...interface{}) module.Module {
	var broker *internalinteractive.SignalBroker
	var revoker internalinteractive.RevokePusher
	for _, dependency := range brokers {
		switch value := dependency.(type) {
		case *internalinteractive.SignalBroker:
			broker = value
		case internalinteractive.RevokePusher:
			revoker = value
		}
	}
	path, handler := interactiveconnect.NewInteractiveDesktopServiceHandler(NewConnectHandler(Deps{Admission: admission, AuthorizeNode: authorizeNode, SignalBroker: broker, RevokePusher: revoker, Logger: logger}))
	return module.Module{Name: "interactive-desktop", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
