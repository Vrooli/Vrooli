package companion

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
	companionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion/companionv1connect"
	"vrooli-bridge/internal/auth"
	internal "vrooli-bridge/internal/companion"
	"vrooli-bridge/internal/module"
)

func Module(manager *internal.Manager) module.Module {
	path, handler := companionconnect.NewCompanionServiceHandler(&connectHandler{manager: manager})
	return module.Module{Name: "companion", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

type connectHandler struct {
	companionconnect.UnimplementedCompanionServiceHandler
	manager *internal.Manager
}

func (h *connectHandler) Install(ctx context.Context, req *connect.Request[companionv1.InstallRequest]) (*connect.Response[companionv1.CompanionOperation], error) {
	if err := owner(ctx); err != nil {
		return nil, err
	}
	var out *companionv1.CompanionOperation
	var err error
	if req.Msg.GetArtifactSourceRef() != "" || req.Msg.GetArtifactName() != "" || req.Msg.GetArtifactDestinationPath() != "" {
		out, err = h.manager.InstallWithArtifact(ctx, req.Msg.GetNodeId(), req.Msg.GetVersion(), req.Msg.GetDisplayId(), req.Msg.GetArtifactSourceRef(), req.Msg.GetArtifactName(), req.Msg.GetArtifactDestinationPath())
	} else {
		out, err = h.manager.Install(req.Msg.GetNodeId(), req.Msg.GetVersion(), req.Msg.GetDisplayId())
	}
	return result(out, err)
}
func (h *connectHandler) Inspect(ctx context.Context, req *connect.Request[companionv1.InspectRequest]) (*connect.Response[companionv1.CompanionOperation], error) {
	if err := owner(ctx); err != nil {
		return nil, err
	}
	out, err := h.manager.Inspect(req.Msg.GetNodeId())
	return result(out, err)
}
func (h *connectHandler) Upgrade(ctx context.Context, req *connect.Request[companionv1.UpgradeRequest]) (*connect.Response[companionv1.CompanionOperation], error) {
	if err := owner(ctx); err != nil {
		return nil, err
	}
	var out *companionv1.CompanionOperation
	var err error
	if req.Msg.GetArtifactSourceRef() != "" || req.Msg.GetArtifactName() != "" || req.Msg.GetArtifactDestinationPath() != "" {
		out, err = h.manager.UpgradeWithArtifact(ctx, req.Msg.GetNodeId(), req.Msg.GetVersion(), req.Msg.GetArtifactSourceRef(), req.Msg.GetArtifactName(), req.Msg.GetArtifactDestinationPath())
	} else {
		out, err = h.manager.Upgrade(req.Msg.GetNodeId(), req.Msg.GetVersion())
	}
	return result(out, err)
}
func (h *connectHandler) Revoke(ctx context.Context, req *connect.Request[companionv1.RevokeRequest]) (*connect.Response[companionv1.CompanionOperation], error) {
	if err := owner(ctx); err != nil {
		return nil, err
	}
	out, err := h.manager.Revoke(req.Msg.GetNodeId(), req.Msg.GetReason())
	return result(out, err)
}
func (h *connectHandler) Remove(ctx context.Context, req *connect.Request[companionv1.RemoveRequest]) (*connect.Response[companionv1.CompanionOperation], error) {
	if err := owner(ctx); err != nil {
		return nil, err
	}
	out, err := h.manager.Remove(req.Msg.GetNodeId())
	return result(out, err)
}
func owner(ctx context.Context) error {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return auth.ToConnectError(err)
	}
	return nil
}
func result(out *companionv1.CompanionOperation, err error) (*connect.Response[companionv1.CompanionOperation], error) {
	if err != nil {
		if errors.Is(err, internal.ErrTargetUnavailable) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}
