package metrics_test

import (
	"context"
	"database/sql"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	metricsH "landing-page-react-vite-api/handlers/metrics"
	internalmetrics "landing-page-react-vite-api/internal/metrics"

	internalvariant "landing-page-react-vite-api/internal/variant"
)

func setup(t *testing.T) (*internalmetrics.Service, *sql.DB) {
	t.Helper()
	db := pgtest.NewDB(t)
	// metrics_events FKs variants(id); the stats join requires active variants.
	pgtest.Apply(t, db, internalvariant.Schema, internalmetrics.Schema)
	_, err := db.Exec(`DELETE FROM metrics_events`)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM variants`)
	require.NoError(t, err)
	seedVariant(t, db, 1, "control", "Control")
	seedVariant(t, db, 2, "variant-a", "Variant A")
	return internalmetrics.NewService(db), db
}

// Wide bounds (±2 days) so freshly-inserted rows land inside the window
// regardless of test-host vs database clock/timezone skew.
func yesterday() string { return time.Now().AddDate(0, 0, -2).Format("2006-01-02") }
func tomorrow() string  { return time.Now().AddDate(0, 0, 2).Format("2006-01-02") }

func seedVariant(t *testing.T, db *sql.DB, id int, slug, name string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO variants (id, slug, name, weight, status) VALUES ($1, $2, $3, 50, 'active')`, id, slug, name)
	require.NoError(t, err)
}

func track(t *testing.T, h *metricsH.Deps, eventType string, variantID int64, sessionID, eventID string, data map[string]interface{}) error {
	t.Helper()
	handler := metricsH.NewConnectHandler(*h)
	var st *structpb.Struct
	if data != nil {
		var err error
		st, err = structpb.NewStruct(data)
		require.NoError(t, err)
	}
	_, err := handler.TrackEvent(context.Background(), connect.NewRequest(&landingv1.TrackEventRequest{
		EventType: eventType, VariantId: variantID, SessionId: sessionID, EventId: eventID, EventData: st,
	}))
	return err
}

func TestTrackEventValid(t *testing.T) {
	svc, db := setup(t)
	deps := &metricsH.Deps{Service: svc}
	require.NoError(t, track(t, deps, "page_view", 1, "test-session-123", "", map[string]interface{}{"page": "/"}))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE session_id = $1`, "test-session-123").Scan(&count))
	require.Equal(t, 1, count)
}

func TestTrackEventIdempotency(t *testing.T) {
	svc, db := setup(t)
	deps := &metricsH.Deps{Service: svc}
	require.NoError(t, track(t, deps, "page_view", 1, "test-session-idem", "unique-event-123", nil))
	require.NoError(t, track(t, deps, "page_view", 1, "test-session-idem", "unique-event-123", nil))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE session_id = $1`, "test-session-idem").Scan(&count))
	require.Equal(t, 1, count)
}

func TestTrackEventInvalidType(t *testing.T) {
	svc, _ := setup(t)
	err := track(t, &metricsH.Deps{Service: svc}, "invalid_type", 1, "s", "", nil)
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestTrackEventMissingRequiredFields(t *testing.T) {
	svc, _ := setup(t)
	err := track(t, &metricsH.Deps{Service: svc}, "page_view", 0, "", "", nil)
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetVariantStats(t *testing.T) {
	svc, _ := setup(t)
	deps := &metricsH.Deps{Service: svc}
	require.NoError(t, track(t, deps, "page_view", 1, "session1", "evt1", nil))
	require.NoError(t, track(t, deps, "page_view", 1, "session2", "evt2", nil))
	require.NoError(t, track(t, deps, "click", 1, "session1", "evt3", map[string]interface{}{"element_type": "cta"}))
	require.NoError(t, track(t, deps, "conversion", 1, "session1", "evt4", nil))
	require.NoError(t, track(t, deps, "download", 1, "session1", "evt_dl", map[string]interface{}{"platform": "windows"}))
	require.NoError(t, track(t, deps, "page_view", 2, "session3", "evt5", nil))

	handler := metricsH.NewConnectHandler(*deps)
	resp, err := handler.GetVariantStats(context.Background(), connect.NewRequest(&landingv1.GetVariantStatsRequest{
		StartDate: yesterday(), EndDate: tomorrow(),
	}))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(resp.Msg.Stats), 2)

	var v1 *landingv1.VariantStats
	for _, s := range resp.Msg.Stats {
		if s.VariantId == 1 {
			v1 = s
		}
	}
	require.NotNil(t, v1)
	require.EqualValues(t, 2, v1.Views)
	require.EqualValues(t, 1, v1.CtaClicks)
	require.EqualValues(t, 1, v1.Conversions)
	require.EqualValues(t, 1, v1.Downloads)
	require.Equal(t, 50.0, v1.ConversionRate)
}

func TestGetVariantStatsFilterBySlug(t *testing.T) {
	svc, _ := setup(t)
	deps := &metricsH.Deps{Service: svc}
	require.NoError(t, track(t, deps, "page_view", 1, "session1", "evt-filter-1", nil))

	handler := metricsH.NewConnectHandler(*deps)
	resp, err := handler.GetVariantStats(context.Background(), connect.NewRequest(&landingv1.GetVariantStatsRequest{Variant: "control"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Stats, 1)
	require.Equal(t, "control", resp.Msg.Stats[0].VariantSlug)
}

func TestGetAnalyticsSummary(t *testing.T) {
	svc, _ := setup(t)
	deps := &metricsH.Deps{Service: svc}
	require.NoError(t, track(t, deps, "page_view", 1, "session1", "sum1", nil))
	require.NoError(t, track(t, deps, "page_view", 1, "session2", "sum2", nil))
	require.NoError(t, track(t, deps, "click", 1, "session1", "sum3", map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}))
	require.NoError(t, track(t, deps, "click", 1, "session2", "sum4", map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}))
	require.NoError(t, track(t, deps, "conversion", 1, "session1", "sum5", nil))
	require.NoError(t, track(t, deps, "download", 1, "session1", "sum6", nil))

	handler := metricsH.NewConnectHandler(*deps)
	resp, err := handler.GetAnalyticsSummary(context.Background(), connect.NewRequest(&landingv1.GetAnalyticsSummaryRequest{
		StartDate: yesterday(), EndDate: tomorrow(),
	}))
	require.NoError(t, err)
	require.EqualValues(t, 2, resp.Msg.TotalVisitors)
	require.EqualValues(t, 1, resp.Msg.TotalDownloads)
	require.NotNil(t, resp.Msg.TopCta)
	require.Equal(t, "hero-cta", *resp.Msg.TopCta)
	require.NotNil(t, resp.Msg.TopCtaCtr)
	require.Equal(t, 100.0, *resp.Msg.TopCtaCtr)
	require.NotEmpty(t, resp.Msg.VariantStats)
}

func TestGenerateEventID(t *testing.T) {
	e1 := internalmetrics.Event{EventType: "page_view", VariantID: 1, SessionID: "session1"}
	e2 := internalmetrics.Event{EventType: "page_view", VariantID: 1, SessionID: "session1"}
	require.Equal(t, internalmetrics.GenerateEventID(e1), internalmetrics.GenerateEventID(e2))

	e3 := internalmetrics.Event{EventType: "page_view", VariantID: 1, SessionID: "session2"}
	require.NotEqual(t, internalmetrics.GenerateEventID(e1), internalmetrics.GenerateEventID(e3))
}
