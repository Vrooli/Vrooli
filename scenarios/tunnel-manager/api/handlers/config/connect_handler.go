package config

import (
	"context"
	"log"

	internalconfig "tunnel-manager/internal/config"

	"connectrpc.com/connect"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
)

// Deps wires the seams the Connect config handler needs.
type Deps struct {
	Service internalconfig.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetConfig(ctx context.Context, _ *connect.Request[configv1.GetConfigRequest]) (*connect.Response[configv1.GetConfigResponse], error) {
	cfg, err := h.deps.Service.GetConfig(ctx)
	if err != nil {
		h.deps.Logger.Printf("config.GetConfig: %v", err)
		return nil, internalconfig.ToConnectError(err)
	}
	return connect.NewResponse(&configv1.GetConfigResponse{Config: domainConfigToProto(cfg)}), nil
}

func (h *connectHandler) Sync(ctx context.Context, req *connect.Request[configv1.SyncRequest]) (*connect.Response[configv1.SyncResponse], error) {
	result, err := h.deps.Service.Sync(ctx, req.Msg.DryRun)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.Sync: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(syncResultToProto(result)), nil
}

func (h *connectHandler) SwitchMode(ctx context.Context, req *connect.Request[configv1.SwitchModeRequest]) (*connect.Response[configv1.SwitchModeResponse], error) {
	prev, cur, err := h.deps.Service.SwitchMode(ctx, modeFromProto(req.Msg.TargetMode))
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.SwitchMode: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.SwitchModeResponse{
		PreviousMode: modeToProto(prev),
		CurrentMode:  modeToProto(cur),
	}), nil
}
