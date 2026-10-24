package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"
)

func TestTimePresetToDurationAndFilterFromPresetAt(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		preset TimePreset
		want   time.Duration
	}{
		{TimePreset6H, 6 * time.Hour},
		{TimePreset12H, 12 * time.Hour},
		{TimePreset24H, 24 * time.Hour},
		{TimePreset7D, 7 * 24 * time.Hour},
		{TimePreset30D, 30 * 24 * time.Hour},
		{TimePreset("unknown"), 24 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(string(tc.preset), func(t *testing.T) {
			if got := TimePresetToDuration(tc.preset); got != tc.want {
				t.Fatalf("duration = %v, want %v", got, tc.want)
			}
			filter := FilterFromPresetAt(tc.preset, now)
			if !filter.Window.End.Equal(now) || !filter.Window.Start.Equal(now.Add(-tc.want)) {
				t.Fatalf("filter = %+v, want %s through %s", filter.Window, now.Add(-tc.want), now)
			}
		})
	}
}

func TestFilterFromPresetUsesAConsistentCurrentWindow(t *testing.T) {
	before := time.Now().Add(-time.Second)
	filter := FilterFromPreset(TimePreset6H)
	after := time.Now().Add(time.Second)
	if !filter.Window.Start.Equal(filter.Window.End.Add(-6 * time.Hour)) {
		t.Fatalf("window=%+v", filter.Window)
	}
	if filter.Window.End.Before(before) || filter.Window.End.After(after) {
		t.Fatalf("end=%s outside current interval", filter.Window.End)
	}
}

func TestStatsOrchestratorSummaryCachesAndExpiresWithInjectedClock(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	fake := mocks.NewFakeStatsRepository()
	fake.StatusCounts = &repository.RunStatusCounts{Complete: 2, Failed: 1, Total: 3}
	fake.SuccessRate = 2.0 / 3.0
	service := NewStatsOrchestrator(fake, func() time.Time { return now })
	filter := FilterFromPresetAt(TimePreset24H, now)
	first, err := service.GetSummary(context.Background(), filter)
	if err != nil || first.StatusCounts.Total != 3 || first.SuccessRate != 2.0/3.0 {
		t.Fatalf("first summary = %+v, err=%v", first, err)
	}
	// A repository failure after the first request must not affect an unexpired
	// cached summary. Advancing the fake clock forces the fresh-query path.
	fake.StatusCountsErr = errors.New("repository unavailable")
	if cached, err := service.GetSummary(context.Background(), filter); err != nil || cached != first {
		t.Fatalf("cached summary = %+v, err=%v", cached, err)
	}
	now = now.Add(31 * time.Second)
	if got, err := service.GetSummary(context.Background(), filter); err == nil || got != nil {
		t.Fatalf("expired summary = %+v, err=%v, want repository error", got, err)
	}
}

func TestStatsOrchestratorForwardsEveryQuery(t *testing.T) {
	ctx := context.Background()
	fake := mocks.NewFakeStatsRepository()
	service := NewStatsOrchestrator(fake)
	filter := repository.StatsFilter{}
	if _, err := service.GetStatusCounts(ctx, filter); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetSuccessRate(ctx, filter); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetDurationStats(ctx, filter); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetCostStats(ctx, filter); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRunnerBreakdown(ctx, filter); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetProfileBreakdown(ctx, filter, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetModelBreakdown(ctx, filter, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetToolUsageStats(ctx, filter, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetModelRunUsage(ctx, filter, "model", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetToolRunUsage(ctx, filter, "tool", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetToolUsageByModel(ctx, filter, "tool", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetErrorPatterns(ctx, filter, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetTimeSeries(ctx, filter, time.Hour); err != nil {
		t.Fatal(err)
	}
	if got := FilterFromTimeRange(time.Time{}, time.Time{}); !got.Window.Start.IsZero() || !got.Window.End.IsZero() {
		t.Fatalf("time range filter = %+v", got)
	}
}
