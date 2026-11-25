package metrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	handler "search-hub/handlers/metrics"
	"search-hub/internal/aigateway"
	localdb "search-hub/internal/database"
	internalmetrics "search-hub/internal/metrics"
	internalregistry "search-hub/internal/registry"
)

// fakeInsights is a hand-written InsightsReader fake.
type fakeInsights struct {
	out       *internalmetrics.Insights
	err       error
	rangeFrom time.Time
	rangeTo   time.Time
}

func (f *fakeInsights) Insights(context.Context, int) (*internalmetrics.Insights, error) {
	return f.out, f.err
}

func (f *fakeInsights) InsightsRange(_ context.Context, from, to time.Time) (*internalmetrics.Insights, error) {
	f.rangeFrom, f.rangeTo = from, to
	return f.out, f.err
}

// fakeLister returns a fixed ACTIVE provider set.
type fakeLister struct {
	providers []*registryv1.ProviderDescriptor
	err       error
}

type fakeEmbeddingTelemetry struct {
	page     internalmetrics.EmbeddingTelemetryPage
	err      error
	from, to time.Time
	limit    int
}

func (f *fakeEmbeddingTelemetry) ReadWindow(_ context.Context, from, to time.Time, limit int) (internalmetrics.EmbeddingTelemetryPage, error) {
	f.from, f.to, f.limit = from, to, limit
	return f.page, f.err
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
			TotalQueries:       10,
			ZeroResultQueries:  2,
			DegradedQueries:    1,
			RerankedQueries:    8,
			LatencyP50Ms:       120,
			LatencyP95Ms:       400,
			WindowFrom:         time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC),
			WindowTo:           time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
			SampleCount:        12,
			MinimumSampleCount: 10,
			SampleSufficient:   true,
			RecentSampleCount:  10,
			RecentLatencyP50Ms: 110,
			RecentLatencyP95Ms: 390,
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
	require.Equal(t, int64(12), msg.GetSampleCount())
	require.True(t, msg.GetSampleSufficient())
	require.Equal(t, "2026-06-23T00:00:00Z", msg.GetWindowFrom())
	require.Equal(t, int64(390), msg.GetRecentLatencyP95Ms())

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

func TestInsightsAcceptsDurationWindowAndUsesRangeReader(t *testing.T) {
	reader := &fakeInsights{out: &internalmetrics.Insights{SampleCount: 10, SampleSufficient: true}}
	client := newClient(t, handler.Deps{
		Insights:      reader,
		RangeInsights: reader,
		Lister:        &fakeLister{},
	})

	_, err := client.Insights(context.Background(), connect.NewRequest(&metricsv1.InsightsRequest{Window: "15m"}))
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().UTC(), reader.rangeTo, 5*time.Second)
	require.WithinDuration(t, reader.rangeTo.Add(-15*time.Minute), reader.rangeFrom, time.Second)
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

func TestEmbeddingTelemetryMapsBoundedMetadataWindow(t *testing.T) {
	reader := &fakeEmbeddingTelemetry{page: internalmetrics.EmbeddingTelemetryPage{
		Truncated:            true,
		HasRetainedWindow:    true,
		RetainedFrom:         time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC),
		RetainedTo:           time.Date(2026, 11, 20, 1, 0, 0, 0, time.UTC),
		ObserverDroppedTotal: 2, ObserverWriteFailures: 1,
		Rows: []internalmetrics.EmbeddingTelemetryRow{{Kind: "query", Role: "embedding.default", Model: "embedder", DurationUS: 1200, VectorCount: 1, CreatedAt: time.Date(2026, 11, 20, 0, 30, 0, 0, time.UTC)}},
	}}
	client := newClient(t, handler.Deps{EmbeddingTelemetry: reader})
	resp, err := client.EmbeddingTelemetry(context.Background(), connect.NewRequest(&metricsv1.EmbeddingTelemetryRequest{
		From: "2026-11-20T00:00:00Z", To: "2026-11-21T00:00:00Z", Limit: 50,
	}))
	require.NoError(t, err)
	require.Equal(t, 50, reader.limit)
	require.Equal(t, time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), reader.from)
	require.Equal(t, time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC), reader.to)
	require.True(t, resp.Msg.GetTruncated())
	require.Equal(t, uint64(2), resp.Msg.GetObserverDroppedTotal())
	require.Equal(t, uint64(1), resp.Msg.GetObserverWriteFailures())
	require.Equal(t, "2026-11-20T00:00:00Z", resp.Msg.GetRetainedFrom())
	require.Len(t, resp.Msg.GetObservations(), 1)
	require.Equal(t, int64(1200), resp.Msg.GetObservations()[0].GetDurationUs())
	require.Len(t, resp.Msg.GetProfiles(), 1)
	require.Equal(t, int64(1), resp.Msg.GetProfiles()[0].GetSampleCount())
	require.Equal(t, int64(1200), resp.Msg.GetProfiles()[0].GetDurationP95Us())
	require.False(t, resp.Msg.GetProfiles()[0].GetComplete(), "truncated pages cannot claim complete summaries")
}

