package metricshttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/analytics"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type ConnectDependencies struct {
	Tracker EventTracker
	Reader  AnalyticsReader
	Traffic TrafficReader
	Revenue RevenueReader
}

type RevenueReader interface {
	GetAdminRevenue() (*metrics.AdminRevenue, error)
}

type RevenueSummaryReader interface {
	GetRevenueSummary() (*metrics.RevenueSummary, error)
}

// seam: interface
// EventTracker records an idempotent analytics event. The production service
// is wired once by the API composition root; tests use a narrow fake.
type EventTracker interface {
	TrackEvent(metrics.Event) error
}

// seam: interface
// AnalyticsReader supplies reporting projections without exposing persistence
// details to the generated transport.
type AnalyticsReader interface {
	GetAnalyticsSummary(time.Time, time.Time) (*metrics.AnalyticsSummary, error)
	GetVariantStats(time.Time, time.Time, string) ([]metrics.VariantStats, error)
}

type TrafficReader interface {
	GetTrafficBreakdown(string, time.Time, time.Time, int) (*metrics.TrafficBreakdown, error)
	GetTrafficSeries(string, time.Time, time.Time, string) (*metrics.TrafficSeries, error)
}

type ConnectHandler struct{ deps ConnectDependencies }

func NewConnectHandler(deps ConnectDependencies) *ConnectHandler { return &ConnectHandler{deps: deps} }

func (h *ConnectHandler) TrackEvent(_ context.Context, request *connect.Request[lpbsv1.TrackEventRequest]) (*connect.Response[lpbsv1.TrackEventResponse], error) {
	input := request.Msg
	slug := strings.TrimSpace(input.GetVariantSlug())
	if slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("variant_slug is required"))
	}
	var data map[string]interface{}
	if input.GetEventData() != nil {
		data = input.GetEventData().AsMap()
	}
	referrerHost := ""
	if parsed, err := url.Parse(input.GetReferrer()); err == nil && parsed.Hostname() != "" {
		referrerHost = parsed.Hostname()
	}
	dimensions := analytics.Enrich(&http.Request{Header: request.Header(), RemoteAddr: ""}, input.GetReferrer())
	if dimensions.ReferrerHost == "" {
		dimensions.ReferrerHost = referrerHost
	}
	if err := h.deps.Tracker.TrackEvent(metrics.Event{EventType: input.GetEventType(), VariantSlug: slug, EventData: data, SessionID: input.GetSessionId(), VisitorID: input.GetVisitorId(), EventID: input.GetEventId(), ReferrerHost: dimensions.ReferrerHost, ReferrerKind: dimensions.ReferrerKind, DeviceClass: dimensions.DeviceClass, CountryCode: dimensions.CountryCode, UTMSource: input.GetUtmSource(), UTMMedium: input.GetUtmMedium(), UTMCampaign: input.GetUtmCampaign(), LandingPath: input.GetLandingPath()}); err != nil {
		var validation *metrics.ValidationError
		if errors.As(err, &validation) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("track event: %w", err))
	}
	return connect.NewResponse(&lpbsv1.TrackEventResponse{Success: true, Message: "Event tracked successfully"}), nil
}

func (h *ConnectHandler) GetAnalyticsSummary(_ context.Context, request *connect.Request[lpbsv1.GetAnalyticsSummaryRequest]) (*connect.Response[lpbsv1.AnalyticsSummary], error) {
	start, end := dates(request.Msg.GetStartDate(), request.Msg.GetEndDate())
	summary, err := h.deps.Reader.GetAnalyticsSummary(start, end)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get analytics summary: %w", err))
	}
	return connect.NewResponse(summaryProto(summary)), nil
}

func (h *ConnectHandler) GetVariantStats(_ context.Context, request *connect.Request[lpbsv1.GetVariantStatsRequest]) (*connect.Response[lpbsv1.GetVariantStatsResponse], error) {
	start, end := dates(request.Msg.GetStartDate(), request.Msg.GetEndDate())
	stats, err := h.deps.Reader.GetVariantStats(start, end, request.Msg.GetVariant())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get variant stats: %w", err))
	}
	result := make([]*lpbsv1.VariantStats, 0, len(stats))
	for _, stat := range stats {
		result = append(result, statProto(stat))
	}
	return connect.NewResponse(&lpbsv1.GetVariantStatsResponse{StartDate: start.Format("2006-01-02"), EndDate: end.Format("2006-01-02"), Stats: result}), nil
}

