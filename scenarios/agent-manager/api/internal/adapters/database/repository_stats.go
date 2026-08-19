package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// StatsRepository Implementation
// ============================================================================

type statsRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.StatsRepository = (*statsRepository)(nil)

type modelRunUsageRow struct {
	RunID        uuid.UUID      `db:"run_id"`
	TaskID       sql.NullString `db:"task_id"`
	TaskTitle    string         `db:"task_title"`
	ProfileID    *uuid.UUID     `db:"profile_id"`
	ProfileName  string         `db:"profile_name"`
	Status       string         `db:"status"`
	CreatedAt    SQLiteTime     `db:"created_at"`
	TotalCostUSD float64        `db:"total_cost_usd"`
	TotalTokens  int64          `db:"total_tokens"`
}

func (r modelRunUsageRow) toRepository() *repository.ModelRunUsage {
	return &repository.ModelRunUsage{
		RunID:        r.RunID,
		TaskID:       parseNullableUUID(r.TaskID),
		TaskTitle:    r.TaskTitle,
		ProfileID:    r.ProfileID,
		ProfileName:  r.ProfileName,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.Time(),
		TotalCostUSD: r.TotalCostUSD,
		TotalTokens:  r.TotalTokens,
	}
}

type toolRunUsageRow struct {
	RunID        uuid.UUID      `db:"run_id"`
	TaskID       sql.NullString `db:"task_id"`
	TaskTitle    string         `db:"task_title"`
	ProfileID    *uuid.UUID     `db:"profile_id"`
	ProfileName  string         `db:"profile_name"`
	Status       string         `db:"status"`
	CreatedAt    SQLiteTime     `db:"created_at"`
	Model        string         `db:"model"`
	CallCount    int            `db:"call_count"`
	SuccessCount int            `db:"success_count"`
	FailedCount  int            `db:"failed_count"`
}

func (r toolRunUsageRow) toRepository() *repository.ToolRunUsage {
	return &repository.ToolRunUsage{
		RunID:        r.RunID,
		TaskID:       parseNullableUUID(r.TaskID),
		TaskTitle:    r.TaskTitle,
		ProfileID:    r.ProfileID,
		ProfileName:  r.ProfileName,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.Time(),
		Model:        r.Model,
		CallCount:    r.CallCount,
		SuccessCount: r.SuccessCount,
		FailedCount:  r.FailedCount,
	}
}

type errorPatternRow struct {
	ErrorCode   string     `db:"error_code"`
	Count       int        `db:"count"`
	LastSeen    SQLiteTime `db:"last_seen"`
	SampleRunID uuid.UUID  `db:"sample_run_id"`
}

func (r errorPatternRow) toRepository() *repository.ErrorPattern {
	return &repository.ErrorPattern{
		ErrorCode:   r.ErrorCode,
		Count:       r.Count,
		LastSeen:    r.LastSeen.Time(),
		SampleRunID: r.SampleRunID,
	}
}

type timeSeriesBucketRow struct {
	Timestamp     SQLiteTime `db:"timestamp"`
	RunsStarted   int        `db:"runs_started"`
	RunsCompleted int        `db:"runs_completed"`
	RunsFailed    int        `db:"runs_failed"`
	RunsCancelled int        `db:"runs_cancelled"`
	TotalCostUSD  float64    `db:"total_cost_usd"`
	AvgDurationMs int64      `db:"avg_duration_ms"`
}

func (r timeSeriesBucketRow) toRepository() *repository.TimeSeriesBucket {
	return &repository.TimeSeriesBucket{
		Timestamp:     r.Timestamp.Time(),
		RunsStarted:   r.RunsStarted,
		RunsCompleted: r.RunsCompleted,
		RunsFailed:    r.RunsFailed,
		RunsCancelled: r.RunsCancelled,
		TotalCostUSD:  r.TotalCostUSD,
		AvgDurationMs: r.AvgDurationMs,
	}
}

