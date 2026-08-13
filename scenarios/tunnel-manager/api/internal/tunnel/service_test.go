package tunnel_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/testutil/mocks"
	"tunnel-manager/internal/tunnel"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

// fakeRepo is a minimal in-memory tunnel.MetricsRepository for service tests
// that don't want the sqlite round-trip. It records the last sample passed to
// Store and serves scripted Latest/Query results.
type fakeRepo struct {
	stored    tunnel.MetricsSample
	latestOut tunnel.MetricsSample
	latestErr error
	queryOut  []tunnel.MetricsSample
	queryFrom time.Time
	queryTo   time.Time
}

func (f *fakeRepo) Store(_ context.Context, s tunnel.MetricsSample) (tunnel.MetricsSample, error) {
	f.stored = s
	if s.ID == "" {
		s.ID = "generated-id"
	}
	return s, nil
}

func (f *fakeRepo) Query(_ context.Context, from, to time.Time) ([]tunnel.MetricsSample, error) {
	f.queryFrom, f.queryTo = from, to
	return f.queryOut, nil
}

func (f *fakeRepo) Latest(_ context.Context) (tunnel.MetricsSample, error) {
	return f.latestOut, f.latestErr
}

func newService(t *testing.T, repo tunnel.MetricsRepository, runner *mocks.FakeCmdRunner, doer *mocks.FakeDoer) tunnel.Service {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return tunnel.NewService(repo, runner.Run, doer, clk, "")
}

// --- GetStatus composite health ---------------------------------------------

func TestGetStatus_Healthy(t *testing.T) {
	repo := &fakeRepo{latestErr: tunnel.ErrNoMetrics{}}
	runner := &mocks.FakeCmdRunner{Out: []byte("active\n")}
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, nil) // /ready -> ok
	svc := newService(t, repo, runner, doer)

	status, latest, err := svc.GetStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, tunnel.StatusHealthy, status.Status)
	require.Equal(t, "active", status.Systemd)
	require.Equal(t, "ok", status.Ready)
	require.Equal(t, 100, status.Score)
	require.Empty(t, status.Message)
	require.Nil(t, latest, "no metrics yet -> nil latest, ErrNoMetrics swallowed")
}

func TestGetStatus_DegradedSystemdInactive(t *testing.T) {
	repo := &fakeRepo{latestErr: tunnel.ErrNoMetrics{}}
	runner := &mocks.FakeCmdRunner{Out: []byte("inactive\n")}
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, nil) // /ready -> ok, so only the -50 systemd penalty applies
	svc := newService(t, repo, runner, doer)

	status, _, err := svc.GetStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, tunnel.StatusDegraded, status.Status)
	require.Equal(t, "inactive", status.Systemd)
	require.Equal(t, 50, status.Score)
}

func TestGetStatus_UnhealthyBothChecksFail(t *testing.T) {
	sample := tunnel.MetricsSample{ID: "m1", HAConnections: 4}
	repo := &fakeRepo{latestOut: sample}
	runner := &mocks.FakeCmdRunner{Out: []byte("inactive\n")}
	doer := &mocks.FakeDoer{}
	doer.AddResponse(503, nil) // /ready -> http_503 (not ok)
	svc := newService(t, repo, runner, doer)

	status, latest, err := svc.GetStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, tunnel.StatusUnhealthy, status.Status)
	require.Equal(t, "http_503", status.Ready)
	require.Equal(t, 20, status.Score)
	require.NotEmpty(t, status.Message)
	require.NotNil(t, latest)
	require.Equal(t, "m1", latest.ID)
}

// --- Scrape parse + persist -------------------------------------------------

func TestScrape_ParsesAndPersists(t *testing.T) {
	repo := &fakeRepo{}
	runner := &mocks.FakeCmdRunner{}
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`# HELP cloudflared_tunnel_ha_connections HA connections
cloudflared_tunnel_ha_connections 4
cloudflared_tunnel_request_errors_total 7
cloudflared_tunnel_active_streams 3
quic_client_smoothed_rtt 12.5
`))
	svc := newService(t, repo, runner, doer)

	got, err := svc.Scrape(context.Background())
	require.NoError(t, err)
	require.Equal(t, 4, got.HAConnections)
	require.InDelta(t, 7.0, got.RequestErrors, 1e-9)
	require.Equal(t, 3, got.ActiveStreams)
	require.InDelta(t, 12.5, got.SmoothedRTTMS, 1e-9)
	require.False(t, got.ScrapedAt.IsZero(), "clock-stamped before persist")
	require.Equal(t, 4, repo.stored.HAConnections, "parsed sample handed to repo")

	require.Len(t, doer.Requests, 1)
	require.Equal(t, "http://127.0.0.1:20241/metrics", doer.Requests[0].URL.String())
}

// --- ListMetrics window passthrough -----------------------------------------

func TestListMetrics_PassesWindow(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(t, repo, &mocks.FakeCmdRunner{}, &mocks.FakeDoer{})
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)

	_, err := svc.ListMetrics(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, from, repo.queryFrom)
	require.Equal(t, to, repo.queryTo)
}