func TestEmbeddingTelemetryReadsThroughOwnedSQLiteStore(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(internalmetrics.Schema),
	))
	stamp := time.Date(2026, 11, 20, 0, 30, 0, 0, time.UTC)
	store := internalmetrics.NewEmbeddingTelemetryStore(db, scheduletest.New(stamp), nil)
	defer store.Close(context.Background())
	store.Observe(aigateway.EmbeddingObservation{Kind: "query", Role: "embedding.default", Model: "test-embedder", Duration: 2 * time.Millisecond, VectorCount: 1})
	require.NoError(t, store.Flush(ctx))
	client := newClient(t, handler.Deps{EmbeddingTelemetry: store})
	resp, err := client.EmbeddingTelemetry(ctx, connect.NewRequest(&metricsv1.EmbeddingTelemetryRequest{
		From: stamp.Add(-time.Minute).Format(time.RFC3339Nano), To: stamp.Add(time.Minute).Format(time.RFC3339Nano), Limit: 10,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetObservations(), 1)
	row := resp.Msg.GetObservations()[0]
	require.Equal(t, "query", row.GetKind())
	require.Equal(t, "embedding.default", row.GetRole())
	require.Equal(t, "test-embedder", row.GetModel())
	require.Equal(t, int64(2000), row.GetDurationUs())
	require.Equal(t, "2026-11-20T00:30:00Z", row.GetCreatedAt())
	require.Len(t, resp.Msg.GetProfiles(), 1)
	require.Equal(t, int64(1), resp.Msg.GetProfiles()[0].GetSampleCount())
	require.True(t, resp.Msg.GetProfiles()[0].GetComplete())
}

func TestEmbeddingTelemetryRejectsBadWindowAndKeepsReadErrorsOpaque(t *testing.T) {
	client := newClient(t, handler.Deps{EmbeddingTelemetry: &fakeEmbeddingTelemetry{err: errors.New("secret db path")}})
	_, err := client.EmbeddingTelemetry(context.Background(), connect.NewRequest(&metricsv1.EmbeddingTelemetryRequest{From: "bad", To: "2026-11-21T00:00:00Z"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = client.EmbeddingTelemetry(context.Background(), connect.NewRequest(&metricsv1.EmbeddingTelemetryRequest{From: "2026-11-21T00:00:00Z", To: "2026-11-20T00:00:00Z"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = client.EmbeddingTelemetry(context.Background(), connect.NewRequest(&metricsv1.EmbeddingTelemetryRequest{From: "2026-11-20T00:00:00Z", To: "2026-11-21T00:00:00Z"}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret")
}

func TestHygieneReportsRetirementAndConcentration(t *testing.T) {
	providers := []*metricsv1.ProviderUtilization{
		{ProviderId: "a.one", ProviderGroup: "a", TimesRouted: 100, TotalHits: 0},
		{ProviderId: "a.two", ProviderGroup: "a", TimesRouted: 2, TotalHits: 4},
		{ProviderId: "a.three", ProviderGroup: "a", TimesRouted: 4, TotalHits: 1},
		{ProviderId: "b.one", ProviderGroup: "b", TimesRouted: 100, TotalHits: 0},
	}
	retire, groups := handler.HygieneReports(providers)
	if len(retire) != 2 || retire[0].GetProviderId() != "a.one" || retire[1].GetProviderId() != "b.one" {
		t.Fatalf("retirement candidates = %#v", retire)
	}
	if len(groups) != 1 || groups[0].GetProviderGroup() != "a" || groups[0].GetActiveLeaves() != 3 {
		t.Fatalf("group advisories = %#v", groups)
	}
}