// GetRunStatusCounts returns counts of runs by status within the time window.
func (r *statsRepository) GetRunStatusCounts(ctx context.Context, filter repository.StatsFilter) (*repository.RunStatusCounts, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN status = 'starting' THEN 1 ELSE 0 END), 0) as starting,
			COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END), 0) as running,
			COALESCE(SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END), 0) as complete,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) as cancelled,
			COALESCE(SUM(CASE WHEN status = 'needs_review' THEN 1 ELSE 0 END), 0) as needs_review,
			COUNT(*) as total
		FROM invocation_read_model_runs
		WHERE created_at >= ? AND created_at < ?`

	query, args = r.appendDurableRunFilters(query, args, filter, "")

	var counts repository.RunStatusCounts
	if err := r.db.GetContext(ctx, &counts, query, args...); err != nil {
		return nil, wrapDBError("get_run_status_counts", "Stats", "", err)
	}
	return &counts, nil
}

// GetSuccessRate returns the ratio of complete runs to terminal runs.
func (r *statsRepository) GetSuccessRate(ctx context.Context, filter repository.StatsFilter) (float64, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			COALESCE(CASE
				WHEN SUM(CASE WHEN status IN ('complete', 'failed', 'cancelled') THEN 1 ELSE 0 END) = 0
				THEN 0.0
				ELSE ROUND(
					CAST(SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) AS REAL) /
					SUM(CASE WHEN status IN ('complete', 'failed', 'cancelled') THEN 1 ELSE 0 END),
					4
				)
			END, 0.0) as success_rate
		FROM invocation_read_model_runs
		WHERE created_at >= ? AND created_at < ?`

	query, args = r.appendDurableRunFilters(query, args, filter, "")

	var rate float64
	if err := r.db.GetContext(ctx, &rate, query, args...); err != nil {
		return 0, wrapDBError("get_success_rate", "Stats", "", err)
	}
	return rate, nil
}

// GetDurationStats returns duration percentile statistics.
func (r *statsRepository) GetDurationStats(ctx context.Context, filter repository.StatsFilter) (*repository.DurationStats, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	// SQLite: use simple AVG, MIN, MAX (no native percentiles)
	query := `
		SELECT
			COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0) as avg_ms,
			COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0) as p50_ms,
			COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0) as p95_ms,
			COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0) as p99_ms,
			COALESCE(MIN(duration_ms), 0) as min_ms,
			COALESCE(MAX(duration_ms), 0) as max_ms,
			COUNT(*) as count
		FROM invocation_read_model_runs
		WHERE created_at >= ? AND created_at < ?
		  AND duration_ms > 0`

	query, args = r.appendDurableRunFilters(query, args, filter, "")

	var stats repository.DurationStats
	if err := r.db.GetContext(ctx, &stats, query, args...); err != nil {
		return nil, wrapDBError("get_duration_stats", "Stats", "", err)
	}
	return &stats, nil
}

// GetCostStats aggregates cost data from metric events.
func (r *statsRepository) GetCostStats(ctx context.Context, filter repository.StatsFilter) (*repository.CostStats, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			COALESCE(SUM(total_cost_usd), 0) as total_cost_usd,
			COALESCE(SUM(input_cost_usd), 0) as input_cost_usd,
			COALESCE(SUM(output_cost_usd), 0) as output_cost_usd,
			COALESCE(SUM(cache_read_cost_usd), 0) as cache_read_cost_usd,
			COALESCE(SUM(cache_creation_cost_usd), 0) as cache_creation_cost_usd,
			COALESCE(AVG(total_cost_usd), 0) as avg_cost_usd,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM invocation_read_model_runs
		WHERE created_at >= ? AND created_at < ?`

	query, args = r.appendDurableRunFilters(query, args, filter, "")

	var stats repository.CostStats
	if err := r.db.GetContext(ctx, &stats, query, args...); err != nil {
		return nil, wrapDBError("get_cost_stats", "Stats", "", err)
	}
	return &stats, nil
}

// GetRunnerBreakdown returns stats grouped by runner type.
func (r *statsRepository) GetRunnerBreakdown(ctx context.Context, filter repository.StatsFilter) ([]*repository.RunnerBreakdown, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			runner_type,
			COUNT(*) as run_count,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(SUM(total_cost_usd), 0) as total_cost_usd,
			COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0) as avg_duration_ms
		FROM invocation_read_model_runs
		WHERE created_at >= ? AND created_at < ?
		  AND runner_type != ''`
	query, args = r.appendDurableRunFilters(query, args, filter, "")
	query += `
		GROUP BY runner_type
		ORDER BY run_count DESC`

	type runnerRow struct {
		RunnerType    string  `db:"runner_type"`
		RunCount      int     `db:"run_count"`
		SuccessCount  int     `db:"success_count"`
		FailedCount   int     `db:"failed_count"`
		TotalCostUSD  float64 `db:"total_cost_usd"`
		AvgDurationMs int64   `db:"avg_duration_ms"`
	}

	var rows []runnerRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_runner_breakdown", "Stats", "", err)
	}

	result := make([]*repository.RunnerBreakdown, len(rows))
	for i, row := range rows {
		result[i] = &repository.RunnerBreakdown{
			RunnerType:    domain.RunnerType(row.RunnerType),
			RunCount:      row.RunCount,
			SuccessCount:  row.SuccessCount,
			FailedCount:   row.FailedCount,
			TotalCostUSD:  row.TotalCostUSD,
			AvgDurationMs: row.AvgDurationMs,
		}
	}
	return result, nil
}

