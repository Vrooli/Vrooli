package config_test

import (
	"context"
	"os"
	"testing"
	"time"

	"tunnel-manager/internal/config"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// fakeRepo is an in-memory ConfigRepository for service tests.
type fakeRepo struct {
	cfg      config.TunnelConfig
	upserted config.TunnelConfig
	upsertN  int
}

func (f *fakeRepo) Get(context.Context) (config.TunnelConfig, error) { return f.cfg, nil }

func (f *fakeRepo) Upsert(_ context.Context, c config.TunnelConfig) (config.TunnelConfig, error) {
	f.upserted = c
	f.upsertN++
	f.cfg = c
	return c, nil
}

// fakeRoutes is an in-memory RoutesReader.
type fakeRoutes struct {
	routes []internalroutes.Route
}

func (f *fakeRoutes) List(context.Context, internalroutes.Tier) ([]internalroutes.Route, error) {
	return f.routes, nil
}

// fakeIngress is an in-memory IngressClient recording the last push.
type fakeIngress struct {
	live   []config.IngressRule
	pushed []config.IngressRule
	pushN  int
}

func (f *fakeIngress) ReadIngress(context.Context) ([]config.IngressRule, error) {
	return f.live, nil
}

func (f *fakeIngress) PushIngress(_ context.Context, r []config.IngressRule) error {
	f.pushed = r
	f.pushN++
	return nil
}

type fakeCredentialStore struct {
	cfg        config.CFConfig
	status     config.CredentialStatus
	resolveN   int
	statusN    int
	saveIn     config.CredentialUpdate
	saveN      int
	deleteIn   []string
	deleteN    int
}

func (f *fakeCredentialStore) Status(context.Context) (config.CredentialStatus, error) {
	f.statusN++
	return f.status, nil
}

func (f *fakeCredentialStore) Resolve(context.Context) (config.CFConfig, error) {
	f.resolveN++
	return f.cfg, nil
}

func (f *fakeCredentialStore) Save(_ context.Context, values config.CredentialUpdate) (config.CredentialStatus, error) {
	f.saveN++
	f.saveIn = values
	return f.status, nil
}

func (f *fakeCredentialStore) Delete(_ context.Context, keys []string) (config.CredentialStatus, error) {
	f.deleteN++
	f.deleteIn = keys
	return f.status, nil
}

func route(sub string, port int, enabled bool) internalroutes.Route {
	return internalroutes.Route{
		Subdomain: sub, Scenario: sub, Domain: internalroutes.DefaultDomain,
		LocalPort: port, Tier: internalroutes.TierLeased, Enabled: enabled,
	}
}

func newSvc(repo config.ConfigRepository, routes config.RoutesReader, ingress config.IngressClient, runner *mocks.FakeCmdRunner) config.Service {
	return config.NewService(config.Deps{
		Repo:    repo,
		Routes:  routes,
		Ingress: ingress,
		Runner:  runner.Run,
		Clock:   mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})
}

// --- Sync diff -------------------------------------------------------------

func TestSync_ComputesAddedAndRemovedFromManifestDomains(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{
		route("agent-manager", 21100, true),
		route("web-console", 3000, true),
		route("disabled-one", 9999, false),
	}}
	// Live ingress has an old hostname (web-console) plus a stale one to be
	// removed (legacy), and lacks agent-manager (to be added).
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "web-console.itsagitime.com", Service: "http://localhost:3000"},
		{Hostname: "legacy.itsagitime.com", Service: "http://localhost:1"},
		{Service: "http_status:404"},
	}}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, []string{"agent-manager.itsagitime.com"}, res.Added)
	require.Equal(t, []string{"legacy.itsagitime.com"}, res.Removed)
	require.False(t, res.NoChanges)
	require.Equal(t, config.ModeRemote, res.Mode)

	// BUG-FIX ASSERTION: hostnames derive from route.Subdomain.Domain
	// (itsagitime.com), never a hardcoded ".vrooli.com" apex.
	require.NotEmpty(t, ingress.pushed)
	for _, r := range ingress.pushed {
		require.NotContains(t, r.Hostname, "vrooli.com", "must not leak hardcoded apex")
	}
	require.Equal(t, 1, ingress.pushN)
	// The disabled route must not appear.
	for _, r := range ingress.pushed {
		require.NotContains(t, r.Hostname, "disabled-one")
	}
}

