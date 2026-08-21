package config_test

import (
	"context"
	"os"
	"testing"
	"time"

	"tunnel-manager/internal/config"
	"tunnel-manager/internal/envreader"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/vrooli/api-core/scheduletest"

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
	cfg      config.CFConfig
	status   config.CredentialStatus
	resolveN int
	statusN  int
	saveIn   config.CredentialUpdate
	saveN    int
	deleteIn []string
	deleteN  int
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

// fakeLedger is an in-memory OwnershipLedger for service tests.
type fakeLedger struct {
	entries map[string]config.LedgerEntry
}

func newFakeLedger(entries ...config.LedgerEntry) *fakeLedger {
	m := make(map[string]config.LedgerEntry, len(entries))
	for _, e := range entries {
		m[e.Hostname] = e
	}
	return &fakeLedger{entries: m}
}

func (f *fakeLedger) List(context.Context) ([]config.LedgerEntry, error) {
	out := make([]config.LedgerEntry, 0, len(f.entries))
	for _, e := range f.entries {
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeLedger) Get(_ context.Context, host string) (config.LedgerEntry, bool, error) {
	e, ok := f.entries[host]
	return e, ok, nil
}

func (f *fakeLedger) Put(_ context.Context, e config.LedgerEntry) error {
	f.entries[e.Hostname] = e
	return nil
}

func (f *fakeLedger) Delete(_ context.Context, host string) (bool, error) {
	_, ok := f.entries[host]
	delete(f.entries, host)
	return ok, nil
}

// fakeRoutesWriter is an in-memory RoutesManager for adopt tests.
type fakeRoutesWriter struct {
	created  []internalroutes.CreateInput
	updated  []internalroutes.UpdateInput
	bySubdom map[string]internalroutes.Route
}

func (f *fakeRoutesWriter) GetBySubdomain(_ context.Context, sub string) (internalroutes.Route, error) {
	if r, ok := f.bySubdom[sub]; ok {
		return r, nil
	}
	return internalroutes.Route{}, internalroutes.ErrRouteNotFound{ID: sub}
}

func (f *fakeRoutesWriter) Create(_ context.Context, in internalroutes.CreateInput) (internalroutes.Route, error) {
	f.created = append(f.created, in)
	return internalroutes.Route{ID: "r-1", Subdomain: in.Subdomain, Domain: in.Domain, Source: in.Source, ServiceTarget: in.ServiceTarget, Scenario: in.Scenario, LocalPort: in.LocalPort}, nil
}

func (f *fakeRoutesWriter) Update(_ context.Context, id string, in internalroutes.UpdateInput) (internalroutes.Route, error) {
	f.updated = append(f.updated, in)
	return internalroutes.Route{ID: id, Subdomain: in.Subdomain, Domain: in.Domain, Source: in.Source, ServiceTarget: in.ServiceTarget, Scenario: in.Scenario, LocalPort: in.LocalPort}, nil
}

// fakeScenarioResolver maps known scenario slugs to their fixed UI port. Slugs
// in `ranged` are known scenarios with NO fixed port (dynamically allocated).
type fakeScenarioResolver struct {
	ports  map[string]int
	ranged map[string]bool
}

func (f fakeScenarioResolver) UIPort(_ context.Context, scenario string) (int, error) {
	if p, ok := f.ports[scenario]; ok {
		return p, nil
	}
	return 0, internalroutes.ErrRouteNotFound{ID: scenario}
}

func (f fakeScenarioResolver) IsScenario(_ context.Context, scenario string) bool {
	_, fixed := f.ports[scenario]
	return fixed || f.ranged[scenario]
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
		Clock:   scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})
}

// --- Sync diff -------------------------------------------------------------

