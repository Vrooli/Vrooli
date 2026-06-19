package probes_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/probes"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// fakeRoutes is a minimal probes.RoutesReader returning a fixed manifest.
type fakeRoutes struct {
	routes []internalroutes.Route
	err    error
}

func (f *fakeRoutes) List(_ context.Context, _ internalroutes.Tier) ([]internalroutes.Route, error) {
	return f.routes, f.err
}

// fakeRepo is an in-memory probes.Repository recording every persisted
// result and serving a scripted LatestPerRoute for classification tests.
type fakeRepo struct {
	persisted []probes.ProbeResult
	listOut   []probes.ProbeResult
	latestOut []probes.LatestPair
}

func (f *fakeRepo) Persist(_ context.Context, r probes.ProbeResult) (probes.ProbeResult, error) {
	if r.ID == "" {
		r.ID = "generated-id"
	}
	f.persisted = append(f.persisted, r)
	return r, nil
}

func (f *fakeRepo) List(_ context.Context, _ string, _ int) ([]probes.ProbeResult, error) {
	return f.listOut, nil
}

func (f *fakeRepo) LatestPerRoute(_ context.Context) ([]probes.LatestPair, error) {
	return f.latestOut, nil
}

func sampleRoute(sub string, port int) internalroutes.Route {
	return internalroutes.Route{
		Subdomain: sub, Scenario: sub, Domain: "itsagitime.com",
		LocalPort: port, Tier: internalroutes.TierLeased, Enabled: true, HealthPath: "/health",
	}
}

// --- RunProbes: per-kind statuses + persistence -----------------------------

func TestRunProbes_StatusMappingAndPersistence(t *testing.T) {
	routes := &fakeRoutes{routes: []internalroutes.Route{sampleRoute("agent-manager", 21100)}}
	repo := &fakeRepo{}
	doer := &mocks.FakeDoer{}
	// Probes run concurrently per route but internal precedes external in
	// the goroutine, so call 1 = internal (200 → up), call 2 = external
	// (500 → down).
	doer.AddResponse(200, []byte("ok"))
	doer.AddResponse(500, []byte("boom"))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))

	svc := probes.NewService(routes, repo, doer, clk)
	results, err := svc.RunProbes(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Len(t, repo.persisted, 2, "both probe results persisted")

	byKind := map[probes.ProbeKind]probes.ProbeResult{}
	for _, r := range results {
		byKind[r.Kind] = r
	}
	require.Equal(t, probes.ProbeStatusUp, byKind[probes.ProbeKindInternal].Status)
	require.Equal(t, probes.ProbeStatusDown, byKind[probes.ProbeKindExternal].Status)
	require.Equal(t, 500, byKind[probes.ProbeKindExternal].StatusCode)

	// Internal probes the local port; external probes the public URL.
	require.Equal(t, "http://localhost:21100/health", doer.Requests[0].URL.String())
	require.Equal(t, "https://agent-manager.itsagitime.com/health", doer.Requests[1].URL.String())
}

func TestRunProbes_TimeoutAndSkipsDisabled(t *testing.T) {
	enabled := sampleRoute("web-console", 3000)
	disabled := sampleRoute("disabled-one", 4000)
	disabled.Enabled = false
	routes := &fakeRoutes{routes: []internalroutes.Route{enabled, disabled}}
	repo := &fakeRepo{}
	doer := &mocks.FakeDoer{}
	clk := mocks.NewFakeClock(time.Time{})

	svc := probes.NewService(routes, repo, doer, clk)

	// A cancelled context makes both probes fail; ctx.Err() != nil maps to
	// timeout rather than down.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err := svc.RunProbes(ctx)
	require.NoError(t, err)
	require.Len(t, results, 2, "only the enabled route was probed (internal+external)")
	for _, r := range results {
		require.Equal(t, "web-console", r.Subdomain)
		require.Equal(t, probes.ProbeStatusTimeout, r.Status)
	}
}

// --- Classify: every FailureClass branch ------------------------------------

func up() *probes.ProbeResult   { return &probes.ProbeResult{Status: probes.ProbeStatusUp} }
func down() *probes.ProbeResult { return &probes.ProbeResult{Status: probes.ProbeStatusDown} }

func TestClassify_AllBranches(t *testing.T) {
	repo := &fakeRepo{latestOut: []probes.LatestPair{
		{Subdomain: "healthy", Internal: up(), External: up()},
		{Subdomain: "tunnel", Internal: up(), External: down()},
		{Subdomain: "drift", Internal: down(), External: up()},
		{Subdomain: "scenario", Internal: down(), External: down()},
	}}
	svc := probes.NewService(&fakeRoutes{}, repo, &mocks.FakeDoer{}, mocks.NewFakeClock(time.Time{}))

	got, err := svc.Classify(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 4)

	bySub := map[string]probes.RouteClassification{}
	for _, c := range got {
		bySub[c.Subdomain] = c
	}
	require.Equal(t, probes.FailureClassHealthy, bySub["healthy"].Classification)
	require.Equal(t, probes.FailureClassTunnelDown, bySub["tunnel"].Classification)
	require.Equal(t, probes.FailureClassConfigDrift, bySub["drift"].Classification)
	require.Equal(t, probes.FailureClassScenarioDown, bySub["scenario"].Classification)
	require.NotEmpty(t, bySub["tunnel"].Assessment)
	require.Equal(t, probes.ProbeStatusUp, bySub["tunnel"].Internal)
	require.Equal(t, probes.ProbeStatusDown, bySub["tunnel"].External)
}
