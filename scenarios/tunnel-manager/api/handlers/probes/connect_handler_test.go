package probes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"tunnel-manager/handlers/probes"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	probesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes"
	probesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes/probes_v1connect"

	internalprobes "tunnel-manager/internal/probes"
)

// fakeService implements internalprobes.Service for handler tests.
type fakeService struct {
	runOut      []internalprobes.ProbeResult
	runErr      error
	listOut     []internalprobes.ProbeResult
	classifyOut []internalprobes.RouteClassification
}

func (f *fakeService) RunProbes(_ context.Context) ([]internalprobes.ProbeResult, error) {
	return f.runOut, f.runErr
}

func (f *fakeService) ListProbes(_ context.Context, _ string, _ int) ([]internalprobes.ProbeResult, error) {
	return f.listOut, nil
}

func (f *fakeService) Classify(_ context.Context) ([]internalprobes.RouteClassification, error) {
	return f.classifyOut, nil
}

func newClient(t *testing.T, svc internalprobes.Service) probesconnect.ProbesServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := probesconnect.NewProbesServiceHandler(probes.NewConnectHandler(probes.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return probesconnect.NewProbesServiceClient(server.Client(), server.URL)
}

func TestHandlerRunProbesMapsEnums(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{runOut: []internalprobes.ProbeResult{
		{ID: "p1", Subdomain: "agent-manager", Kind: internalprobes.ProbeKindInternal, Status: internalprobes.ProbeStatusUp, LatencyMS: 11, StatusCode: 200, CreatedAt: now},
		{ID: "p2", Subdomain: "agent-manager", Kind: internalprobes.ProbeKindExternal, Status: internalprobes.ProbeStatusDown, StatusCode: 502, ErrorMsg: "bad gateway", CreatedAt: now},
	}})

	resp, err := client.RunProbes(context.Background(), connect.NewRequest(&probesv1.RunProbesRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 2)
	require.Equal(t, probesv1.ProbeKind_PROBE_KIND_INTERNAL, resp.Msg.Results[0].Kind)
	require.Equal(t, probesv1.ProbeStatus_PROBE_STATUS_UP, resp.Msg.Results[0].Status)
	require.EqualValues(t, 200, resp.Msg.Results[0].StatusCode)
	require.Equal(t, probesv1.ProbeKind_PROBE_KIND_EXTERNAL, resp.Msg.Results[1].Kind)
	require.Equal(t, probesv1.ProbeStatus_PROBE_STATUS_DOWN, resp.Msg.Results[1].Status)
	require.Equal(t, "bad gateway", resp.Msg.Results[1].ErrorMsg)
}

func TestHandlerClassifyMapsFailureClass(t *testing.T) {
	client := newClient(t, &fakeService{classifyOut: []internalprobes.RouteClassification{
		{Subdomain: "agent-manager", Classification: internalprobes.FailureClassTunnelDown, Internal: internalprobes.ProbeStatusUp, External: internalprobes.ProbeStatusDown, Assessment: "local up, tunnel down"},
	}})

	resp, err := client.Classify(context.Background(), connect.NewRequest(&probesv1.ClassifyRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Classifications, 1)
	c := resp.Msg.Classifications[0]
	require.Equal(t, probesv1.FailureClass_FAILURE_CLASS_TUNNEL_DOWN, c.Classification)
	require.Equal(t, probesv1.ProbeStatus_PROBE_STATUS_UP, c.Internal)
	require.Equal(t, probesv1.ProbeStatus_PROBE_STATUS_DOWN, c.External)
	require.Equal(t, "local up, tunnel down", c.Assessment)
}

func TestHandlerRunProbesInternalError(t *testing.T) {
	client := newClient(t, &fakeService{runErr: errors.New("manifest read failed")})
	_, err := client.RunProbes(context.Background(), connect.NewRequest(&probesv1.RunProbesRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
