package sources

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	agentapiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lighting-control/internal/policy"
)

type fakeMetrics struct {
	current     *metricsv1.GetCurrentMetricsResponse
	detailed    *metricsv1.GetDetailedMetricsResponse
	err         error
	detailedErr error
}

type metricsFunctions struct {
	current  func(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error)
	detailed func(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error)
}

func (f metricsFunctions) GetCurrentMetrics(ctx context.Context, req *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error) {
	return f.current(ctx, req)
}

func (f metricsFunctions) GetDetailedMetrics(ctx context.Context, req *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error) {
	return f.detailed(ctx, req)
}

func (f fakeMetrics) GetCurrentMetrics(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.current), nil
}

func (f fakeMetrics) GetDetailedMetrics(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error) {
	if f.detailedErr != nil {
		return nil, f.detailedErr
	}
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.detailed), nil
}

func TestReadPreservesFreshLoadMetricsWhenHealthDetailIsUnavailable(t *testing.T) {
	now := time.Date(2026, 11, 10, 17, 0, 0, 0, time.UTC)
	sources := &Sources{
		resolve: func(_ context.Context, scenario string) (string, error) {
			if scenario == "system-monitor" {
				return "http://monitor", nil
			}
			return "", errors.New("not configured")
		},
		client: &http.Client{}, now: func() time.Time { return now },
		newMetricsReader: func(*http.Client, string) metricsReader {
			return fakeMetrics{
				current:     &metricsv1.GetCurrentMetricsResponse{Metrics: &metricsv1.MetricsResponse{Timestamp: timestamppb.New(now), Cpu: &metricsv1.MetricValue{State: &metricsv1.MetricValue_Measured{Measured: 87}}}},
				detailedErr: errors.New("detailed endpoint unavailable"),
			}
		},
		newActivityReader: func(*http.Client, string) activityReader { return &fakeActivity{} },
	}

	got := sources.Read(context.Background())
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.SystemMonitor.State)
	require.Equal(t, float64(87), got.SystemMonitor.Measurements["cpu_usage_percent"])
	require.NotContains(t, got.SystemMonitor.Measurements, "memory_pressure_level")
	require.Contains(t, got.SystemHealth.Reason, "read health detail")
}

func TestSystemMonitorPressureReadIsNotBlockedBySlowLoadRead(t *testing.T) {
	now := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	currentStarted := make(chan struct{})
	releaseCurrent := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(releaseCurrent) }) })
	detailedFinished := make(chan struct{})
	reader := metricsFunctions{
		current: func(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error) {
			close(currentStarted)
			<-releaseCurrent
			return nil, errors.New("load read exceeded its budget")
		},
		detailed: func(ctx context.Context, _ *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error) {
			defer close(detailedFinished)
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return connect.NewResponse(&metricsv1.GetDetailedMetricsResponse{Metrics: &metricsv1.DetailedMetrics{MemoryDetails: &metricsv1.MemoryMetrics{
				MemoryPressure: &metricsv1.MemoryPressureReading{Level: "critical", ObservedAt: timestamppb.New(now)},
			}}}), nil
		},
	}
	sources := &Sources{
		resolve:          func(context.Context, string) (string, error) { return "http://monitor", nil },
		client:           &http.Client{},
		now:              func() time.Time { return now },
		newMetricsReader: func(*http.Client, string) metricsReader { return reader },
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	type result struct {
		load, health policy.SourceInput
	}
	readDone := make(chan result, 1)
	go func() {
		load, health := sources.readSystemMonitor(ctx)
		readDone <- result{load: load, health: health}
	}()
	<-currentStarted
	select {
	case <-detailedFinished:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("pressure read waited for the blocked utilization request")
	}
	releaseOnce.Do(func() { close(releaseCurrent) })
	got := <-readDone
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, got.load.State)
	require.Contains(t, got.load.Reason, "load read exceeded")
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.health.State)
	require.Equal(t, float64(3), got.health.Measurements["memory_pressure_level"])
}

func TestNilDetailedMetricsResponseOnlyDegradesPressureEvidence(t *testing.T) {
	now := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	reader := metricsFunctions{
		current: func(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error) {
			return connect.NewResponse(&metricsv1.GetCurrentMetricsResponse{Metrics: &metricsv1.MetricsResponse{
				Timestamp: timestamppb.New(now),
				Cpu:       &metricsv1.MetricValue{State: &metricsv1.MetricValue_Measured{Measured: 50}},
			}}), nil
		},
		detailed: func(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error) {
			return nil, nil
		},
	}
	sources := &Sources{
		resolve:          func(context.Context, string) (string, error) { return "http://monitor", nil },
		client:           &http.Client{},
		now:              func() time.Time { return now },
		newMetricsReader: func(*http.Client, string) metricsReader { return reader },
	}

	load, health := sources.readSystemMonitor(context.Background())

	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, load.State)
	require.Equal(t, float64(50), load.Measurements["cpu_usage_percent"])
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, health.State)
	require.Equal(t, "read health detail: response unavailable", health.Reason)
}

