package sources

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNilCurrentMetricsResponseOnlyDegradesLoadEvidence(t *testing.T) {
	now := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	reader := metricsFunctions{
		current: func(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error) {
			return nil, nil
		},
		detailed: func(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error) {
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

	load, health := sources.readSystemMonitor(context.Background())

	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, load.State)
	require.Empty(t, load.Measurements)
	require.Equal(t, "read current metrics: response unavailable", load.Reason)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_FRESH, health.State)
	require.Equal(t, float64(3), health.Measurements["memory_pressure_level"])
	require.Equal(t, now, health.SourceTime)
}
