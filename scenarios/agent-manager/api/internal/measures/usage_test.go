package measures

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/usageforecast"

	"agent-manager/internal/pricing"

	"connectrpc.com/connect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

type quotaReadings struct {
	spans    []pricing.QuotaReadingSpan
	provider string
	since    time.Time
}

func (q *quotaReadings) QuotaReadingSpans(_ context.Context, provider string, since time.Time) ([]pricing.QuotaReadingSpan, error) {
	q.provider, q.since = provider, since
	return q.spans, nil
}

// codexReadings is a weekly window that climbed one point every 40 minutes
// for a day, followed by a completed earlier window that reset early.
func codexReadings(now time.Time) []pricing.QuotaReadingSpan {
	minutes := int64(10080)
	var out []pricing.QuotaReadingSpan
	add := func(reset time.Time, from time.Time, percents int, step time.Duration) {
		for p := 1; p <= percents; p++ {
			percent, r := float64(p), reset
			first := from.Add(time.Duration(p-1) * step)
			out = append(out, pricing.QuotaReadingSpan{Provider: "openai", Pool: "primary", Window: "10080m", WindowMinutes: &minutes, ResetAt: &r, UsedPercent: &percent, Standing: pricing.QuotaStandingAvailable, FirstAt: first, LastAt: first.Add(step - time.Second), Samples: 3})
		}
	}
	currentStart := now.Add(-24 * time.Hour)
	previousStart := currentStart.Add(-48 * time.Hour)
	add(previousStart.Add(7*24*time.Hour), previousStart.Add(time.Minute), 40, 70*time.Minute)
	add(currentStart.Add(7*24*time.Hour), currentStart.Add(time.Minute), 36, 40*time.Minute)
	return out
}

func TestUsageForecastProceduresReadQuotaReadings(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	source := &quotaReadings{spans: codexReadings(now)}
	h := NewHandler(&fakeStore{}, func() time.Time { return now })
	if _, err := h.UsageForecast(context.Background(), connect.NewRequest(&measurepb.UsageForecastRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("without readings the forecast is unavailable, got %v", err)
	}
	h.SetUsageForecast(source)

	forecast, err := h.UsageForecast(context.Background(), connect.NewRequest(&measurepb.UsageForecastRequest{Provider: "openai"}))
	if err != nil {
		t.Fatal(err)
	}
	if source.provider != "openai" || !source.since.Equal(now.Add(-usageforecast.CurrentLookback)) {
		t.Fatalf("read provider=%q since=%v", source.provider, source.since)
	}
	if len(forecast.Msg.GetWindows()) != 1 {
		t.Fatalf("windows=%v", forecast.Msg.GetWindows())
	}
	w := forecast.Msg.GetWindows()[0]
	if w.GetStatus() != "will_exhaust" || w.GetUsedPercent() != 36 || !w.GetHasPercent() || w.GetWindowMinutes() != 10080 {
		t.Fatalf("window=%v", w)
	}
	if !w.GetEarlyReset() || w.GetPreviousFinalPercent() != 40 || w.GetProjectedExhaustionAt() == "" || w.GetExhaustionBandStart() == "" || w.GetExhaustionBandEnd() == "" {
		t.Fatalf("projection/early reset fields=%v", w)
	}
	if len(w.GetBurn()) != 3 || w.GetBurn()[2].GetHorizon() != "24h" || w.GetCentralPercentPerHour() <= 1 || w.GetLastingPercentPerHour() <= 0 || len(w.GetSeries()) != 37 {
		t.Fatalf("burn=%v central=%v lasting=%v series=%d", w.GetBurn(), w.GetCentralPercentPerHour(), w.GetLastingPercentPerHour(), len(w.GetSeries()))
	}
	if w.GetLastReadingPercent() != 36 || w.GetObservedAt() == "" || w.GetCalibrationFactor() != 1 || w.GetCalibrationWindows() != 1 {
		t.Fatalf("last reading / calibration fields=%v", w)
	}
	if p := forecast.Msg.GetProvenance(); p.GetSourceTable() != "quota_observations" || p.GetRowCount() != int64(len(source.spans)) || forecast.Msg.GetDefinitionId() == "" {
		t.Fatalf("provenance=%v definition=%q", p, forecast.Msg.GetDefinitionId())
	}

	backtest, err := h.UsageForecastBacktest(context.Background(), connect.NewRequest(&measurepb.UsageForecastBacktestRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !source.since.IsZero() {
		t.Fatalf("the backtest reads the whole history, since=%v", source.since)
	}
	if len(backtest.Msg.GetWindows()) != 1 || !backtest.Msg.GetWindows()[0].GetEarlyReset() || backtest.Msg.GetWindows()[0].GetFinalPercent() != 40 || len(backtest.Msg.GetSummaries()) != 4 {
		t.Fatalf("backtest=%v", backtest.Msg)
	}
	if s := backtest.Msg.GetSummaries(); s[0].GetPopulation() != usageforecast.PopulationRecent || s[1].GetPopulation() != usageforecast.PopulationAll {
		t.Fatalf("each checkpoint reports the last-5-windows population, then all windows: %v", s)
	}
	if v := backtest.Msg.GetValidity(); v.GetState() != "unreliable" || v.GetSampleSize() != 1 {
		t.Fatalf("one scored window must be reported unreliable: %v", v)
	}
}
