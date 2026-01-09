// Package repository provides in-memory implementations for testing.
//
// This file contains the in-memory StatsRepository implementation.
// Unlike the PostgreSQL implementation which computes aggregations from
// runs and events, this implementation allows direct seeding of results
// for deterministic testing.
package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
)

// =============================================================================
// In-Memory Stats Repository
// =============================================================================

// MemoryStatsRepository is an in-memory implementation of StatsRepository.
// It supports two modes:
// 1. Direct seeding: Set SeededXxx fields for deterministic test responses
// 2. Compute mode: Pass RunRepo and EventRepo to compute from actual data
type MemoryStatsRepository struct {
	mu sync.RWMutex

	// Optional: compute stats from these repositories
	RunRepo   RunRepository
	EventRepo EventRepository

	// Seeded responses for testing - when set, these take precedence
	SeededStatusCounts     *RunStatusCounts
	SeededSuccessRate      *float64
	SeededDurationStats    *DurationStats
	SeededCostStats        *CostStats
	SeededRunnerBreakdown  []*RunnerBreakdown
	SeededProfileBreakdown []*ProfileBreakdown
	SeededModelBreakdown   []*ModelBreakdown
	SeededToolUsage        []*ToolUsageStats
	SeededModelRunUsage    []*ModelRunUsage
	SeededToolRunUsage     []*ToolRunUsage
	SeededToolUsageModels  []*ToolUsageModelBreakdown
	SeededErrorPatterns    []*ErrorPattern
	SeededTimeSeries       []*TimeSeriesBucket
}

// NewMemoryStatsRepository creates a new in-memory stats repository.
func NewMemoryStatsRepository() *MemoryStatsRepository {
	return &MemoryStatsRepository{}
}

// NewMemoryStatsRepositoryWithRepos creates a stats repository that computes from actual repos.
func NewMemoryStatsRepositoryWithRepos(runRepo RunRepository, eventRepo EventRepository) *MemoryStatsRepository {
	return &MemoryStatsRepository{
		RunRepo:   runRepo,
		EventRepo: eventRepo,
	}
}

func (r *MemoryStatsRepository) GetRunStatusCounts(ctx context.Context, filter StatsFilter) (*RunStatusCounts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededStatusCounts != nil {
		return r.SeededStatusCounts, nil
	}

	// Compute from runs if repository is available
	if r.RunRepo != nil {
		runs, err := r.RunRepo.List(ctx, RunListFilter{})
		if err != nil {
			return nil, err
		}

		counts := &RunStatusCounts{}
		for _, run := range runs {
			if !r.runInWindow(run, filter.Window) {
				continue
			}
			if !r.runMatchesFilter(run, filter) {
				continue
			}

			counts.Total++
			switch run.Status {
			case domain.RunStatusPending:
				counts.Pending++
			case domain.RunStatusStarting:
				counts.Starting++
			case domain.RunStatusRunning:
				counts.Running++
			case domain.RunStatusComplete:
				counts.Complete++
			case domain.RunStatusFailed:
				counts.Failed++
			case domain.RunStatusCancelled:
				counts.Cancelled++
			case domain.RunStatusNeedsReview:
				counts.NeedsReview++
			}
		}
		return counts, nil
	}

	// Default empty response
	return &RunStatusCounts{}, nil
}

func (r *MemoryStatsRepository) GetSuccessRate(ctx context.Context, filter StatsFilter) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededSuccessRate != nil {
		return *r.SeededSuccessRate, nil
	}

	// Compute from status counts
	counts, err := r.GetRunStatusCounts(ctx, filter)
	if err != nil {
		return 0, err
	}

	terminal := counts.Complete + counts.Failed + counts.Cancelled
	if terminal == 0 {
		return 0, nil
	}

	return float64(counts.Complete) / float64(terminal), nil
}

func (r *MemoryStatsRepository) GetDurationStats(ctx context.Context, filter StatsFilter) (*DurationStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededDurationStats != nil {
		return r.SeededDurationStats, nil
	}

	// Compute from runs if repository is available
	if r.RunRepo != nil {
		runs, err := r.RunRepo.List(ctx, RunListFilter{})
		if err != nil {
			return nil, err
		}

		var durations []int64
		for _, run := range runs {
			if !r.runInWindow(run, filter.Window) || !r.runMatchesFilter(run, filter) {
				continue
			}
			if !run.Status.IsTerminal() || run.EndedAt == nil || run.StartedAt == nil {
				continue
			}

			durationMs := run.EndedAt.Sub(*run.StartedAt).Milliseconds()
			if durationMs > 0 {
				durations = append(durations, durationMs)
			}
		}

		if len(durations) == 0 {
			return &DurationStats{}, nil
		}

		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

		stats := &DurationStats{
			Count: len(durations),
			MinMs: durations[0],
			MaxMs: durations[len(durations)-1],
		}

		// Calculate average
		var sum int64
		for _, d := range durations {
			sum += d
		}
		stats.AvgMs = sum / int64(len(durations))

		// Calculate percentiles
		stats.P50Ms = durations[len(durations)*50/100]
		stats.P95Ms = durations[len(durations)*95/100]
		if len(durations) > 100 {
			stats.P99Ms = durations[len(durations)*99/100]
		} else {
			stats.P99Ms = stats.MaxMs
		}

		return stats, nil
	}

	return &DurationStats{}, nil
}

