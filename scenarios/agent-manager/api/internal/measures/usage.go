package measures

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"

	"connectrpc.com/connect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const (
	UsageForecast         = "usage.forecast"
	UsageForecastBacktest = "usage.forecast_backtest"
)

// SetUsageForecast connects the provider quota readings the usage forecast
// reads. Without it the usage procedures report unavailable.
func (h *Handler) SetUsageForecast(source pricing.QuotaReadingSource) { h.quotaReadings = source }

// SetUsageActivity connects the provider activity that lets a stale window be
// idle; without it such a window is unknown.
func (h *Handler) SetUsageActivity(source usageforecast.ActivitySource) { h.usageActivity = source }

func (h *Handler) usageSpans(ctx context.Context, provider string, since time.Time) ([]pricing.QuotaReadingSpan, error) {
	if h.quotaReadings == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("provider quota readings are unavailable"))
	}
	spans, err := h.quotaReadings.QuotaReadingSpans(ctx, strings.TrimSpace(provider), since)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return spans, nil
}

func usageProvenance(spans []pricing.QuotaReadingSpan, since, now time.Time, provider string) *measurepb.MeasureProvenance {
	provenance := &measurepb.MeasureProvenance{SourceTable: "quota_observations", WindowEnd: now.UTC().Format(time.RFC3339Nano), RowCount: int64(len(spans)), ExecutedQuery: "SELECT reading spans GROUP BY provider, pool, window, reset minute, used_percent, standing FROM quota_observations WHERE observed_at >= window_start"}
	if !since.IsZero() {
		provenance.WindowStart = since.UTC().Format(time.RFC3339Nano)
	}
	if provider != "" {
		provenance.AppliedFilters = []*measurepb.MeasureFilter{{Field: "provider", Value: provider}}
	}
	return provenance
}

