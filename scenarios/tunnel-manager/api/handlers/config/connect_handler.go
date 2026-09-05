package config

import (
	"context"
	"log"

	"tunnel-manager/internal/authz"
	internalconfig "tunnel-manager/internal/config"

	"connectrpc.com/connect"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
)

// Deps wires the seams the Connect config handler needs.
type Deps struct {
	Service    internalconfig.Service
	Logger     *log.Logger
	Authorizer authz.Authorizer
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Authorizer == nil {
		d.Authorizer = authz.AllowLocalOperator()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetConfig(ctx context.Context, _ *connect.Request[configv1.GetConfigRequest]) (*connect.Response[configv1.GetConfigResponse], error) {
	state, err := h.deps.Service.GetConfigState(ctx)
	if err != nil {
		h.deps.Logger.Printf("config.GetConfig: %v", err)
		return nil, internalconfig.ToConnectError(err)
	}
	return connect.NewResponse(&configv1.GetConfigResponse{
		Config:    domainConfigToProto(state.Config),
		Readiness: readinessToProto(state.Readiness),
	}), nil
}

func (h *connectHandler) GetCredentialStatus(ctx context.Context, _ *connect.Request[configv1.GetCredentialStatusRequest]) (*connect.Response[configv1.GetCredentialStatusResponse], error) {
	status, err := h.deps.Service.GetCredentialStatus(ctx)
	if err != nil {
		h.deps.Logger.Printf("config.GetCredentialStatus: %v", err)
		return nil, internalconfig.ToConnectError(err)
	}
	return connect.NewResponse(&configv1.GetCredentialStatusResponse{
		Status: credentialStatusToProto(status),
	}), nil
}

func (h *connectHandler) VerifyCredentials(ctx context.Context, _ *connect.Request[configv1.VerifyCredentialsRequest]) (*connect.Response[configv1.VerifyCredentialsResponse], error) {
	verification, err := h.deps.Service.VerifyCredentials(ctx)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.VerifyCredentials: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(verificationToProto(verification)), nil
}

func (h *connectHandler) BootstrapCloudflare(ctx context.Context, req *connect.Request[configv1.BootstrapCloudflareRequest]) (*connect.Response[configv1.BootstrapCloudflareResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigCredentials, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	result, err := h.deps.Service.BootstrapCloudflare(ctx, internalconfig.BootstrapRequest{
		APIToken: req.Msg.ApiToken, AccountID: req.Msg.AccountId, TunnelID: req.Msg.TunnelId,
		TunnelName: req.Msg.TunnelName, DryRun: req.Msg.DryRun,
	})
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.BootstrapCloudflare: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.BootstrapCloudflareResponse{
		AccountId: result.AccountID, TunnelId: result.TunnelID, Adopted: result.Adopted,
		Created: result.Created, Written: result.Written,
	}), nil
}

func (h *connectHandler) SetCloudflareCredentials(ctx context.Context, req *connect.Request[configv1.SetCloudflareCredentialsRequest]) (*connect.Response[configv1.SetCloudflareCredentialsResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigCredentials, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	status, err := h.deps.Service.SetCloudflareCredentials(ctx, internalconfig.CredentialUpdate{
		AccountID: req.Msg.AccountId,
		TunnelID:  req.Msg.TunnelId,
		APIToken:  req.Msg.ApiToken,
	})
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.SetCloudflareCredentials: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.SetCloudflareCredentialsResponse{
		Status: credentialStatusToProto(status),
	}), nil
}

func (h *connectHandler) ClearCloudflareCredentials(ctx context.Context, req *connect.Request[configv1.ClearCloudflareCredentialsRequest]) (*connect.Response[configv1.ClearCloudflareCredentialsResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigCredentials, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	status, err := h.deps.Service.ClearCloudflareCredentials(ctx, req.Msg.Fields)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.ClearCloudflareCredentials: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.ClearCloudflareCredentialsResponse{
		Status: credentialStatusToProto(status),
	}), nil
}

func (h *connectHandler) Sync(ctx context.Context, req *connect.Request[configv1.SyncRequest]) (*connect.Response[configv1.SyncResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigSync, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	result, err := h.deps.Service.Sync(ctx, req.Msg.DryRun, req.Msg.Prune)
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
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigSwitchMode, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
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

func (h *connectHandler) GetDrift(ctx context.Context, _ *connect.Request[configv1.GetDriftRequest]) (*connect.Response[configv1.GetDriftResponse], error) {
	rep, err := h.deps.Service.GetDrift(ctx)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.GetDrift: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(driftReportToProto(rep)), nil
}

func (h *connectHandler) AdoptIngress(ctx context.Context, req *connect.Request[configv1.AdoptIngressRequest]) (*connect.Response[configv1.AdoptIngressResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigSync, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	entry, err := h.deps.Service.AdoptIngress(ctx, req.Msg.Hostname, req.Msg.Scenario, req.Msg.Target)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.AdoptIngress: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.AdoptIngressResponse{Entry: ingressEntryToProto(entry)}), nil
}

func (h *connectHandler) IgnoreIngress(ctx context.Context, req *connect.Request[configv1.IgnoreIngressRequest]) (*connect.Response[configv1.IgnoreIngressResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigSync, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	entry, err := h.deps.Service.IgnoreIngress(ctx, req.Msg.Hostname, req.Msg.Note)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.IgnoreIngress: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.IgnoreIngressResponse{Entry: ingressEntryToProto(entry)}), nil
}

func (h *connectHandler) PruneIngress(ctx context.Context, req *connect.Request[configv1.PruneIngressRequest]) (*connect.Response[configv1.PruneIngressResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigSync, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	pruned, err := h.deps.Service.PruneIngress(ctx, req.Msg.Hostname)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.PruneIngress: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.PruneIngressResponse{Pruned: pruned}), nil
}

func (h *connectHandler) SetPublicExposure(ctx context.Context, req *connect.Request[configv1.SetPublicExposureRequest]) (*connect.Response[configv1.SetPublicExposureResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationConfigSync, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	cfg, err := h.deps.Service.SetPublicExposure(ctx, req.Msg.Enabled)
	if err != nil {
		connectErr := internalconfig.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("config.SetPublicExposure: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&configv1.SetPublicExposureResponse{Config: domainConfigToProto(cfg)}), nil
}

func (h *connectHandler) GetAccessStatus(ctx context.Context, _ *connect.Request[configv1.GetAccessStatusRequest]) (*connect.Response[configv1.GetAccessStatusResponse], error) {
	status, err := h.deps.Service.GetAccessStatus(ctx)
	if err != nil {
		h.deps.Logger.Printf("config.GetAccessStatus: %v", err)
		return nil, internalconfig.ToConnectError(err)
	}
	return connect.NewResponse(&configv1.GetAccessStatusResponse{Status: accessStatusToProto(status)}), nil
}