func (r *MemoryStatsRepository) GetCostStats(ctx context.Context, filter StatsFilter) (*CostStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededCostStats != nil {
		return r.SeededCostStats, nil
	}

	// For in-memory, we'd need to iterate through all run events
	// This is simplified - production uses SQL aggregation
	return &CostStats{}, nil
}

func (r *MemoryStatsRepository) GetRunnerBreakdown(ctx context.Context, filter StatsFilter) ([]*RunnerBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededRunnerBreakdown != nil {
		return r.SeededRunnerBreakdown, nil
	}

	// Compute from runs if repository is available
	if r.RunRepo != nil {
		runs, err := r.RunRepo.List(ctx, RunListFilter{})
		if err != nil {
			return nil, err
		}

		breakdown := make(map[domain.RunnerType]*RunnerBreakdown)
		for _, run := range runs {
			if !r.runInWindow(run, filter.Window) || !r.runMatchesFilter(run, filter) {
				continue
			}

			runnerType := domain.RunnerType("")
			if run.ResolvedConfig != nil {
				runnerType = run.ResolvedConfig.RunnerType
			}

			if breakdown[runnerType] == nil {
				breakdown[runnerType] = &RunnerBreakdown{RunnerType: runnerType}
			}

			b := breakdown[runnerType]
			b.RunCount++
			if run.Status == domain.RunStatusComplete {
				b.SuccessCount++
			} else if run.Status == domain.RunStatusFailed {
				b.FailedCount++
			}

			if run.EndedAt != nil && run.StartedAt != nil {
				durationMs := run.EndedAt.Sub(*run.StartedAt).Milliseconds()
				b.AvgDurationMs = (b.AvgDurationMs*int64(b.RunCount-1) + durationMs) / int64(b.RunCount)
			}
		}

		result := make([]*RunnerBreakdown, 0, len(breakdown))
		for _, b := range breakdown {
			result = append(result, b)
		}
		return result, nil
	}

	return []*RunnerBreakdown{}, nil
}