func TestSync_DryRunAppliesNothing(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, []string{"agent-manager.itsagitime.com"}, res.Added)
	require.False(t, res.NoChanges)
	require.Equal(t, 0, ingress.pushN, "dry-run pushes nothing")
}

func TestSync_NoChangesPath(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "agent-manager.itsagitime.com", Service: "http://localhost:21100"},
		{Service: "http_status:404"},
	}}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), false)
	require.NoError(t, err)
	require.True(t, res.NoChanges)
	require.Empty(t, res.Added)
	require.Empty(t, res.Removed)
	require.Equal(t, 0, ingress.pushN, "no drift applies nothing")
}

// --- Remote unavailable ----------------------------------------------------

func TestSync_RemoteWithoutCredsUnavailable(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	svc := newSvc(repo, routes, nil /* no IngressClient */, &mocks.FakeCmdRunner{})

	_, err := svc.Sync(context.Background(), false)
	var unavailable config.ErrRemoteUnavailable
	require.ErrorAs(t, err, &unavailable)
}

func TestSync_DryRunRemoteWithoutCredsReturnsSetupReport(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	svc := newSvc(repo, routes, nil /* no IngressClient */, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), true)
	require.NoError(t, err)
	require.True(t, res.SetupRequired)
	require.Equal(t, config.ModeRemote, res.Mode)
	require.ElementsMatch(t, []string{"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_TUNNEL_ID", "CLOUDFLARE_API_TOKEN"}, res.MissingFields)
	require.Contains(t, res.Message, "Remote mode is unavailable")
}

func TestSwitchMode_RemoteWithoutCredsUnavailable(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	svc := newSvc(repo, routes, nil, &mocks.FakeCmdRunner{})

	_, _, err := svc.SwitchMode(context.Background(), config.ModeRemote)
	var unavailable config.ErrRemoteUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.Equal(t, 0, repo.upsertN, "failed switch does not persist the new mode")
}

// --- SwitchMode happy paths -----------------------------------------------

func TestSwitchMode_RemotePushesAndPersists(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	prev, cur, err := svc.SwitchMode(context.Background(), config.ModeRemote)
	require.NoError(t, err)
	require.Equal(t, config.ModeLocal, prev)
	require.Equal(t, config.ModeRemote, cur)
	require.Equal(t, 1, ingress.pushN)
	require.Equal(t, config.ModeRemote, repo.upserted.Mode)
	for _, r := range ingress.pushed {
		require.NotContains(t, r.Hostname, "vrooli.com")
	}
}

func TestSwitchMode_LocalWritesAndRestarts(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.yml"
	runner := &mocks.FakeCmdRunner{}
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote, TunnelID: "tid-1"}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}

	svc := config.NewService(config.Deps{
		Repo:            repo,
		Routes:          routes,
		Ingress:         &fakeIngress{},
		Runner:          runner.Run,
		Clock:           mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
		LocalConfigPath: cfgPath,
	})

	prev, cur, err := svc.SwitchMode(context.Background(), config.ModeLocal)
	require.NoError(t, err)
	require.Equal(t, config.ModeRemote, prev)
	require.Equal(t, config.ModeLocal, cur)
	require.Equal(t, 1, runner.CallCount(), "local switch restarts cloudflared")
	require.Equal(t, "sudo", runner.Calls[0].Name)
	require.Equal(t, []string{"systemctl", "restart", "cloudflared"}, runner.Calls[0].Args)

	written, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	require.Contains(t, string(written), "agent-manager.itsagitime.com")
	require.NotContains(t, string(written), "vrooli.com", "no hardcoded apex in generated config.yml")
	require.Contains(t, string(written), "http_status:404")
}

