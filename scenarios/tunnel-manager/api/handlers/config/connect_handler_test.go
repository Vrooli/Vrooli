package config_test

import (
	"context"
	"testing"

	"tunnel-manager/handlers/config"
	"tunnel-manager/internal/authz"

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
	getOut       internalconfig.TunnelConfig
	ready        internalconfig.ConfigReadiness
	getErr       error
	credOut      internalconfig.CredentialStatus
	credErr      error
	verifyOut    internalconfig.CredentialVerification
	verifyErr    error
	bootstrapOut internalconfig.BootstrapResult
	bootstrapErr error
	bootstrapIn  internalconfig.BootstrapRequest
	setIn        internalconfig.CredentialUpdate
	setCalls     int
	clearIn      []string
	clearCalls   int
	syncOut      internalconfig.SyncResult
	syncErr      error
	syncDry      bool
	syncPrune    bool
	syncCalls    int
	swPrev       internalconfig.Mode
	swCur        internalconfig.Mode
	swErr        error
	swTgt        internalconfig.Mode
	swCalls      int
	driftOut     internalconfig.DriftReport
	driftErr     error
	driftCalls   int

	adoptOut      internalconfig.IngressEntry
	adoptErr      error
	adoptCalls    int
	adoptHost     string
	adoptScenario string
	adoptTarget   string

	ignoreOut   internalconfig.IngressEntry
	ignoreErr   error
	ignoreCalls int
	ignoreHost  string
	ignoreNote  string

	pruneOut   bool
	pruneErr   error
	pruneCalls int
	pruneHost  string

	setExposureOut   internalconfig.TunnelConfig
	setExposureErr   error
	setExposureCalls int
	setExposureArg   bool

	accessStatusOut   internalconfig.AccessStatus
	accessStatusErr   error
	accessStatusCalls int
}

func (f *fakeService) GetConfig(context.Context) (internalconfig.TunnelConfig, error) {
	return f.getOut, f.getErr
}

func (f *fakeService) GetConfigState(context.Context) (internalconfig.ConfigState, error) {
	return internalconfig.ConfigState{Config: f.getOut, Readiness: f.ready}, f.getErr
}

func (f *fakeService) GetCredentialStatus(context.Context) (internalconfig.CredentialStatus, error) {
	return f.credOut, f.credErr
}

func (f *fakeService) VerifyCredentials(context.Context) (internalconfig.CredentialVerification, error) {
	return f.verifyOut, f.verifyErr
}

func (f *fakeService) BootstrapCloudflare(_ context.Context, request internalconfig.BootstrapRequest) (internalconfig.BootstrapResult, error) {
	f.bootstrapIn = request
	return f.bootstrapOut, f.bootstrapErr
}

func (f *fakeService) SetCloudflareCredentials(_ context.Context, values internalconfig.CredentialUpdate) (internalconfig.CredentialStatus, error) {
	f.setCalls++
	f.setIn = values
	return f.credOut, f.credErr
}

func (f *fakeService) ClearCloudflareCredentials(_ context.Context, keys []string) (internalconfig.CredentialStatus, error) {
	f.clearCalls++
	f.clearIn = keys
	return f.credOut, f.credErr
}

func (f *fakeService) Sync(_ context.Context, dryRun, prune bool) (internalconfig.SyncResult, error) {
	f.syncCalls++
	f.syncDry = dryRun
	f.syncPrune = prune
	return f.syncOut, f.syncErr
}

func (f *fakeService) SwitchMode(_ context.Context, target internalconfig.Mode) (internalconfig.Mode, internalconfig.Mode, error) {
	f.swCalls++
	f.swTgt = target
	return f.swPrev, f.swCur, f.swErr
}

func (f *fakeService) GetDrift(context.Context) (internalconfig.DriftReport, error) {
	f.driftCalls++
	return f.driftOut, f.driftErr
}

func (f *fakeService) AdoptIngress(_ context.Context, hostname, scenario, target string) (internalconfig.IngressEntry, error) {
	f.adoptCalls++
	f.adoptHost, f.adoptScenario, f.adoptTarget = hostname, scenario, target
	return f.adoptOut, f.adoptErr
}

