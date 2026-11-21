package desktop

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"device-control/internal/desktopwebrtc"
	"device-control/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	desktopconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

// Module exposes the browser-operated desktop session contract. The manager
// owns all lease, controller, and epoch decisions; this adapter only maps
// Connect requests to that policy boundary.
func Module(manager *desktopwebrtc.Manager) module.Module {
	h := &handler{manager: manager}
	return module.Module{
		Name: "desktop-session",
		Mount: func(r *mux.Router) {
			path, service := desktopconnect.NewDesktopSessionServiceHandler(h, connect.WithReadMaxBytes(128*1024))
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: service})
		},
	}
}

type handler struct {
	desktopconnect.UnimplementedDesktopSessionServiceHandler
	manager *desktopwebrtc.Manager
}

func (h *handler) GetReadiness(ctx context.Context, req *connect.Request[desktopv1.GetReadinessRequest]) (*connect.Response[desktopv1.DesktopReadiness], error) {
	if req == nil || req.Msg == nil || req.Msg.GetSurface() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	readiness, err := h.manager.Readiness(ctx, req.Msg.GetSurface(), req.Msg.GetDisplayId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(readiness), nil
}

func (h *handler) OpenSession(ctx context.Context, req *connect.Request[desktopv1.OpenSessionRequest]) (*connect.Response[desktopv1.DesktopSession], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	session, err := h.manager.Open(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(session), nil
}

func (h *handler) AttachViewer(ctx context.Context, req *connect.Request[desktopv1.AttachViewerRequest]) (*connect.Response[desktopv1.DesktopSession], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	session, err := h.manager.AttachViewer(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(session), nil
}

func (h *handler) TakeControl(ctx context.Context, req *connect.Request[desktopv1.TakeControlRequest]) (*connect.Response[desktopv1.DesktopSession], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	session, err := h.manager.TakeControl(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(session), nil
}

func (h *handler) Signal(ctx context.Context, req *connect.Request[desktopv1.DesktopSignalRequest]) (*connect.Response[desktopv1.DesktopSignalResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	response, err := h.manager.Signal(ctx, req.Msg)
	if err != nil {
		return connect.NewResponse(response), mapError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *handler) Input(ctx context.Context, req *connect.Request[desktopv1.InputRequest]) (*connect.Response[desktopv1.InputReceipt], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	receipt, err := h.manager.Input(ctx, req.Msg)
	if err != nil {
		return connect.NewResponse(receipt), mapError(err)
	}
	return connect.NewResponse(receipt), nil
}

func (h *handler) Clipboard(ctx context.Context, req *connect.Request[desktopv1.ClipboardRequest]) (*connect.Response[desktopv1.ClipboardReceipt], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	receipt, err := h.manager.Clipboard(ctx, req.Msg)
	if err != nil {
		return connect.NewResponse(receipt), mapError(err)
	}
	return connect.NewResponse(receipt), nil
}

func (h *handler) CloseSession(ctx context.Context, req *connect.Request[desktopv1.CloseSessionRequest]) (*connect.Response[desktopv1.CloseSessionResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, desktopwebrtc.ErrInvalidCall)
	}
	closed, err := h.manager.Close(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(closed), nil
}

func mapError(err error) error {
	code := connect.CodeFailedPrecondition
	switch {
	case errors.Is(err, desktopwebrtc.ErrInvalidCall):
		code = connect.CodeInvalidArgument
	case errors.Is(err, desktopwebrtc.ErrNoSession):
		code = connect.CodeNotFound
	case errors.Is(err, desktopwebrtc.ErrNotControl), errors.Is(err, desktopwebrtc.ErrClipboard):
		code = connect.CodePermissionDenied
	case errors.Is(err, desktopwebrtc.ErrRevoked), errors.Is(err, desktopwebrtc.ErrStaleEpoch):
		code = connect.CodeFailedPrecondition
	}
	return connect.NewError(code, err)
}
