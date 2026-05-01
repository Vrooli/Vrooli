package fixtures

import (
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
)

// StatsSnapshot groups the stats values most handler/orchestrator tests seed
// together.
type StatsSnapshot struct {
	StatusCounts     *repository.RunStatusCounts
	SuccessRate      float64
	DurationStats    *repository.DurationStats
	CostStats        *repository.CostStats
	RunnerBreakdown  []*repository.RunnerBreakdown
	ProfileBreakdown []*repository.ProfileBreakdown
	ModelBreakdown   []*repository.ModelBreakdown
	TimeSeries       []*repository.TimeSeriesBucket
}

// StatsSnapshotOpt mutates a StatsSnapshot during construction.
type StatsSnapshotOpt func(*StatsSnapshot)

// NewStatsSnapshot returns representative aggregate stats for handler tests.
func NewStatsSnapshot(opts ...StatsSnapshotOpt) StatsSnapshot {
	now := time.Now().Truncate(time.Hour)
	snapshot := StatsSnapshot{
		StatusCounts: &repository.RunStatusCounts{
			Pending:     2,
			Running:     3,
			Complete:    17,
			Failed:      3,
			Cancelled:   1,
			NeedsReview: 1,
			Total:       27,
		},
		SuccessRate: 0.85,
		DurationStats: &repository.DurationStats{
			AvgMs: 45000,
			P50Ms: 30000,
			P95Ms: 120000,
			P99Ms: 180000,
			MinMs: 5000,
			MaxMs: 200000,
			Count: 20,
		},
		CostStats: &repository.CostStats{
			TotalCostUSD:    12.50,
			AvgCostUSD:      0.625,
			InputTokens:     500000,
			OutputTokens:    100000,
			CacheReadTokens: 50000,
			TotalTokens:     650000,
		},
		RunnerBreakdown: []*repository.RunnerBreakdown{
			{
				RunnerType:    domain.RunnerTypeClaudeCode,
				RunCount:      15,
				SuccessCount:  12,
				FailedCount:   2,
				TotalCostUSD:  10.00,
				AvgDurationMs: 40000,
			},
			{
				RunnerType:    domain.RunnerTypeCodex,
				RunCount:      5,
				SuccessCount:  4,
				FailedCount:   1,
				TotalCostUSD:  2.50,
				AvgDurationMs: 55000,
			},
		},
		TimeSeries: []*repository.TimeSeriesBucket{
			{
				Timestamp:     now.Add(-2 * time.Hour),
				RunsStarted:   5,
				RunsCompleted: 4,
				RunsFailed:    1,
				TotalCostUSD:  3.00,
				AvgDurationMs: 35000,
			},
			{
				Timestamp:     now.Add(-1 * time.Hour),
				RunsStarted:   8,
				RunsCompleted: 7,
				RunsFailed:    0,
				TotalCostUSD:  5.50,
				AvgDurationMs: 42000,
			},
		},
	}
	for _, opt := range opts {
		opt(&snapshot)
	}
	return snapshot
}

func WithStatsStatusCounts(counts *repository.RunStatusCounts) StatsSnapshotOpt {
	return func(s *StatsSnapshot) { s.StatusCounts = counts }
}

func WithStatsSuccessRate(rate float64) StatsSnapshotOpt {
	return func(s *StatsSnapshot) { s.SuccessRate = rate }
}

func WithStatsTimeSeries(buckets ...*repository.TimeSeriesBucket) StatsSnapshotOpt {
	return func(s *StatsSnapshot) {
		s.TimeSeries = append([]*repository.TimeSeriesBucket(nil), buckets...)
	}
}
