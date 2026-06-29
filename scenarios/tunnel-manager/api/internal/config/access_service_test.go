package config_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"tunnel-manager/internal/config"
	"tunnel-manager/internal/manifest"
	"tunnel-manager/internal/testutil/mocks"
)

// fakeAccess records EnsurePublicBypass/RemovePublicBypass calls for
// service-level assertions.
type fakeAccess struct {
	ensured   []string
	removed   []string
	created   map[string]bool // host -> report Created
	ensureErr error
}

func (f *fakeAccess) EnsurePublicBypass(_ context.Context, host string) (config.AccessResult, error) {
	f.ensured = append(f.ensured, host)
	if f.ensureErr != nil {
		return config.AccessResult{}, f.ensureErr
	}
	created := true
	if f.created != nil {
		created = f.created[host]
	}
	return config.AccessResult{AppID: "app-" + host, PolicyID: "pol-" + host, Created: created}, nil
}

func (f *fakeAccess) RemovePublicBypass(_ context.Context, host string) (bool, error) {
	f.removed = append(f.removed, host)
	return true, nil
}

func (f *fakeAccess) LookupPublicBypass(_ context.Context, host string) (config.AccessApp, bool, error) {
	return config.AccessApp{}, false, nil
}

// fakeAccessLedger is an in-memory AccessLedger.
type fakeAccessLedger struct {
	entries map[string]config.AccessAppEntry
}

func newFakeAccessLedger(hosts ...string) *fakeAccessLedger {
	m := map[string]config.AccessAppEntry{}
	for _, h := range hosts {
		m[h] = config.AccessAppEntry{Host: h, AppID: "app-" + h, PolicyID: "pol-" + h}
	}
	return &fakeAccessLedger{entries: m}
}

func (f *fakeAccessLedger) List(context.Context) ([]config.AccessAppEntry, error) {
	out := make([]config.AccessAppEntry, 0, len(f.entries))
	for _, e := range f.entries {
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeAccessLedger) Get(_ context.Context, host string) (config.AccessAppEntry, bool, error) {
	e, ok := f.entries[host]
	return e, ok, nil
}

func (f *fakeAccessLedger) Put(_ context.Context, e config.AccessAppEntry) error {
	f.entries[e.Host] = e
	return nil
}

func (f *fakeAccessLedger) Delete(_ context.Context, host string) (bool, error) {
	_, ok := f.entries[host]
	delete(f.entries, host)
	return ok, nil
}

func exposedRoute(sub string, port int, exposure manifest.PublicExposure) manifest.Route {
	return manifest.Route{
		Subdomain: sub, Scenario: sub, Domain: manifest.DefaultDomain,
		LocalPort: port, Tier: manifest.TierLeased, Enabled: true,
		PublicExposure: exposure,
	}
}

func newSvcWithAccess(cfg config.TunnelConfig, routes []manifest.Route, ingress config.IngressClient, access config.AccessClient, accessLedger config.AccessLedger) config.Service {
	return config.NewService(config.Deps{
		Repo:         &fakeRepo{cfg: cfg},
		Routes:       &fakeRoutes{routes: routes},
		Ingress:      ingress,
		Ledger:       newFakeLedger(),
		Access:       access,
		AccessLedger: accessLedger,
		Runner:       (&mocks.FakeCmdRunner{}).Run,
		Clock:        mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})
}

// Global switch ON: a desired host (inherit) gets a /public bypass ensured, and
// the TM-created app is recorded in the access ledger.
func TestSync_EnsuresBypassWhenGloballyEnabled(t *testing.T) {
	cfg := config.TunnelConfig{Mode: config.ModeRemote, PublicExposureEnabled: true}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	access := &fakeAccess{}
	ledger := newFakeAccessLedger()

	svc := newSvcWithAccess(cfg, []manifest.Route{exposedRoute("web-console", 21100, manifest.PublicExposureInherit)}, ingress, access, ledger)
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)

	require.Equal(t, []string{"web-console.itsagitime.com"}, access.ensured)
	_, ok, _ := ledger.Get(context.Background(), "web-console.itsagitime.com")
	require.True(t, ok, "TM-created bypass app recorded in the access ledger")
}

