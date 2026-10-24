package trends

import (
	"context"
	"testing"
	"time"
)

func TestPreviousWindowTrendIsOptInAndPolarityIsSeparate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	policy := Policy{Enabled: true, WindowSeconds: 7 * 24 * 60 * 60, Comparison: "previous_window", Aggregation: "latest", Direction: "higher_is_better", MinimumObservations: 2, NeutralPercent: 1}
	for i, value := range []float64{100, 110} {
		_ = store.Record(ctx, Observation{MetricID: "visitors", Source: "lpbs", Value: value, Observed: at.Add(-time.Duration(13-i) * 24 * time.Hour)})
	}
	for i, value := range []float64{120, 130} {
		_ = store.Record(ctx, Observation{MetricID: "visitors", Source: "lpbs", Value: value, Observed: at.Add(-time.Duration(6-i) * 24 * time.Hour)})
	}
	result, err := store.Trend(ctx, "visitors", "lpbs", policy, at)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "meaningful" || result.Movement != "up" || result.Polarity != "favorable" || result.Percent == nil || *result.Percent != 18.181818181818183 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTrendSuppressesInsufficientHistoryAndDisabledPolicy(t *testing.T) {
	store := NewMemoryStore()
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	policy := Policy{Enabled: true, WindowSeconds: 3600, Comparison: "previous_window", MinimumObservations: 2}
	_ = store.Record(context.Background(), Observation{MetricID: "x", Source: "s", Value: 1, Observed: at.Add(-30 * time.Minute)})
	got, _ := store.Trend(context.Background(), "x", "s", policy, at)
	if got.State != "insufficient_data" {
		t.Fatalf("state=%q", got.State)
	}
	policy.Enabled = false
	got, _ = store.Trend(context.Background(), "x", "s", policy, at)
	if got.State != "not_applicable" {
		t.Fatalf("disabled state=%q", got.State)
	}
}