// Sync is ADDITIVE by default: it adds desired hostnames and PRESERVES
// pre-existing foreign/unmanaged live entries rather than dropping them. The
// old destructive full-replace is gone.
func TestSync_AdditivePreservesForeignIngress(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{
		route("agent-manager", 21100, true),
		route("web-console", 3000, true),
		route("disabled-one", 9999, false),
	}}
	// Live ingress has a managed hostname (web-console), a foreign one
	// (legacy) TM did not author, and lacks agent-manager (to be added).
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "web-console.itsagitime.com", Service: "http://localhost:3000"},
		{Hostname: "legacy.itsagitime.com", Service: "http://localhost:1"},
		{Service: "http_status:404"},
	}}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.Equal(t, []string{"agent-manager.itsagitime.com"}, res.Added)
	require.Empty(t, res.Removed, "additive sync removes nothing")
	require.Equal(t, []string{"legacy.itsagitime.com"}, res.DriftUnmanaged, "foreign hostname surfaced as drift")
	require.False(t, res.NoChanges)
	require.Equal(t, config.ModeRemote, res.Mode)

	// The published set is the UNION: foreign legacy entry survives, desired
	// agent-manager added, never a hardcoded ".vrooli.com" apex, disabled
	// routes excluded.
	require.Equal(t, 1, ingress.pushN)
	pushedHosts := map[string]bool{}
	for _, r := range ingress.pushed {
		pushedHosts[r.Hostname] = true
		require.NotContains(t, r.Hostname, "vrooli.com", "must not leak hardcoded apex")
		require.NotContains(t, r.Hostname, "disabled-one")
	}
	require.True(t, pushedHosts["legacy.itsagitime.com"], "foreign ingress must survive an additive sync")
	require.True(t, pushedHosts["agent-manager.itsagitime.com"])
	require.True(t, pushedHosts["web-console.itsagitime.com"])
}

// TestSync_LocalModeMergePreservesForeignConfigYAML: a local-mode sync reads
// the existing cloudflared config.yml, merges TM's desired entries onto it, and
// writes back — foreign entries the operator added survive the round-trip.
func TestSync_LocalModeMergePreservesForeignConfigYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.yml"
	seed := "tunnel: tid-1\ningress:\n" +
		"  - hostname: foreign.itsagitime.com\n    service: http://localhost:7000\n" +
		"  - service: http_status:404\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(seed), 0o644))

	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal, TunnelID: "tid-1"}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	runner := &mocks.FakeCmdRunner{}
	// No remote client/credentials wired: local mode is only effective when
	// Cloudflare is unavailable. With complete credentials TM reconciles against
	// the real remote tunnel instead (see effectiveMode), so a genuine
	// local-config.yml round-trip requires remote to be absent.
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: routes, Ledger: newFakeLedger(),
		Runner:          runner.Run,
		Clock:           scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
		LocalConfigPath: cfgPath,
	})

	res, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.Equal(t, []string{"agent-manager.itsagitime.com"}, res.Added)
	require.Equal(t, []string{"foreign.itsagitime.com"}, res.DriftUnmanaged)
	require.Equal(t, 1, runner.CallCount(), "local sync restarts cloudflared")

	written, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	require.Contains(t, string(written), "foreign.itsagitime.com", "foreign config.yml entry must survive an additive sync")
	require.Contains(t, string(written), "agent-manager.itsagitime.com")
	require.Contains(t, string(written), "http_status:404")
}

func TestSync_DryRunAppliesNothing(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), true, false)
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

	res, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.True(t, res.NoChanges)
	require.Empty(t, res.Added)
	require.Empty(t, res.Removed)
	require.Equal(t, 0, ingress.pushN, "no drift applies nothing")
}

