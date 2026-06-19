package tunnel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"tunnel-manager/handlers/tunnel"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	tunnelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel"
	tunnelconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel/tunnel_v1connect"

	internaltunnel "tunnel-manager/internal/tunnel"
)

// fakeService implements internaltunnel.Service for handler tests.
type fakeService struct {
	statusOut internaltunnel.TunnelStatus
	latestOut *internaltunnel.MetricsSample
	statusErr error

	listOut []internaltunnel.MetricsSample
	listErr error

	scrapeOut internaltunnel.MetricsSample
	scrapeErr error
}

func (f *fakeService) GetStatus(_ context.Context) (internaltunnel.TunnelStatus, *internaltunnel.MetricsSample, error) {
	return f.statusOut, f.latestOut, f.statusErr
}

func (f *fakeService) ListMetrics(_ context.Context, _, _ time.Time) ([]internaltunnel.MetricsSample, error) {
	return f.listOut, f.listErr
}

func (f *fakeService) Scrape(_ context.Context) (internaltunnel.MetricsSample, error) {
	return f.scrapeOut, f.scrapeErr
}

func newClient(t *testing.T, svc internaltunnel.Service) tunnelconnect.TunnelServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := tunnelconnect.NewTunnelServiceHandler(tunnel.NewConnectHandler(tunnel.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return tunnelconnect.NewTunnelServiceClient(server.Client(), server.URL)
}

func TestHandlerGetStatusMapsSnapshot(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{
		statusOut: internaltunnel.TunnelStatus{
			Status: internaltunnel.StatusDegraded, Systemd: "inactive", Ready: "ok",
			ReadyLatencyMS: 12, Score: 50, Message: "tunnel is experiencing issues", CheckedAt: now,
		},
		latestOut: &internaltunnel.MetricsSample{ID: "m1", HAConnections: 4, SmoothedRTTMS: 9.5, ScrapedAt: now},
	})

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&tunnelv1.GetStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, "degraded", resp.Msg.Status.Status)
	require.Equal(t, "inactive", resp.Msg.Status.Systemd)
	require.Equal(t, int32(50), resp.Msg.Status.Score)
	require.Equal(t, int32(12), resp.Msg.Status.ReadyLatencyMs)
	require.NotNil(t, resp.Msg.LatestMetrics)
	require.Equal(t, "m1", resp.Msg.LatestMetrics.Id)
	require.Equal(t, int32(4), resp.Msg.LatestMetrics.HaConnections)
}

func TestHandlerGetStatusNilLatest(t *testing.T) {
	client := newClient(t, &fakeService{
		statusOut: internaltunnel.TunnelStatus{Status: internaltunnel.StatusHealthy, Systemd: "active", Ready: "ok", Score: 100, CheckedAt: time.Now()},
		latestOut: nil,
	})
	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&tunnelv1.GetStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, "healthy", resp.Msg.Status.Status)
	require.Nil(t, resp.Msg.LatestMetrics)
}

func TestHandlerScrapeMapsSample(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{
		scrapeOut: internaltunnel.MetricsSample{ID: "s1", HAConnections: 6, RequestErrors: 3, ActiveStreams: 2, SmoothedRTTMS: 14.25, ScrapedAt: now},
	})
	resp, err := client.Scrape(context.Background(), connect.NewRequest(&tunnelv1.ScrapeRequest{}))
	require.NoError(t, err)
	require.Equal(t, "s1", resp.Msg.Sample.Id)
	require.Equal(t, int32(6), resp.Msg.Sample.HaConnections)
	require.InDelta(t, 14.25, resp.Msg.Sample.SmoothedRttMs, 1e-9)
}

func TestHandlerScrapeInternalError(t *testing.T) {
	client := newClient(t, &fakeService{scrapeErr: errors.New("metrics endpoint returned 503")})
	_, err := client.Scrape(context.Background(), connect.NewRequest(&tunnelv1.ScrapeRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
