package metrics_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	handler "search-hub/handlers/metrics"
	internalmetrics "search-hub/internal/metrics"
	internalregistry "search-hub/internal/registry"
)

// fakeInsights is a hand-written InsightsReader fake.
type fakeInsights struct {
	out *internalmetrics.Insights
	err error
}

func (f *fakeInsights) Insights(context.Context, int) (*internalmetrics.Insights, error) {
	return f.out, f.err
}

// fakeLister returns a fixed ACTIVE provider set.
type fakeLister struct {
	providers []*registryv1.ProviderDescriptor
	err       error
}

func (f *fakeLister) List(context.Context, internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	return f.providers, f.err
}

func newClient(t *testing.T, d handler.Deps) metricsconnect.MetricsServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	d.Logger = logger
	path, h := metricsconnect.NewMetricsServiceHandler(handler.NewConnectHandler(d))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return metricsconnect.NewMetricsServiceClient(server.Client(), server.URL)
}

func activeLeaf(id, group, typ string) *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: id, ProviderGroup: group, Type: typ,
		State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
	}
}

func TestInsightsReconcilesUnderUtilized(t *testing.T) {
	// cli-health routed 5×; ui-health registered but never routed-to.
	client := newClient(t, handler.Deps{
		Insights: &fakeInsights{out: &internalmetrics.Insights{
			TotalQueries:      10,
			ZeroResultQueries: 2,
			DegradedQueries:   1,
			RerankedQueries:   8,
			LatencyP50Ms:      120,
			LatencyP95Ms:      400,
			ProviderUsage: []internalmetrics.ProviderUsage{
				{
					ProviderID:      "cli-health.commands",
					TimesRouted:     5,
					TotalHits:       12,
					LatencyP50Ms:    90,
					LatencyP95Ms:    300,
					DegradedCount:   2,
					DegradationRate: 0.4,
					DegradationReasons: []internalmetrics.ProviderDegradationReason{
						{Reason: "timeout", Count: 2},
					},
				},
			},
		}},
		Lister: &fakeLister{providers: []*registryv1.ProviderDescriptor{
			activeLeaf("cli-health.commands", "cli-health", "command"),
			activeLeaf("ui-health.surfaces", "ui-health", "component"),
		}},
	})

	resp, err := client.Insights(context.Background(), connect.NewRequest(&metricsv1.InsightsRequest{}))
	require.NoError(t, err)
	msg := resp.Msg

	require.Equal(t, int64(10), msg.GetTotalQueries())
	require.InDelta(t, 0.2, msg.GetZeroResultRate(), 1e-9)
	require.Equal(t, int64(120), msg.GetLatencyP50Ms())
	require.Equal(t, int64(400), msg.GetLatencyP95Ms())

	byID := map[string]*metricsv1.ProviderUtilization{}
	for _, p := range msg.GetProviders() {
		byID[p.GetProviderId()] = p
	}
	require.Len(t, byID, 2, "every ACTIVE leaf appears, even with no telemetry")
	require.Equal(t, int64(5), byID["cli-health.commands"].GetTimesRouted())
	require.False(t, byID["cli-health.commands"].GetUnderUtilized())
	require.Equal(t, int64(300), byID["cli-health.commands"].GetLatencyP95Ms())
	require.Equal(t, int64(2), byID["cli-health.commands"].GetDegradedCount())
	require.InDelta(t, 0.4, byID["cli-health.commands"].GetDegradationRate(), 1e-9)
	require.Equal(t, "timeout", byID["cli-health.commands"].GetDegradationReasons()[0].GetReason())
	require.Equal(t, int64(0), byID["ui-health.surfaces"].GetTimesRouted())
	require.True(t, byID["ui-health.surfaces"].GetUnderUtilized(), "registered-but-never-routed ⇒ under-utilized")
	require.Equal(t, "component", byID["ui-health.surfaces"].GetType())
}

func TestInsightsZeroQueriesRateIsZero(t *testing.T) {
	client := newClient(t, handler.Deps{
		Insights: &fakeInsights{out: &internalmetrics.Insights{}},
		Lister:   &fakeLister{},
	})
	resp, err := client.Insights(context.Background(), connect.NewRequest(&metricsv1.InsightsRequest{}))
	require.NoError(t, err)
	require.Equal(t, float64(0), resp.Msg.GetZeroResultRate(), "no divide-by-zero on an empty window")
}

func TestInsightsStoreErrorIsOpaque(t *testing.T) {
	client := newClient(t, handler.Deps{
		Insights: &fakeInsights{err: errors.New("telemetry db on fire: /var/lib/secret")},
		Lister:   &fakeLister{},
	})
	_, err := client.Insights(context.Background(), connect.NewRequest(&metricsv1.InsightsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret")
}