func (h *Handler) UsageForecast(ctx context.Context, req *connect.Request[measurepb.UsageForecastRequest]) (*connect.Response[measurepb.UsageForecastResponse], error) {
	now := h.now().UTC()
	since := now.Add(-usageforecast.CurrentLookback)
	spans, err := h.usageSpans(ctx, req.Msg.GetProvider(), since)
	if err != nil {
		return nil, err
	}
	response := &measurepb.UsageForecastResponse{GeneratedAt: now.Format(time.RFC3339), Provenance: usageProvenance(spans, since, now, req.Msg.GetProvider()), DefinitionId: definitionID(UsageForecast)}
	for _, forecast := range usageforecast.CurrentWithActivity(ctx, spans, now, h.usageActivity) {
		response.Windows = append(response.Windows, protoWindowForecast(forecast))
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) UsageForecastBacktest(ctx context.Context, req *connect.Request[measurepb.UsageForecastBacktestRequest]) (*connect.Response[measurepb.UsageForecastBacktestResponse], error) {
	now := h.now().UTC()
	spans, err := h.usageSpans(ctx, req.Msg.GetProvider(), time.Time{})
	if err != nil {
		return nil, err
	}
	report := usageforecast.Backtest(spans, now)
	response := &measurepb.UsageForecastBacktestResponse{
		UnobservedWindows: report.Unobserved,
		Limitation:        "scores only completed windows whose newest reading is within 6 hours of the window end; each checkpoint sees only readings and windows completed by then, but projects to the window's actual end, as the linear baseline does; history is limited to the readings in quota_observations",
		Provenance:        usageProvenance(spans, time.Time{}, now, req.Msg.GetProvider()),
		DefinitionId:      definitionID(UsageForecastBacktest),
	}
	scored := int64(-1)
	for _, summary := range report.Summaries {
		response.Summaries = append(response.Summaries, &measurepb.UsageBacktestSummary{
			Population: summary.Population, ElapsedFraction: summary.Fraction, Windows: summary.Windows,
			MedianAbsErrorPoints: summary.MedianAbsErrorPoints, BaselineMedianAbsErrorPoints: summary.BaselineMedianAbsErrorPts,
			HourErrorWindows: summary.HourErrorWindows, MedianAbsErrorHours: summary.MedianAbsErrorHours, BaselineMedianAbsErrorHours: summary.BaselineMedianAbsErrorHrs,
			OutcomeAccuracy: summary.OutcomeAccuracy, BaselineOutcomeAccuracy: summary.BaselineOutcomeAccuracy, BeatsBaseline: summary.BeatsBaseline,
		})
		if summary.Population == usageforecast.PopulationRecent && (scored < 0 || summary.Windows < scored) {
			scored = summary.Windows
		}
	}
	response.Validity = h.validityForSample(max(scored, 0))
	for _, w := range report.Windows {
		row := &measurepb.UsageBacktestWindow{
			Provider: w.Window.Provider, Pool: w.Window.Pool, Window: w.Window.Window,
			WindowStart: rfc3339(w.Window.Start), EndedAt: rfc3339(w.EndedAt), ResetsAt: rfc3339(w.Window.ResetAt),
			EarlyReset: w.EarlyReset, FinalPercent: w.Final, ActualExhaustionAt: rfc3339Ptr(w.ActualExhaustion), OutcomeObserved: w.OutcomeObserved,
		}
		for _, c := range w.Checkpoints {
			row.Checkpoints = append(row.Checkpoints, &measurepb.UsageBacktestCheckpoint{
				ElapsedFraction: c.Fraction, AsOf: rfc3339(c.AsOf), UsedPercentAt: c.UsedAt,
				PredictedFinalPercent: c.PredictedFinal, BaselinePredictedFinalPercent: c.BaselineFinal,
				AbsErrorPoints: c.AbsErrorPoints, BaselineAbsErrorPoints: c.BaselineAbsError,
				PredictedExhaustionAt: rfc3339Ptr(c.PredictedExhaustion), BaselinePredictedExhaustionAt: rfc3339Ptr(c.BaselineExhaustion),
				AbsErrorHours: c.AbsErrorHours, BaselineAbsErrorHours: c.BaselineAbsErrorHours,
				OutcomeCorrect: c.OutcomeCorrect, BaselineOutcomeCorrect: c.BaselineOutcomeOK,
				CalibrationFactor: c.Calibration.Factor, CalibrationWindows: int64(c.Calibration.Windows),
			})
		}
		response.Windows = append(response.Windows, row)
	}
	return connect.NewResponse(response), nil
}

func protoWindowForecast(f usageforecast.Forecast) *measurepb.UsageWindowForecast {
	w := f.Window
	row := &measurepb.UsageWindowForecast{
		Provider: w.Provider, Pool: w.Pool, Window: w.Window, WindowMinutes: w.WindowMinutes,
		Status: f.Status, StatusReason: f.Reason, HasPercent: w.HasPercent, UsedPercent: f.Used, Standing: string(w.Standing),
		WindowStart: rfc3339(w.Start), ResetsAt: rfc3339(w.ResetAt), ObservedAt: rfc3339(f.AsOf), SampleCount: w.Samples,
		CentralPercentPerHour: f.Central, LastReadingPercent: w.LastPercent,
		CalibrationFactor: f.Calibration.Factor, CalibrationWindows: int64(f.Calibration.Windows),
		ProjectedExhaustionAt: rfc3339Ptr(f.Projected), ExhaustionBandStart: rfc3339Ptr(f.BandStart), ExhaustionBandEnd: rfc3339Ptr(f.BandEnd),
		EarlyReset: f.EarlyReset,
	}
	if f.Lasting != nil {
		row.LastingPercentPerHour = *f.Lasting
	}
	if f.Previous != nil {
		row.PreviousResetsAt, row.PreviousEndedAt, row.PreviousFinalPercent = rfc3339(f.Previous.ResetAt), rfc3339(f.Previous.EndedAt), f.Previous.Final
	}
	for _, b := range f.Burn {
		row.Burn = append(row.Burn, &measurepb.UsageBurnRate{Horizon: fmt.Sprintf("%dh", int(b.Horizon.Hours())), Available: b.Available, Stale: b.Stale, PercentPerHour: b.Rate, SpanHours: b.Span.Hours()})
	}
	for _, p := range w.Series() {
		row.Series = append(row.Series, &measurepb.UsagePercentPoint{At: rfc3339(p.At), UsedPercent: p.Percent})
	}
	return row
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func rfc3339Ptr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return rfc3339(*t)
}