// TestSync_PruneRemovesOrphanedOnly: a batch --prune removes ledger-managed
// hostnames whose routes are gone (orphaned), but never touches genuine
// unmanaged drift.
func TestSync_PruneRemovesOrphanedOnly(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "agent-manager.itsagitime.com", Service: "http://localhost:21100"},
		{Hostname: "orphan.itsagitime.com", Service: "http://localhost:2"}, // ledger MANAGED, route gone
		{Hostname: "foreign.example.com", Service: "http://localhost:3"},   // UNMANAGED
		{Service: "http_status:404"},
	}}
	ledger := newFakeLedger(config.LedgerEntry{Hostname: "orphan.itsagitime.com", Owner: config.OwnerManaged})
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: routes, Ingress: ingress, Ledger: ledger,
		Runner: (&mocks.FakeCmdRunner{}).Run,
		Clock:  scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	res, err := svc.Sync(context.Background(), false, true)
	require.NoError(t, err)
	require.Equal(t, []string{"orphan.itsagitime.com"}, res.Orphaned)
	require.Equal(t, []string{"orphan.itsagitime.com"}, res.Pruned)
	require.Equal(t, []string{"foreign.example.com"}, res.DriftUnmanaged)

	pushedHosts := map[string]bool{}
	for _, r := range ingress.pushed {
		pushedHosts[r.Hostname] = true
	}
	require.False(t, pushedHosts["orphan.itsagitime.com"], "orphaned entry pruned")
	require.True(t, pushedHosts["foreign.example.com"], "unmanaged drift left intact even with --prune")
	require.True(t, pushedHosts["agent-manager.itsagitime.com"])

	// The pruned hostname's ledger record is cleared.
	_, found, err := ledger.Get(context.Background(), "orphan.itsagitime.com")
	require.NoError(t, err)
	require.False(t, found)
}

// TestSync_RecordsManagedOwnership: applying a scenario route records a
// MANAGED ledger entry so a later manifest removal classifies it as orphaned.
func TestSync_RecordsManagedOwnership(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	ledger := newFakeLedger()
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: routes, Ingress: ingress, Ledger: ledger,
		Runner: (&mocks.FakeCmdRunner{}).Run,
		Clock:  scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	got, found, err := ledger.Get(context.Background(), "agent-manager.itsagitime.com")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, config.OwnerManaged, got.Owner)
	require.Equal(t, "agent-manager", got.Scenario)
}

// --- Remote unavailable ----------------------------------------------------

func TestSync_RemoteWithoutCredsUnavailable(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	svc := newSvc(repo, routes, nil /* no IngressClient */, &mocks.FakeCmdRunner{})

	_, err := svc.Sync(context.Background(), false, false)
	var unavailable config.ErrRemoteUnavailable
	require.ErrorAs(t, err, &unavailable)
}

func TestSync_DryRunRemoteWithoutCredsReturnsSetupReport(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	svc := newSvc(repo, routes, nil /* no IngressClient */, &mocks.FakeCmdRunner{})

	res, err := svc.Sync(context.Background(), true, false)
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

// SwitchMode is now PURE — it persists the mode and performs ZERO ingress
// writes. The push/restart that used to happen here was the central footgun
// (a good token meant an immediate overwrite of the live tunnel). Apply is now
// only ever an explicit Sync.
func TestSwitchMode_RemotePersistsWithoutPushing(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{}
	svc := newSvc(repo, routes, ingress, &mocks.FakeCmdRunner{})

	prev, cur, err := svc.SwitchMode(context.Background(), config.ModeRemote)
	require.NoError(t, err)
	require.Equal(t, config.ModeLocal, prev)
	require.Equal(t, config.ModeRemote, cur)
	require.Equal(t, 0, ingress.pushN, "switching mode must never push ingress")
	require.Equal(t, config.ModeRemote, repo.upserted.Mode)
}

func TestSwitchMode_LocalPersistsWithoutWritingOrRestarting(t *testing.T) {
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
		Clock:           scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
		LocalConfigPath: cfgPath,
	})

	prev, cur, err := svc.SwitchMode(context.Background(), config.ModeLocal)
	require.NoError(t, err)
	require.Equal(t, config.ModeRemote, prev)
	require.Equal(t, config.ModeLocal, cur)
	require.Equal(t, 0, runner.CallCount(), "switching mode must not restart cloudflared")
	require.Equal(t, config.ModeLocal, repo.upserted.Mode)

	// The local config.yml is NOT written on switch — only an explicit Sync
	// writes it.
	_, err = os.ReadFile(cfgPath)
	require.True(t, os.IsNotExist(err), "switching mode must not write config.yml")
}

// --- GetDrift read model ---------------------------------------------------

