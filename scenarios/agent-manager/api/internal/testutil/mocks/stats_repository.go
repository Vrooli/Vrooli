package mocks

import (
	"context"
	"time"

	"agent-manager/internal/repository"
)

var _ repository.StatsRepository = (*FakeStatsRepository)(nil)

// FakeStatsRepository is a reusable fake for stats aggregation tests.
type FakeStatsRepository struct {
	StatusCounts     *repository.RunStatusCounts
	SuccessRate      float64
	DurationStats    *repository.DurationStats
	CostStats        *repository.CostStats
	RunnerBreakdown  []*repository.RunnerBreakdown
	ProfileBreakdown []*repository.ProfileBreakdown
	ModelBreakdown   []*repository.ModelBreakdown
	ToolUsageStats   []*repository.ToolUsageStats
	ModelRunUsage    []*repository.ModelRunUsage
	ToolRunUsage     []*repository.ToolRunUsage
	ToolUsageByModel []*repository.ToolUsageModelBreakdown
	ErrorPatterns    []*repository.ErrorPattern
	TimeSeries       []*repository.TimeSeriesBucket
	PopularModels    []string
	RecentModels     []string

	StatusCountsErr     error
	SuccessRateErr      error
	DurationStatsErr    error
	CostStatsErr        error
	RunnerBreakdownErr  error
	ProfileBreakdownErr error
	ModelBreakdownErr   error
	ToolUsageStatsErr   error
	ModelRunUsageErr    error
	ToolRunUsageErr     error
	ToolUsageByModelErr error
	ErrorPatternsErr    error
	TimeSeriesErr       error
	PopularModelsErr    error
	RecentModelsErr     error
}

func NewFakeStatsRepository() *FakeStatsRepository {
	return &FakeStatsRepository{}
}

func (f *FakeStatsRepository) GetRunStatusCounts(context.Context, repository.StatsFilter) (*repository.RunStatusCounts, error) {
	if f.StatusCountsErr != nil {
		return nil, f.StatusCountsErr
	}
	if f.StatusCounts != nil {
		return f.StatusCounts, nil
	}
	return &repository.RunStatusCounts{}, nil
}

func (f *FakeStatsRepository) GetSuccessRate(context.Context, repository.StatsFilter) (float64, error) {
	if f.SuccessRateErr != nil {
		return 0, f.SuccessRateErr
	}
	return f.SuccessRate, nil
}

func (f *FakeStatsRepository) GetDurationStats(context.Context, repository.StatsFilter) (*repository.DurationStats, error) {
	if f.DurationStatsErr != nil {
		return nil, f.DurationStatsErr
	}
	if f.DurationStats != nil {
		return f.DurationStats, nil
	}
	return &repository.DurationStats{}, nil
}

func (f *FakeStatsRepository) GetCostStats(context.Context, repository.StatsFilter) (*repository.CostStats, error) {
	if f.CostStatsErr != nil {
		return nil, f.CostStatsErr
	}
	if f.CostStats != nil {
		return f.CostStats, nil
	}
	return &repository.CostStats{}, nil
}

func (f *FakeStatsRepository) GetRunnerBreakdown(context.Context, repository.StatsFilter) ([]*repository.RunnerBreakdown, error) {
	if f.RunnerBreakdownErr != nil {
		return nil, f.RunnerBreakdownErr
	}
	return append([]*repository.RunnerBreakdown(nil), f.RunnerBreakdown...), nil
}

func (f *FakeStatsRepository) GetProfileBreakdown(context.Context, repository.StatsFilter, int) ([]*repository.ProfileBreakdown, error) {
	if f.ProfileBreakdownErr != nil {
		return nil, f.ProfileBreakdownErr
	}
	return append([]*repository.ProfileBreakdown(nil), f.ProfileBreakdown...), nil
}

func (f *FakeStatsRepository) GetModelBreakdown(context.Context, repository.StatsFilter, int) ([]*repository.ModelBreakdown, error) {
	if f.ModelBreakdownErr != nil {
		return nil, f.ModelBreakdownErr
	}
	return append([]*repository.ModelBreakdown(nil), f.ModelBreakdown...), nil
}

func (f *FakeStatsRepository) GetToolUsageStats(context.Context, repository.StatsFilter, int) ([]*repository.ToolUsageStats, error) {
	if f.ToolUsageStatsErr != nil {
		return nil, f.ToolUsageStatsErr
	}
	return append([]*repository.ToolUsageStats(nil), f.ToolUsageStats...), nil
}

func (f *FakeStatsRepository) GetModelRunUsage(context.Context, repository.StatsFilter, string, int) ([]*repository.ModelRunUsage, error) {
	if f.ModelRunUsageErr != nil {
		return nil, f.ModelRunUsageErr
	}
	return append([]*repository.ModelRunUsage(nil), f.ModelRunUsage...), nil
}

func (f *FakeStatsRepository) GetToolRunUsage(context.Context, repository.StatsFilter, string, int) ([]*repository.ToolRunUsage, error) {
	if f.ToolRunUsageErr != nil {
		return nil, f.ToolRunUsageErr
	}
	return append([]*repository.ToolRunUsage(nil), f.ToolRunUsage...), nil
}

func (f *FakeStatsRepository) GetToolUsageByModel(context.Context, repository.StatsFilter, string, int) ([]*repository.ToolUsageModelBreakdown, error) {
	if f.ToolUsageByModelErr != nil {
		return nil, f.ToolUsageByModelErr
	}
	return append([]*repository.ToolUsageModelBreakdown(nil), f.ToolUsageByModel...), nil
}

func (f *FakeStatsRepository) GetErrorPatterns(context.Context, repository.StatsFilter, int) ([]*repository.ErrorPattern, error) {
	if f.ErrorPatternsErr != nil {
		return nil, f.ErrorPatternsErr
	}
	return append([]*repository.ErrorPattern(nil), f.ErrorPatterns...), nil
}

func (f *FakeStatsRepository) GetTimeSeries(context.Context, repository.StatsFilter, time.Duration) ([]*repository.TimeSeriesBucket, error) {
	if f.TimeSeriesErr != nil {
		return nil, f.TimeSeriesErr
	}
	return append([]*repository.TimeSeriesBucket(nil), f.TimeSeries...), nil
}

func (f *FakeStatsRepository) GetPopularModels(context.Context, time.Time, int) ([]string, error) {
	if f.PopularModelsErr != nil {
		return nil, f.PopularModelsErr
	}
	return append([]string(nil), f.PopularModels...), nil
}

func (f *FakeStatsRepository) GetRecentModels(context.Context, int) ([]string, error) {
	if f.RecentModelsErr != nil {
		return nil, f.RecentModelsErr
	}
	return append([]string(nil), f.RecentModels...), nil
}