// GetProfileBreakdown returns stats grouped by profile.
func (r *statsRepository) GetProfileBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ProfileBreakdown, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			r.profile_id as profile_id,
			COALESCE(p.name, 'Unknown') as profile_name,
			COUNT(*) as run_count,
			SUM(CASE WHEN r.status = 'complete' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN r.status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(SUM(r.total_cost_usd), 0) as total_cost_usd
		FROM invocation_read_model_runs r
		LEFT JOIN agent_profiles p ON r.profile_id = p.id
		WHERE r.created_at >= ? AND r.created_at < ?
		  AND r.profile_id != ''`
	query, args = r.appendDurableRunFilters(query, args, filter, "r")
	query += `
		GROUP BY r.profile_id, p.name
		ORDER BY run_count DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []*repository.ProfileBreakdown
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_profile_breakdown", "Stats", "", err)
	}
	return rows, nil
}

// GetModelBreakdown returns stats grouped by model.
func (r *statsRepository) GetModelBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ModelBreakdown, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			model,
			COUNT(*) as run_count,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as success_count,
			COALESCE(SUM(total_cost_usd), 0) as total_cost_usd,
			COALESCE(SUM(input_cost_usd), 0) as input_cost_usd,
			COALESCE(SUM(output_cost_usd), 0) as output_cost_usd,
			COALESCE(SUM(cache_read_cost_usd), 0) as cache_read_cost_usd,
			COALESCE(SUM(cache_creation_cost_usd), 0) as cache_creation_cost_usd,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM invocation_read_model_runs
		WHERE created_at >= ? AND created_at < ?`
	query, args = r.appendDurableRunFilters(query, args, filter, "")
	query += `
		GROUP BY model
		ORDER BY run_count DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []*repository.ModelBreakdown
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_model_breakdown", "Stats", "", err)
	}
	return rows, nil
}

// GetToolUsageStats aggregates tool call events.
func (r *statsRepository) GetToolUsageStats(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ToolUsageStats, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			CASE WHEN f.tool_name = '' THEN 'unknown' ELSE f.tool_name END as tool_name,
			COUNT(*) as call_count,
			SUM(CASE WHEN f.outcome = 'success' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN f.outcome = 'failure' THEN 1 ELSE 0 END) as failed_count
		FROM invocation_read_model_facts f
		JOIN invocation_read_model_runs r ON r.run_id = f.run_id
		WHERE r.created_at >= ? AND r.created_at < ?`
	query, args = r.appendDurableRunFilters(query, args, filter, "r")
	query += `
		GROUP BY CASE WHEN f.tool_name = '' THEN 'unknown' ELSE f.tool_name END
		ORDER BY call_count DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []*repository.ToolUsageStats
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_tool_usage_stats", "Stats", "", err)
	}
	return rows, nil
}

// GetModelRunUsage returns run-level usage for a specific model.
func (r *statsRepository) GetModelRunUsage(ctx context.Context, filter repository.StatsFilter, model string, limit int) ([]*repository.ModelRunUsage, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End), model, limit}

	query := `
		SELECT
			irm.run_id as run_id,
			r.task_id as task_id,
			COALESCE(t.title, 'Unknown Task') as task_title,
			irm.profile_id as profile_id,
			COALESCE(p.name, 'Unknown Profile') as profile_name,
			irm.status as status,
			irm.created_at as created_at,
			irm.total_cost_usd as total_cost_usd,
			irm.total_tokens as total_tokens
		FROM invocation_read_model_runs irm
		JOIN runs r ON r.id = irm.run_id
		LEFT JOIN tasks t ON r.task_id = t.id
		LEFT JOIN agent_profiles p ON irm.profile_id = p.id
		WHERE irm.created_at >= ? AND irm.created_at < ?
		  AND irm.model = ?
		ORDER BY irm.created_at DESC
		LIMIT ?`

	var rows []modelRunUsageRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_model_run_usage", "Stats", "", err)
	}
	result := make([]*repository.ModelRunUsage, len(rows))
	for i, row := range rows {
		result[i] = row.toRepository()
	}
	return result, nil
}