func TestGetDrift_ClassifiesDesiredLiveAndLedger(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{
		route("agent-manager", 21100, true), // desired & live → MANAGED
		route("new-one", 21200, true),       // desired & !live → MISSING
	}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "agent-manager.itsagitime.com", Service: "http://localhost:21100"},
		{Hostname: "foreign.example.com", Service: "http://localhost:1"}, // UNMANAGED
		{Hostname: "kept.itsagitime.com", Service: "http://localhost:2"}, // IGNORED
		{Service: "http_status:404"},
	}}
	ledger := newFakeLedger(config.LedgerEntry{Hostname: "kept.itsagitime.com", Owner: config.OwnerIgnored})
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: routes, Ingress: ingress, Ledger: ledger,
		Runner: (&mocks.FakeCmdRunner{}).Run,
		Clock:  scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	rep, err := svc.GetDrift(context.Background())
	require.NoError(t, err)
	require.Equal(t, config.ModeRemote, rep.Mode)
	require.Equal(t, 1, rep.Counts[config.StateManaged])
	require.Equal(t, 1, rep.Counts[config.StateMissing])
	require.Equal(t, 1, rep.Counts[config.StateUnmanaged])
	require.Equal(t, 1, rep.Counts[config.StateIgnored])
	require.Equal(t, 0, ingress.pushN, "GetDrift performs reads only")
}

// TestGetDrift_LocalDefaultWithCredentialsReadsRemote is the regression test
// for the reported bug: a tunnel set up via the Cloudflare dashboard, with all
// credentials configured but the persisted mode still the local default, must
// show the drift view reconciled against the REAL remote tunnel — not against
// an empty local ~/.cloudflared/config.yml. Before the fix, GetDrift read the
// (absent) local config.yml, so every desired route showed as MISSING ("not
// registered") and the pre-existing remote ingress was invisible.
func TestGetDrift_LocalDefaultWithCredentialsReadsRemote(t *testing.T) {
	// Persisted mode is the local default — the user never flipped the toggle.
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal}}
	routes := &fakeRoutes{routes: []internalroutes.Route{
		route("agent-manager", 21100, true), // desired; live on Cloudflare → MANAGED
	}}
	// The real remote tunnel already has the managed route plus other ingress
	// the operator registered directly on Cloudflare.
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "agent-manager.itsagitime.com", Service: "http://localhost:21100"},
		{Hostname: "app-monitor.itsagitime.com", Service: "http://localhost:5000"}, // UNMANAGED
		{Hostname: "web-console.itsagitime.com", Service: "http://localhost:6000"}, // UNMANAGED
		{Service: "http_status:404"},
	}}
	// A wired Ingress client (≈ complete credentials) means remote is available.
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: routes, Ingress: ingress, Ledger: newFakeLedger(),
		Runner: (&mocks.FakeCmdRunner{}).Run,
		Clock:  scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	rep, err := svc.GetDrift(context.Background())
	require.NoError(t, err)
	require.Equal(t, config.ModeRemote, rep.Mode, "credentials present ⇒ reconcile against the real remote tunnel")
	require.Equal(t, 1, rep.Counts[config.StateManaged], "the desired route is live on Cloudflare")
	require.Equal(t, 2, rep.Counts[config.StateUnmanaged], "pre-existing remote ingress is surfaced as drift, not hidden")
	require.Equal(t, 0, rep.Counts[config.StateMissing], "desired routes are NOT falsely reported missing")
	require.Equal(t, 0, ingress.pushN, "GetDrift performs reads only")
}

// --- Drift actions: adopt / ignore / prune --------------------------------

func driftSvc(repo config.ConfigRepository, routes config.RoutesReader, writer config.RoutesManager, ingress config.IngressClient, ledger config.OwnershipLedger) config.Service {
	return config.NewService(config.Deps{
		Repo: repo, Routes: routes, RoutesWriter: writer, Ingress: ingress, Ledger: ledger,
		Runner: (&mocks.FakeCmdRunner{}).Run,
		Clock:  scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})
}

