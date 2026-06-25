package exposure_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"tunnel-manager/internal/exposure"
	internalroutes "tunnel-manager/internal/manifest"
	"tunnel-manager/internal/testutil/mocks"
)

// fakeAssigner records EnsureFixed/Release calls and reports whether it assigned.
type fakeAssigner struct {
	assignByTM bool
	ensured    []string
	released   []string
	ensureErr  error
}

func (f *fakeAssigner) EnsureFixed(_ context.Context, scenario string) (bool, error) {
	f.ensured = append(f.ensured, scenario)
	return f.assignByTM, f.ensureErr
}

func (f *fakeAssigner) Release(_ context.Context, scenario string) error {
	f.released = append(f.released, scenario)
	return nil
}

// memOwnership is an in-memory PortOwnership.
type memOwnership struct{ owned map[string]bool }

func newMemOwnership() *memOwnership { return &memOwnership{owned: map[string]bool{}} }

func (m *memOwnership) Record(_ context.Context, s string) error { m.owned[s] = true; return nil }
func (m *memOwnership) Owned(_ context.Context, s string) (bool, error) {
	return m.owned[s], nil
}
func (m *memOwnership) Clear(_ context.Context, s string) error { delete(m.owned, s); return nil }

func svcWithAssigner(t *testing.T, m *fakeManifest, repo *fakeRepo, ing *fakeIngress, ports *fakePorts, assigner exposure.PortAssigner, owner exposure.PortOwnership) (exposure.Service, *mocks.FakeClock) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	svc := exposure.NewService(repo, m, ing, &fakeRunner{}, ports, func() []string { return nil }, clk,
		exposure.WithPortAssigner(assigner, owner))
	return svc, clk
}

// Expose assigns a fixed port for a ranged scenario and records TM ownership.
func TestExpose_AssignsFixedPortForRangedScenario(t *testing.T) {
	m := newFakeManifest()
	repo := newFakeRepo()
	// The resolver only resolves the port AFTER assignment makes it fixed; model
	// that with a resolver that returns a port (assignment already reflected).
	ports := &fakePorts{port: 21242}
	assigner := &fakeAssigner{assignByTM: true}
	owner := newMemOwnership()

	svc, _ := svcWithAssigner(t, m, repo, &fakeIngress{}, ports, assigner, owner)
	_, _, err := svc.Expose(context.Background(), exposure.ExposeInput{Scenario: "ranged-scn"})
	require.NoError(t, err)
	require.Equal(t, []string{"ranged-scn"}, assigner.ensured, "expose ensured a fixed port")
	require.True(t, owner.owned["ranged-scn"], "TM-assigned port recorded as owned")
}

// Revoke releases a TM-assigned fixed port and clears ownership.
func TestRevoke_ReleasesTMAssignedPort(t *testing.T) {
	leasedRoute := internalroutes.Route{ID: "r1", Subdomain: "ranged-scn", Scenario: "ranged-scn", Domain: internalroutes.DefaultDomain, LocalPort: 21242, Tier: internalroutes.TierLeased, Enabled: true}
	m := newFakeManifest(leasedRoute)
	repo := newFakeRepo()
	assigner := &fakeAssigner{}
	owner := newMemOwnership()
	owner.owned["ranged-scn"] = true // TM assigned it earlier.

	svc, clk := svcWithAssigner(t, m, repo, &fakeIngress{}, &fakePorts{port: 21242}, assigner, owner)
	lease, _ := repo.Create(context.Background(), exposure.Lease{ID: "la", Scenario: "ranged-scn", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})

	_, err := svc.RevokeLease(context.Background(), lease.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"ranged-scn"}, assigner.released, "TM-assigned port released on revoke")
	require.False(t, owner.owned["ranged-scn"], "ownership cleared after release")
}

// Revoke never releases a port TM did not assign (hand-pinned fixed port).
func TestRevoke_NeverReleasesHandPinnedPort(t *testing.T) {
	leasedRoute := internalroutes.Route{ID: "r1", Subdomain: "pinned", Scenario: "pinned", Domain: internalroutes.DefaultDomain, LocalPort: 21242, Tier: internalroutes.TierLeased, Enabled: true}
	m := newFakeManifest(leasedRoute)
	repo := newFakeRepo()
	assigner := &fakeAssigner{}
	owner := newMemOwnership() // empty: TM did not assign this scenario's port.

	svc, clk := svcWithAssigner(t, m, repo, &fakeIngress{}, &fakePorts{port: 21242}, assigner, owner)
	lease, _ := repo.Create(context.Background(), exposure.Lease{ID: "lb", Scenario: "pinned", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})

	_, err := svc.RevokeLease(context.Background(), lease.ID)
	require.NoError(t, err)
	require.Empty(t, assigner.released, "a hand-pinned fixed port is never reverted")
}