// GetToolRunUsage returns run-level usage for a specific tool.
func (r *statsRepository) GetToolRunUsage(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolRunUsage, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}
	toolPredicate := "f.tool_name = ?"
	if toolName == "unknown" {
		toolPredicate = "(f.tool_name = '' OR f.tool_name = 'unknown')"
	} else {
		args = append(args, toolName)
	}
	query := `SELECT irm.run_id, r.task_id, COALESCE(t.title, 'Unknown Task') AS task_title, irm.profile_id, COALESCE(p.name, 'Unknown Profile') AS profile_name, irm.status, irm.created_at, irm.model,
		COUNT(*) AS call_count, SUM(CASE WHEN f.outcome = 'success' THEN 1 ELSE 0 END) AS success_count, SUM(CASE WHEN f.outcome = 'failure' THEN 1 ELSE 0 END) AS failed_count
		FROM invocation_read_model_facts f JOIN invocation_read_model_runs irm ON irm.run_id = f.run_id JOIN runs r ON r.id = irm.run_id LEFT JOIN tasks t ON r.task_id = t.id LEFT JOIN agent_profiles p ON irm.profile_id = p.id
		WHERE irm.created_at >= ? AND irm.created_at < ? AND ` + toolPredicate + `
		GROUP BY irm.run_id, r.task_id, t.title, irm.profile_id, p.name, irm.status, irm.created_at, irm.model ORDER BY call_count DESC, irm.created_at DESC LIMIT ?`
	args = append(args, limit)

	var rows []toolRunUsageRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_tool_run_usage", "Stats", "", err)
	}
	result := make([]*repository.ToolRunUsage, len(rows))
	for i, row := range rows {
		result[i] = row.toRepository()
	}
	return result, nil
}

// GetToolUsageByModel returns tool usage grouped by model.
func (r *statsRepository) GetToolUsageByModel(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolUsageModelBreakdown, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}
	toolPredicate := "f.tool_name = ?"
	if toolName == "unknown" {
		toolPredicate = "(f.tool_name = '' OR f.tool_name = 'unknown')"
	} else {
		args = append(args, toolName)
	}
	query := `SELECT irm.model, COUNT(DISTINCT irm.run_id) AS run_count, COUNT(*) AS call_count, SUM(CASE WHEN f.outcome = 'success' THEN 1 ELSE 0 END) AS success_count, SUM(CASE WHEN f.outcome = 'failure' THEN 1 ELSE 0 END) AS failed_count FROM invocation_read_model_facts f JOIN invocation_read_model_runs irm ON irm.run_id = f.run_id WHERE irm.created_at >= ? AND irm.created_at < ? AND ` + toolPredicate + ` GROUP BY irm.model ORDER BY call_count DESC LIMIT ?`
	args = append(args, limit)

	var rows []*repository.ToolUsageModelBreakdown
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_tool_usage_by_model", "Stats", "", err)
	}
	return rows, nil
}