func TestAdoptIngress_AsExternalCreatesRouteAndLedger(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{{Hostname: "api.itsagitime.com", Service: "http://127.0.0.1:9000"}}}
	writer := &fakeRoutesWriter{bySubdom: map[string]internalroutes.Route{}}
	ledger := newFakeLedger()
	svc := driftSvc(repo, &fakeRoutes{}, writer, ingress, ledger)

	entry, err := svc.AdoptIngress(context.Background(), "api.itsagitime.com", "", "")
	require.NoError(t, err)
	require.Equal(t, config.StateExternalOK, entry.State)
	require.Equal(t, config.SourceExternal, entry.Source)
	require.Len(t, writer.created, 1)
	require.Equal(t, internalroutes.SourceExternal, writer.created[0].Source)
	require.Equal(t, "http://127.0.0.1:9000", writer.created[0].ServiceTarget, "adopts the live target when none supplied")
	got, found, _ := ledger.Get(context.Background(), "api.itsagitime.com")
	require.True(t, found)
	require.Equal(t, config.OwnerExternal, got.Owner)
}

func TestAdoptIngress_AsScenarioParsesPort(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{{Hostname: "web.itsagitime.com", Service: "http://localhost:3000"}}}
	writer := &fakeRoutesWriter{bySubdom: map[string]internalroutes.Route{}}
	ledger := newFakeLedger()
	svc := driftSvc(repo, &fakeRoutes{}, writer, ingress, ledger)

	entry, err := svc.AdoptIngress(context.Background(), "web.itsagitime.com", "web-console", "")
	require.NoError(t, err)
	require.Equal(t, config.StateManaged, entry.State)
	require.Equal(t, internalroutes.SourceScenario, writer.created[0].Source)
	require.Equal(t, 3000, writer.created[0].LocalPort)
	require.Equal(t, "web-console", writer.created[0].Scenario)
}

// TestAdoptIngress_BareAdoptAutoDetectsScenario is the regression for the
// reported bug: adopting a drift hostname from the UI (bare adopt — no explicit
// scenario or target) for a hostname whose subdomain matches a known scenario
// must produce a SCENARIO route with the scenario's real UI port, NOT an
// external route with port 0.
func TestAdoptIngress_BareAdoptAutoDetectsScenario(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "agent-inbox.itsagitime.com", Service: "http://localhost:21237"},
	}}
	writer := &fakeRoutesWriter{bySubdom: map[string]internalroutes.Route{}}
	ledger := newFakeLedger()
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: &fakeRoutes{}, RoutesWriter: writer, Ledger: ledger, Ingress: ingress,
		Scenarios: fakeScenarioResolver{ports: map[string]int{"agent-inbox": 21237}},
		Runner:    (&mocks.FakeCmdRunner{}).Run,
		Clock:     scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	entry, err := svc.AdoptIngress(context.Background(), "agent-inbox.itsagitime.com", "", "")
	require.NoError(t, err)
	require.Equal(t, config.StateManaged, entry.State, "a known-scenario hostname adopts as a managed scenario route")
	require.Equal(t, config.SourceScenario, entry.Source)
	require.Len(t, writer.created, 1)
	require.Equal(t, internalroutes.SourceScenario, writer.created[0].Source)
	require.Equal(t, "agent-inbox", writer.created[0].Scenario)
	require.Equal(t, 21237, writer.created[0].LocalPort, "uses the scenario's real UI port, not 0")
	require.Empty(t, writer.created[0].ServiceTarget, "scenario routes derive their target from the port")
	got, found, _ := ledger.Get(context.Background(), "agent-inbox.itsagitime.com")
	require.True(t, found)
	require.Equal(t, config.OwnerManaged, got.Owner)
	require.Equal(t, "agent-inbox", got.Scenario)
}

// TestAdoptIngress_BareAdoptRangedPortScenarioUsesLivePort: a known scenario
// whose UI port is ranged (no fixed port in service.json) still adopts as a
// scenario route, using the live localhost port — not as an external route.
func TestAdoptIngress_BareAdoptRangedPortScenarioUsesLivePort(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "scenario-completeness-scoring.itsagitime.com", Service: "http://localhost:21242"},
	}}
	writer := &fakeRoutesWriter{bySubdom: map[string]internalroutes.Route{}}
	ledger := newFakeLedger()
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: &fakeRoutes{}, RoutesWriter: writer, Ledger: ledger, Ingress: ingress,
		Scenarios: fakeScenarioResolver{ranged: map[string]bool{"scenario-completeness-scoring": true}},
		Runner:    (&mocks.FakeCmdRunner{}).Run,
		Clock:     scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	entry, err := svc.AdoptIngress(context.Background(), "scenario-completeness-scoring.itsagitime.com", "", "")
	require.NoError(t, err)
	require.Equal(t, config.StateManaged, entry.State)
	require.Equal(t, internalroutes.SourceScenario, writer.created[0].Source)
	require.Equal(t, 21242, writer.created[0].LocalPort, "ranged-port scenario uses the live localhost port")
	require.Equal(t, "scenario-completeness-scoring", writer.created[0].Scenario)
}