func (f *fakeService) IgnoreIngress(_ context.Context, hostname, note string) (internalconfig.IngressEntry, error) {
	f.ignoreCalls++
	f.ignoreHost, f.ignoreNote = hostname, note
	return f.ignoreOut, f.ignoreErr
}

func (f *fakeService) PruneIngress(_ context.Context, hostname string) (bool, error) {
	f.pruneCalls++
	f.pruneHost = hostname
	return f.pruneOut, f.pruneErr
}

func (f *fakeService) SetPublicExposure(_ context.Context, enabled bool) (internalconfig.TunnelConfig, error) {
	f.setExposureCalls++
	f.setExposureArg = enabled
	return f.setExposureOut, f.setExposureErr
}

func (f *fakeService) GetAccessStatus(context.Context) (internalconfig.AccessStatus, error) {
	f.accessStatusCalls++
	return f.accessStatusOut, f.accessStatusErr
}

func newClient(t *testing.T, svc internalconfig.Service) configconnect.ConfigServiceClient {
	t.Helper()
	return newClientWithAuthorizer(t, svc, nil)
}

func newClientWithAuthorizer(t *testing.T, svc internalconfig.Service, authorizer authz.Authorizer) configconnect.ConfigServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := configconnect.NewConfigServiceHandler(config.NewConnectHandler(config.Deps{
		Service:    svc,
		Logger:     logger,
		Authorizer: authorizer,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return configconnect.NewConfigServiceClient(server.Client(), server.URL)
}

func TestHandlerGetConfigMapsMode(t *testing.T) {
	client := newClient(t, &fakeService{getOut: internalconfig.TunnelConfig{
		Mode: internalconfig.ModeRemote, TunnelID: "tid", AccountID: "acct", PromEndpoint: "127.0.0.1:20241",
	}, ready: internalconfig.ConfigReadiness{
		DesiredMode:      internalconfig.ModeRemote,
		RemoteAvailable:  true,
		CredentialSource: "credential-authority",
		CredentialRef:    "vrooli/tunnel-manager:cloudflare-api-token",
		CredentialStatus: internalconfig.CredentialStatus{Fields: []internalconfig.CredentialFieldStatus{{
			Name: "CLOUDFLARE_API_TOKEN", Present: true, Source: "credential-authority", Ref: "vrooli/tunnel-manager:cloudflare-api-token",
		}}},
		LocalConfigPath: "/tmp/config.yml",
		SyncReady:       true,
		ModeReason:      "ready",
	}})

	resp, err := client.GetConfig(context.Background(), connect.NewRequest(&configv1.GetConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, configv1.Mode_MODE_REMOTE, resp.Msg.Config.Mode)
	require.Equal(t, "tid", resp.Msg.Config.TunnelId)
	require.Equal(t, "127.0.0.1:20241", resp.Msg.Config.PromEndpoint)
	require.Equal(t, configv1.Mode_MODE_REMOTE, resp.Msg.Readiness.DesiredMode)
	require.True(t, resp.Msg.Readiness.RemoteAvailable)
	require.Equal(t, "vrooli/tunnel-manager:cloudflare-api-token", resp.Msg.Readiness.CredentialRef)
	require.Len(t, resp.Msg.Readiness.CredentialFields, 1)
	require.Equal(t, "CLOUDFLARE_API_TOKEN", resp.Msg.Readiness.CredentialFields[0].Name)
}

func TestHandlerGetCredentialStatusRedactsTokenValue(t *testing.T) {
	fake := &fakeService{credOut: internalconfig.CredentialStatus{
		Ready:  true,
		Source: "credential-authority",
		Ref:    "vrooli/tunnel-manager:cloudflare-api-token",
		Fields: []internalconfig.CredentialFieldStatus{{
			Name: "CLOUDFLARE_API_TOKEN", Present: true, Source: "credential-authority", Ref: "vrooli/tunnel-manager:cloudflare-api-token", Writable: true,
		}},
	}}
	client := newClient(t, fake)

	resp, err := client.GetCredentialStatus(context.Background(), connect.NewRequest(&configv1.GetCredentialStatusRequest{}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Status.Ready)
	require.Equal(t, "vrooli/tunnel-manager:cloudflare-api-token", resp.Msg.Status.Ref)
	require.NotContains(t, resp.Msg.String(), "secret-token")
}

func TestHandlerSetCloudflareCredentialsRequiresOperatorTokenWhenEnforced(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})

	_, err := client.SetCloudflareCredentials(context.Background(), connect.NewRequest(&configv1.SetCloudflareCredentialsRequest{
		AccountId: "acct", TunnelId: "tun", ApiToken: "token",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, fake.setCalls, "denied credential write must not reach service")
}

func TestHandlerSetCloudflareCredentialsPassesWriteOnlyValues(t *testing.T) {
	fake := &fakeService{credOut: internalconfig.CredentialStatus{Ready: true, Source: "credential-authority"}}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})
	req := connect.NewRequest(&configv1.SetCloudflareCredentialsRequest{
		AccountId: "acct", TunnelId: "tun", ApiToken: "secret-token",
	})
	req.Header().Set("Authorization", "Bearer secret")

	resp, err := client.SetCloudflareCredentials(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, internalconfig.CredentialUpdate{AccountID: "acct", TunnelID: "tun", APIToken: "secret-token"}, fake.setIn)
	require.True(t, resp.Msg.Status.Ready)
	require.NotContains(t, resp.Msg.String(), "secret-token")
}

func TestHandlerClearCloudflareCredentialsRequiresOperatorTokenWhenEnforced(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})

	_, err := client.ClearCloudflareCredentials(context.Background(), connect.NewRequest(&configv1.ClearCloudflareCredentialsRequest{
		Fields: []string{"api_token"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, fake.clearCalls, "denied credential clear must not reach service")
}

func TestHandlerClearCloudflareCredentialsPassesFields(t *testing.T) {
	fake := &fakeService{credOut: internalconfig.CredentialStatus{Source: "missing"}}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})
	req := connect.NewRequest(&configv1.ClearCloudflareCredentialsRequest{Fields: []string{"api_token"}})
	req.Header().Set("X-Vrooli-Operator-Token", "secret")

	_, err := client.ClearCloudflareCredentials(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, []string{"api_token"}, fake.clearIn)
}

func TestHandlerSyncMapsResponse(t *testing.T) {
	fake := &fakeService{syncOut: internalconfig.SyncResult{
		Mode:    internalconfig.ModeLocal,
		Added:   []string{"agent-manager.itsagitime.com"},
		Removed: []string{"legacy.itsagitime.com"},
		Message: "dry-run complete",
	}}
	client := newClient(t, fake)

	resp, err := client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{DryRun: true}))
	require.NoError(t, err)
	require.True(t, fake.syncDry, "dry_run propagated to service")
	require.Equal(t, configv1.Mode_MODE_LOCAL, resp.Msg.Mode)
	require.Equal(t, []string{"agent-manager.itsagitime.com"}, resp.Msg.Added)
	require.Equal(t, []string{"legacy.itsagitime.com"}, resp.Msg.Removed)
	require.False(t, resp.Msg.NoChanges)
	require.Equal(t, "dry-run complete", resp.Msg.Message)
}