func TestSwitchMode_UnknownTargetInvalid(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeRoutes{}, &fakeIngress{}, &mocks.FakeCmdRunner{})
	_, _, err := svc.SwitchMode(context.Background(), config.Mode("sideways"))
	var invalid config.ErrInvalidConfig
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "target_mode", invalid.Field)
}

func TestSync_RemoteReresolvesCredentialsBeforeLiveRead(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	store := &fakeCredentialStore{
		cfg: config.CFConfig{AccountID: "acct", TunnelID: "tun", APIToken: "tok"},
		status: config.CredentialStatus{
			Ready:  true,
			Source: "file:scenario",
			Fields: []config.CredentialFieldStatus{
				{Name: "CLOUDFLARE_ACCOUNT_ID", Present: true, Source: "file:scenario", Writable: true},
				{Name: "CLOUDFLARE_TUNNEL_ID", Present: true, Source: "file:scenario", Writable: true},
				{Name: "CLOUDFLARE_API_TOKEN", Present: true, Source: "file:scenario", Ref: "file:scenario:cloudflare.api_token", Writable: true},
			},
		},
	}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	svc := config.NewService(config.Deps{
		Repo:            repo,
		Routes:          routes,
		Ingress:         ingress,
		CredentialStore: store,
		Runner:          (&mocks.FakeCmdRunner{}).Run,
		Clock:           mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	_, err := svc.Sync(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, 2, store.resolveN, "dry-run checks remote availability then reads live ingress through the current credential store")
}

func TestCredentialMutationsDelegateToStore(t *testing.T) {
	store := &fakeCredentialStore{status: config.CredentialStatus{Ready: true, Source: "file:scenario"}}
	svc := config.NewService(config.Deps{
		Repo:            &fakeRepo{},
		Routes:          &fakeRoutes{},
		CredentialStore: store,
	})

	status, err := svc.SetCloudflareCredentials(context.Background(), config.CredentialUpdate{
		AccountID: "acct", TunnelID: "tun", APIToken: "tok",
	})
	require.NoError(t, err)
	require.True(t, status.Ready)
	require.Equal(t, config.CredentialUpdate{AccountID: "acct", TunnelID: "tun", APIToken: "tok"}, store.saveIn)
	require.Equal(t, 1, store.saveN)

	_, err = svc.ClearCloudflareCredentials(context.Background(), []string{"api_token"})
	require.NoError(t, err)
	require.Equal(t, []string{"api_token"}, store.deleteIn)
	require.Equal(t, 1, store.deleteN)
}

// --- GetConfig -------------------------------------------------------------

func TestGetConfig_PassesThrough(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote, TunnelID: "tid"}}
	svc := newSvc(repo, &fakeRoutes{}, &fakeIngress{}, &mocks.FakeCmdRunner{})
	got, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "tid", got.TunnelID)
	require.Equal(t, config.ModeRemote, got.Mode)
}

func TestGetConfigState_ReportsLocalReadinessByDefault(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal}}
	svc := newSvc(repo, &fakeRoutes{}, nil, &mocks.FakeCmdRunner{})

	state, err := svc.GetConfigState(context.Background())
	require.NoError(t, err)
	require.Equal(t, config.ModeLocal, state.Readiness.DesiredMode)
	require.False(t, state.Readiness.RemoteAvailable)
	require.True(t, state.Readiness.SyncReady)
	require.Equal(t, "missing", state.Readiness.CredentialSource)
	require.NotEmpty(t, state.Readiness.LocalConfigPath)
	require.ElementsMatch(t, []string{"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_TUNNEL_ID", "CLOUDFLARE_API_TOKEN"}, state.Readiness.MissingFields)
}
