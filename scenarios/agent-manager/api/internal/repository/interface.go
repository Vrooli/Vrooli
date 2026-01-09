// Package repository provides persistence interfaces for domain entities.
//
// This package defines the SEAM for data persistence. All domain entities
// are persisted through these interfaces, enabling:
// - Different storage backends (PostgreSQL, SQLite, in-memory)
// - Easy testing with mock repositories
// - Database migrations without changing domain logic
package repository

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Common Types
// -----------------------------------------------------------------------------

// ListFilter specifies pagination for list operations.
type ListFilter struct {
	Limit  int
	Offset int
}

// RunListFilter extends ListFilter with run-specific filters.
type RunListFilter struct {
	ListFilter
	TaskID         *uuid.UUID
	AgentProfileID *uuid.UUID
	Status         *domain.RunStatus
	TagPrefix      string // Filter runs by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
}

// -----------------------------------------------------------------------------
// ProfileRepository - AgentProfile persistence
// -----------------------------------------------------------------------------

// ProfileRepository provides persistence for AgentProfile entities.
type ProfileRepository interface {
	// Create stores a new agent profile.
	Create(ctx context.Context, profile *domain.AgentProfile) error

	// Get retrieves a profile by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error)

	// GetByName retrieves a profile by name.
	GetByName(ctx context.Context, name string) (*domain.AgentProfile, error)

	// GetByKey retrieves a profile by key.
	GetByKey(ctx context.Context, key string) (*domain.AgentProfile, error)

	// List retrieves profiles with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.AgentProfile, error)

	// Update modifies an existing profile.
	Update(ctx context.Context, profile *domain.AgentProfile) error

	// Delete removes a profile by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// -----------------------------------------------------------------------------
// TaskRepository - Task persistence
// -----------------------------------------------------------------------------

// TaskRepository provides persistence for Task entities.
type TaskRepository interface {
	// Create stores a new task.
	Create(ctx context.Context, task *domain.Task) error

	// Get retrieves a task by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Task, error)

	// List retrieves tasks with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.Task, error)

	// ListByStatus retrieves tasks with a specific status.
	ListByStatus(ctx context.Context, status domain.TaskStatus, filter ListFilter) ([]*domain.Task, error)

	// Update modifies an existing task.
	Update(ctx context.Context, task *domain.Task) error

	// Delete removes a task by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// -----------------------------------------------------------------------------
// RunRepository - Run persistence
// -----------------------------------------------------------------------------

// RunRepository provides persistence for Run entities.
type RunRepository interface {
	// Create stores a new run.
	Create(ctx context.Context, run *domain.Run) error

	// Get retrieves a run by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Run, error)

	// List retrieves runs with optional filtering.
	List(ctx context.Context, filter RunListFilter) ([]*domain.Run, error)

	// ListByTask retrieves runs for a specific task.
	ListByTask(ctx context.Context, taskID uuid.UUID, filter ListFilter) ([]*domain.Run, error)

	// Update modifies an existing run.
	Update(ctx context.Context, run *domain.Run) error

	// Delete removes a run by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByStatus returns the count of runs by status.
	CountByStatus(ctx context.Context, status domain.RunStatus) (int, error)
}

// -----------------------------------------------------------------------------
// EventRepository - RunEvent persistence
// -----------------------------------------------------------------------------

// EventRepository provides persistence for RunEvent entities.
type EventRepository interface {
	// Append adds events to a run's event stream.
	Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error

	// Get retrieves events for a run.
	Get(ctx context.Context, runID uuid.UUID, afterSequence int64, limit int) ([]*domain.RunEvent, error)

	// GetByType retrieves events of specific types.
	GetByType(ctx context.Context, runID uuid.UUID, types []domain.RunEventType, limit int) ([]*domain.RunEvent, error)

	// Count returns the number of events for a run.
	Count(ctx context.Context, runID uuid.UUID) (int64, error)

	// Delete removes all events for a run.
	Delete(ctx context.Context, runID uuid.UUID) error
}

// -----------------------------------------------------------------------------
// PolicyRepository - Policy persistence
// -----------------------------------------------------------------------------

// PolicyRepository provides persistence for Policy entities.
type PolicyRepository interface {
	// Create stores a new policy.
	Create(ctx context.Context, policy *domain.Policy) error

	// Get retrieves a policy by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Policy, error)

	// List retrieves policies with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.Policy, error)

	// ListEnabled retrieves only enabled policies.
	ListEnabled(ctx context.Context) ([]*domain.Policy, error)

	// Update modifies an existing policy.
	Update(ctx context.Context, policy *domain.Policy) error

	// Delete removes a policy by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByScope finds policies matching a scope pattern.
	FindByScope(ctx context.Context, scopePath string) ([]*domain.Policy, error)
}

// -----------------------------------------------------------------------------
// LockRepository - ScopeLock persistence
// -----------------------------------------------------------------------------

// LockRepository provides persistence for ScopeLock entities.
type LockRepository interface {
	// Acquire attempts to acquire a lock.
	Acquire(ctx context.Context, lock *domain.ScopeLock) error

	// Release releases a lock by ID.
	Release(ctx context.Context, id uuid.UUID) error

	// ReleaseByRun releases all locks for a run.
	ReleaseByRun(ctx context.Context, runID uuid.UUID) error

	// Check finds overlapping locks for a scope.
	Check(ctx context.Context, scopePath, projectRoot string) ([]*domain.ScopeLock, error)

	// Refresh extends a lock's expiration.
	Refresh(ctx context.Context, id uuid.UUID, newExpiry int64) error

	// CleanupExpired removes expired locks.
	CleanupExpired(ctx context.Context) (int, error)
}

// -----------------------------------------------------------------------------
// CheckpointRepository - Run checkpoint persistence
// -----------------------------------------------------------------------------

// CheckpointRepository provides persistence for run checkpoints.
// Checkpoints enable safe interruption and resumption of runs.
type CheckpointRepository interface {
	// Save stores a checkpoint, overwriting any existing checkpoint for the run.
	Save(ctx context.Context, checkpoint *domain.RunCheckpoint) error

	// Get retrieves the latest checkpoint for a run.
	Get(ctx context.Context, runID uuid.UUID) (*domain.RunCheckpoint, error)

	// Delete removes all checkpoints for a run.
	Delete(ctx context.Context, runID uuid.UUID) error

	// ListStale finds runs with checkpoints older than the given duration.
	// This helps identify runs that may have stalled.
	ListStale(ctx context.Context, olderThan time.Duration) ([]*domain.RunCheckpoint, error)

	// Heartbeat updates the last heartbeat time for a checkpoint.
	Heartbeat(ctx context.Context, runID uuid.UUID) error
}

// -----------------------------------------------------------------------------
// IdempotencyRepository - Idempotency record persistence
// -----------------------------------------------------------------------------

// IdempotencyRepository provides persistence for idempotency records.
// This enables replay-safe operations that can be safely retried.
type IdempotencyRepository interface {
	// Check looks up an existing idempotency record.
	// Returns nil, nil if no record exists.
	Check(ctx context.Context, key string) (*domain.IdempotencyRecord, error)

	// Reserve creates a pending idempotency record.
	// Returns an error if the key already exists.
	Reserve(ctx context.Context, key string, ttl time.Duration) (*domain.IdempotencyRecord, error)

	// Complete marks an idempotency record as complete with the result.
	Complete(ctx context.Context, key string, entityID uuid.UUID, entityType string, response []byte) error

	// Fail marks an idempotency record as failed (allows retry).
	Fail(ctx context.Context, key string) error

	// CleanupExpired removes expired idempotency records.
	CleanupExpired(ctx context.Context) (int, error)
}

// -----------------------------------------------------------------------------
// StatsRepository - Aggregation queries for analytics
// -----------------------------------------------------------------------------

// StatsTimeWindow defines a time range for aggregation queries.
type StatsTimeWindow struct {
	Start time.Time
	End   time.Time
}

// StatsFilter specifies filtering options for stats queries.
type StatsFilter struct {
	Window      StatsTimeWindow
	RunnerTypes []domain.RunnerType // Filter by specific runners
	ProfileIDs  []uuid.UUID         // Filter by specific profiles
	Models      []string            // Filter by specific models
	TagPrefix   string              // Filter by tag prefix
}

// RunStatusCounts contains counts by run status.
type RunStatusCounts struct {
	Pending     int `json:"pending" db:"pending"`
	Starting    int `json:"starting" db:"starting"`
	Running     int `json:"running" db:"running"`
	Complete    int `json:"complete" db:"complete"`
	Failed      int `json:"failed" db:"failed"`
	Cancelled   int `json:"cancelled" db:"cancelled"`
	NeedsReview int `json:"needsReview" db:"needs_review"`
	Total       int `json:"total" db:"total"`
}

// DurationStats contains duration percentile statistics.
type DurationStats struct {
	AvgMs int64 `json:"avgMs" db:"avg_ms"`
	P50Ms int64 `json:"p50Ms" db:"p50_ms"`
	P95Ms int64 `json:"p95Ms" db:"p95_ms"`
	P99Ms int64 `json:"p99Ms" db:"p99_ms"`
	MinMs int64 `json:"minMs" db:"min_ms"`
	MaxMs int64 `json:"maxMs" db:"max_ms"`
	Count int   `json:"count" db:"count"`
}

// CostStats contains cost aggregation data.
type CostStats struct {
	TotalCostUSD              float64 `json:"totalCostUsd" db:"total_cost_usd"`
	TotalCostUSDAuthoritative float64 `json:"totalCostUsdAuthoritative" db:"total_cost_usd_authoritative"`
	TotalCostUSDEstimated     float64 `json:"totalCostUsdEstimated" db:"total_cost_usd_estimated"`
	TotalCostUSDUnknown       float64 `json:"totalCostUsdUnknown" db:"total_cost_usd_unknown"`
	InputCostUSD              float64 `json:"inputCostUsd" db:"input_cost_usd"`
	OutputCostUSD             float64 `json:"outputCostUsd" db:"output_cost_usd"`
	CacheReadCostUSD          float64 `json:"cacheReadCostUsd" db:"cache_read_cost_usd"`
	CacheCreationCostUSD      float64 `json:"cacheCreationCostUsd" db:"cache_creation_cost_usd"`
	AvgCostUSD                float64 `json:"avgCostUsd" db:"avg_cost_usd"`
	InputTokens               int64   `json:"inputTokens" db:"input_tokens"`
	OutputTokens              int64   `json:"outputTokens" db:"output_tokens"`
	CacheReadTokens           int64   `json:"cacheReadTokens" db:"cache_read_tokens"`
	TotalTokens               int64   `json:"totalTokens" db:"total_tokens"`
}

// RunnerBreakdown contains stats grouped by runner type.
type RunnerBreakdown struct {
	RunnerType    domain.RunnerType `json:"runnerType" db:"runner_type"`
	RunCount      int               `json:"runCount" db:"run_count"`
	SuccessCount  int               `json:"successCount" db:"success_count"`
	FailedCount   int               `json:"failedCount" db:"failed_count"`
	TotalCostUSD  float64           `json:"totalCostUsd" db:"total_cost_usd"`
	AvgDurationMs int64             `json:"avgDurationMs" db:"avg_duration_ms"`
}

// ProfileBreakdown contains stats grouped by agent profile.
type ProfileBreakdown struct {
	ProfileID    uuid.UUID `json:"profileId" db:"profile_id"`
	ProfileName  string    `json:"profileName" db:"profile_name"`
	RunCount     int       `json:"runCount" db:"run_count"`
	SuccessCount int       `json:"successCount" db:"success_count"`
	FailedCount  int       `json:"failedCount" db:"failed_count"`
	TotalCostUSD float64   `json:"totalCostUsd" db:"total_cost_usd"`
}

// ModelBreakdown contains stats grouped by model.
type ModelBreakdown struct {
	Model                     string  `json:"model" db:"model"`
	RunCount                  int     `json:"runCount" db:"run_count"`
	SuccessCount              int     `json:"successCount" db:"success_count"`
	TotalCostUSD              float64 `json:"totalCostUsd" db:"total_cost_usd"`
	TotalCostUSDAuthoritative float64 `json:"totalCostUsdAuthoritative" db:"total_cost_usd_authoritative"`
	TotalCostUSDEstimated     float64 `json:"totalCostUsdEstimated" db:"total_cost_usd_estimated"`
	TotalCostUSDUnknown       float64 `json:"totalCostUsdUnknown" db:"total_cost_usd_unknown"`
	InputCostUSD              float64 `json:"inputCostUsd" db:"input_cost_usd"`
	OutputCostUSD             float64 `json:"outputCostUsd" db:"output_cost_usd"`
	CacheReadCostUSD          float64 `json:"cacheReadCostUsd" db:"cache_read_cost_usd"`
	CacheCreationCostUSD      float64 `json:"cacheCreationCostUsd" db:"cache_creation_cost_usd"`
	TotalTokens               int64   `json:"totalTokens" db:"total_tokens"`
}

// ToolUsageStats contains tool call frequency data.
type ToolUsageStats struct {
	ToolName     string `json:"toolName" db:"tool_name"`
	CallCount    int    `json:"callCount" db:"call_count"`
	SuccessCount int    `json:"successCount" db:"success_count"`
	FailedCount  int    `json:"failedCount" db:"failed_count"`
}

// ToolUsageModelBreakdown contains tool usage grouped by model.
type ToolUsageModelBreakdown struct {
	Model        string `json:"model" db:"model"`
	RunCount     int    `json:"runCount" db:"run_count"`
	CallCount    int    `json:"callCount" db:"call_count"`
	SuccessCount int    `json:"successCount" db:"success_count"`
	FailedCount  int    `json:"failedCount" db:"failed_count"`
}

// ModelRunUsage contains run-level details for a specific model.
type ModelRunUsage struct {
	RunID        uuid.UUID  `json:"runId" db:"run_id"`
	TaskID       uuid.UUID  `json:"taskId" db:"task_id"`
	TaskTitle    string     `json:"taskTitle" db:"task_title"`
	ProfileID    *uuid.UUID `json:"profileId" db:"profile_id"`
	ProfileName  string     `json:"profileName" db:"profile_name"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	TotalCostUSD float64    `json:"totalCostUsd" db:"total_cost_usd"`
	TotalTokens  int64      `json:"totalTokens" db:"total_tokens"`
}

// ToolRunUsage contains run-level tool usage details.
type ToolRunUsage struct {
	RunID        uuid.UUID  `json:"runId" db:"run_id"`
	TaskID       uuid.UUID  `json:"taskId" db:"task_id"`
	TaskTitle    string     `json:"taskTitle" db:"task_title"`
	ProfileID    *uuid.UUID `json:"profileId" db:"profile_id"`
	ProfileName  string     `json:"profileName" db:"profile_name"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	Model        string     `json:"model" db:"model"`
	CallCount    int        `json:"callCount" db:"call_count"`
	SuccessCount int        `json:"successCount" db:"success_count"`
	FailedCount  int        `json:"failedCount" db:"failed_count"`
}

// ErrorPattern contains error frequency data.
type ErrorPattern struct {
	ErrorCode   string    `json:"errorCode" db:"error_code"`
	Count       int       `json:"count" db:"count"`
	LastSeen    time.Time `json:"lastSeen" db:"last_seen"`
	SampleRunID uuid.UUID `json:"sampleRunId" db:"sample_run_id"`
}

// TimeSeriesBucket contains time-bucketed data for charts.
type TimeSeriesBucket struct {
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	RunsStarted   int       `json:"runsStarted" db:"runs_started"`
	RunsCompleted int       `json:"runsCompleted" db:"runs_completed"`
	RunsFailed    int       `json:"runsFailed" db:"runs_failed"`
	RunsCancelled int       `json:"runsCancelled" db:"runs_cancelled"`
	TotalCostUSD  float64   `json:"totalCostUsd" db:"total_cost_usd"`
	AvgDurationMs int64     `json:"avgDurationMs" db:"avg_duration_ms"`
}

// -----------------------------------------------------------------------------
// InvestigationSettingsRepository - Investigation settings persistence
// -----------------------------------------------------------------------------

// InvestigationSettingsRepository provides persistence for investigation settings.
// This is a singleton table - only one row exists.
type InvestigationSettingsRepository interface {
	// Get retrieves the current investigation settings.
	// Returns default settings if none exist.
	Get(ctx context.Context) (*domain.InvestigationSettings, error)

	// Update modifies the investigation settings.
	Update(ctx context.Context, settings *domain.InvestigationSettings) error

	// Reset restores settings to defaults.
	Reset(ctx context.Context) error
}

// StatsRepository provides aggregation queries for analytics.
// This is the primary SEAM for testing stats functionality.
type StatsRepository interface {
	// GetRunStatusCounts returns counts of runs by status within the time window.
	GetRunStatusCounts(ctx context.Context, filter StatsFilter) (*RunStatusCounts, error)

	// GetSuccessRate returns the ratio of complete runs to terminal runs.
	GetSuccessRate(ctx context.Context, filter StatsFilter) (float64, error)

	// GetDurationStats returns duration percentile statistics.
	GetDurationStats(ctx context.Context, filter StatsFilter) (*DurationStats, error)

	// GetCostStats aggregates cost data from metric events.
	GetCostStats(ctx context.Context, filter StatsFilter) (*CostStats, error)

	// GetRunnerBreakdown returns stats grouped by runner type.
	GetRunnerBreakdown(ctx context.Context, filter StatsFilter) ([]*RunnerBreakdown, error)

	// GetProfileBreakdown returns stats grouped by profile.
	GetProfileBreakdown(ctx context.Context, filter StatsFilter, limit int) ([]*ProfileBreakdown, error)

	// GetModelBreakdown returns stats grouped by model.
	GetModelBreakdown(ctx context.Context, filter StatsFilter, limit int) ([]*ModelBreakdown, error)

	// GetToolUsageStats aggregates tool call events.
	GetToolUsageStats(ctx context.Context, filter StatsFilter, limit int) ([]*ToolUsageStats, error)

	// GetModelRunUsage returns run-level usage for a specific model.
	GetModelRunUsage(ctx context.Context, filter StatsFilter, model string, limit int) ([]*ModelRunUsage, error)

	// GetToolRunUsage returns run-level usage for a specific tool.
	GetToolRunUsage(ctx context.Context, filter StatsFilter, toolName string, limit int) ([]*ToolRunUsage, error)

	// GetToolUsageByModel returns tool usage grouped by model.
	GetToolUsageByModel(ctx context.Context, filter StatsFilter, toolName string, limit int) ([]*ToolUsageModelBreakdown, error)

	// GetErrorPatterns aggregates error events.
	GetErrorPatterns(ctx context.Context, filter StatsFilter, limit int) ([]*ErrorPattern, error)

	// GetTimeSeries returns time-bucketed data for charts.
	GetTimeSeries(ctx context.Context, filter StatsFilter, bucketDuration time.Duration) ([]*TimeSeriesBucket, error)

	// GetPopularModels returns the most used models by run count within a time window.
	GetPopularModels(ctx context.Context, since time.Time, limit int) ([]string, error)

	// GetRecentModels returns recently used models ordered by most recent use (system-wide).
	GetRecentModels(ctx context.Context, limit int) ([]string, error)
}
