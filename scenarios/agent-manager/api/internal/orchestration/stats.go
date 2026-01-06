// Package orchestration provides stats service methods.
//
// Stats provides aggregated analytics data for runs, cost, duration,
// and other metrics. The stats service uses the StatsRepository for
// database aggregation queries.
package orchestration

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/repository"
)

// StatsSummary contains all stats data for a single API response.
type StatsSummary struct {
	StatusCounts     *repository.RunStatusCounts   `json:"statusCounts"`
	SuccessRate      float64                       `json:"successRate"`
	Duration         *repository.DurationStats     `json:"duration"`
	Cost             *repository.CostStats         `json:"cost"`
	RunnerBreakdown  []*repository.RunnerBreakdown `json:"runnerBreakdown"`
	ProfileBreakdown []*repository.ProfileBreakdown `json:"profileBreakdown,omitempty"`
	ModelBreakdown   []*repository.ModelBreakdown  `json:"modelBreakdown,omitempty"`
	ToolUsage        []*repository.ToolUsageStats  `json:"toolUsage,omitempty"`
	ErrorPatterns    []*repository.ErrorPattern    `json:"errorPatterns,omitempty"`
}

// TimePreset defines common time window presets.
type TimePreset string

const (
	TimePreset6H  TimePreset = "6h"
	TimePreset12H TimePreset = "12h"
	TimePreset24H TimePreset = "24h"
	TimePreset7D  TimePreset = "7d"
	TimePreset30D TimePreset = "30d"
)

// TimePresetToDuration converts a preset to a duration.
func TimePresetToDuration(preset TimePreset) time.Duration {
	switch preset {
	case TimePreset6H:
		return 6 * time.Hour
	case TimePreset12H:
		return 12 * time.Hour
	case TimePreset24H:
		return 24 * time.Hour
	case TimePreset7D:
		return 7 * 24 * time.Hour
	case TimePreset30D:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// StatsService provides analytics operations.
type StatsService interface {
	// GetSummary returns a complete stats summary with parallel query execution.
	GetSummary(ctx context.Context, filter repository.StatsFilter) (*StatsSummary, error)

	// GetStatusCounts returns run counts by status.
	GetStatusCounts(ctx context.Context, filter repository.StatsFilter) (*repository.RunStatusCounts, error)

	// GetSuccessRate returns the success rate.
	GetSuccessRate(ctx context.Context, filter repository.StatsFilter) (float64, error)

	// GetDurationStats returns duration percentile statistics.
	GetDurationStats(ctx context.Context, filter repository.StatsFilter) (*repository.DurationStats, error)

	// GetCostStats returns cost aggregation data.
	GetCostStats(ctx context.Context, filter repository.StatsFilter) (*repository.CostStats, error)

	// GetRunnerBreakdown returns stats grouped by runner type.
	GetRunnerBreakdown(ctx context.Context, filter repository.StatsFilter) ([]*repository.RunnerBreakdown, error)

	// GetProfileBreakdown returns stats grouped by profile.
	GetProfileBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ProfileBreakdown, error)

	// GetModelBreakdown returns stats grouped by model.
	GetModelBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ModelBreakdown, error)

	// GetToolUsageStats returns tool call frequency data.
	GetToolUsageStats(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ToolUsageStats, error)

	// GetModelRunUsage returns run-level usage for a specific model.
	GetModelRunUsage(ctx context.Context, filter repository.StatsFilter, model string, limit int) ([]*repository.ModelRunUsage, error)

	// GetToolRunUsage returns run-level usage for a specific tool.
	GetToolRunUsage(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolRunUsage, error)

	// GetToolUsageByModel returns tool usage grouped by model.
	GetToolUsageByModel(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolUsageModelBreakdown, error)

	// GetErrorPatterns returns error frequency data.
	GetErrorPatterns(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ErrorPattern, error)

	// GetTimeSeries returns time-bucketed data for charts.
	GetTimeSeries(ctx context.Context, filter repository.StatsFilter, bucketDuration time.Duration) ([]*repository.TimeSeriesBucket, error)
}