// TestAdoptIngress_BareAdoptUnknownSubdomainFallsBackToExternal: a hostname
// whose subdomain is NOT a known scenario still adopts (as external, pointing
// at the live service) so genuinely-foreign ingress remains manageable.
func TestAdoptIngress_BareAdoptUnknownSubdomainFallsBackToExternal(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "grafana.itsagitime.com", Service: "http://localhost:3000"},
	}}
	writer := &fakeRoutesWriter{bySubdom: map[string]internalroutes.Route{}}
	ledger := newFakeLedger()
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: &fakeRoutes{}, RoutesWriter: writer, Ledger: ledger, Ingress: ingress,
		Scenarios: fakeScenarioResolver{ports: map[string]int{"agent-inbox": 21237}},
		Runner:    (&mocks.FakeCmdRunner{}).Run,
		Clock:     scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	entry, err := svc.AdoptIngress(context.Background(), "grafana.itsagitime.com", "", "")
	require.NoError(t, err)
	require.Equal(t, config.StateExternalOK, entry.State)
	require.Equal(t, internalroutes.SourceExternal, writer.created[0].Source)
	require.Equal(t, "http://localhost:3000", writer.created[0].ServiceTarget)
}

// TestAdoptIngress_ReadoptRepairsExistingRoute: re-adopting a hostname that was
// previously mis-classified (external, port 0) updates the existing route in
// place instead of failing on the unique-subdomain conflict — this is how the
// already-adopted-as-external rows get repaired into scenario routes.
func TestAdoptIngress_ReadoptRepairsExistingRoute(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "agent-inbox.itsagitime.com", Service: "http://localhost:21237"},
	}}
	// An existing, wrongly-external route for the same subdomain.
	writer := &fakeRoutesWriter{bySubdom: map[string]internalroutes.Route{
		"agent-inbox": {ID: "r-existing", Subdomain: "agent-inbox", Domain: internalroutes.DefaultDomain, Source: internalroutes.SourceExternal, ServiceTarget: "http://localhost:21237"},
	}}
	ledger := newFakeLedger(config.LedgerEntry{Hostname: "agent-inbox.itsagitime.com", Owner: config.OwnerExternal})
	svc := config.NewService(config.Deps{
		Repo: repo, Routes: &fakeRoutes{}, RoutesWriter: writer, Ledger: ledger, Ingress: ingress,
		Scenarios: fakeScenarioResolver{ports: map[string]int{"agent-inbox": 21237}},
		Runner:    (&mocks.FakeCmdRunner{}).Run,
		Clock:     scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	entry, err := svc.AdoptIngress(context.Background(), "agent-inbox.itsagitime.com", "", "")
	require.NoError(t, err)
	require.Equal(t, config.StateManaged, entry.State)
	require.Empty(t, writer.created, "must not create a conflicting route")
	require.Len(t, writer.updated, 1, "repairs the existing route in place")
	require.Equal(t, internalroutes.SourceScenario, writer.updated[0].Source)
	require.Equal(t, 21237, writer.updated[0].LocalPort)
	require.Equal(t, "agent-inbox", writer.updated[0].Scenario)
	got, _, _ := ledger.Get(context.Background(), "agent-inbox.itsagitime.com")
	require.Equal(t, config.OwnerManaged, got.Owner, "ledger flips EXTERNAL → MANAGED")
}

func TestIgnoreIngress_RecordsLedgerNoApply(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{{Hostname: "legacy.itsagitime.com", Service: "http://localhost:1"}}}
	ledger := newFakeLedger()
	svc := driftSvc(repo, &fakeRoutes{}, nil, ingress, ledger)

	entry, err := svc.IgnoreIngress(context.Background(), "legacy.itsagitime.com", "operator dashboard")
	require.NoError(t, err)
	require.Equal(t, config.StateIgnored, entry.State)
	require.Equal(t, 0, ingress.pushN, "ignore never applies")
	got, found, _ := ledger.Get(context.Background(), "legacy.itsagitime.com")
	require.True(t, found)
	require.Equal(t, config.OwnerIgnored, got.Owner)
	require.Equal(t, "operator dashboard", got.Note)
}

