package config_test

import (
	"context"
	"testing"

	"tunnel-manager/handlers/config"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"

	internalconfig "tunnel-manager/internal/config"
)

// fakeService implements internalconfig.Service for handler tests.
type fakeService struct {
	getOut  internalconfig.TunnelConfig
	getErr  error
	syncOut internalconfig.SyncResult
	syncErr error
	syncDry bool
	swPrev  internalconfig.Mode
	swCur   internalconfig.Mode
	swErr   error
	swTgt   internalconfig.Mode
}

func (f *fakeService) GetConfig(context.Context) (internalconfig.TunnelConfig, error) {
	return f.getOut, f.getErr
}

func (f *fakeService) Sync(_ context.Context, dryRun bool) (internalconfig.SyncResult, error) {
	f.syncDry = dryRun
	return f.syncOut, f.syncErr
}

func (f *fakeService) SwitchMode(_ context.Context, target internalconfig.Mode) (internalconfig.Mode, internalconfig.Mode, error) {
	f.swTgt = target
	return f.swPrev, f.swCur, f.swErr
}

func newClient(t *testing.T, svc internalconfig.Service) configconnect.ConfigServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := configconnect.NewConfigServiceHandler(config.NewConnectHandler(config.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return configconnect.NewConfigServiceClient(server.Client(), server.URL)
}

func TestHandlerGetConfigMapsMode(t *testing.T) {
	client := newClient(t, &fakeService{getOut: internalconfig.TunnelConfig{
		Mode: internalconfig.ModeRemote, TunnelID: "tid", AccountID: "acct", PromEndpoint: "127.0.0.1:20241",
	}})

	resp, err := client.GetConfig(context.Background(), connect.NewRequest(&configv1.GetConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, configv1.Mode_MODE_REMOTE, resp.Msg.Config.Mode)
	require.Equal(t, "tid", resp.Msg.Config.TunnelId)
	require.Equal(t, "127.0.0.1:20241", resp.Msg.Config.PromEndpoint)
}

func TestHandlerSyncMapsResponse(t *testing.T) {
	fake := &fakeService{syncOut: internalconfig.SyncResult{
		Mode:    internalconfig.ModeLocal,
		Added:   []string{"agent-manager.itsagitime.com"},
		Removed: []string{"legacy.itsagitime.com"},
	}}
	client := newClient(t, fake)

	resp, err := client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{DryRun: true}))
	require.NoError(t, err)
	require.True(t, fake.syncDry, "dry_run propagated to service")
	require.Equal(t, configv1.Mode_MODE_LOCAL, resp.Msg.Mode)
	require.Equal(t, []string{"agent-manager.itsagitime.com"}, resp.Msg.Added)
	require.Equal(t, []string{"legacy.itsagitime.com"}, resp.Msg.Removed)
	require.False(t, resp.Msg.NoChanges)
}

func TestHandlerSyncRemoteUnavailableMapsFailedPrecondition(t *testing.T) {
	client := newClient(t, &fakeService{syncErr: internalconfig.ErrRemoteUnavailable{}})
	_, err := client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestHandlerSwitchModeMapsModes(t *testing.T) {
	fake := &fakeService{swPrev: internalconfig.ModeRemote, swCur: internalconfig.ModeLocal}
	client := newClient(t, fake)

	resp, err := client.SwitchMode(context.Background(), connect.NewRequest(&configv1.SwitchModeRequest{
		TargetMode: configv1.Mode_MODE_LOCAL,
	}))
	require.NoError(t, err)
	require.Equal(t, internalconfig.ModeLocal, fake.swTgt, "target mode propagated to service")
	require.Equal(t, configv1.Mode_MODE_REMOTE, resp.Msg.PreviousMode)
	require.Equal(t, configv1.Mode_MODE_LOCAL, resp.Msg.CurrentMode)
}

func TestHandlerSwitchModeInvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{swErr: internalconfig.ErrInvalidConfig{Field: "target_mode", Reason: "unknown"}})
	_, err := client.SwitchMode(context.Background(), connect.NewRequest(&configv1.SwitchModeRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
