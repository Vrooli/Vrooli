package exposure_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"tunnel-manager/internal/exposure"
	internalroutes "tunnel-manager/internal/routes"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

// fakeManifest is an in-memory routes manifest satisfying exposure.Manifest.
type fakeManifest struct {
	routes    map[string]internalroutes.Route
	nextID    int
	createErr error
}

func newFakeManifest(seed ...internalroutes.Route) *fakeManifest {
	m := &fakeManifest{routes: map[string]internalroutes.Route{}}
	for _, r := range seed {
		m.routes[r.ID] = r
	}
	return m
}

func (m *fakeManifest) List(_ context.Context, tier internalroutes.Tier) ([]internalroutes.Route, error) {
	var out []internalroutes.Route
	for _, r := range m.routes {
		if tier == "" || r.Tier == tier {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *fakeManifest) Create(_ context.Context, in internalroutes.CreateInput) (internalroutes.Route, error) {
	if m.createErr != nil {
		return internalroutes.Route{}, m.createErr
	}
	m.nextID++
	id := fmt.Sprintf("gen-route-%d", m.nextID)
	tier := in.Tier
	if tier == "" {
		tier = internalroutes.TierLeased
	}
	r := internalroutes.Route{
		ID: id, Subdomain: in.Subdomain, Scenario: in.Scenario,
		Domain: internalroutes.DefaultDomain, LocalPort: in.LocalPort,
		Tier: tier, Enabled: true, HealthPath: internalroutes.DefaultHealthPath,
	}
	m.routes[id] = r
	return r, nil
}

func (m *fakeManifest) Update(_ context.Context, id string, in internalroutes.UpdateInput) (internalroutes.Route, error) {
	r, ok := m.routes[id]
	if !ok {
		return internalroutes.Route{}, internalroutes.ErrRouteNotFound{ID: id}
	}
	if in.Tier != "" {
		r.Tier = in.Tier
	}
	if in.Enabled != nil {
		r.Enabled = *in.Enabled
	}
	m.routes[id] = r
	return r, nil
}

func (m *fakeManifest) Delete(_ context.Context, id string) (bool, error) {
	if _, ok := m.routes[id]; !ok {
		return false, nil
	}
	delete(m.routes, id)
	return true, nil
}

// fakeRepo is an in-memory leases repository.
type fakeRepo struct {
	leases map[string]exposure.Lease
	nextID int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{leases: map[string]exposure.Lease{}} }

func (f *fakeRepo) Create(_ context.Context, l exposure.Lease) (exposure.Lease, error) {
	f.nextID++
	if l.ID == "" {
		l.ID = fmt.Sprintf("gen-lease-%d", f.nextID)
	}
	if l.Status == "" {
		l.Status = exposure.LeaseActive
	}
	f.leases[l.ID] = l
	return l, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (exposure.Lease, error) {
	l, ok := f.leases[id]
	if !ok {
		return exposure.Lease{}, exposure.ErrLeaseNotFound{ID: id}
	}
	return l, nil
}

func (f *fakeRepo) ActiveForScenario(_ context.Context, scenario string) (exposure.Lease, error) {
	for _, l := range f.leases {
		if l.Scenario == scenario && l.Status == exposure.LeaseActive {
			return l, nil
		}
	}
	return exposure.Lease{}, exposure.ErrLeaseNotFound{ID: scenario}
}

func (f *fakeRepo) Update(_ context.Context, l exposure.Lease) (exposure.Lease, error) {
	if _, ok := f.leases[l.ID]; !ok {
		return exposure.Lease{}, exposure.ErrLeaseNotFound{ID: l.ID}
	}
	f.leases[l.ID] = l
	return l, nil
}

func (f *fakeRepo) List(_ context.Context, status exposure.LeaseStatus) ([]exposure.Lease, error) {
	var out []exposure.Lease
	for _, l := range f.leases {
		if status == "" || l.Status == status {
			out = append(out, l)
		}
	}
	return out, nil
}

// seam fakes
type fakeIngress struct {
	calls int
	err   error
}

func (f *fakeIngress) Reconcile(context.Context) error { f.calls++; return f.err }

type fakeRunner struct {
	started []string
	err     error
}

func (f *fakeRunner) EnsureRunning(_ context.Context, s string) error {
	f.started = append(f.started, s)
	return f.err
}

type fakePorts struct {
	port int
	err  error
}

func (f *fakePorts) UIPort(context.Context, string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.port, nil
}

func newSvc(t *testing.T, m *fakeManifest, repo *fakeRepo, ing *fakeIngress, run *fakeRunner, ports *fakePorts, core []string) (exposure.Service, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	svc := exposure.NewService(repo, m, ing, run, ports, func() []string { return core }, clk)
	return svc, clk
}

func TestExpose_CreatesRouteRunsAndLeases(t *testing.T) {
	m := newFakeManifest()
	repo := newFakeRepo()
	ing := &fakeIngress{}
	run := &fakeRunner{}
	svc, clk := newSvc(t, m, repo, ing, run, &fakePorts{port: 21233}, nil)

	lease, url, localPort, portAssigned, err := svc.Expose(context.Background(), exposure.ExposeInput{Scenario: "web-console", RequestedBy: "tester"})
	require.NoError(t, err)
	require.Equal(t, "https://web-console.itsagitime.com", url)
	require.Equal(t, 21233, localPort)
	require.False(t, portAssigned, "pre-fixed scenario should not report assignment on expose")
	require.Equal(t, exposure.LeaseActive, lease.Status)
	require.Equal(t, clk.Now().UTC().Add(exposure.DefaultTTL), lease.ExpiresAt)
	require.Equal(t, []string{"web-console"}, run.started, "ensured running")
	require.Equal(t, 1, ing.calls, "ingress reconciled")
	require.Len(t, m.routes, 1, "one leased route created")
}

func TestExpose_IdempotentExtendsExistingLease(t *testing.T) {
	m := newFakeManifest()
	repo := newFakeRepo()
	svc, _ := newSvc(t, m, repo, &fakeIngress{}, &fakeRunner{}, &fakePorts{port: 3000}, nil)

	first, _, _, _, err := svc.Expose(context.Background(), exposure.ExposeInput{Scenario: "svc"})
	require.NoError(t, err)
	second, _, _, _, err := svc.Expose(context.Background(), exposure.ExposeInput{Scenario: "svc"})
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID, "same lease reused")
	require.Equal(t, 1, second.ExtendedCount, "extended, not duplicated")
	all, _ := repo.List(context.Background(), "")
	require.Len(t, all, 1, "no duplicate lease")
	require.Len(t, m.routes, 1, "no duplicate route")
}

func TestExpose_PortUnresolvedFailsBeforeLease(t *testing.T) {
	m := newFakeManifest()
	repo := newFakeRepo()
	run := &fakeRunner{}
	svc, _ := newSvc(t, m, repo, &fakeIngress{}, run, &fakePorts{err: exposure.ErrPortUnresolved{Scenario: "x", Reason: "ranged"}}, nil)

	_, _, _, _, err := svc.Expose(context.Background(), exposure.ExposeInput{Scenario: "x"})
	var portErr exposure.ErrPortUnresolved
	require.ErrorAs(t, err, &portErr)
	require.Empty(t, run.started, "did not start before route resolved")
	all, _ := repo.List(context.Background(), "")
	require.Empty(t, all, "no orphan lease")
}

func TestRevoke_NonCoreRetracts_CoreStays(t *testing.T) {
	leasedRoute := internalroutes.Route{ID: "r1", Subdomain: "leasedsvc", Scenario: "leasedsvc", Domain: internalroutes.DefaultDomain, LocalPort: 1000, Tier: internalroutes.TierLeased, Enabled: true}
	coreRoute := internalroutes.Route{ID: "r2", Subdomain: "coresvc", Scenario: "coresvc", Domain: internalroutes.DefaultDomain, LocalPort: 2000, Tier: internalroutes.TierCore, Enabled: true}
	m := newFakeManifest(leasedRoute, coreRoute)
	repo := newFakeRepo()
	ing := &fakeIngress{}
	svc, clk := newSvc(t, m, repo, ing, &fakeRunner{}, &fakePorts{}, []string{"coresvc"})

	leased, _ := repo.Create(context.Background(), exposure.Lease{ID: "la", Scenario: "leasedsvc", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})
	core, _ := repo.Create(context.Background(), exposure.Lease{ID: "lb", Scenario: "coresvc", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})

	retracted, err := svc.RevokeLease(context.Background(), leased.ID)
	require.NoError(t, err)
	require.True(t, retracted, "non-core lease retracts ingress")
	_, ok := m.routes["r1"]
	require.False(t, ok, "leased route deleted")

	retracted, err = svc.RevokeLease(context.Background(), core.ID)
	require.NoError(t, err)
	require.False(t, retracted, "core scenario stays exposed")
	_, ok = m.routes["r2"]
	require.True(t, ok, "core route preserved")
}

func TestUnexpose_RevokesActiveLeaseByScenario(t *testing.T) {
	leasedRoute := internalroutes.Route{ID: "r1", Subdomain: "leasedsvc", Scenario: "leasedsvc", Domain: internalroutes.DefaultDomain, LocalPort: 1000, Tier: internalroutes.TierLeased, Enabled: true}
	m := newFakeManifest(leasedRoute)
	repo := newFakeRepo()
	ing := &fakeIngress{}
	svc, clk := newSvc(t, m, repo, ing, &fakeRunner{}, &fakePorts{}, nil)

	lease, _ := repo.Create(context.Background(), exposure.Lease{ID: "la", Scenario: "leasedsvc", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})

	retracted, leaseID, err := svc.Unexpose(context.Background(), "leasedsvc")
	require.NoError(t, err)
	require.True(t, retracted, "non-core scenario retracts ingress")
	require.Equal(t, lease.ID, leaseID, "returns the revoked lease id")
	_, ok := m.routes["r1"]
	require.False(t, ok, "leased route deleted")
}

func TestUnexpose_NoActiveLeaseErrors(t *testing.T) {
	m := newFakeManifest()
	repo := newFakeRepo()
	svc, _ := newSvc(t, m, repo, &fakeIngress{}, &fakeRunner{}, &fakePorts{}, nil)

	_, _, err := svc.Unexpose(context.Background(), "never-exposed")
	require.Error(t, err, "no active lease for the scenario surfaces an error")
}

func TestReconcile_EnsuresCoreAndReapsExpired(t *testing.T) {
	// An expired leased route + lease, and a missing core scenario.
	leasedRoute := internalroutes.Route{ID: "r1", Subdomain: "old", Scenario: "old", Domain: internalroutes.DefaultDomain, LocalPort: 1000, Tier: internalroutes.TierLeased, Enabled: true}
	m := newFakeManifest(leasedRoute)
	repo := newFakeRepo()
	ing := &fakeIngress{}
	svc, clk := newSvc(t, m, repo, ing, &fakeRunner{}, &fakePorts{port: 21238}, []string{"agent-manager"})

	_, _ = repo.Create(context.Background(), exposure.Lease{ID: "expired", Scenario: "old", ExpiresAt: clk.Now().Add(-time.Hour), Status: exposure.LeaseActive})

	coreEnsured, reaped, err := svc.Reconcile(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, coreEnsured, "agent-manager core route created")
	require.Equal(t, 1, reaped, "expired lease reaped")

	// Core route exists, expired leased route retracted.
	var haveCore bool
	for _, r := range m.routes {
		if r.Scenario == "agent-manager" && r.Tier == internalroutes.TierCore {
			haveCore = true
		}
	}
	require.True(t, haveCore)
	_, stillThere := m.routes["r1"]
	require.False(t, stillThere, "expired non-core route removed")
	require.GreaterOrEqual(t, ing.calls, 1, "ingress reconciled after changes")
}

func TestIsExposed_CoreAlways_LeasedNeedsActive(t *testing.T) {
	coreRoute := internalroutes.Route{ID: "r2", Subdomain: "coresvc", Scenario: "coresvc", Domain: internalroutes.DefaultDomain, LocalPort: 2000, Tier: internalroutes.TierCore, Enabled: true}
	leasedRoute := internalroutes.Route{ID: "r1", Subdomain: "leasedsvc", Scenario: "leasedsvc", Domain: internalroutes.DefaultDomain, LocalPort: 1000, Tier: internalroutes.TierLeased, Enabled: true}
	m := newFakeManifest(coreRoute, leasedRoute)
	repo := newFakeRepo()
	svc, clk := newSvc(t, m, repo, &fakeIngress{}, &fakeRunner{}, &fakePorts{}, []string{"coresvc"})

	exposed, url, err := svc.IsExposed(context.Background(), "coresvc")
	require.NoError(t, err)
	require.True(t, exposed)
	require.Equal(t, "https://coresvc.itsagitime.com", url)

	exposed, _, err = svc.IsExposed(context.Background(), "leasedsvc")
	require.NoError(t, err)
	require.False(t, exposed, "leased route without active lease is not exposed")

	_, _ = repo.Create(context.Background(), exposure.Lease{ID: "la", Scenario: "leasedsvc", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})
	exposed, _, err = svc.IsExposed(context.Background(), "leasedsvc")
	require.NoError(t, err)
	require.True(t, exposed, "leased route with active lease is exposed")
}

func TestExtendLease_NotFound(t *testing.T) {
	svc, _ := newSvc(t, newFakeManifest(), newFakeRepo(), &fakeIngress{}, &fakeRunner{}, &fakePorts{}, nil)
	_, err := svc.ExtendLease(context.Background(), "ghost", time.Hour)
	var nf exposure.ErrLeaseNotFound
	require.ErrorAs(t, err, &nf)
}

func TestExpose_IngressFailurePropagates(t *testing.T) {
	svc, _ := newSvc(t, newFakeManifest(), newFakeRepo(), &fakeIngress{err: errors.New("cf down")}, &fakeRunner{}, &fakePorts{port: 1}, nil)
	_, _, _, _, err := svc.Expose(context.Background(), exposure.ExposeInput{Scenario: "x"})
	require.Error(t, err)
}
