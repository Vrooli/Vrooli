package main

import (
	"context"
	"testing"
	"time"

	"knowledge-observatory/internal/quality"
	qualitymocks "knowledge-observatory/internal/quality/mocks"
)

// TestRetentionCollapsesOldSamplesAndKeepsRecentOnes is the guard on the
// unbounded-growth defect: quality_metrics grew to 1.23M rows because nothing
// ever reclaimed it.
func TestRetentionCollapsesOldSamplesAndKeepsRecentOnes(t *testing.T) {
	repo := qualitymocks.New()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()

	// Two old days, four samples each, for two collections.
	for _, collection := range []string{"alpha", "beta"} {
		for _, day := range []time.Time{now.AddDate(0, 0, -90), now.AddDate(0, 0, -60)} {
			for h := range 4 {
				if _, err := repo.InsertMetric(ctx, quality.Metric{
					CollectionName: collection,
					MeasuredAt:     day.Add(time.Duration(h) * time.Hour),
				}); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	// Six recent samples that must survive untouched.
	for h := range 6 {
		if _, err := repo.InsertMetric(ctx, quality.Metric{
			CollectionName: "alpha",
			MeasuredAt:     now.Add(-time.Duration(h) * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}

	before, _ := repo.CountMetrics(ctx)
	if before != 22 {
		t.Fatalf("seeded %d rows, want 22", before)
	}

	r := &Retention{Repo: repo, Now: func() time.Time { return now }}
	if err := r.ApplyOnce(ctx); err != nil {
		t.Fatalf("apply: %v", err)
	}

	after, _ := repo.CountMetrics(ctx)
	// 2 collections x 2 old days = 4 survivors, plus the 6 recent samples.
	if after != 10 {
		t.Errorf("kept %d rows, want 10 (4 downsampled + 6 recent)", after)
	}

	// The recent window must keep full resolution.
	recent := 0
	for _, m := range repo.Metrics {
		if m.MeasuredAt.After(now.AddDate(0, 0, -30)) {
			recent++
		}
	}
	if recent != 6 {
		t.Errorf("recent samples = %d, want all 6 kept at full resolution", recent)
	}
}

func TestRetentionIsIdempotent(t *testing.T) {
	repo := qualitymocks.New()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()

	for h := range 5 {
		if _, err := repo.InsertMetric(ctx, quality.Metric{
			CollectionName: "alpha",
			MeasuredAt:     now.AddDate(0, 0, -90).Add(time.Duration(h) * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}

	r := &Retention{Repo: repo, Now: func() time.Time { return now }}
	if err := r.ApplyOnce(ctx); err != nil {
		t.Fatal(err)
	}
	first, _ := repo.CountMetrics(ctx)
	if err := r.ApplyOnce(ctx); err != nil {
		t.Fatal(err)
	}
	second, _ := repo.CountMetrics(ctx)

	if first != 1 || second != 1 {
		t.Errorf("counts = %d then %d, want 1 then 1", first, second)
	}
}

// TestRetentionDefaultsToThirtyDays pins the operator-approved window so a
// later edit cannot silently shrink or widen it.
func TestRetentionDefaultsToThirtyDays(t *testing.T) {
	r := &Retention{}
	if got := r.window(); got != 30*24*time.Hour {
		t.Errorf("window = %v, want 30 days", got)
	}
	if got := r.interval(); got <= 0 {
		t.Errorf("interval = %v, want a positive default", got)
	}
}

func TestRetentionWithoutRepositoryIsInert(t *testing.T) {
	var r *Retention
	if err := r.ApplyOnce(context.Background()); err != nil {
		t.Errorf("nil retention should be inert, got %v", err)
	}
	if err := (&Retention{}).ApplyOnce(context.Background()); err != nil {
		t.Errorf("retention without a repo should be inert, got %v", err)
	}
}