// statsOrchestrator implements StatsService.
type statsOrchestrator struct {
	stats repository.StatsRepository

	// Cache for expensive queries
	cacheMu       sync.RWMutex
	summaryCache  map[string]*cachedSummary
	cacheDuration time.Duration
}

type cachedSummary struct {
	summary   *StatsSummary
	expiresAt time.Time
}

// NewStatsOrchestrator creates a stats orchestrator.
func NewStatsOrchestrator(stats repository.StatsRepository) StatsService {
	return &statsOrchestrator{
		stats:         stats,
		summaryCache:  make(map[string]*cachedSummary),
		cacheDuration: 30 * time.Second,
	}
}

// GetSummary returns a complete stats summary with parallel query execution.
func (o *statsOrchestrator) GetSummary(ctx context.Context, filter repository.StatsFilter) (*StatsSummary, error) {
	// Generate cache key from filter
	cacheKey := o.cacheKey(filter)

	// Check cache
	o.cacheMu.RLock()
	if cached, ok := o.summaryCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		o.cacheMu.RUnlock()
		return cached.summary, nil
	}
	o.cacheMu.RUnlock()

	// Execute queries in parallel
	var (
		wg               sync.WaitGroup
		statusCounts     *repository.RunStatusCounts
		successRate      float64
		durationStats    *repository.DurationStats
		costStats        *repository.CostStats
		runnerBreakdown  []*repository.RunnerBreakdown
		profileBreakdown []*repository.ProfileBreakdown
		modelBreakdown   []*repository.ModelBreakdown
		toolUsage        []*repository.ToolUsageStats
		errorPatterns    []*repository.ErrorPattern

		statusErr, successErr, durationErr, costErr    error
		runnerErr, profileErr, modelErr                error
		toolErr, errorErr                              error
	)

	wg.Add(9)

	go func() {
		defer wg.Done()
		statusCounts, statusErr = o.stats.GetRunStatusCounts(ctx, filter)
	}()

	go func() {
		defer wg.Done()
		successRate, successErr = o.stats.GetSuccessRate(ctx, filter)
	}()

	go func() {
		defer wg.Done()
		durationStats, durationErr = o.stats.GetDurationStats(ctx, filter)
	}()

	go func() {
		defer wg.Done()
		costStats, costErr = o.stats.GetCostStats(ctx, filter)
	}()

	go func() {
		defer wg.Done()
		runnerBreakdown, runnerErr = o.stats.GetRunnerBreakdown(ctx, filter)
	}()

	go func() {
		defer wg.Done()
		profileBreakdown, profileErr = o.stats.GetProfileBreakdown(ctx, filter, 10)
	}()

	go func() {
		defer wg.Done()
		modelBreakdown, modelErr = o.stats.GetModelBreakdown(ctx, filter, 10)
	}()

	go func() {
		defer wg.Done()
		toolUsage, toolErr = o.stats.GetToolUsageStats(ctx, filter, 20)
	}()

	go func() {
		defer wg.Done()
		errorPatterns, errorErr = o.stats.GetErrorPatterns(ctx, filter, 10)
	}()

	wg.Wait()

	// Check for errors - return first error encountered
	for _, err := range []error{statusErr, successErr, durationErr, costErr, runnerErr, profileErr, modelErr, toolErr, errorErr} {
		if err != nil {
			return nil, err
		}
	}

	summary := &StatsSummary{
		StatusCounts:     statusCounts,
		SuccessRate:      successRate,
		Duration:         durationStats,
		Cost:             costStats,
		RunnerBreakdown:  runnerBreakdown,
		ProfileBreakdown: profileBreakdown,
		ModelBreakdown:   modelBreakdown,
		ToolUsage:        toolUsage,
		ErrorPatterns:    errorPatterns,
	}

	// Cache the result
	o.cacheMu.Lock()
	o.summaryCache[cacheKey] = &cachedSummary{
		summary:   summary,
		expiresAt: time.Now().Add(o.cacheDuration),
	}
	o.cacheMu.Unlock()

	return summary, nil
}

