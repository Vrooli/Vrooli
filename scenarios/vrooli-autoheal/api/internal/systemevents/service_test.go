package systemevents

import (
	"context"
	"testing"
	"time"
)

// fakeStore satisfies the Store interface for service tests. It embeds
// MemoryCursorStore so the CursorStore half is provided for free.
type fakeStore struct {
	*MemoryCursorStore
	upserts int
}

func newFakeStore() *fakeStore {
	return &fakeStore{MemoryCursorStore: NewMemoryCursorStore()}
}

func (f *fakeStore) UpsertSystemEvents(_ context.Context, events []Event) (int, int, error) {
	f.upserts++
	return len(events), 0, nil
}
func (f *fakeStore) UpsertSystemEventSource(_ context.Context, _ SourceStatus) error { return nil }
func (f *fakeStore) ListSystemEvents(_ context.Context, _ Filters) (*Response, error) {
	return &Response{}, nil
}
func (f *fakeStore) GetSystemEventSources(_ context.Context) ([]SourceStatus, error) { return nil, nil }

// countingCollector records how many times Collect runs and exposes an
// ExecsAvoided count for the observability assertion.
type countingCollector struct {
	calls   int
	avoided int64
}

func (c *countingCollector) Collect(_ context.Context) ([]Event, []SourceStatus) {
	c.calls++
	return nil, nil
}

func (c *countingCollector) ExecsAvoided() int64 { return c.avoided }

func TestIngestIfDueThrottlesToInterval(t *testing.T) {
	store := newFakeStore()
	collector := &countingCollector{}
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	clock := now
	svc := NewServiceWithCollectors(store, []Collector{collector}, func() time.Time { return clock })
	svc.SetIngestInterval(300 * time.Second)

	// First call: nothing ran yet, so it ingests.
	if _, ran, err := svc.IngestIfDue(context.Background()); err != nil || !ran {
		t.Fatalf("first IngestIfDue ran=%v err=%v, want ran=true", ran, err)
	}
	// Immediately again (no time elapsed): throttled, no ingest.
	if _, ran, _ := svc.IngestIfDue(context.Background()); ran {
		t.Fatal("second IngestIfDue ran within interval, want throttled")
	}
	// Advance just under the interval: still throttled.
	clock = now.Add(299 * time.Second)
	if _, ran, _ := svc.IngestIfDue(context.Background()); ran {
		t.Fatal("IngestIfDue ran before interval elapsed")
	}
	// Advance past the interval: ingests again.
	clock = now.Add(301 * time.Second)
	if _, ran, _ := svc.IngestIfDue(context.Background()); !ran {
		t.Fatal("IngestIfDue did not run after interval elapsed")
	}
	if collector.calls != 2 {
		t.Fatalf("collector ran %d times, want 2 (once per due window)", collector.calls)
	}
}

func TestIngestAlwaysRunsRegardlessOfThrottle(t *testing.T) {
	store := newFakeStore()
	collector := &countingCollector{}
	svc := NewServiceWithCollectors(store, []Collector{collector}, nil)

	// Explicit Ingest (the force path used by the refresh endpoint / startup)
	// always runs, even back-to-back.
	for i := 0; i < 3; i++ {
		if _, err := svc.Ingest(context.Background()); err != nil {
			t.Fatalf("Ingest %d: %v", i, err)
		}
	}
	if collector.calls != 3 {
		t.Fatalf("collector ran %d times, want 3 (Ingest is unthrottled)", collector.calls)
	}
}

func TestSetIngestIntervalClamped(t *testing.T) {
	svc := NewServiceWithCollectors(newFakeStore(), nil, nil)
	svc.SetIngestInterval(10 * time.Second) // below min
	if svc.interval != MinIngestInterval {
		t.Errorf("interval = %s, want clamped to %s", svc.interval, MinIngestInterval)
	}
	svc.SetIngestInterval(2 * time.Hour) // above max
	if svc.interval != MaxIngestInterval {
		t.Errorf("interval = %s, want clamped to %s", svc.interval, MaxIngestInterval)
	}
	svc.SetIngestInterval(0) // reset to default
	if svc.interval != DefaultIngestInterval {
		t.Errorf("interval = %s, want default %s", svc.interval, DefaultIngestInterval)
	}
}

func TestExecsAvoidedSurfacedFromCollector(t *testing.T) {
	collector := &countingCollector{avoided: 7}
	svc := NewServiceWithCollectors(newFakeStore(), []Collector{collector}, nil)
	if got := svc.ExecsAvoided(); got != 7 {
		t.Fatalf("ExecsAvoided = %d, want 7", got)
	}
	summary, err := svc.Ingest(context.Background())
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if summary.ExecsAvoided != 7 {
		t.Fatalf("summary.ExecsAvoided = %d, want 7", summary.ExecsAvoided)
	}
}
