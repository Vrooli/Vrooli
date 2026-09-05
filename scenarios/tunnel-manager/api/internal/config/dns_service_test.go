package config_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"tunnel-manager/internal/config"
	internalroutes "tunnel-manager/internal/manifest"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

// fakeDNS records EnsureRecord/RemoveRecord calls for service-level assertions.
type fakeDNS struct {
	ensured  []string
	removed  []string
	created  map[string]bool // hostname -> report Created
	ensueErr error
}

func (f *fakeDNS) EnsureRecord(_ context.Context, hostname string) (config.DNSResult, error) {
	f.ensured = append(f.ensured, hostname)
	if f.ensueErr != nil {
		return config.DNSResult{}, f.ensueErr
	}
	created := true
	if f.created != nil {
		created = f.created[hostname]
	}
	return config.DNSResult{RecordID: "rec-" + hostname, Created: created}, nil
}

func (f *fakeDNS) RemoveRecord(_ context.Context, hostname string) (bool, error) {
	f.removed = append(f.removed, hostname)
	return true, nil
}

// fakeDNSLedger is an in-memory DNSLedger.
type fakeDNSLedger struct {
	entries map[string]config.DNSRecordEntry
}

func newFakeDNSLedger(hosts ...string) *fakeDNSLedger {
	m := map[string]config.DNSRecordEntry{}
	for _, h := range hosts {
		m[h] = config.DNSRecordEntry{Hostname: h, RecordID: "rec-" + h}
	}
	return &fakeDNSLedger{entries: m}
}

func (f *fakeDNSLedger) List(context.Context) ([]config.DNSRecordEntry, error) {
	out := make([]config.DNSRecordEntry, 0, len(f.entries))
	for _, e := range f.entries {
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeDNSLedger) Get(_ context.Context, host string) (config.DNSRecordEntry, bool, error) {
	e, ok := f.entries[host]
	return e, ok, nil
}

func (f *fakeDNSLedger) Put(_ context.Context, e config.DNSRecordEntry) error {
	f.entries[e.Hostname] = e
	return nil
}

func (f *fakeDNSLedger) Delete(_ context.Context, host string) (bool, error) {
	_, ok := f.entries[host]
	delete(f.entries, host)
	return ok, nil
}

func newSvcWithDNS(repo config.ConfigRepository, routes config.RoutesReader, ingress config.IngressClient, ledger config.OwnershipLedger, dns config.DNSClient, dnsLedger config.DNSLedger) config.Service {
	return config.NewService(config.Deps{
		Repo:      repo,
		Routes:    routes,
		Ingress:   ingress,
		Ledger:    ledger,
		DNS:       dns,
		DNSLedger: dnsLedger,
		Runner:    (&mocks.FakeCmdRunner{}).Run,
		Clock:     scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
	})
}

// A remote-mode sync ensures a proxied CNAME for each desired managed hostname
// and records the records TM created in the DNS ledger.
func TestSync_EnsuresDNSForDesiredHostnames(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	dns := &fakeDNS{}
	dnsLedger := newFakeDNSLedger()

	svc := newSvcWithDNS(repo, routes, ingress, newFakeLedger(), dns, dnsLedger)
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)

	require.Equal(t, []string{"agent-manager.itsagitime.com"}, dns.ensured, "DNS ensured for the desired managed hostname")
	_, ok, _ := dnsLedger.Get(context.Background(), "agent-manager.itsagitime.com")
	require.True(t, ok, "TM-created record recorded in the DNS ledger")
}

// SMOKE / REGRESSION GUARD for the headline bug this plan fixes: a freshly
// "exposed" hostname must get BOTH an ingress rule AND a DNS CNAME, or the
// public URL is NXDOMAIN (ingress live, DNS missing — the exact failure the
// live test hit). This asserts the full handshake at the seam: the ingress push
// publishes the new hostname and EnsureRecord is called for it. A regression
// that drops DNS automation makes dns.ensured empty and fails here.
func TestSmoke_NewHostnameGetsIngressAndDNS(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("fresh-scenario", 21242, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	dns := &fakeDNS{}

	svc := newSvcWithDNS(repo, routes, ingress, newFakeLedger(), dns, newFakeDNSLedger())
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)

	host := "fresh-scenario.itsagitime.com"
	// Ingress published the new hostname...
	var pushedHost bool
	for _, r := range ingress.pushed {
		if r.Hostname == host {
			pushedHost = true
		}
	}
	require.True(t, pushedHost, "ingress push must include the freshly-exposed hostname")
	// ...AND DNS was ensured for it (the fix for the NXDOMAIN regression).
	require.Contains(t, dns.ensured, host, "a freshly-exposed hostname must get a DNS CNAME, not just ingress")
}

