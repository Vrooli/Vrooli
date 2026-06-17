package measures

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"image-tools/internal/testutil/db"
)

func newRecorder(t *testing.T) *Recorder {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return NewRecorder(d)
}

// TestRecordAndAggregate proves samples are recorded per op/model and that
// latency percentiles, throughput, and outcome counts are queryable.
func TestRecordAndAggregate(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)

	// 10 upscale runs with durations 100..1000ms, all succeeded.
	for i := 1; i <= 10; i++ {
		if err := r.Record(ctx, Sample{
			Operation:  "upscale",
			ModelID:    "real-esrgan",
			Tier:       "local-cpu",
			State:      "succeeded",
			DurationMS: int64(i * 100),
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	// One failed resize.
	if err := r.Record(ctx, Sample{Operation: "resize", State: "failed", DurationMS: 5}); err != nil {
		t.Fatalf("record resize: %v", err)
	}

	stats, err := r.Stats(ctx, "")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(stats))
	}

	var up OpStat
	for _, s := range stats {
		if s.Operation == "upscale" {
			up = s
		}
	}
	if up.Count != 10 || up.Succeeded != 10 {
		t.Fatalf("upscale count/succeeded = %d/%d, want 10/10", up.Count, up.Succeeded)
	}
	// Nearest-rank p50 of 100..1000 (n=10) = rank 5 → 500; p95 = rank ceil(9.5)=10 → 1000.
	if up.LatencyP50MS != 500 {
		t.Fatalf("p50 = %d, want 500", up.LatencyP50MS)
	}
	if up.LatencyP95MS != 1000 {
		t.Fatalf("p95 = %d, want 1000", up.LatencyP95MS)
	}
}

// TestStatsFilterByOperation proves the operation filter narrows results.
func TestStatsFilterByOperation(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	_ = r.Record(ctx, Sample{Operation: "resize", State: "succeeded", DurationMS: 10})
	_ = r.Record(ctx, Sample{Operation: "crop", State: "succeeded", DurationMS: 20})

	stats, err := r.Stats(ctx, "crop")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 1 || stats[0].Operation != "crop" {
		t.Fatalf("expected only crop, got %+v", stats)
	}
}

// TestObserveRecordsOpLevelFacts proves the job-driven path records latency +
// queue wait + outcome without a model id.
func TestObserveRecordsOpLevelFacts(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	if err := r.Observe(ctx, JobLike{Operation: "denoise", State: "succeeded", DurationMS: 250, QueueMS: 40}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	stats, err := r.Stats(ctx, "denoise")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 1 || stats[0].LatencyP50MS != 250 || stats[0].QueueWaitP50 != 40 {
		t.Fatalf("unexpected denoise stat: %+v", stats)
	}
}

// TestFallbackCounted proves fallback-tier usage is tallied.
func TestFallbackCounted(t *testing.T) {
	ctx := context.Background()
	r := newRecorder(t)
	_ = r.Record(ctx, Sample{Operation: "text_to_image", State: "succeeded", DurationMS: 9000, Tier: "local-cpu", FallbackUsed: true})
	_ = r.Record(ctx, Sample{Operation: "text_to_image", State: "succeeded", DurationMS: 800, Tier: "local-gpu"})

	stats, err := r.Stats(ctx, "text_to_image")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats[0].FallbackCount != 1 {
		t.Fatalf("fallback count = %d, want 1", stats[0].FallbackCount)
	}
}
