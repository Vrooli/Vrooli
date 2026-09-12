package metricshttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type connectTracker struct {
	event metrics.Event
	err   error
}

func (f *connectTracker) TrackEvent(event metrics.Event) error { f.event = event; return f.err }

type connectReader struct {
	summary     *metrics.AnalyticsSummary
	stats       []metrics.VariantStats
	err         error
	start, end  time.Time
	variantSlug string
}

type connectRevenue struct{ summary *metrics.RevenueSummary }

func (f *connectRevenue) GetAdminRevenue() (*metrics.AdminRevenue, error) {
	return &metrics.AdminRevenue{}, nil
}
func (f *connectRevenue) GetRevenueSummary() (*metrics.RevenueSummary, error) { return f.summary, nil }

func (f *connectReader) GetAnalyticsSummary(start, end time.Time) (*metrics.AnalyticsSummary, error) {
	f.start, f.end = start, end
	return f.summary, f.err
}

func (f *connectReader) GetVariantStats(start, end time.Time, variantSlug string) ([]metrics.VariantStats, error) {
	f.start, f.end, f.variantSlug = start, end, variantSlug
	return f.stats, f.err
}

func TestConnectTrackEventRequiresVariantSlug(t *testing.T) {
	handler := NewConnectHandler(ConnectDependencies{Tracker: &connectTracker{}})
	_, err := handler.TrackEvent(context.Background(), connect.NewRequest(&lpbsv1.TrackEventRequest{EventType: "page_view", SessionId: "session"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestConnectTrackEventMapsSlugAndPayload(t *testing.T) {
	tracker := &connectTracker{}
	handler := NewConnectHandler(ConnectDependencies{Tracker: tracker})
	response, err := handler.TrackEvent(context.Background(), connect.NewRequest(&lpbsv1.TrackEventRequest{EventType: "page_view", VariantSlug: "control", SessionId: "session", EventId: "event"}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetSuccess() || tracker.event.VariantSlug != "control" || tracker.event.SessionID != "session" {
		t.Fatalf("unexpected event: %+v", tracker.event)
	}
}

func TestConnectTrackEventMapsDomainValidationFailures(t *testing.T) {
	handler := NewConnectHandler(ConnectDependencies{Tracker: &connectTracker{err: &metrics.ValidationError{Field: "event_type", Reason: "event type is required"}}})
	_, err := handler.TrackEvent(context.Background(), connect.NewRequest(&lpbsv1.TrackEventRequest{EventType: "unknown", VariantSlug: "control", SessionId: "session"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestConnectAnalyticsSummaryPreservesRequestedWindowAndProjection(t *testing.T) {
	observedAt := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	reader := &connectReader{summary: &metrics.AnalyticsSummary{TotalVisitors: 12, TotalDownloads: 3, ObservedAt: &observedAt, VariantStats: []metrics.VariantStats{{VariantSlug: "control", VariantName: "Control", Views: 10, CTAClicks: 4, Conversions: 2, Downloads: 1, ConversionRate: 20}}}}
	handler := NewConnectHandler(ConnectDependencies{Reader: reader})
	response, err := handler.GetAnalyticsSummary(context.Background(), connect.NewRequest(&lpbsv1.GetAnalyticsSummaryRequest{StartDate: "2026-01-01", EndDate: "2026-01-31"}))
	if err != nil {
		t.Fatal(err)
	}
	if reader.start.Format("2006-01-02") != "2026-01-01" || reader.end.Format("2006-01-02") != "2026-01-31" {
		t.Fatalf("reader window = %s..%s", reader.start, reader.end)
	}
	if response.Msg.GetTotalVisitors() != 12 || response.Msg.GetVariantStats()[0].GetCtaClicks() != 4 {
		t.Fatalf("summary projection = %+v", response.Msg)
	}
	if response.Msg.GetObservedAt() == nil || !response.Msg.GetObservedAt().AsTime().Equal(observedAt) {
		t.Fatalf("observed_at = %v, want %v", response.Msg.GetObservedAt(), observedAt)
	}
}

func TestConnectRevenueSummaryMapsTypedRollup(t *testing.T) {
	observedAt := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	reader := &connectRevenue{summary: &metrics.RevenueSummary{
		Currency: "usd", MRRMinor: 1200, RevenueTodayMinor: 250, RevenueWindowMinor: 900,
		SampleSize: 4, ChurnRatePercent: 25, MRRUnit: "minor_currency",
		RevenueTodayUnit: "minor_currency", RevenueWindowUnit: "minor_currency", CreditUnit: "credits",
		ObservedAt: &observedAt,
	}}
	handler := NewConnectHandler(ConnectDependencies{Revenue: reader})
	response, err := handler.GetRevenueSummary(context.Background(), connect.NewRequest(&lpbsv1.GetRevenueSummaryRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetMrrMinor() != 1200 || response.Msg.GetSampleSize() != 4 || response.Msg.GetCurrency() != "usd" {
		t.Fatalf("summary projection = %+v", response.Msg)
	}
	if response.Msg.GetObservedAt() == nil || !response.Msg.GetObservedAt().AsTime().Equal(observedAt) {
		t.Fatalf("observed_at = %v, want %v", response.Msg.GetObservedAt(), observedAt)
	}
}

func TestRegisterConnectRoutesMountsRevenueSummaryProcedure(t *testing.T) {
	observedAt := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	router := mux.NewRouter()
	RegisterConnectRoutes(router, ConnectDependencies{Revenue: &connectRevenue{summary: &metrics.RevenueSummary{
		Currency: "usd", MRRMinor: 1200, ObservedAt: &observedAt,
	}}}, func(next http.HandlerFunc) http.HandlerFunc { return next })
	server := httptest.NewServer(router)
	defer server.Close()

	client := lpbsconnect.NewAdminRevenueServiceClient(http.DefaultClient, server.URL)
	response, err := client.GetRevenueSummary(context.Background(), connect.NewRequest(&lpbsv1.GetRevenueSummaryRequest{}))
	if err != nil {
		t.Fatalf("GetRevenueSummary() error = %v", err)
	}
	if response.Msg.GetMrrMinor() != 1200 || response.Msg.GetCurrency() != "usd" {
		t.Fatalf("route projection = %+v", response.Msg)
	}
}

// [REQ:METRIC-DETAIL] The generated analytics edge preserves the variant-detail
// projection used by the operator analytics view.
func TestConnectVariantStatsMapsFilterAndServiceFailures(t *testing.T) {
	reader := &connectReader{stats: []metrics.VariantStats{{VariantSlug: "campaign", Views: 7}}}
	handler := NewConnectHandler(ConnectDependencies{Reader: reader})
	response, err := handler.GetVariantStats(context.Background(), connect.NewRequest(&lpbsv1.GetVariantStatsRequest{StartDate: "2026-02-01", EndDate: "2026-02-02", Variant: "campaign"}))
	if err != nil {
		t.Fatal(err)
	}
	if reader.variantSlug != "campaign" || response.Msg.GetStats()[0].GetViews() != 7 {
		t.Fatalf("stats projection = %+v, filter=%q", response.Msg, reader.variantSlug)
	}
	reader.err = errors.New("database unavailable")
	_, err = handler.GetVariantStats(context.Background(), connect.NewRequest(&lpbsv1.GetVariantStatsRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
}