// GetErrorPatterns aggregates error events.
func (r *statsRepository) GetErrorPatterns(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ErrorPattern, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}

	query := `
		SELECT
			e.error_code,
			COUNT(*) as count,
			MAX(e.occurred_at) as last_seen,
			(SELECT recent.run_id FROM invocation_read_model_errors recent WHERE recent.error_code = e.error_code ORDER BY recent.occurred_at DESC, recent.run_id ASC LIMIT 1) as sample_run_id
		FROM invocation_read_model_errors e
		JOIN invocation_read_model_runs r ON r.run_id = e.run_id
		WHERE r.created_at >= ? AND r.created_at < ?`
	query, args = r.appendDurableRunFilters(query, args, filter, "r")
	query += `
		GROUP BY e.error_code
		ORDER BY count DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []errorPatternRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_error_patterns", "Stats", "", err)
	}
	result := make([]*repository.ErrorPattern, len(rows))
	for i, row := range rows {
		result[i] = row.toRepository()
	}
	return result, nil
}

// GetTimeSeries returns time-bucketed data for charts.
func (r *statsRepository) GetTimeSeries(ctx context.Context, filter repository.StatsFilter, bucketDuration time.Duration) ([]*repository.TimeSeriesBucket, error) {
	args := []interface{}{SQLiteTime(filter.Window.Start), SQLiteTime(filter.Window.End)}
	bucketSQL := getBucketSQL(bucketDuration)

	query := fmt.Sprintf(`
		SELECT
			%s as timestamp,
			COUNT(DISTINCT r.run_id) as runs_started,
			COUNT(DISTINCT CASE WHEN r.status = 'complete' THEN r.run_id END) as runs_completed,
			COUNT(DISTINCT CASE WHEN r.status = 'failed' THEN r.run_id END) as runs_failed,
			COUNT(DISTINCT CASE WHEN r.status = 'cancelled' THEN r.run_id END) as runs_cancelled,
			COALESCE(SUM(r.total_cost_usd), 0) as total_cost_usd,
			COALESCE(CAST(AVG(r.duration_ms) AS INTEGER), 0) as avg_duration_ms
		FROM invocation_read_model_runs r
		WHERE r.created_at >= ? AND r.created_at < ?
		GROUP BY %s
		ORDER BY timestamp ASC`, bucketSQL, bucketSQL)

	query, args = r.appendDurableRunFilters(query, args, filter, "r")

	var rows []timeSeriesBucketRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_time_series", "Stats", "", err)
	}
	result := make([]*repository.TimeSeriesBucket, len(rows))
	for i, row := range rows {
		result[i] = row.toRepository()
	}
	return result, nil
}

// getBucketSQL returns the SQLite expression for time bucketing.
func getBucketSQL(bucketDuration time.Duration) string {
	switch {
	case bucketDuration >= 24*time.Hour:
		return "date(r.created_at)"
	case bucketDuration >= 6*time.Hour:
		// 6-hour buckets for 7d view
		return "strftime('%Y-%m-%d ', r.created_at) || printf('%02d', (cast(strftime('%H', r.created_at) as integer) / 6) * 6) || ':00:00'"
	case bucketDuration >= time.Hour:
		return "strftime('%Y-%m-%d %H:00:00', r.created_at)"
	default:
		return "strftime('%Y-%m-%d %H:%M:00', r.created_at)"
	}
}

// appendDurableRunFilters applies the legacy StatsFilter directly to the
// projected run schema. Compatibility endpoints therefore share the same
// authoritative dimensions as typed measures instead of reopening run JSON.
func (r *statsRepository) appendDurableRunFilters(query string, args []interface{}, filter repository.StatsFilter, alias string) (string, []interface{}) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	conditions := []string{}
	if len(filter.RunIDs) > 0 {
		placeholders := make([]string, len(filter.RunIDs))
		for i, id := range filter.RunIDs {
			placeholders[i] = "?"
			args = append(args, id.String())
		}
		conditions = append(conditions, fmt.Sprintf("%srun_id IN (%s)", prefix, strings.Join(placeholders, ",")))
	}
	if len(filter.RunnerTypes) > 0 {
		placeholders := make([]string, len(filter.RunnerTypes))
		for i, value := range filter.RunnerTypes {
			placeholders[i] = "?"
			args = append(args, string(value))
		}
		conditions = append(conditions, fmt.Sprintf("%srunner_type IN (%s)", prefix, strings.Join(placeholders, ",")))
	}
	if len(filter.ProfileIDs) > 0 {
		placeholders := make([]string, len(filter.ProfileIDs))
		for i, id := range filter.ProfileIDs {
			placeholders[i] = "?"
			args = append(args, id.String())
		}
		conditions = append(conditions, fmt.Sprintf("%sprofile_id IN (%s)", prefix, strings.Join(placeholders, ",")))
	}
	if len(filter.Models) > 0 {
		placeholders := make([]string, len(filter.Models))
		for i, value := range filter.Models {
			placeholders[i] = "?"
			args = append(args, value)
		}
		conditions = append(conditions, fmt.Sprintf("%smodel IN (%s)", prefix, strings.Join(placeholders, ",")))
	}
	if filter.TagPrefix != "" {
		conditions = append(conditions, fmt.Sprintf("%stag LIKE ?", prefix))
		args = append(args, filter.TagPrefix+"%")
	}
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	return query, args
}

// GetPopularModels returns the most used models by run count within a time window.
func (r *statsRepository) GetPopularModels(ctx context.Context, since time.Time, limit int) ([]string, error) {
	args := []interface{}{SQLiteTime(since), limit}

	query := `
		SELECT model
		FROM invocation_read_model_runs
		WHERE created_at >= ?
		  AND model != ''
		  AND model != 'unknown'
		GROUP BY model
		ORDER BY COUNT(*) DESC
		LIMIT ?`

	var models []string
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, wrapDBError("get_popular_models", "Stats", "", err)
	}
	return models, nil
}

// GetRecentModels returns recently used models ordered by most recent use (system-wide).
func (r *statsRepository) GetRecentModels(ctx context.Context, limit int) ([]string, error) {
	args := []interface{}{limit}

	// SQLite: Group by model, get max created_at, order by that
	query := `
		SELECT model
		FROM invocation_read_model_runs
		WHERE model != ''
		  AND model != 'unknown'
		GROUP BY model
		ORDER BY MAX(created_at) DESC
		LIMIT ?`

	var models []string
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, wrapDBError("get_recent_models", "Stats", "", err)
	}
	return models, nil
}
