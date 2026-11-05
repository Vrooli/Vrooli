package measures

import (
	"context"
	"testing"
)

// countProjection is a trivial Projection: it counts events per EventType. It
// stands in for an adopter's watermark-folded aggregate.
type countProjection struct {
	counts map[string]int
}

func newCountProjection() *countProjection { return &countProjection{counts: map[string]int{}} }

func (p *countProjection) Apply(e Event) { p.counts[e.EventType]++ }
func (p *countProjection) Reset()        { p.counts = map[string]int{} }

func TestMemoryEventLogAppendSinceMaxID(t *testing.T) {
	ctx := context.Background()
	log := NewMemoryEventLog()
	for i := 0; i < 3; i++ {
		if _, err := log.Append(ctx, Event{EntityType: "backlog", EntityID: "b1", EventType: "backlog.created"}); err != nil {
			t.Fatal(err)
		}
	}
	max, _ := log.MaxID(ctx)
	if max != 3 {
		t.Errorf("MaxID = %d, want 3", max)
	}
	since, _ := log.Since(ctx, 1, 0)
	if len(since) != 2 || since[0].ID != 2 {
		t.Errorf("Since(1) = %+v, want IDs 2,3", since)
	}
	all, _ := log.All(ctx)
	if len(all) != 3 {
		t.Errorf("All len = %d, want 3", len(all))
	}
}

func TestReadModelRebuildAndIncrementalRefresh(t *testing.T) {
	ctx := context.Background()
	log := NewMemoryEventLog()
	_, _ = log.Append(ctx, Event{EventType: "a"})
	_, _ = log.Append(ctx, Event{EventType: "a"})
	_, _ = log.Append(ctx, Event{EventType: "b"})

	proj := newCountProjection()
	rm := NewReadModel(log, proj, WithBatchSize(2)) // small batch exercises paging

	if err := rm.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
	if proj.counts["a"] != 2 || proj.counts["b"] != 1 {
		t.Fatalf("after rebuild counts = %+v", proj.counts)
	}
	if rm.Watermark() != 3 {
		t.Errorf("watermark = %d, want 3", rm.Watermark())
	}

	// Append more, then incremental refresh folds only the new events.
	_, _ = log.Append(ctx, Event{EventType: "a"})
	if err := rm.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if proj.counts["a"] != 3 {
		t.Errorf("after refresh a = %d, want 3", proj.counts["a"])
	}
	if rm.Watermark() != 4 {
		t.Errorf("watermark = %d, want 4", rm.Watermark())
	}

	// Refresh with no new events is a no-op.
	if err := rm.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if proj.counts["a"] != 3 {
		t.Errorf("idempotent refresh changed counts: %+v", proj.counts)
	}
}