func (h *ConnectHandler) GetTrafficBreakdown(_ context.Context, request *connect.Request[lpbsv1.GetTrafficBreakdownRequest]) (*connect.Response[lpbsv1.GetTrafficBreakdownResponse], error) {
	start, end := dates(request.Msg.GetStartDate(), request.Msg.GetEndDate())
	if h.deps.Traffic == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("traffic reader is unavailable"))
	}
	result, err := h.deps.Traffic.GetTrafficBreakdown(strings.ToLower(request.Msg.GetDimension().String()), start, end, int(request.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("get traffic breakdown: %w", err))
	}
	rows := make([]*lpbsv1.TrafficBreakdownRow, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, &lpbsv1.TrafficBreakdownRow{Key: row.Key, Label: row.Label, Sessions: row.Sessions, Conversions: row.Conversions, RevenueMinor: row.RevenueMinor, Share: row.Share})
	}
	return connect.NewResponse(&lpbsv1.GetTrafficBreakdownResponse{Rows: rows, TotalSessions: result.TotalSessions, Exhaustive: result.Exhaustive, Currency: result.Currency, ObservedAt: timestamppb.New(result.ObservedAt)}), nil
}

func (h *ConnectHandler) GetTrafficSeries(_ context.Context, request *connect.Request[lpbsv1.GetTrafficSeriesRequest]) (*connect.Response[lpbsv1.GetTrafficSeriesResponse], error) {
	start, end := dates(request.Msg.GetStartDate(), request.Msg.GetEndDate())
	if h.deps.Traffic == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("traffic reader is unavailable"))
	}
	result, err := h.deps.Traffic.GetTrafficSeries(request.Msg.GetMetric(), start, end, request.Msg.GetBucket())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("get traffic series: %w", err))
	}
	points := make([]*lpbsv1.TrafficSeriesPoint, 0, len(result.Points))
	for _, point := range result.Points {
		points = append(points, &lpbsv1.TrafficSeriesPoint{BucketStart: point.BucketStart.Format(time.RFC3339), Value: point.Value})
	}
	return connect.NewResponse(&lpbsv1.GetTrafficSeriesResponse{Points: points, Unit: result.Unit, ObservedAt: timestamppb.New(result.ObservedAt)}), nil
}

func (h *ConnectHandler) GetRevenue(_ context.Context, _ *connect.Request[lpbsv1.GetAdminRevenueRequest]) (*connect.Response[lpbsv1.AdminRevenue], error) {
	if h.deps.Revenue == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("revenue reader is unavailable"))
	}
	revenue, err := h.deps.Revenue.GetAdminRevenue()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get admin revenue: %w", err))
	}
	result := &lpbsv1.AdminRevenue{Mrr: revenue.MRR, MrrUnit: revenue.MRRUnit, Today: revenue.Today, TodayUnit: revenue.TodayUnit, Currency: revenue.Currency, SampleSize: revenue.SampleSize}
	if revenue.ObservedAt != nil {
		result.ObservedAt = timestamppb.New(*revenue.ObservedAt)
	}
	return connect.NewResponse(result), nil
}

func (h *ConnectHandler) GetRevenueSummary(_ context.Context, _ *connect.Request[lpbsv1.GetRevenueSummaryRequest]) (*connect.Response[lpbsv1.RevenueSummary], error) {
	reader, ok := h.deps.Revenue.(RevenueSummaryReader)
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("revenue summary reader is unavailable"))
	}
	summary, err := reader.GetRevenueSummary()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get revenue summary: %w", err))
	}
	result := &lpbsv1.RevenueSummary{
		Currency: summary.Currency, MrrMinor: summary.MRRMinor, RevenueTodayMinor: summary.RevenueTodayMinor,
		RevenueWindowMinor: summary.RevenueWindowMinor, ActiveSubscriptions: summary.ActiveSubscriptions,
		SubscriptionsChurnedWindow: summary.SubscriptionsChurnedWindow, ChurnRatePercent: summary.ChurnRatePercent,
		CreditBalanceTotal: summary.CreditBalanceTotal, CreditBurnedWindow: summary.CreditBurnedWindow,
		UsageRecordsWindow: summary.UsageRecordsWindow, SampleSize: summary.SampleSize,
		TrialsWithoutPaymentMethod: summary.TrialsWithoutPaymentMethod,
		MrrUnit:                    summary.MRRUnit, RevenueTodayUnit: summary.RevenueTodayUnit,
		RevenueWindowUnit: summary.RevenueWindowUnit, CreditUnit: summary.CreditUnit,
		CurrencyExcludedCount: summary.CurrencyExcludedCount,
	}
	if summary.ObservedAt != nil {
		result.ObservedAt = timestamppb.New(*summary.ObservedAt)
	}
	return connect.NewResponse(result), nil
}

