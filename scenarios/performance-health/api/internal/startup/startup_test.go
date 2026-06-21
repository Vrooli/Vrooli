package startup

import (
	"context"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fakeRunner struct {
	m   Measurement
	err error
}

func (f fakeRunner) Measure(context.Context, string, time.Duration) (Measurement, error) {
	return f.m, f.err
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db := openSQLite(t)
	return NewStore(db)
}

// [REQ:PH-STARTUP-001] Benchmark measures via the runner seam and persists the
// measurement to the trend store.
func TestBenchmarkPersists(t *testing.T) {
	store := newTestStore(t)
	svc := NewService(fakeRunner{m: Measurement{Scenario: "demo", TimeToHealthyMs: 1200, Healthy: true}}, store, "performance-health")
	m, err := svc.Benchmark(context.Background(), "demo", 0)
	if err != nil {
		t.Fatalf("Benchmark: %v", err)
	}
	if !m.Healthy || m.TimeToHealthyMs != 1200 {
		t.Fatalf("unexpected measurement: %#v", m)
	}
	series, err := store.Series(context.Background(), "demo", 10)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 persisted measurement, got %d", len(series))
	}
}

func TestTrendReadsNewestFirst(t *testing.T) {
	store := newTestStore(t)
	for i, ms := range []int64{100, 200, 300} {
		if err := store.Insert(context.Background(), Measurement{
			Scenario:        "demo",
			CapturedAt:      time.Now().Add(time.Duration(i) * time.Second),
			TimeToHealthyMs: ms,
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	svc := NewService(fakeRunner{}, store, "performance-health")
	series, err := svc.Trend(context.Background(), "demo", 10)
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}
	if len(series) != 3 || series[0].TimeToHealthyMs != 300 {
		t.Fatalf("expected newest-first ordering, got %#v", series)
	}
}