func TestHandlerSyncRemoteUnavailableMapsFailedPrecondition(t *testing.T) {
	client := newClient(t, &fakeService{syncErr: internalconfig.ErrRemoteUnavailable{}})
	_, err := client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestHandlerSyncRequiresOperatorTokenWhenEnforced(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})

	_, err := client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, fake.syncCalls, "denied sync must not reach the service")
}

func TestHandlerSyncAcceptsOperatorBearer(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})
	req := connect.NewRequest(&configv1.SyncRequest{DryRun: true})
	req.Header().Set("Authorization", "Bearer secret")

	_, err := client.Sync(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 1, fake.syncCalls)
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

func TestHandlerGetDriftMapsReport(t *testing.T) {
	fake := &fakeService{driftOut: internalconfig.DriftReport{
		Mode: internalconfig.ModeRemote,
		Entries: []internalconfig.IngressEntry{
			{Hostname: "a.itsagitime.com", State: internalconfig.StateManaged, Source: internalconfig.SourceScenario, Scenario: "agent-manager", ServiceTarget: "http://localhost:21100"},
			{Hostname: "b.example.com", State: internalconfig.StateUnmanaged},
		},
		Counts: map[internalconfig.OwnershipState]int{
			internalconfig.StateManaged:   1,
			internalconfig.StateUnmanaged: 1,
		},
	}}
	client := newClient(t, fake)

	resp, err := client.GetDrift(context.Background(), connect.NewRequest(&configv1.GetDriftRequest{}))
	require.NoError(t, err)
	require.Equal(t, 1, fake.driftCalls)
	require.Equal(t, configv1.Mode_MODE_REMOTE, resp.Msg.Mode)
	require.Len(t, resp.Msg.Entries, 2)
	require.Equal(t, configv1.OwnershipState_OWNERSHIP_STATE_MANAGED, resp.Msg.Entries[0].State)
	require.Equal(t, configv1.IngressSource_INGRESS_SOURCE_SCENARIO, resp.Msg.Entries[0].Source)
	require.Equal(t, configv1.OwnershipState_OWNERSHIP_STATE_UNMANAGED, resp.Msg.Entries[1].State)
	require.Equal(t, int32(1), resp.Msg.Counts.Managed)
	require.Equal(t, int32(1), resp.Msg.Counts.Unmanaged)
}