func dates(start, end string) (time.Time, time.Time) {
	now := time.Now()
	from := now.AddDate(0, 0, -7)
	if parsed, err := time.Parse("2006-01-02", start); err == nil {
		from = parsed
	}
	if parsed, err := time.Parse("2006-01-02", end); err == nil {
		// Date-only requests include the complete end date. Without this,
		// `end_date=YYYY-MM-DD` silently excludes every event after midnight.
		now = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return from, now
}

func statProto(stat metrics.VariantStats) *lpbsv1.VariantStats {
	return &lpbsv1.VariantStats{VariantSlug: stat.VariantSlug, VariantName: stat.VariantName, Views: stat.Views, CtaClicks: stat.CTAClicks, Conversions: stat.Conversions, Downloads: stat.Downloads, Exposures: stat.Exposures, ConversionRate: stat.ConversionRate}
}

func summaryProto(summary *metrics.AnalyticsSummary) *lpbsv1.AnalyticsSummary {
	if summary == nil {
		return &lpbsv1.AnalyticsSummary{}
	}
	stats := make([]*lpbsv1.VariantStats, 0, len(summary.VariantStats))
	for _, stat := range summary.VariantStats {
		stats = append(stats, statProto(stat))
	}
	result := &lpbsv1.AnalyticsSummary{TotalVisitors: summary.TotalVisitors, TotalDownloads: summary.TotalDownloads, VariantStats: stats}
	if summary.ObservedAt != nil {
		result.ObservedAt = timestamppb.New(*summary.ObservedAt)
	}
	return result
}

func RegisterConnectRoutes(router *mux.Router, deps ConnectDependencies, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewMetricsServiceHandler(NewConnectHandler(deps))
	_, revenueService := lpbsconnect.NewAdminRevenueServiceHandler(NewConnectHandler(deps))
	mount := func(path string, middleware func(http.HandlerFunc) http.HandlerFunc) {
		handler := http.HandlerFunc(service.ServeHTTP)
		if middleware != nil {
			handler = middleware(handler)
		}
		router.Handle(path, handler).Methods(http.MethodPost)
	}
	mount(lpbsconnect.MetricsServiceTrackEventProcedure, nil)
	mount(lpbsconnect.MetricsServiceGetAnalyticsSummaryProcedure, requireAdmin)
	mount(lpbsconnect.MetricsServiceGetVariantStatsProcedure, requireAdmin)
	mount(lpbsconnect.MetricsServiceGetTrafficBreakdownProcedure, requireAdmin)
	mount(lpbsconnect.MetricsServiceGetTrafficSeriesProcedure, requireAdmin)
	revenueHandler := http.HandlerFunc(revenueService.ServeHTTP)
	router.Handle(lpbsconnect.AdminRevenueServiceGetRevenueProcedure, requireAdmin(revenueHandler)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdminRevenueServiceGetRevenueSummaryProcedure, requireAdmin(revenueHandler)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/admin/dashboard/traffic-breakdown", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if deps.Traffic == nil {
			http.Error(w, "traffic reader is unavailable", http.StatusPreconditionFailed)
			return
		}
		start, end := dates(r.URL.Query().Get("start_date"), r.URL.Query().Get("end_date"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		breakdown, err := deps.Traffic.GetTrafficBreakdown(r.URL.Query().Get("dimension"), start, end, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(breakdown)
	})).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/admin/dashboard/revenue", requireAdmin(func(w http.ResponseWriter, _ *http.Request) {
		if deps.Revenue == nil {
			http.Error(w, "revenue reader is unavailable", http.StatusPreconditionFailed)
			return
		}
		revenue, err := deps.Revenue.GetAdminRevenue()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(revenue)
	})).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/admin/dashboard/revenue/summary", requireAdmin(func(w http.ResponseWriter, _ *http.Request) {
		reader, ok := deps.Revenue.(RevenueSummaryReader)
		if !ok {
			http.Error(w, "revenue summary reader is unavailable", http.StatusPreconditionFailed)
			return
		}
		summary, err := reader.GetRevenueSummary()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	})).Methods(http.MethodGet)
}
