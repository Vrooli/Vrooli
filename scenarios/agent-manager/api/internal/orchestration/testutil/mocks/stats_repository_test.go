package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/repository"
)

func TestFakeStatsRepository_Defaults(t *testing.T) {
	stats := NewFakeStatsRepository()

	counts, err := stats.GetRunStatusCounts(context.Background(), repository.StatsFilter{})
	if err != nil {
		t.Fatalf("GetRunStatusCounts returned error: %v", err)
	}
	if counts == nil {
		t.Fatal("expected default status counts")
	}

	duration, err := stats.GetDurationStats(context.Background(), repository.StatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats returned error: %v", err)
	}
	if duration == nil {
		t.Fatal("expected default duration stats")
	}
}

func TestFakeStatsRepository_ErrorKnobs(t *testing.T) {
	stats := NewFakeStatsRepository()
	want := errors.New("stats unavailable")
	stats.StatusCountsErr = want

	_, err := stats.GetRunStatusCounts(context.Background(), repository.StatsFilter{})
	if !errors.Is(err, want) {
		t.Fatalf("expected configured error, got %v", err)
	}
}

func TestFakeStatsRepository_ReturnsSliceCopies(t *testing.T) {
	stats := NewFakeStatsRepository()
	stats.PopularModels = []string{"claude"}

	got, err := stats.GetPopularModels(context.Background(), timeNow(), 10)
	if err != nil {
		t.Fatalf("GetPopularModels returned error: %v", err)
	}
	got[0] = "changed"

	if stats.PopularModels[0] != "claude" {
		t.Fatal("expected stored model slice to be protected from caller mutation")
	}
}

func TestFakeStatsRepositoryForwardsConfiguredValuesAndErrorsAcrossQueries(t *testing.T) {
	ctx := context.Background()
	filter := repository.StatsFilter{}
	want := errors.New("injected")
	stats := NewFakeStatsRepository()
	stats.SuccessRate = 0.75
	if got, err := stats.GetSuccessRate(ctx, filter); err != nil || got != 0.75 {
		t.Fatalf("success rate=%v err=%v", got, err)
	}
	stats.SuccessRateErr = want
	if _, err := stats.GetSuccessRate(ctx, filter); !errors.Is(err, want) {
		t.Fatalf("success error=%v", err)
	}

	cases := []struct {
		name   string
		setErr func()
		call   func() error
	}{
		{"duration", func() { stats.DurationStatsErr = want }, func() error { _, err := stats.GetDurationStats(ctx, filter); return err }},
		{"cost", func() { stats.CostStatsErr = want }, func() error { _, err := stats.GetCostStats(ctx, filter); return err }},
		{"runner", func() { stats.RunnerBreakdownErr = want }, func() error { _, err := stats.GetRunnerBreakdown(ctx, filter); return err }},
		{"profile", func() { stats.ProfileBreakdownErr = want }, func() error { _, err := stats.GetProfileBreakdown(ctx, filter, 1); return err }},
		{"model", func() { stats.ModelBreakdownErr = want }, func() error { _, err := stats.GetModelBreakdown(ctx, filter, 1); return err }},
		{"tool", func() { stats.ToolUsageStatsErr = want }, func() error { _, err := stats.GetToolUsageStats(ctx, filter, 1); return err }},
		{"model-runs", func() { stats.ModelRunUsageErr = want }, func() error { _, err := stats.GetModelRunUsage(ctx, filter, "model", 1); return err }},
		{"tool-runs", func() { stats.ToolRunUsageErr = want }, func() error { _, err := stats.GetToolRunUsage(ctx, filter, "tool", 1); return err }},
		{"tool-by-model", func() { stats.ToolUsageByModelErr = want }, func() error { _, err := stats.GetToolUsageByModel(ctx, filter, "tool", 1); return err }},
		{"patterns", func() { stats.ErrorPatternsErr = want }, func() error { _, err := stats.GetErrorPatterns(ctx, filter, 1); return err }},
		{"series", func() { stats.TimeSeriesErr = want }, func() error { _, err := stats.GetTimeSeries(ctx, filter, time.Minute); return err }},
		{"popular", func() { stats.PopularModelsErr = want }, func() error { _, err := stats.GetPopularModels(ctx, time.Time{}, 1); return err }},
		{"recent", func() { stats.RecentModelsErr = want }, func() error { _, err := stats.GetRecentModels(ctx, 1); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setErr()
			if err := tc.call(); !errors.Is(err, want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func timeNow() time.Time {
	return time.Time{}
}
