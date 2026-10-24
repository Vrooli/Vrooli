package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
)

func TestRunDurationStatisticsUsesNearestRankPercentiles(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}
	occurred := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	for _, duration := range []int64{10, 20, 100} {
		started := occurred
		ended := started.Add(time.Duration(duration) * time.Millisecond)
		if err := repo.ReplaceRun(context.Background(), invocationreadmodel.RunFact{
			RunID: "duration-" + string(rune(duration)), OccurredAt: occurred,
			CreatedAt: occurred, StartedAt: &started, EndedAt: &ended,
			DurationMS: duration, Status: "complete", ProjectedAt: occurred,
		}); err != nil {
			t.Fatalf("seed duration %d: %v", duration, err)
		}
	}

	stats, err := repo.RunDurationStatistics(context.Background(), invocationreadmodel.Filter{
		From: &occurred, To: ptrTime(occurred.Add(time.Hour)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Count != 3 || stats.MinDurationMS != 10 || stats.MaxDurationMS != 100 ||
		stats.P50DurationMS != 20 || stats.P95DurationMS != 100 || stats.P99DurationMS != 100 {
		t.Fatalf("duration stats = %+v", stats)
	}
}