type fakeActivity struct {
	response *agentapiv1.GetRuntimeActivityResponse
	err      error
}

func (f *fakeActivity) GetRuntimeActivity(context.Context, *connect.Request[agentapiv1.GetRuntimeActivityRequest]) (*connect.Response[agentapiv1.GetRuntimeActivityResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response == nil {
		return connect.NewResponse(&agentapiv1.GetRuntimeActivityResponse{}), nil
	}
	return connect.NewResponse(f.response), nil
}

func TestReadUsesOwnerFactsAndCountsOnlyActiveRuns(t *testing.T) {
	now := time.Date(2026, 11, 10, 17, 0, 0, 0, time.UTC)
	count := uint32(1)
	activity := &fakeActivity{response: &agentapiv1.GetRuntimeActivityResponse{ObservedAt: timestamppb.New(now.Add(-time.Second)), Complete: true, ExecutingRuns: &count, ObservedExecutingRuns: count, LogicalQueuedRuns: 100}}

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
		newActivityReader: func(_ *http.Client, base string) activityReader {
			require.Equal(t, "http://127.0.0.1/agent-manager", base)
			return activity
		},
	}

	got := sources.Read(context.Background())
	require.Equal(t, now, got.Now)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.SystemMonitor.State)
	require.Equal(t, now.Add(-2*time.Second), got.SystemMonitor.SourceTime)
	require.Equal(t, float64(61), got.SystemMonitor.Measurements["cpu_usage_percent"])
	require.Equal(t, float64(49), got.SystemMonitor.Measurements["gpu_usage_percent"])
	require.Equal(t, float64(2), got.SystemHealth.Measurements["memory_pressure_level"])
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.AgentManager.State)
	require.Equal(t, now.Add(-time.Second), got.AgentManager.SourceTime)
	require.Equal(t, now, got.AgentManager.ObservedAt)
	require.Equal(t, float64(1), got.AgentManager.Measurements["active_runs"])
	require.Equal(t, float64(100), got.AgentManager.Measurements["logical_queued_runs"])

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
		newActivityReader: func(*http.Client, string) activityReader { return &fakeActivity{} },
	}

	got := sources.Read(context.Background())
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, got.SystemMonitor.State)
	require.Empty(t, got.SystemMonitor.Measurements)
	require.Contains(t, got.SystemMonitor.Reason, "no measured CPU or GPU")
}

func TestLoadAndPressureKeepIndependentFreshness(t *testing.T) {
	now := time.Date(2026, 11, 11, 1, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name         string
		cpu          *metricsv1.MetricValue
		pressureTime *timestamppb.Timestamp
		wantHealth   policyv1.EvidenceState
	}{
		{"unmeasured CPU with valid pressure", nil, timestamppb.New(now), policyv1.EvidenceState_EVIDENCE_STATE_FRESH},
		{"old pressure retains old timestamp", &metricsv1.MetricValue{State: &metricsv1.MetricValue_Measured{Measured: 90}}, timestamppb.New(now.Add(-time.Hour)), policyv1.EvidenceState_EVIDENCE_STATE_FRESH},
		{"missing pressure timestamp", &metricsv1.MetricValue{State: &metricsv1.MetricValue_Measured{Measured: 90}}, nil, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sources := &Sources{
				resolve: func(context.Context, string) (string, error) { return "http://owner", nil },
				client:  &http.Client{}, now: func() time.Time { return now },
				newMetricsReader: func(*http.Client, string) metricsReader {
					return fakeMetrics{
						current: &metricsv1.GetCurrentMetricsResponse{Metrics: &metricsv1.MetricsResponse{Timestamp: timestamppb.New(now), Cpu: tt.cpu}},
						detailed: &metricsv1.GetDetailedMetricsResponse{Metrics: &metricsv1.DetailedMetrics{MemoryDetails: &metricsv1.MemoryMetrics{
							MemoryPressure: &metricsv1.MemoryPressureReading{Level: "critical", ObservedAt: tt.pressureTime},
						}}},
					}
				},
				newActivityReader: func(*http.Client, string) activityReader { return &fakeActivity{} },
			}
			got := sources.Read(context.Background())
			require.Equal(t, tt.wantHealth, got.SystemHealth.State)
			if tt.cpu != nil {
				require.Equal(t, now, got.SystemMonitor.SourceTime)
				require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, got.SystemMonitor.State)
			}
			if tt.pressureTime != nil {
				require.Equal(t, tt.pressureTime.AsTime(), got.SystemHealth.SourceTime)
				require.Equal(t, float64(3), got.SystemHealth.Measurements["memory_pressure_level"])
			} else {
				require.NotContains(t, got.SystemHealth.Measurements, "memory_pressure_level")
			}
		})
	}
}
