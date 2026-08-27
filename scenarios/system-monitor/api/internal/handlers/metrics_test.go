package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/nodeclient"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	handlermocks "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetCurrentMetrics_Success(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{
			CPUUsage:       42.5,
			MemoryUsage:    65.3,
			TCPConnections: 120,
			Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithActive(true)

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.HandleGetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)

	// Verify JSON contains expected fields.
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["cpuUsage"] == nil && body["cpu_usage"] == nil {
		t.Error("expected cpuUsage or cpu_usage in response")
	}
}

func TestGetCurrentMetrics_Fresh(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{
			CPUUsage:    10.0,
			MemoryUsage: 20.0,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithFreshMetrics(&models.MetricsResponse{
			CPUUsage:    55.5,
			MemoryUsage: 77.7,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithActive(true)

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current?fresh=true", nil)
	w := httptest.NewRecorder()

	handler.HandleGetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetCurrentMetrics_Error(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithError(fmt.Errorf("collection failed"))
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.HandleGetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusInternalServerError)
}

// brokenResponseWriter simulates a ResponseWriter whose Write method fails.
type brokenResponseWriter struct {
	header http.Header
}

func newBrokenResponseWriter() *brokenResponseWriter {
	return &brokenResponseWriter{header: make(http.Header)}
}

func (w *brokenResponseWriter) Header() http.Header         { return w.header }
func (w *brokenResponseWriter) WriteHeader(_ int)           {}
func (w *brokenResponseWriter) Write(_ []byte) (int, error) { return 0, fmt.Errorf("broken pipe") }

func TestGetCurrentMetrics_WriteError_NoPanic(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{
			CPUUsage:    42.5,
			MemoryUsage: 65.3,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithActive(true)

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := newBrokenResponseWriter()

	// This should not panic even though Write returns an error.
	handler.HandleGetCurrentMetrics(w, req)
}

type machineRegistryServer struct {
	registryconnect.UnimplementedNodeRegistryServiceHandler
}

func (machineRegistryServer) ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	return connect.NewResponse(&registryv1.ListNodesResponse{Nodes: []*registryv1.Node{{
		Id: "mac-node", Name: "Mac mini", Os: "darwin", Arch: "amd64", Online: true,
		HeartbeatFresh: true, ChannelHeld: true, ProtocolCompatible: true, Dispatchable: true,
		Scopes: []string{"system-monitor:read", "system-monitor:write"},
		Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, LastSeenAt: timestamppb.Now(),
	}}}), nil
}

type remoteMetricsRelayServer struct {
	command string
	args    []string
}

func (s *remoteMetricsRelayServer) Call(_ context.Context, request *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	s.command = request.Msg.GetCommand()
	s.args = append([]string(nil), request.Msg.GetArgs()...)
	payload, err := protojson.Marshal(&metricspb.GetCurrentMetricsResponse{Metrics: &metricspb.MetricsResponse{CpuUsage: 12.5}})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&relayv1.RelayCallResponse{Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, Data: payload}), nil
}

func TestRemoteMetricsAndMachinesUseSharedNodeClient(t *testing.T) {
	registryPath, registryHandler := registryconnect.NewNodeRegistryServiceHandler(machineRegistryServer{})
	relayServer := &remoteMetricsRelayServer{}
	relayPath, relayHandler := relayconnect.NewRelayServiceHandler(relayServer)
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, registryPath):
			registryHandler.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, relayPath):
			relayHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer bridge.Close()

	client := nodeclient.New(nodeclient.Config{BridgeURL: bridge.URL})
	handler := NewMetricsHandler(&config.Config{}, handlermocks.NewMonitorQuerier(), slog.Default())
	handler.SetNodeClient(client)

	machines := httptest.NewRecorder()
	handler.HandleGetMachines(machines, httptest.NewRequest(http.MethodGet, "/api/v1/machines", nil))
	testutil.AssertStatusCode(t, machines.Code, http.StatusOK)
	if !strings.Contains(machines.Body.String(), "mac-node") {
		t.Fatalf("machines response = %s", machines.Body.String())
	}
	if !strings.Contains(machines.Body.String(), "Read and operate; destructive actions withheld") {
		t.Fatalf("machines response omitted the operator-facing grant: %s", machines.Body.String())
	}
	if !strings.Contains(machines.Body.String(), "system-monitor:read") || !strings.Contains(machines.Body.String(), "system-monitor:write") {
		t.Fatalf("machines response omitted concrete grant scopes: %s", machines.Body.String())
	}

	metrics := httptest.NewRecorder()
	handler.HandleGetCurrentMetrics(metrics, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current?node=mac-node&fresh=1", nil))
	testutil.AssertStatusCode(t, metrics.Code, http.StatusOK)
	if !strings.Contains(metrics.Body.String(), "12.5") {
		t.Fatalf("remote metrics response = %s", metrics.Body.String())
	}
	if got, want := relayServer.command, "system-monitor metrics current"; got != want {
		t.Fatalf("relay command = %q, want %q", got, want)
	}
	if got, want := relayServer.args, []string{"--json", "--fresh"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("relay args = %#v, want %#v", got, want)
	}
}

func TestGetMetricsTimeline_Success(t *testing.T) {
	gpu := 45.0
	mock := handlermocks.NewMonitorQuerier().
		WithTimelineResponse(&models.MetricsTimelineResponse{
			WindowSeconds:         120,
			SampleIntervalSeconds: 5,
			Samples: []models.MetricTimelineSample{
				{
					Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					CPUUsage:       42.5,
					MemoryUsage:    65.3,
					TCPConnections: 120,
					GPUUsage:       &gpu,
				},
				{
					Timestamp:      time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC),
					CPUUsage:       44.1,
					MemoryUsage:    66.0,
					TCPConnections: 118,
				},
			},
		})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline?window=120", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)

	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	samples, ok := body["samples"].([]interface{})
	if !ok {
		t.Fatal("expected samples array in response")
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
}

func TestGetMetricsTimeline_EmptySamples(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithTimelineResponse(&models.MetricsTimelineResponse{
			WindowSeconds:         120,
			SampleIntervalSeconds: 5,
			Samples:               []models.MetricTimelineSample{},
		})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetMetricsTimeline_CustomWindow(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithTimelineResponse(&models.MetricsTimelineResponse{
			WindowSeconds:         300,
			SampleIntervalSeconds: 10,
			Samples:               []models.MetricTimelineSample{},
		})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline?window=300&interval=10", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetMetricsTimeline_Error(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithError(fmt.Errorf("timeline retrieval failed"))
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline?window=120", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusInternalServerError)
}

func TestGetDiskDetailConnectReturnsCleanupManagerHandoff(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithDiskDetail(&models.DiskDetailResponse{
			Partitions: []models.DiskPartitionInfo{
				{
					Device:         "/dev/test",
					MountPoint:     "/",
					SizeBytes:      100,
					UsedBytes:      90,
					AvailableBytes: 10,
					UsePercent:     90,
				},
			},
			ActiveMount: "/",
			Depth:       2,
			Notes:       []string{"Suggested handoff: storage-manager cleanup plan --profile conservative"},
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	res, err := handler.GetDiskDetail(context.Background(), connect.NewRequest(&metricspb.GetDiskDetailRequest{}))
	if err != nil {
		t.Fatalf("GetDiskDetail returned error: %v", err)
	}
	if res.Msg.GetData().GetPartitions()[0].GetUsePercent() != 90 {
		t.Fatalf("disk pressure = %v, want 90", res.Msg.GetData().GetPartitions()[0].GetUsePercent())
	}
	notes := res.Msg.GetData().GetNotes()
	if len(notes) != 1 || notes[0] != "Suggested handoff: storage-manager cleanup plan --profile conservative" {
		t.Fatalf("notes = %v, want storage-manager handoff", notes)
	}
}

var _ MonitorQuerier = (*handlermocks.MonitorQuerier)(nil)
