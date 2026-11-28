package trends

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type toggledStore struct {
	available bool
	rows      []Observation
}

func (s *toggledStore) Record(_ context.Context, o Observation) error {
	if !s.available {
		return errors.New("injected storage loss")
	}
	for _, row := range s.rows {
		if row.MetricID == o.MetricID && row.Source == o.Source && row.Observed.Equal(o.Observed) {
			return nil
		}
	}
	s.rows = append(s.rows, o)
	return nil
}
func (s *toggledStore) Trend(_ context.Context, metric, source string, p Policy, at time.Time) (Result, error) {
	if !s.available {
		return Result{}, errors.New("injected storage loss")
	}
	return calculate(s.rows, p, at), nil
}
func (s *toggledStore) Close() error { return nil }

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

func TestResilientStoreRetainsObservationAcrossPrimaryLossAndFlushesOnce(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	primary := &toggledStore{available: true}
	store := NewResilientStore(primary)
	policy := Policy{Enabled: true, WindowSeconds: 10 * 60, Comparison: "previous_window", Aggregation: "latest", MinimumObservations: 1}
	previous := Observation{MetricID: "visitors", Source: "swarm", Value: 90, Observed: at.Add(-15 * time.Minute)}
	if err := store.Record(ctx, previous); err != nil {
		t.Fatal(err)
	}

	primary.available = false
	retained := Observation{MetricID: "visitors", Source: "swarm", Value: 120, Observed: at.Add(-5 * time.Minute)}
	if err := store.Record(ctx, retained); err != nil {
		t.Fatalf("buffer during provider loss: %v", err)
	}
	if err := store.Record(ctx, Observation{MetricID: retained.MetricID, Source: retained.Source, Value: 999, Observed: retained.Observed}); err != nil {
		t.Fatalf("duplicate observation during provider loss: %v", err)
	}
	duringLoss, err := store.Trend(ctx, "visitors", "swarm", policy, at)
	if err != nil || duringLoss.State != "provider_unavailable" || duringLoss.Observations != 1 {
		t.Fatalf("loss state=%+v err=%v", duringLoss, err)
	}

	primary.available = true
	recovered, err := store.Trend(ctx, "visitors", "swarm", policy, at)
	if err != nil || recovered.State != "meaningful" || recovered.Delta != 30 || len(primary.rows) != 2 || primary.rows[1].Value != retained.Value {
		t.Fatalf("recovered=%+v persisted=%s err=%v", recovered, fmt.Sprint(primary.rows), err)
	}
	_, _ = store.Trend(ctx, "visitors", "swarm", policy, at)
	if len(primary.rows) != 2 {
		t.Fatalf("recovery replayed a saved observation: %d rows", len(primary.rows))
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close recovered store: %v", err)
	}
}