// Re-syncing an already-published hostname must not duplicate the DNS record:
// EnsureRecord is idempotent (a pre-existing record is left as-is, Created=false),
// so no second ledger row is created.
func TestSync_DNSIdempotentOnResync(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("fresh-scenario", 21242, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	host := "fresh-scenario.itsagitime.com"
	// Model the record already existing: EnsureRecord reports Created=false.
	dns := &fakeDNS{created: map[string]bool{host: false}}
	dnsLedger := newFakeDNSLedger()

	svc := newSvcWithDNS(repo, routes, ingress, newFakeLedger(), dns, dnsLedger)
	_, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.Contains(t, dns.ensured, host, "ensure is still called (idempotent check)")
	_, recorded, _ := dnsLedger.Get(context.Background(), host)
	require.False(t, recorded, "a pre-existing record is not recorded as TM-created")
}

// REGRESSION GUARD: an already-published hostname whose ingress rule already
// matches the desired manifest (so the reconcile reports NoChanges) must STILL
// get its DNS CNAME ensured. The original bug folded reconcileDNS inside
// applyAdditive, which Sync skips entirely on NoChanges — so any host whose
// ingress predated DNS automation (or any re-expose) stayed NXDOMAIN forever.
// This is the exact live failure: ingress live, DNS missing, NoChanges short-
// circuit. The fix runs an independent DNS pass on the remote path even when
// ingress is unchanged; a regression that restores the short-circuit makes
// dns.ensured empty and fails here.
func TestSync_EnsuresDNSWhenIngressAlreadyMatches(t *testing.T) {
	host := "agent-manager.itsagitime.com"
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	// Live ingress already contains the exact desired rule (+ catch-all), so the
	// reconcile yields NoChanges — nothing to add, nothing to prune.
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: host, Service: "http://localhost:21100"},
		{Service: "http_status:404"},
	}}
	dns := &fakeDNS{}
	dnsLedger := newFakeDNSLedger()

	svc := newSvcWithDNS(repo, routes, ingress, newFakeLedger(), dns, dnsLedger)
	result, err := svc.Sync(context.Background(), false, false)
	require.NoError(t, err)
	require.True(t, result.NoChanges, "ingress already matches ⇒ NoChanges (the short-circuit path)")

	require.Contains(t, dns.ensured, host, "DNS must still be ensured for an already-published host on NoChanges")
	require.Empty(t, ingress.pushed, "ingress is untouched when it already matches — only DNS reconciles")
	_, recorded, _ := dnsLedger.Get(context.Background(), host)
	require.True(t, recorded, "a newly-created CNAME on the NoChanges path is recorded in the DNS ledger")
}

// A dry-run never touches DNS.
func TestSync_DryRunSkipsDNS(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: []internalroutes.Route{route("agent-manager", 21100, true)}}
	ingress := &fakeIngress{live: []config.IngressRule{{Service: "http_status:404"}}}
	dns := &fakeDNS{}

	svc := newSvcWithDNS(repo, routes, ingress, newFakeLedger(), dns, newFakeDNSLedger())
	_, err := svc.Sync(context.Background(), true, false)
	require.NoError(t, err)
	require.Empty(t, dns.ensured, "dry-run must not create DNS records")
}

// Prune removes the CNAME ONLY for hostnames the DNS ledger attributes to TM.
func TestSync_PruneRemovesOnlyTMOwnedDNS(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	// No routes desired, so a previously-managed live hostname is orphaned.
	routes := &fakeRoutes{routes: nil}
	orphan := "old-scenario.itsagitime.com"
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: orphan, Service: "http://localhost:9"},
		{Service: "http_status:404"},
	}}
	// Ingress ledger marks the orphan MANAGED (TM authored its ingress) so prune
	// classifies it as orphaned rather than unmanaged drift.
	ingressLedger := newFakeLedger(config.LedgerEntry{Hostname: orphan, Owner: config.OwnerManaged, Scenario: "old-scenario"})
	dns := &fakeDNS{}
	dnsLedger := newFakeDNSLedger(orphan) // TM created the CNAME too.

	svc := newSvcWithDNS(repo, routes, ingress, ingressLedger, dns, dnsLedger)
	_, err := svc.Sync(context.Background(), false, true)
	require.NoError(t, err)

	require.Equal(t, []string{orphan}, dns.removed, "TM-owned orphan CNAME removed on prune")
	_, ok, _ := dnsLedger.Get(context.Background(), orphan)
	require.False(t, ok, "DNS ledger row cleared after removal")
}

// A prune for a hostname TM did NOT create in DNS never deletes the record.
func TestSync_PruneSkipsForeignDNS(t *testing.T) {
	repo := &fakeRepo{cfg: config.TunnelConfig{Mode: config.ModeRemote}}
	routes := &fakeRoutes{routes: nil}
	orphan := "external.itsagitime.com"
	ingress := &fakeIngress{live: []config.IngressRule{
		{Hostname: orphan, Service: "http://localhost:9"},
		{Service: "http_status:404"},
	}}
	ingressLedger := newFakeLedger(config.LedgerEntry{Hostname: orphan, Owner: config.OwnerManaged, Scenario: "external"})
	dns := &fakeDNS{}
	dnsLedger := newFakeDNSLedger() // empty: TM did NOT create this CNAME.

	svc := newSvcWithDNS(repo, routes, ingress, ingressLedger, dns, dnsLedger)
	_, err := svc.Sync(context.Background(), false, true)
	require.NoError(t, err)
	require.Empty(t, dns.removed, "never delete a DNS record TM did not create")
}