// Global switch OFF + all inherit: no bypass is ever created (the default-dark
// posture). The capability stays a no-op for the vast majority of installs.
func TestSync_NoBypassWhenGloballyDisabled(t *testing.T) {
	cfg := config.TunnelConfig{Mode: config.ModeRemote, PublicExposureEnabled: false}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	access := &fakeAccess{}

	svc := newSvcWithAccess(cfg, []manifest.Route{exposedRoute("web-console", 21100, manifest.PublicExposureInherit)}, ingress, access, newFakeAccessLedger())
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.Empty(t, access.ensured, "no bypass when globally disabled and route inherits")
}

// A per-route override of `enabled` forces the bypass on even when the global
// switch is off.
func TestSync_RouteOverrideEnablesBypass(t *testing.T) {
	cfg := config.TunnelConfig{Mode: config.ModeRemote, PublicExposureEnabled: false}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	access := &fakeAccess{}

	svc := newSvcWithAccess(cfg, []manifest.Route{exposedRoute("web-console", 21100, manifest.PublicExposureEnabled)}, ingress, access, newFakeAccessLedger())
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.Equal(t, []string{"web-console.itsagitime.com"}, access.ensured, "route override `enabled` wins over global off")
}

// A per-route override of `disabled` forces the bypass off even when the global
// switch is on — and removes a previously-created app (ledger-gated).
func TestSync_RouteOverrideDisablesAndRemoves(t *testing.T) {
	cfg := config.TunnelConfig{Mode: config.ModeRemote, PublicExposureEnabled: true}
	host := "web-console.itsagitime.com"
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: host, Service: "http://localhost:21100"},
		{Service: "http_status:404"},
	}}
	access := &fakeAccess{}
	ledger := newFakeAccessLedger(host) // TM previously created the bypass.

	svc := newSvcWithAccess(cfg, []manifest.Route{exposedRoute("web-console", 21100, manifest.PublicExposureDisabled)}, ingress, access, ledger)
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.Empty(t, access.ensured, "disabled route is never ensured")
	require.Equal(t, []string{host}, access.removed, "disabled route's previously-created bypass is removed")
	_, ok, _ := ledger.Get(context.Background(), host)
	require.False(t, ok, "access ledger row cleared after removal")
}

// A dry-run never mutates Access.
func TestSync_DryRunSkipsAccess(t *testing.T) {
	cfg := config.TunnelConfig{Mode: config.ModeRemote, PublicExposureEnabled: true}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	access := &fakeAccess{}

	svc := newSvcWithAccess(cfg, []manifest.Route{exposedRoute("web-console", 21100, manifest.PublicExposureInherit)}, ingress, access, newFakeAccessLedger())
	_, err := svc.Sync(context.Background(), true, false)
	require.NoError(t, err)
	require.Empty(t, access.ensured, "dry-run must not create Access apps")
	require.Empty(t, access.removed)
}

// GetAccessStatus computes the dry-run plan + per-host effective decisions with
// no mutation.
func TestGetAccessStatus_PlanAndHosts(t *testing.T) {
	cfg := config.TunnelConfig{Mode: config.ModeRemote, PublicExposureEnabled: true}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	access := &fakeAccess{}

	svc := newSvcWithAccess(cfg, []manifest.Route{
		exposedRoute("web-console", 21100, manifest.PublicExposureInherit),
		exposedRoute("secret-app", 21101, manifest.PublicExposureDisabled),
	}, ingress, access, newFakeAccessLedger())

	st, err := svc.GetAccessStatus(context.Background())
	require.NoError(t, err)
	require.True(t, st.Enabled)
	require.True(t, st.Configured)
	require.Equal(t, []string{"web-console.itsagitime.com"}, st.ToCreate, "inherit+global-on is a create candidate")
	require.Empty(t, st.ToRemove)
	require.Empty(t, access.ensured, "status is pure — no mutation")

	byHost := map[string]config.AccessHostState{}
	for _, h := range st.Hosts {
		byHost[h.Host] = h
	}
	require.True(t, byHost["web-console.itsagitime.com"].EffectiveBypass)
	require.False(t, byHost["secret-app.itsagitime.com"].EffectiveBypass, "disabled route has no effective bypass")
}

// SetPublicExposure persists the global switch without writing to Cloudflare.
func TestSetPublicExposure_Persists(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	svc := config.NewService(config.Deps{
		Repo:   repo,
		Routes: &fakeRoutes{},
		Clock:  mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})
	updated, err := svc.SetPublicExposure(context.Background(), true)
	require.NoError(t, err)
	require.True(t, updated.PublicExposureEnabled)
	require.True(t, repo.cfg.PublicExposureEnabled, "flag persisted to the repo")
}
