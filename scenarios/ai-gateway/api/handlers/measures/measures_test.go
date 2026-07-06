package measures_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"

	measuresH "ai-gateway/handlers/measures"
	"ai-gateway/internal/routing"
	testdb "ai-gateway/internal/testutil/db"
)

func newRepo(t *testing.T) *routing.SQLRepository {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(routing.Schema)))
	return routing.NewSQLRepository(db)
}

func seedEvidence(t *testing.T, repo *routing.SQLRepository, status string, fallback bool, rejection, capacity string, latency int64, createdAt time.Time) {
	t.Helper()
	ev := &routingv1.RouteEvidence{
		EventId:          "rt-" + createdAt.Format("20060102T150405.000000000"),
		RequestId:        "req",
		Scenario:         "fixture",
		Operation:        "summarize",
		Role:             "chat.default",
		Status:           status,
		FallbackUsed:     fallback,
		RejectionReason:  rejection,
		CapacityVerdict:  capacity,
		LatencyMs:        latency,
		PromptRedacted:   true,
		ResponseRedacted: true,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339Nano),
	}
	require.NoError(t, repo.Create(context.Background(), ev))
}

func TestMeasuresAggregateMathOverWindow(t *testing.T) { // [REQ:AIGW-ROUTE-MEASURES]
	repo := newRepo(t)
	base := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	seedEvidence(t, repo, "succeeded", false, "", "", 100, base)
	seedEvidence(t, repo, "succeeded", true, "", "", 200, base.Add(time.Minute))
	seedEvidence(t, repo, "failed", false, "", "", 50, base.Add(2*time.Minute))
	seedEvidence(t, repo, "blocked", false, "provider_breaker_open", "", 5, base.Add(3*time.Minute))
	seedEvidence(t, repo, "blocked", false, "", "insufficient_capacity", 5, base.Add(4*time.Minute))
	// Out-of-window event (a day earlier) must be excluded.
	seedEvidence(t, repo, "succeeded", false, "", "", 9999, base.Add(-24*time.Hour))

	from := base.Add(-time.Hour)
	to := base.Add(time.Hour)
	agg, err := repo.Aggregate(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, int64(5), agg.Total)
	require.Equal(t, int64(2), agg.Succeeded)
	require.Equal(t, int64(1), agg.Failed)
	require.Equal(t, int64(1), agg.FallbackUsed)
	require.Equal(t, int64(1), agg.BreakerOpen)
	require.Equal(t, int64(1), agg.CapacityRejected)
}

func TestMeasuresEmptyWindow(t *testing.T) { // [REQ:AIGW-ROUTE-MEASURES]
	repo := newRepo(t)
	from := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	agg, err := repo.Aggregate(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, routing.RouteAggregate{}, agg)
	p95, err := repo.LatencyP95(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, int64(0), p95)
}

func TestMeasuresLatencyP95(t *testing.T) { // [REQ:AIGW-ROUTE-MEASURES]
	repo := newRepo(t)
	base := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= 100; i++ {
		seedEvidence(t, repo, "succeeded", false, "", "", int64(i), base.Add(time.Duration(i)*time.Second))
	}
	p95, err := repo.LatencyP95(context.Background(), base, base.Add(2*time.Hour))
	require.NoError(t, err)
	// Nearest-rank p95 over 1..100 = value at offset floor(0.95*99)=94 → 95.
	require.Equal(t, int64(95), p95)
}

func TestConnectRPCsShareComputePath(t *testing.T) { // [REQ:AIGW-ROUTE-MEASURES]
	repo := newRepo(t)
	base := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	seedEvidence(t, repo, "succeeded", true, "", "", 100, base)
	seedEvidence(t, repo, "succeeded", false, "", "", 100, base.Add(time.Minute))
	seedEvidence(t, repo, "failed", false, "", "", 100, base.Add(2*time.Minute))

	// now within the same week so the default this_week window covers base.
	now := func() time.Time { return base.Add(time.Hour) }
	h := measuresH.NewHandler(repo, now)
	ctx := context.Background()
	req := func() *connect.Request[measuresv1.RouteMeasureRequest] {
		return connect.NewRequest(&measuresv1.RouteMeasureRequest{})
	}

	total, err := h.CountRouteEvents(ctx, req())
	require.NoError(t, err)
	require.Equal(t, int64(3), total.Msg.GetCount())

	rate, err := h.RouteSuccessRate(ctx, req())
	require.NoError(t, err)
	require.InDelta(t, 2.0/3.0, rate.Msg.GetRate(), 1e-4)

	fb, err := h.RouteFallbackRate(ctx, req())
	require.NoError(t, err)
	require.InDelta(t, 0.5, fb.Msg.GetRate(), 1e-4) // 1 fallback / 2 succeeded
}
