package sources

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	agentapiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	agentdomainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeMetrics struct {
	current  *metricsv1.GetCurrentMetricsResponse
	detailed *metricsv1.GetDetailedMetricsResponse
	err      error
}

func (f fakeMetrics) GetCurrentMetrics(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.current), nil
}

func (f fakeMetrics) GetDetailedMetrics(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.detailed), nil
}

type fakeRuns struct {
	statuses []agentdomainv1.RunStatus
	calls    []agentdomainv1.RunStatus
	err      error
}

func (f *fakeRuns) ListRuns(_ context.Context, req *connect.Request[agentapiv1.ListRunsRequest]) (*connect.Response[agentapiv1.ListRunsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	status := req.Msg.GetStatus()
	f.calls = append(f.calls, status)
	var runs []*agentdomainv1.Run
	for _, candidate := range f.statuses {
		if candidate == status {
			runs = append(runs, &agentdomainv1.Run{Status: candidate})
		}
	}
	return connect.NewResponse(&agentapiv1.ListRunsResponse{Runs: runs}), nil
}

func TestReadUsesOwnerFactsAndCountsOnlyActiveRuns(t *testing.T) {
	now := time.Date(2026, 11, 10, 17, 0, 0, 0, time.UTC)
	runs := &fakeRuns{statuses: []agentdomainv1.RunStatus{
		agentdomainv1.RunStatus_RUN_STATUS_PENDING,
		agentdomainv1.RunStatus_RUN_STATUS_STARTING,
		agentdomainv1.RunStatus_RUN_STATUS_RUNNING,
		agentdomainv1.RunStatus_RUN_STATUS_COMPLETE,
		agentdomainv1.RunStatus_RUN_STATUS_FAILED,
	}}
	metricTime := timestamppb.New(now.Add(-2 * time.Second))
	sources := &Sources{
		resolve: func(_ context.Context, scenario string) (string, error) {
			return "http://127.0.0.1/" + scenario + "/", nil
		},
		client: &http.Client{},
		now:    func() time.Time { return now },
		newMetricsReader: func(_ *http.Client, base string) metricsReader {
			require.Equal(t, "http://127.0.0.1/system-monitor", base)
			return fakeMetrics{
				current: &metricsv1.GetCurrentMetricsResponse{Metrics: &metricsv1.MetricsResponse{
					Timestamp: metricTime,
					Cpu:       &metricsv1.MetricValue{State: &metricsv1.MetricValue_Measured{Measured: 61}},
					Gpu:       &metricsv1.MetricValue{State: &metricsv1.MetricValue_Measured{Measured: 49}},
				}},
				detailed: &metricsv1.GetDetailedMetricsResponse{Metrics: &metricsv1.DetailedMetrics{MemoryDetails: &metricsv1.MemoryMetrics{
					MemoryPressure: &metricsv1.MemoryPressureReading{Level: "elevated", ObservedAt: timestamppb.New(now.Add(-time.Second))},
				}}},
			}
		},
		newRunsReader: func(_ *http.Client, base string) runsReader {
			require.Equal(t, "http://127.0.0.1/agent-manager", base)
			return runs
		},
	}

	got := sources.Read(context.Background())
	require.Equal(t, now, got.Now)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.SystemMonitor.State)
	require.Equal(t, now.Add(-2*time.Second), got.SystemMonitor.SourceTime)
	require.Equal(t, float64(61), got.SystemMonitor.Measurements["cpu_usage_percent"])
	require.Equal(t, float64(49), got.SystemMonitor.Measurements["gpu_usage_percent"])
	require.Equal(t, float64(2), got.SystemMonitor.Measurements["memory_pressure_level"])
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.AgentManager.State)
	require.True(t, got.AgentManager.SourceTime.IsZero(), "Agent Manager does not return a snapshot timestamp")
	require.Equal(t, now, got.AgentManager.ObservedAt)
	require.Equal(t, float64(3), got.AgentManager.Measurements["active_runs"])
	require.Equal(t, []agentdomainv1.RunStatus{
		agentdomainv1.RunStatus_RUN_STATUS_PENDING,
		agentdomainv1.RunStatus_RUN_STATUS_STARTING,
		agentdomainv1.RunStatus_RUN_STATUS_RUNNING,
	}, runs.calls)
}

func TestReadReportsUnavailableInsteadOfZeroWhenOwnerCannotResolve(t *testing.T) {
	now := time.Now().UTC()
	sources := &Sources{
		resolve: func(_ context.Context, scenario string) (string, error) {
			return "", errors.New("not running: " + scenario)
		},
		now: func() time.Time { return now },
	}

	got := sources.Read(context.Background())
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, got.SystemMonitor.State)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, got.AgentManager.State)
	require.Empty(t, got.SystemMonitor.Measurements)
	require.Empty(t, got.AgentManager.Measurements)
	require.Contains(t, got.SystemMonitor.Reason, "not running")
	require.Contains(t, got.AgentManager.Reason, "not running")
}

func TestReadDegradesWhenMetricsAreUnmeasuredOrPressureIsMissing(t *testing.T) {
	now := time.Now().UTC()
	sources := &Sources{
		resolve: func(context.Context, string) (string, error) { return "http://monitor", nil },
		client:  &http.Client{},
		now:     func() time.Time { return now },
		newMetricsReader: func(*http.Client, string) metricsReader {
			return fakeMetrics{
				current: &metricsv1.GetCurrentMetricsResponse{Metrics: &metricsv1.MetricsResponse{
					Timestamp: timestamppb.New(now),
					Cpu:       &metricsv1.MetricValue{State: &metricsv1.MetricValue_UnsupportedReason{UnsupportedReason: "not available"}},
				}},
				detailed: &metricsv1.GetDetailedMetricsResponse{Metrics: &metricsv1.DetailedMetrics{}},
			}
		},
		newRunsReader: func(*http.Client, string) runsReader { return &fakeRuns{} },
	}

	got := sources.Read(context.Background())
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, got.SystemMonitor.State)
	require.Empty(t, got.SystemMonitor.Measurements)
	require.Contains(t, got.SystemMonitor.Reason, "no measured CPU or GPU")
}
