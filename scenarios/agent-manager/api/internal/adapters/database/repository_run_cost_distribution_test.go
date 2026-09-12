package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
)

func TestRunCostDistributionUsesNearestRankAndFixedBuckets(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationReadModelRepository{db: db}

	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	occurred := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

	seed := func(id string, tokens int64, cost float64) {
		t.Helper()
		fact := invocationreadmodel.RunFact{
			RunID:         id,
			OccurredAt:    occurred,
			CreatedAt:     occurred,
			Status:        "complete",
			ProfileID:     "profile",
			RunnerType:    "codex",
			Model:         "test-model",
			Tag:           "test",
			WorkloadKind:  "adhoc",
			TotalTokens:   tokens,
			TotalCostUSD:  cost,
			CostTimeBasis: "terminal_projection",
			TimeBasis:     "ingestion",
			ProjectedAt:   occurred,
		}
		if err := repo.ReplaceRun(context.Background(), fact); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	seed("run-50k", 50_000, 0.01)
	seed("run-200k", 200_000, 0.02)
	seed("run-2m", 2_000_000, 0.10)
	seed("run-30m", 30_000_000, 1.00)
	// A run without observed tokens must be excluded from the distribution.
	seed("run-no-usage", 0, 0)

	dist, err := repo.RunCostDistribution(context.Background(), invocationreadmodel.Filter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("RunCostDistribution: %v", err)
	}
	if dist.SampleSize != 4 {
		t.Fatalf("sample size = %d, want 4", dist.SampleSize)
	}
	if dist.P50Tokens != 200_000 || dist.P90Tokens != 30_000_000 || dist.P95Tokens != 30_000_000 || dist.P99Tokens != 30_000_000 || dist.MaxTokens != 30_000_000 {
		t.Fatalf("token percentiles = p50 %d p90 %d p95 %d p99 %d max %d", dist.P50Tokens, dist.P90Tokens, dist.P95Tokens, dist.P99Tokens, dist.MaxTokens)
	}
	if dist.P50CostUSD != 0.02 || dist.P90CostUSD != 1.00 || dist.MaxCostUSD != 1.00 {
		t.Fatalf("cost percentiles = p50 %.4f p90 %.4f max %.4f", dist.P50CostUSD, dist.P90CostUSD, dist.MaxCostUSD)
	}
	want := map[string]int64{"<100K": 1, "100K-500K": 1, "500K-1M": 0, "1M-5M": 1, "5M-25M": 0, "25M+": 1}
	if len(dist.TokenBuckets) != len(want) {
		t.Fatalf("bucket count = %d, want %d", len(dist.TokenBuckets), len(want))
	}
	for _, bucket := range dist.TokenBuckets {
		if bucket.RunCount != want[bucket.Label] {
			t.Fatalf("bucket %q = %d, want %d", bucket.Label, bucket.RunCount, want[bucket.Label])
		}
	}
}