func TestHandlerAdoptIngressPassesArgs(t *testing.T) {
	fake := &fakeService{adoptOut: internalconfig.IngressEntry{Hostname: "api.itsagitime.com", State: internalconfig.StateExternalOK, Source: internalconfig.SourceExternal}}
	client := newClient(t, fake)

	resp, err := client.AdoptIngress(context.Background(), connect.NewRequest(&configv1.AdoptIngressRequest{
		Hostname: "api.itsagitime.com", Target: "http://127.0.0.1:9000",
	}))
	require.NoError(t, err)
	require.Equal(t, "api.itsagitime.com", fake.adoptHost)
	require.Equal(t, "http://127.0.0.1:9000", fake.adoptTarget)
	require.Equal(t, configv1.OwnershipState_OWNERSHIP_STATE_EXTERNAL_OK, resp.Msg.Entry.State)
}

func TestHandlerAdoptIngressRequiresOperatorTokenWhenEnforced(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})
	_, err := client.AdoptIngress(context.Background(), connect.NewRequest(&configv1.AdoptIngressRequest{Hostname: "x.itsagitime.com"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, fake.adoptCalls, "denied adopt must not reach the service")
}

func TestHandlerIgnoreIngressPassesNote(t *testing.T) {
	fake := &fakeService{ignoreOut: internalconfig.IngressEntry{Hostname: "legacy.itsagitime.com", State: internalconfig.StateIgnored}}
	client := newClient(t, fake)
	resp, err := client.IgnoreIngress(context.Background(), connect.NewRequest(&configv1.IgnoreIngressRequest{Hostname: "legacy.itsagitime.com", Note: "dashboard"}))
	require.NoError(t, err)
	require.Equal(t, "legacy.itsagitime.com", fake.ignoreHost)
	require.Equal(t, "dashboard", fake.ignoreNote)
	require.Equal(t, configv1.OwnershipState_OWNERSHIP_STATE_IGNORED, resp.Msg.Entry.State)
}

func TestHandlerPruneIngressReturnsPruned(t *testing.T) {
	fake := &fakeService{pruneOut: true}
	client := newClient(t, fake)
	resp, err := client.PruneIngress(context.Background(), connect.NewRequest(&configv1.PruneIngressRequest{Hostname: "legacy.itsagitime.com"}))
	require.NoError(t, err)
	require.Equal(t, "legacy.itsagitime.com", fake.pruneHost)
	require.True(t, resp.Msg.Pruned)
}