func (r *MemoryStatsRepository) GetProfileBreakdown(ctx context.Context, filter StatsFilter, limit int) ([]*ProfileBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededProfileBreakdown != nil {
		result := r.SeededProfileBreakdown
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ProfileBreakdown{}, nil
}

func (r *MemoryStatsRepository) GetModelBreakdown(ctx context.Context, filter StatsFilter, limit int) ([]*ModelBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededModelBreakdown != nil {
		result := r.SeededModelBreakdown
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ModelBreakdown{}, nil
}

func (r *MemoryStatsRepository) GetToolUsageStats(ctx context.Context, filter StatsFilter, limit int) ([]*ToolUsageStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededToolUsage != nil {
		result := r.SeededToolUsage
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ToolUsageStats{}, nil
}

func (r *MemoryStatsRepository) GetModelRunUsage(ctx context.Context, filter StatsFilter, model string, limit int) ([]*ModelRunUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.SeededModelRunUsage != nil {
		result := r.SeededModelRunUsage
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ModelRunUsage{}, nil
}

func (r *MemoryStatsRepository) GetToolRunUsage(ctx context.Context, filter StatsFilter, toolName string, limit int) ([]*ToolRunUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.SeededToolRunUsage != nil {
		result := r.SeededToolRunUsage
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ToolRunUsage{}, nil
}

func (r *MemoryStatsRepository) GetToolUsageByModel(ctx context.Context, filter StatsFilter, toolName string, limit int) ([]*ToolUsageModelBreakdown, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.SeededToolUsageModels != nil {
		result := r.SeededToolUsageModels
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ToolUsageModelBreakdown{}, nil
}

func (r *MemoryStatsRepository) GetErrorPatterns(ctx context.Context, filter StatsFilter, limit int) ([]*ErrorPattern, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededErrorPatterns != nil {
		result := r.SeededErrorPatterns
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	return []*ErrorPattern{}, nil
}

func (r *MemoryStatsRepository) GetTimeSeries(ctx context.Context, filter StatsFilter, bucketDuration time.Duration) ([]*TimeSeriesBucket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return seeded data if available
	if r.SeededTimeSeries != nil {
		return r.SeededTimeSeries, nil
	}

	// Compute from runs if repository is available
	if r.RunRepo != nil {
		runs, err := r.RunRepo.List(ctx, RunListFilter{})
		if err != nil {
			return nil, err
		}

		buckets := make(map[time.Time]*TimeSeriesBucket)

		for _, run := range runs {
			if !r.runInWindow(run, filter.Window) || !r.runMatchesFilter(run, filter) {
				continue
			}

			// Truncate to bucket
			bucketTime := run.CreatedAt.Truncate(bucketDuration)

			if buckets[bucketTime] == nil {
				buckets[bucketTime] = &TimeSeriesBucket{Timestamp: bucketTime}
			}

			b := buckets[bucketTime]
			b.RunsStarted++

			switch run.Status {
			case domain.RunStatusComplete:
				b.RunsCompleted++
			case domain.RunStatusFailed:
				b.RunsFailed++
			case domain.RunStatusCancelled:
				b.RunsCancelled++
			}
		}

		// Sort by timestamp
		result := make([]*TimeSeriesBucket, 0, len(buckets))
		for _, b := range buckets {
			result = append(result, b)
		}
		sort.Slice(result, func(i, j int) bool {
			return result[i].Timestamp.Before(result[j].Timestamp)
		})

		return result, nil
	}

	return []*TimeSeriesBucket{}, nil
}

// Helper methods

func (r *MemoryStatsRepository) runInWindow(run *domain.Run, window StatsTimeWindow) bool {
	if window.Start.IsZero() && window.End.IsZero() {
		return true
	}
	if !window.Start.IsZero() && run.CreatedAt.Before(window.Start) {
		return false
	}
	if !window.End.IsZero() && run.CreatedAt.After(window.End) {
		return false
	}
	return true
}

func (r *MemoryStatsRepository) runMatchesFilter(run *domain.Run, filter StatsFilter) bool {
	// Filter by runner types
	if len(filter.RunnerTypes) > 0 {
		if run.ResolvedConfig == nil {
			return false
		}
		found := false
		for _, rt := range filter.RunnerTypes {
			if run.ResolvedConfig.RunnerType == rt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by profile IDs
	if len(filter.ProfileIDs) > 0 && run.AgentProfileID != nil {
		found := false
		for _, pid := range filter.ProfileIDs {
			if *run.AgentProfileID == pid {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by models
	if len(filter.Models) > 0 {
		if run.ResolvedConfig == nil {
			return false
		}
		found := false
		for _, m := range filter.Models {
			if run.ResolvedConfig.Model == m {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by tag prefix
	if filter.TagPrefix != "" {
		tag := run.GetTag()
		if !strings.HasPrefix(tag, filter.TagPrefix) {
			return false
		}
	}

	return true
}

// GetPopularModels returns the most used models by run count within a time window.
func (r *MemoryStatsRepository) GetPopularModels(ctx context.Context, since time.Time, limit int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Need run repository to compute
	if r.RunRepo == nil {
		return []string{}, nil
	}

	runs, err := r.RunRepo.List(ctx, RunListFilter{})
	if err != nil {
		return nil, err
	}

	// Count runs per model
	modelCounts := make(map[string]int)
	for _, run := range runs {
		if run.CreatedAt.Before(since) {
			continue
		}
		if run.ResolvedConfig == nil {
			continue
		}
		model := run.ResolvedConfig.Model
		if model == "" {
			continue
		}
		modelCounts[model]++
	}

	// Sort by count descending
	type modelCount struct {
		model string
		count int
	}
	counts := make([]modelCount, 0, len(modelCounts))
	for m, c := range modelCounts {
		counts = append(counts, modelCount{model: m, count: c})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	// Take top N
	result := make([]string, 0, limit)
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, counts[i].model)
	}
	return result, nil
}

// GetRecentModels returns recently used models ordered by most recent use.
func (r *MemoryStatsRepository) GetRecentModels(ctx context.Context, limit int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Need run repository to compute
	if r.RunRepo == nil {
		return []string{}, nil
	}

	runs, err := r.RunRepo.List(ctx, RunListFilter{})
	if err != nil {
		return nil, err
	}

	// Track most recent use per model
	modelLastUsed := make(map[string]time.Time)
	for _, run := range runs {
		if run.ResolvedConfig == nil {
			continue
		}
		model := run.ResolvedConfig.Model
		if model == "" {
			continue
		}
		if existing, ok := modelLastUsed[model]; !ok || run.CreatedAt.After(existing) {
			modelLastUsed[model] = run.CreatedAt
		}
	}

	// Sort by recency descending
	type modelTime struct {
		model    string
		lastUsed time.Time
	}
	models := make([]modelTime, 0, len(modelLastUsed))
	for m, t := range modelLastUsed {
		models = append(models, modelTime{model: m, lastUsed: t})
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].lastUsed.After(models[j].lastUsed)
	})

	// Take top N
	result := make([]string, 0, limit)
	for i := 0; i < len(models) && i < limit; i++ {
		result = append(result, models[i].model)
	}
	return result, nil
}

// Verify interface compliance
var _ StatsRepository = (*MemoryStatsRepository)(nil)
