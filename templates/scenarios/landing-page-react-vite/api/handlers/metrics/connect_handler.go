package metrics

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalmetrics "landing-page-react-vite-api/internal/metrics"
)

// Deps wires the metrics Connect handler.
type Deps struct {
	Service *internalmetrics.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the MetricsService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) TrackEvent(ctx context.Context, req *connect.Request[landingv1.TrackEventRequest]) (*connect.Response[landingv1.TrackEventResponse], error) {
	m := req.Msg
	var eventData map[string]interface{}
	if m.EventData != nil {
		eventData = m.EventData.AsMap()
	}
	err := h.deps.Service.TrackEvent(internalmetrics.Event{
		EventType: m.EventType,
		VariantID: int(m.VariantId),
		EventData: eventData,
		SessionID: m.SessionId,
		VisitorID: m.VisitorId,
		EventID:   m.EventId,
	})
	if err != nil {
		var validationErr *internalmetrics.ValidationError
		if errors.As(err, &validationErr) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(validationErr.Reason))
		}
		h.deps.Logger.Printf("metrics.TrackEvent: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to track event"))
	}
	return connect.NewResponse(&landingv1.TrackEventResponse{Success: true, Message: "Event tracked successfully"}), nil
}

func (h *connectHandler) GetAnalyticsSummary(ctx context.Context, req *connect.Request[landingv1.GetAnalyticsSummaryRequest]) (*connect.Response[landingv1.AnalyticsSummary], error) {
	start, end := resolveWindow(req.Msg.StartDate, req.Msg.EndDate)
	summary, err := h.deps.Service.GetAnalyticsSummary(start, end)
	if err != nil {
		h.deps.Logger.Printf("metrics.GetAnalyticsSummary: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to fetch analytics summary"))
	}
	out := &landingv1.AnalyticsSummary{
		TotalVisitors:  summary.TotalVisitors,
		TotalDownloads: summary.TotalDownloads,
		VariantStats:   statsToProto(summary.VariantStats),
	}
	if summary.TopCTA != "" {
		topCTA := summary.TopCTA
		ctr := summary.TopCTACTR
		out.TopCta = &topCTA
		out.TopCtaCtr = &ctr
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetVariantStats(ctx context.Context, req *connect.Request[landingv1.GetVariantStatsRequest]) (*connect.Response[landingv1.GetVariantStatsResponse], error) {
	start, end := resolveWindow(req.Msg.StartDate, req.Msg.EndDate)
	stats, err := h.deps.Service.GetVariantStats(start, end, req.Msg.Variant)
	if err != nil {
		h.deps.Logger.Printf("metrics.GetVariantStats: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to fetch variant stats"))
	}
	return connect.NewResponse(&landingv1.GetVariantStatsResponse{
		StartDate: start.Format("2006-01-02"),
		EndDate:   end.Format("2006-01-02"),
		Stats:     statsToProto(stats),
	}), nil
}

// resolveWindow parses YYYY-MM-DD bounds, defaulting to the last 7 days.
func resolveWindow(startStr, endStr string) (time.Time, time.Time) {
	end := time.Now()
	start := end.AddDate(0, 0, -7)
	if startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			start = parsed
		}
	}
	if endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			end = parsed
		}
	}
	return start, end
}

func statsToProto(stats []internalmetrics.VariantStats) []*landingv1.VariantStats {
	out := make([]*landingv1.VariantStats, 0, len(stats))
	for _, s := range stats {
		out = append(out, &landingv1.VariantStats{
			VariantId:      int64(s.VariantID),
			VariantSlug:    s.VariantSlug,
			VariantName:    s.VariantName,
			Views:          s.Views,
			CtaClicks:      s.CTAClicks,
			Conversions:    s.Conversions,
			Downloads:      s.Downloads,
			ConversionRate: s.ConversionRate,
		})
	}
	return out
}