func TestPruneIngress_RemovesNamedHostnameOnly(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: "drop.itsagitime.com", Service: "http://localhost:1"},
		{Hostname: "keep.itsagitime.com", Service: "http://localhost:2"},
		{Service: "http_status:404"},
	}}
	ledger := newFakeLedger(config.LedgerEntry{Hostname: "drop.itsagitime.com", Owner: config.OwnerExternal})
	svc := driftSvc(repo, &fakeRoutes{}, nil, ingress, ledger)

	pruned, err := svc.PruneIngress(context.Background(), "drop.itsagitime.com")
	require.NoError(t, err)
	require.True(t, pruned)
	pushedHosts := map[string]bool{}
	for _, r := range ingress.pushed {
		pushedHosts[r.Hostname] = true
	}
	require.False(t, pushedHosts["drop.itsagitime.com"])
	require.True(t, pushedHosts["keep.itsagitime.com"])
	_, found, _ := ledger.Get(context.Background(), "drop.itsagitime.com")
	require.False(t, found, "prune clears the ledger record")
}

func TestPruneIngress_MissingHostnameReturnsFalse(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	svc := driftSvc(repo, &fakeRoutes{}, nil, &fakeIngress{}, newFakeLedger())
	pruned, err := svc.PruneIngress(context.Background(), "ghost.itsagitime.com")
	require.NoError(t, err)
	require.False(t, pruned)
}

func TestSwitchMode_SameModeIsNoOp(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeLocal}}
	svc := newSvc(repo, &fakeRoutes{}, &fakeIngress{}, &mocks.FakeCmdRunner{})

	prev, cur, err := svc.SwitchMode(context.Background(), config.ModeLocal)
	require.NoError(t, err)
	require.Equal(t, config.ModeLocal, prev)
	require.Equal(t, config.ModeLocal, cur)
	require.Equal(t, 0, repo.upsertN, "switching to the current mode persists nothing")
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
			Source: "credential-authority",
			Fields: []config.CredentialFieldStatus{
				{Name: "CLOUDFLARE_ACCOUNT_ID", Present: true, Source: "credential-authority", Writable: true},
				{Name: "CLOUDFLARE_TUNNEL_ID", Present: true, Source: "credential-authority", Writable: true},
				{Name: "CLOUDFLARE_API_TOKEN", Present: true, Source: "credential-authority", Ref: "vrooli/tunnel-manager:cloudflare-api-token", Writable: true},
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
		Clock:           scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})

	_, err := svc.Sync(context.Background(), true, false)
	require.NoError(t, err)
	require.Equal(t, 2, store.resolveN, "dry-run checks remote availability then reads live ingress through the current credential store")
}

func TestCredentialMutationsDelegateToStore(t *testing.T) {
	store := &fakeCredentialStore{status: config.CredentialStatus{Ready: true, Source: "credential-authority"}}
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

func TestGetConfig_UsesManagedResourceEndpointWhenPersistedValueIsLegacyDefault(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote, PromEndpoint: config.DefaultPromEndpoint}}
	svc := config.NewService(config.Deps{
		Repo:    repo,
		Routes:  &fakeRoutes{},
		Ingress: &fakeIngress{},
		Runner:  (&mocks.FakeCmdRunner{}).Run,
		Clock:   scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
		Env: envreader.Func(func(key string) string {
			if key == "TUNNEL_METRICS_URL" {
				return "http://127.0.0.1:20242"
			}
			return ""
		}),
	})

	got, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:20242", got.PromEndpoint)
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