// GetStatusCounts returns run counts by status.
func (o *statsOrchestrator) GetStatusCounts(ctx context.Context, filter repository.StatsFilter) (*repository.RunStatusCounts, error) {
	return o.stats.GetRunStatusCounts(ctx, filter)
}

// GetSuccessRate returns the success rate.
func (o *statsOrchestrator) GetSuccessRate(ctx context.Context, filter repository.StatsFilter) (float64, error) {
	return o.stats.GetSuccessRate(ctx, filter)
}

// GetDurationStats returns duration percentile statistics.
func (o *statsOrchestrator) GetDurationStats(ctx context.Context, filter repository.StatsFilter) (*repository.DurationStats, error) {
	return o.stats.GetDurationStats(ctx, filter)
}

// GetCostStats returns cost aggregation data.
func (o *statsOrchestrator) GetCostStats(ctx context.Context, filter repository.StatsFilter) (*repository.CostStats, error) {
	return o.stats.GetCostStats(ctx, filter)
}

// GetRunnerBreakdown returns stats grouped by runner type.
func (o *statsOrchestrator) GetRunnerBreakdown(ctx context.Context, filter repository.StatsFilter) ([]*repository.RunnerBreakdown, error) {
	return o.stats.GetRunnerBreakdown(ctx, filter)
}

// GetProfileBreakdown returns stats grouped by profile.
func (o *statsOrchestrator) GetProfileBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ProfileBreakdown, error) {
	return o.stats.GetProfileBreakdown(ctx, filter, limit)
}

// GetModelBreakdown returns stats grouped by model.
func (o *statsOrchestrator) GetModelBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ModelBreakdown, error) {
	return o.stats.GetModelBreakdown(ctx, filter, limit)
}

// GetToolUsageStats returns tool call frequency data.
func (o *statsOrchestrator) GetToolUsageStats(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ToolUsageStats, error) {
	return o.stats.GetToolUsageStats(ctx, filter, limit)
}

// GetModelRunUsage returns run-level usage for a specific model.
func (o *statsOrchestrator) GetModelRunUsage(ctx context.Context, filter repository.StatsFilter, model string, limit int) ([]*repository.ModelRunUsage, error) {
	return o.stats.GetModelRunUsage(ctx, filter, model, limit)
}

// GetToolRunUsage returns run-level usage for a specific tool.
func (o *statsOrchestrator) GetToolRunUsage(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolRunUsage, error) {
	return o.stats.GetToolRunUsage(ctx, filter, toolName, limit)
}

// GetToolUsageByModel returns tool usage grouped by model.
func (o *statsOrchestrator) GetToolUsageByModel(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolUsageModelBreakdown, error) {
	return o.stats.GetToolUsageByModel(ctx, filter, toolName, limit)
}

// GetErrorPatterns returns error frequency data.
func (o *statsOrchestrator) GetErrorPatterns(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ErrorPattern, error) {
	return o.stats.GetErrorPatterns(ctx, filter, limit)
}

// GetTimeSeries returns time-bucketed data for charts.
func (o *statsOrchestrator) GetTimeSeries(ctx context.Context, filter repository.StatsFilter, bucketDuration time.Duration) ([]*repository.TimeSeriesBucket, error) {
	return o.stats.GetTimeSeries(ctx, filter, bucketDuration)
}

// cacheKey generates a cache key from the filter.
func (o *statsOrchestrator) cacheKey(filter repository.StatsFilter) string {
	// Simple key based on time window - could be more sophisticated
	return filter.Window.Start.Format(time.RFC3339) + "-" + filter.Window.End.Format(time.RFC3339)
}

// FilterFromPreset creates a StatsFilter from a time preset.
func FilterFromPreset(preset TimePreset) repository.StatsFilter {
	duration := TimePresetToDuration(preset)
	now := time.Now()
	return repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-duration),
			End:   now,
		},
	}
}

// FilterFromTimeRange creates a StatsFilter from a time range.
func FilterFromTimeRange(start, end time.Time) repository.StatsFilter {
	return repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: start,
			End:   end,
		},
	}
}
