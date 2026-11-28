package measures

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	gomeasures "github.com/vrooli/measures-go"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	scsmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/measures"

	"scenario-completeness-scoring/internal/scoring"
)

type fakeStore struct {
	count        int64
	average      float64
	series       []scoring.ScoreSeriesPoint
	lastRank     int
	lastWindow   scoring.MeasureWindow
	countCalls   int
	averageCalls int
	seriesCalls  int
}

func (f *fakeStore) CountLatestBelowRung(_ context.Context, rank int, window scoring.MeasureWindow) (int64, error) {
	f.lastRank = rank
	f.lastWindow = window
	f.countCalls++
	return f.count, nil
}

func (f *fakeStore) AverageLatestComposite(_ context.Context, window scoring.MeasureWindow) (float64, bool, error) {
	f.lastWindow = window
	f.averageCalls++
	return f.average, true, nil
}

func (f *fakeStore) FleetScoreSeries(_ context.Context, window scoring.MeasureWindow) ([]scoring.ScoreSeriesPoint, error) {
	f.lastWindow = window
	f.seriesCalls++
	return f.series, nil
}

func fixedNow() time.Time { return time.Date(2026, 6, 9, 15, 0, 0, 0, time.UTC) }

func TestDeclarationsValidFullTier(t *testing.T) {
	decls := Declarations()
	if len(decls) != 3 {
		t.Fatalf("want 3 declarations, got %d", len(decls))
	}
	for _, d := range decls {
		if err := d.Validate(); err != nil {
			t.Errorf("declaration %s invalid: %v", d.Name, err)
		}
		if d.Domain != "scoring" {
			t.Errorf("declaration %s domain = %q, want scoring", d.Name, d.Domain)
		}
		if d.Effect != gomeasures.EffectRead || !d.RunEligible {
			t.Errorf("declaration %s must be read + run-eligible", d.Name)
		}
		for _, name := range d.ParamNames() {
			p := d.Params[name]
			if !p.IsCanonical() && !p.IsConstrained() {
				t.Errorf("declaration %s param %s is not full-tier: %+v", d.Name, name, p)
			}
		}
	}
}

func TestRegistryExecuteCountBelowRung(t *testing.T) {
	store := &fakeStore{count: 12}
	reg, err := NewRegistry(store, fixedNow)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	res, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{
		Measure: MeasureFleetBelowRung,
		Params: map[string]string{
			"window": "this_week",
			"rung":   scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R3.String(),
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if res.Value != "12" {
		t.Fatalf("Execute() value = %q, want 12", res.Value)
	}
	if store.lastRank != 3 {
		t.Fatalf("rank = %d, want 3", store.lastRank)
	}
	if strings.TrimSpace(res.Provenance.ExecutedQuery) == "" {
		t.Fatal("provenance.executed_query must be set")
	}
	if res.Provenance.ComputedAt.IsZero() {
		t.Fatal("provenance.computed_at must be stamped")
	}
}

func TestRegistryDefaultWindow(t *testing.T) {
	store := &fakeStore{average: 74.25}
	reg, _ := NewRegistry(store, fixedNow)
	res, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureAverageComposite})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if res.Value != "74.25" {
		t.Fatalf("Execute() value = %q, want 74.25", res.Value)
	}
	if !store.lastWindow.To.Equal(fixedNow()) {
		t.Fatalf("default window to = %v, want %v", store.lastWindow.To, fixedNow())
	}
}

func TestConnectHandlerSharesAggregates(t *testing.T) {
	store := &fakeStore{
		count:   4,
		average: 88.5,
		series: []scoring.ScoreSeriesPoint{{
			Bucket: time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
			Score:  88.5,
			Count:  2,
		}},
	}
	h := NewHandler(store, fixedNow)
	ctx := context.Background()

	count, err := h.CountFleetBelowRung(ctx, connect.NewRequest(&scsmeasuresv1.CountFleetBelowRungRequest{}))
	if err != nil {
		t.Fatalf("CountFleetBelowRung() error = %v", err)
	}
	if count.Msg.GetCount() != 4 || store.lastRank != 2 {
		t.Fatalf("CountFleetBelowRung() = %d rank %d, want 4 rank 2", count.Msg.GetCount(), store.lastRank)
	}

	avg, err := h.AverageComposite(ctx, connect.NewRequest(&scsmeasuresv1.AverageCompositeRequest{}))
	if err != nil {
		t.Fatalf("AverageComposite() error = %v", err)
	}
	if avg.Msg.GetAverage() != 88.5 {
		t.Fatalf("AverageComposite() = %v, want 88.5", avg.Msg.GetAverage())
	}

	series, err := h.ScoreSeries(ctx, connect.NewRequest(&scsmeasuresv1.ScoreSeriesRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D}},
	}))
	if err != nil {
		t.Fatalf("ScoreSeries() error = %v", err)
	}
	if len(series.Msg.GetPoints()) != 1 || series.Msg.GetPoints()[0].GetBucket() != "2026-06-09" {
		t.Fatalf("ScoreSeries() points = %+v, want 2026-06-09", series.Msg.GetPoints())
	}
}
