package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
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

// GetRunStatusCounts returns counts of runs by status within the time window.
func (r *statsRepository) GetRunStatusCounts(ctx context.Context, filter repository.StatsFilter) (*repository.RunStatusCounts, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				COUNT(*) FILTER (WHERE status = 'pending') as pending,
				COUNT(*) FILTER (WHERE status = 'starting') as starting,
				COUNT(*) FILTER (WHERE status = 'running') as running,
				COUNT(*) FILTER (WHERE status = 'complete') as complete,
				COUNT(*) FILTER (WHERE status = 'failed') as failed,
				COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled,
				COUNT(*) FILTER (WHERE status = 'needs_review') as needs_review,
				COUNT(*) as total
			FROM runs
			WHERE created_at >= $1 AND created_at < $2`
	} else {
		// SQLite doesn't have FILTER, use CASE WHEN
		query = `
			SELECT
				SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
				SUM(CASE WHEN status = 'starting' THEN 1 ELSE 0 END) as starting,
				SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running,
				SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as complete,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
				SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled,
				SUM(CASE WHEN status = 'needs_review' THEN 1 ELSE 0 END) as needs_review,
				COUNT(*) as total
			FROM runs
			WHERE created_at >= ? AND created_at < ?`
	}

	query, args = r.appendFilters(query, args, filter)
	query = r.db.Rebind(query)

	var counts repository.RunStatusCounts
	if err := r.db.GetContext(ctx, &counts, query, args...); err != nil {
		return nil, wrapDBError("get_run_status_counts", "Stats", "", err)
	}
	return &counts, nil
}

// GetSuccessRate returns the ratio of complete runs to terminal runs.
func (r *statsRepository) GetSuccessRate(ctx context.Context, filter repository.StatsFilter) (float64, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				CASE
					WHEN COUNT(*) FILTER (WHERE status IN ('complete', 'failed', 'cancelled')) = 0
					THEN 0
					ELSE ROUND(
						COUNT(*) FILTER (WHERE status = 'complete')::numeric /
						NULLIF(COUNT(*) FILTER (WHERE status IN ('complete', 'failed', 'cancelled')), 0)::numeric,
						4
					)
				END as success_rate
			FROM runs
			WHERE created_at >= $1 AND created_at < $2`
	} else {
		query = `
			SELECT
				CASE
					WHEN SUM(CASE WHEN status IN ('complete', 'failed', 'cancelled') THEN 1 ELSE 0 END) = 0
					THEN 0.0
					ELSE ROUND(
						CAST(SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) AS REAL) /
						SUM(CASE WHEN status IN ('complete', 'failed', 'cancelled') THEN 1 ELSE 0 END),
						4
					)
				END as success_rate
			FROM runs
			WHERE created_at >= ? AND created_at < ?`
	}

	query, args = r.appendFilters(query, args, filter)
	query = r.db.Rebind(query)

	var rate float64
	if err := r.db.GetContext(ctx, &rate, query, args...); err != nil {
		return 0, wrapDBError("get_success_rate", "Stats", "", err)
	}
	return rate, nil
}

// GetDurationStats returns duration percentile statistics.
func (r *statsRepository) GetDurationStats(ctx context.Context, filter repository.StatsFilter) (*repository.DurationStats, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		// PostgreSQL has PERCENTILE_CONT for accurate percentiles
		query = `
			SELECT
				COALESCE(AVG(EXTRACT(EPOCH FROM (ended_at - started_at)) * 1000)::bigint, 0) as avg_ms,
				COALESCE((PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (ended_at - started_at)) * 1000))::bigint, 0) as p50_ms,
				COALESCE((PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (ended_at - started_at)) * 1000))::bigint, 0) as p95_ms,
				COALESCE((PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (ended_at - started_at)) * 1000))::bigint, 0) as p99_ms,
				COALESCE(MIN(EXTRACT(EPOCH FROM (ended_at - started_at)) * 1000)::bigint, 0) as min_ms,
				COALESCE(MAX(EXTRACT(EPOCH FROM (ended_at - started_at)) * 1000)::bigint, 0) as max_ms,
				COUNT(*) as count
			FROM runs
			WHERE created_at >= $1 AND created_at < $2
			  AND started_at IS NOT NULL
			  AND ended_at IS NOT NULL`
	} else {
		// SQLite: use simple AVG, MIN, MAX (no native percentiles)
		query = `
			SELECT
				COALESCE(CAST(AVG((julianday(ended_at) - julianday(started_at)) * 86400000) AS INTEGER), 0) as avg_ms,
				COALESCE(CAST(AVG((julianday(ended_at) - julianday(started_at)) * 86400000) AS INTEGER), 0) as p50_ms,
				COALESCE(CAST(AVG((julianday(ended_at) - julianday(started_at)) * 86400000) AS INTEGER), 0) as p95_ms,
				COALESCE(CAST(AVG((julianday(ended_at) - julianday(started_at)) * 86400000) AS INTEGER), 0) as p99_ms,
				COALESCE(CAST(MIN((julianday(ended_at) - julianday(started_at)) * 86400000) AS INTEGER), 0) as min_ms,
				COALESCE(CAST(MAX((julianday(ended_at) - julianday(started_at)) * 86400000) AS INTEGER), 0) as max_ms,
				COUNT(*) as count
			FROM runs
			WHERE created_at >= ? AND created_at < ?
			  AND started_at IS NOT NULL
			  AND ended_at IS NOT NULL`
	}

	query, args = r.appendFilters(query, args, filter)
	query = r.db.Rebind(query)

	var stats repository.DurationStats
	if err := r.db.GetContext(ctx, &stats, query, args...); err != nil {
		return nil, wrapDBError("get_duration_stats", "Stats", "", err)
	}
	return &stats, nil
}

// GetCostStats aggregates cost data from metric events.
func (r *statsRepository) GetCostStats(ctx context.Context, filter repository.StatsFilter) (*repository.CostStats, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		// Join runs with run_events to get cost data from metric events
		query = `
			SELECT
				COALESCE(SUM((e.data->>'totalCostUsd')::numeric), 0) as total_cost_usd,
				COALESCE(SUM(CASE WHEN e.data->>'costSource' IN ('runner_reported', 'provider_usage_api') THEN (e.data->>'totalCostUsd')::numeric ELSE 0 END), 0) as total_cost_usd_authoritative,
				COALESCE(SUM(CASE WHEN e.data->>'costSource' = 'pricing_table_estimate' THEN (e.data->>'totalCostUsd')::numeric ELSE 0 END), 0) as total_cost_usd_estimated,
				COALESCE(SUM(CASE WHEN e.data->>'costSource' IS NULL OR e.data->>'costSource' = '' OR e.data->>'costSource' = 'unknown' THEN (e.data->>'totalCostUsd')::numeric ELSE 0 END), 0) as total_cost_usd_unknown,
				COALESCE(SUM((e.data->>'inputCostUsd')::numeric), 0) as input_cost_usd,
				COALESCE(SUM((e.data->>'outputCostUsd')::numeric), 0) as output_cost_usd,
				COALESCE(SUM((e.data->>'cacheReadCostUsd')::numeric), 0) as cache_read_cost_usd,
				COALESCE(SUM((e.data->>'cacheCreationCostUsd')::numeric), 0) as cache_creation_cost_usd,
				COALESCE(AVG((e.data->>'totalCostUsd')::numeric), 0) as avg_cost_usd,
				COALESCE(SUM((e.data->>'inputTokens')::bigint), 0) as input_tokens,
				COALESCE(SUM((e.data->>'outputTokens')::bigint), 0) as output_tokens,
				COALESCE(SUM((e.data->>'cacheReadTokens')::bigint), 0) as cache_read_tokens,
				COALESCE(SUM(COALESCE((e.data->>'inputTokens')::bigint, 0) + COALESCE((e.data->>'outputTokens')::bigint, 0)), 0) as total_tokens
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= $1 AND r.created_at < $2`
	} else {
		query = `
			SELECT
				COALESCE(SUM(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as total_cost_usd,
				COALESCE(SUM(CASE WHEN json_extract(e.data, '$.costSource') IN ('runner_reported', 'provider_usage_api') THEN CAST(json_extract(e.data, '$.totalCostUsd') AS REAL) ELSE 0 END), 0) as total_cost_usd_authoritative,
				COALESCE(SUM(CASE WHEN json_extract(e.data, '$.costSource') = 'pricing_table_estimate' THEN CAST(json_extract(e.data, '$.totalCostUsd') AS REAL) ELSE 0 END), 0) as total_cost_usd_estimated,
				COALESCE(SUM(CASE WHEN json_extract(e.data, '$.costSource') IS NULL OR json_extract(e.data, '$.costSource') = '' OR json_extract(e.data, '$.costSource') = 'unknown' THEN CAST(json_extract(e.data, '$.totalCostUsd') AS REAL) ELSE 0 END), 0) as total_cost_usd_unknown,
				COALESCE(SUM(CAST(json_extract(e.data, '$.inputCostUsd') AS REAL)), 0) as input_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.outputCostUsd') AS REAL)), 0) as output_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.cacheReadCostUsd') AS REAL)), 0) as cache_read_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.cacheCreationCostUsd') AS REAL)), 0) as cache_creation_cost_usd,
				COALESCE(AVG(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as avg_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.inputTokens') AS INTEGER)), 0) as input_tokens,
				COALESCE(SUM(CAST(json_extract(e.data, '$.outputTokens') AS INTEGER)), 0) as output_tokens,
				COALESCE(SUM(CAST(json_extract(e.data, '$.cacheReadTokens') AS INTEGER)), 0) as cache_read_tokens,
				COALESCE(SUM(
					COALESCE(CAST(json_extract(e.data, '$.inputTokens') AS INTEGER), 0) +
					COALESCE(CAST(json_extract(e.data, '$.outputTokens') AS INTEGER), 0)
				), 0) as total_tokens
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= ? AND r.created_at < ?`
	}

	query, args = r.appendFiltersWithAlias(query, args, filter, "r")
	query = r.db.Rebind(query)

	var stats repository.CostStats
	if err := r.db.GetContext(ctx, &stats, query, args...); err != nil {
		return nil, wrapDBError("get_cost_stats", "Stats", "", err)
	}
	return &stats, nil
}

// GetRunnerBreakdown returns stats grouped by runner type.
func (r *statsRepository) GetRunnerBreakdown(ctx context.Context, filter repository.StatsFilter) ([]*repository.RunnerBreakdown, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				r.resolved_config->>'runnerType' as runner_type,
				COUNT(*) as run_count,
				COUNT(*) FILTER (WHERE r.status = 'complete') as success_count,
				COUNT(*) FILTER (WHERE r.status = 'failed') as failed_count,
				COALESCE(SUM((e.data->>'totalCostUsd')::numeric), 0) as total_cost_usd,
				COALESCE(AVG(EXTRACT(EPOCH FROM (r.ended_at - r.started_at)) * 1000)::bigint, 0) as avg_duration_ms
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= $1 AND r.created_at < $2
			  AND r.resolved_config IS NOT NULL
			  AND r.resolved_config->>'runnerType' IS NOT NULL
			GROUP BY r.resolved_config->>'runnerType'
			ORDER BY run_count DESC`
	} else {
		query = `
			SELECT
				json_extract(r.resolved_config, '$.runnerType') as runner_type,
				COUNT(*) as run_count,
				SUM(CASE WHEN r.status = 'complete' THEN 1 ELSE 0 END) as success_count,
				SUM(CASE WHEN r.status = 'failed' THEN 1 ELSE 0 END) as failed_count,
				COALESCE(SUM(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as total_cost_usd,
				COALESCE(CAST(AVG((julianday(r.ended_at) - julianday(r.started_at)) * 86400000) AS INTEGER), 0) as avg_duration_ms
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= ? AND r.created_at < ?
			  AND r.resolved_config IS NOT NULL
			  AND json_extract(r.resolved_config, '$.runnerType') IS NOT NULL
			GROUP BY json_extract(r.resolved_config, '$.runnerType')
			ORDER BY run_count DESC`
	}

	query = r.db.Rebind(query)

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
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				r.agent_profile_id as profile_id,
				COALESCE(p.name, 'Unknown') as profile_name,
				COUNT(*) as run_count,
				COUNT(*) FILTER (WHERE r.status = 'complete') as success_count,
				COUNT(*) FILTER (WHERE r.status = 'failed') as failed_count,
				COALESCE(SUM((e.data->>'totalCostUsd')::numeric), 0) as total_cost_usd
			FROM runs r
			LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= $1 AND r.created_at < $2
			  AND r.agent_profile_id IS NOT NULL
			GROUP BY r.agent_profile_id, p.name
			ORDER BY run_count DESC
			LIMIT $3`
		args = append(args, limit)
	} else {
		query = `
			SELECT
				r.agent_profile_id as profile_id,
				COALESCE(p.name, 'Unknown') as profile_name,
				COUNT(*) as run_count,
				SUM(CASE WHEN r.status = 'complete' THEN 1 ELSE 0 END) as success_count,
				SUM(CASE WHEN r.status = 'failed' THEN 1 ELSE 0 END) as failed_count,
				COALESCE(SUM(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as total_cost_usd
			FROM runs r
			LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= ? AND r.created_at < ?
			  AND r.agent_profile_id IS NOT NULL
			GROUP BY r.agent_profile_id, p.name
			ORDER BY run_count DESC
			LIMIT ?`
		args = append(args, limit)
	}

	query = r.db.Rebind(query)

	var rows []*repository.ProfileBreakdown
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_profile_breakdown", "Stats", "", err)
	}
	return rows, nil
}

// GetModelBreakdown returns stats grouped by model.
func (r *statsRepository) GetModelBreakdown(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ModelBreakdown, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				COALESCE(r.resolved_config->>'model', 'unknown') as model,
				COUNT(*) as run_count,
				COUNT(*) FILTER (WHERE r.status = 'complete') as success_count,
				COALESCE(SUM((e.data->>'totalCostUsd')::numeric), 0) as total_cost_usd,
				COALESCE(SUM(CASE WHEN e.data->>'costSource' IN ('runner_reported', 'provider_usage_api') THEN (e.data->>'totalCostUsd')::numeric ELSE 0 END), 0) as total_cost_usd_authoritative,
				COALESCE(SUM(CASE WHEN e.data->>'costSource' = 'pricing_table_estimate' THEN (e.data->>'totalCostUsd')::numeric ELSE 0 END), 0) as total_cost_usd_estimated,
				COALESCE(SUM(CASE WHEN e.data->>'costSource' IS NULL OR e.data->>'costSource' = '' OR e.data->>'costSource' = 'unknown' THEN (e.data->>'totalCostUsd')::numeric ELSE 0 END), 0) as total_cost_usd_unknown,
				COALESCE(SUM((e.data->>'inputCostUsd')::numeric), 0) as input_cost_usd,
				COALESCE(SUM((e.data->>'outputCostUsd')::numeric), 0) as output_cost_usd,
				COALESCE(SUM((e.data->>'cacheReadCostUsd')::numeric), 0) as cache_read_cost_usd,
				COALESCE(SUM((e.data->>'cacheCreationCostUsd')::numeric), 0) as cache_creation_cost_usd,
				COALESCE(SUM(COALESCE((e.data->>'inputTokens')::bigint, 0) + COALESCE((e.data->>'outputTokens')::bigint, 0)), 0) as total_tokens
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= $1 AND r.created_at < $2
			  AND r.resolved_config IS NOT NULL
			GROUP BY r.resolved_config->>'model'
			ORDER BY run_count DESC
			LIMIT $3`
		args = append(args, limit)
	} else {
		query = `
			SELECT
				COALESCE(json_extract(r.resolved_config, '$.model'), 'unknown') as model,
				COUNT(*) as run_count,
				SUM(CASE WHEN r.status = 'complete' THEN 1 ELSE 0 END) as success_count,
				COALESCE(SUM(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as total_cost_usd,
				COALESCE(SUM(CASE WHEN json_extract(e.data, '$.costSource') IN ('runner_reported', 'provider_usage_api') THEN CAST(json_extract(e.data, '$.totalCostUsd') AS REAL) ELSE 0 END), 0) as total_cost_usd_authoritative,
				COALESCE(SUM(CASE WHEN json_extract(e.data, '$.costSource') = 'pricing_table_estimate' THEN CAST(json_extract(e.data, '$.totalCostUsd') AS REAL) ELSE 0 END), 0) as total_cost_usd_estimated,
				COALESCE(SUM(CASE WHEN json_extract(e.data, '$.costSource') IS NULL OR json_extract(e.data, '$.costSource') = '' OR json_extract(e.data, '$.costSource') = 'unknown' THEN CAST(json_extract(e.data, '$.totalCostUsd') AS REAL) ELSE 0 END), 0) as total_cost_usd_unknown,
				COALESCE(SUM(CAST(json_extract(e.data, '$.inputCostUsd') AS REAL)), 0) as input_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.outputCostUsd') AS REAL)), 0) as output_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.cacheReadCostUsd') AS REAL)), 0) as cache_read_cost_usd,
				COALESCE(SUM(CAST(json_extract(e.data, '$.cacheCreationCostUsd') AS REAL)), 0) as cache_creation_cost_usd,
				COALESCE(SUM(
					COALESCE(CAST(json_extract(e.data, '$.inputTokens') AS INTEGER), 0) +
					COALESCE(CAST(json_extract(e.data, '$.outputTokens') AS INTEGER), 0)
				), 0) as total_tokens
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= ? AND r.created_at < ?
			  AND r.resolved_config IS NOT NULL
			GROUP BY json_extract(r.resolved_config, '$.model')
			ORDER BY run_count DESC
			LIMIT ?`
		args = append(args, limit)
	}

	query = r.db.Rebind(query)

	var rows []*repository.ModelBreakdown
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_model_breakdown", "Stats", "", err)
	}
	return rows, nil
}

// GetToolUsageStats aggregates tool call events.
func (r *statsRepository) GetToolUsageStats(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ToolUsageStats, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		// Use NULLIF to convert empty string to NULL, then COALESCE to convert NULL to 'unknown'
		// This normalizes both NULL and empty string toolNames to 'unknown'
		query = `
			SELECT
				COALESCE(NULLIF(e.data->>'toolName', ''), 'unknown') as tool_name,
				COUNT(*) as call_count,
				COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = true) as success_count,
				COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = false) as failed_count
			FROM run_events e
			JOIN runs r ON e.run_id = r.id
			WHERE r.created_at >= $1 AND r.created_at < $2
			  AND e.event_type IN ('tool_call', 'tool_result')
			GROUP BY COALESCE(NULLIF(e.data->>'toolName', ''), 'unknown')
			ORDER BY call_count DESC
			LIMIT $3`
		args = append(args, limit)
	} else {
		// SQLite: Use CASE to normalize empty/NULL to 'unknown'
		query = `
			SELECT
				CASE
					WHEN json_extract(e.data, '$.toolName') IS NULL OR json_extract(e.data, '$.toolName') = ''
					THEN 'unknown'
					ELSE json_extract(e.data, '$.toolName')
				END as tool_name,
				COUNT(*) as call_count,
				SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 1 THEN 1 ELSE 0 END) as success_count,
				SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 0 THEN 1 ELSE 0 END) as failed_count
			FROM run_events e
			JOIN runs r ON e.run_id = r.id
			WHERE r.created_at >= ? AND r.created_at < ?
			  AND e.event_type IN ('tool_call', 'tool_result')
			GROUP BY CASE
				WHEN json_extract(e.data, '$.toolName') IS NULL OR json_extract(e.data, '$.toolName') = ''
				THEN 'unknown'
				ELSE json_extract(e.data, '$.toolName')
			END
			ORDER BY call_count DESC
			LIMIT ?`
		args = append(args, limit)
	}

	query = r.db.Rebind(query)

	var rows []*repository.ToolUsageStats
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_tool_usage_stats", "Stats", "", err)
	}
	return rows, nil
}

// GetModelRunUsage returns run-level usage for a specific model.
func (r *statsRepository) GetModelRunUsage(ctx context.Context, filter repository.StatsFilter, model string, limit int) ([]*repository.ModelRunUsage, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End, model, limit}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				r.id as run_id,
				r.task_id as task_id,
				COALESCE(t.title, 'Unknown Task') as task_title,
				r.agent_profile_id as profile_id,
				COALESCE(p.name, 'Unknown Profile') as profile_name,
				r.status as status,
				r.created_at as created_at,
				COALESCE(SUM((e.data->>'totalCostUsd')::numeric), 0) as total_cost_usd,
				COALESCE(SUM(COALESCE((e.data->>'inputTokens')::bigint, 0) + COALESCE((e.data->>'outputTokens')::bigint, 0)), 0) as total_tokens
			FROM runs r
			LEFT JOIN tasks t ON r.task_id = t.id
			LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= $1 AND r.created_at < $2
			  AND COALESCE(r.resolved_config->>'model', 'unknown') = $3
			GROUP BY r.id, r.task_id, t.title, r.agent_profile_id, p.name, r.status, r.created_at
			ORDER BY r.created_at DESC
			LIMIT $4`
	} else {
		query = `
			SELECT
				r.id as run_id,
				r.task_id as task_id,
				COALESCE(t.title, 'Unknown Task') as task_title,
				r.agent_profile_id as profile_id,
				COALESCE(p.name, 'Unknown Profile') as profile_name,
				r.status as status,
				r.created_at as created_at,
				COALESCE(SUM(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as total_cost_usd,
				COALESCE(SUM(
					COALESCE(CAST(json_extract(e.data, '$.inputTokens') AS INTEGER), 0) +
					COALESCE(CAST(json_extract(e.data, '$.outputTokens') AS INTEGER), 0)
				), 0) as total_tokens
			FROM runs r
			LEFT JOIN tasks t ON r.task_id = t.id
			LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= ? AND r.created_at < ?
			  AND COALESCE(json_extract(r.resolved_config, '$.model'), 'unknown') = ?
			GROUP BY r.id, r.task_id, t.title, r.agent_profile_id, p.name, r.status, r.created_at
			ORDER BY r.created_at DESC
			LIMIT ?`
	}

	query = r.db.Rebind(query)

	var rows []*repository.ModelRunUsage
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_model_run_usage", "Stats", "", err)
	}
	return rows, nil
}

// GetToolRunUsage returns run-level usage for a specific tool.
func (r *statsRepository) GetToolRunUsage(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolRunUsage, error) {
	var query string
	var args []interface{}

	// When toolName is "unknown", match events where toolName is NULL, empty, or "unknown"
	// This corresponds to the normalization done in GetToolUsageStats
	isUnknown := toolName == "unknown"

	if r.db.Dialect() == DialectPostgres {
		if isUnknown {
			args = []interface{}{filter.Window.Start, filter.Window.End, limit}
			query = `
				SELECT
					r.id as run_id,
					r.task_id as task_id,
					COALESCE(t.title, 'Unknown Task') as task_title,
					r.agent_profile_id as profile_id,
					COALESCE(p.name, 'Unknown Profile') as profile_name,
					r.status as status,
					r.created_at as created_at,
					COALESCE(r.resolved_config->>'model', 'unknown') as model,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_call') as call_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = true) as success_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = false) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				LEFT JOIN tasks t ON r.task_id = t.id
				LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
				WHERE r.created_at >= $1 AND r.created_at < $2
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND (e.data->>'toolName' IS NULL OR e.data->>'toolName' = '' OR e.data->>'toolName' = 'unknown')
				GROUP BY r.id, r.task_id, t.title, r.agent_profile_id, p.name, r.status, r.created_at, r.resolved_config
				ORDER BY call_count DESC, r.created_at DESC
				LIMIT $3`
		} else {
			args = []interface{}{filter.Window.Start, filter.Window.End, toolName, limit}
			query = `
				SELECT
					r.id as run_id,
					r.task_id as task_id,
					COALESCE(t.title, 'Unknown Task') as task_title,
					r.agent_profile_id as profile_id,
					COALESCE(p.name, 'Unknown Profile') as profile_name,
					r.status as status,
					r.created_at as created_at,
					COALESCE(r.resolved_config->>'model', 'unknown') as model,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_call') as call_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = true) as success_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = false) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				LEFT JOIN tasks t ON r.task_id = t.id
				LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
				WHERE r.created_at >= $1 AND r.created_at < $2
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND e.data->>'toolName' = $3
				GROUP BY r.id, r.task_id, t.title, r.agent_profile_id, p.name, r.status, r.created_at, r.resolved_config
				ORDER BY call_count DESC, r.created_at DESC
				LIMIT $4`
		}
	} else {
		if isUnknown {
			args = []interface{}{filter.Window.Start, filter.Window.End, limit}
			query = `
				SELECT
					r.id as run_id,
					r.task_id as task_id,
					COALESCE(t.title, 'Unknown Task') as task_title,
					r.agent_profile_id as profile_id,
					COALESCE(p.name, 'Unknown Profile') as profile_name,
					r.status as status,
					r.created_at as created_at,
					COALESCE(json_extract(r.resolved_config, '$.model'), 'unknown') as model,
					SUM(CASE WHEN e.event_type = 'tool_call' THEN 1 ELSE 0 END) as call_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 1 THEN 1 ELSE 0 END) as success_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 0 THEN 1 ELSE 0 END) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				LEFT JOIN tasks t ON r.task_id = t.id
				LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
				WHERE r.created_at >= ? AND r.created_at < ?
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND (json_extract(e.data, '$.toolName') IS NULL OR json_extract(e.data, '$.toolName') = '' OR json_extract(e.data, '$.toolName') = 'unknown')
				GROUP BY r.id, r.task_id, t.title, r.agent_profile_id, p.name, r.status, r.created_at, r.resolved_config
				ORDER BY call_count DESC, r.created_at DESC
				LIMIT ?`
		} else {
			args = []interface{}{filter.Window.Start, filter.Window.End, toolName, limit}
			query = `
				SELECT
					r.id as run_id,
					r.task_id as task_id,
					COALESCE(t.title, 'Unknown Task') as task_title,
					r.agent_profile_id as profile_id,
					COALESCE(p.name, 'Unknown Profile') as profile_name,
					r.status as status,
					r.created_at as created_at,
					COALESCE(json_extract(r.resolved_config, '$.model'), 'unknown') as model,
					SUM(CASE WHEN e.event_type = 'tool_call' THEN 1 ELSE 0 END) as call_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 1 THEN 1 ELSE 0 END) as success_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 0 THEN 1 ELSE 0 END) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				LEFT JOIN tasks t ON r.task_id = t.id
				LEFT JOIN agent_profiles p ON r.agent_profile_id = p.id
				WHERE r.created_at >= ? AND r.created_at < ?
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND json_extract(e.data, '$.toolName') = ?
				GROUP BY r.id, r.task_id, t.title, r.agent_profile_id, p.name, r.status, r.created_at, r.resolved_config
				ORDER BY call_count DESC, r.created_at DESC
				LIMIT ?`
		}
	}

	query = r.db.Rebind(query)

	var rows []*repository.ToolRunUsage
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_tool_run_usage", "Stats", "", err)
	}
	return rows, nil
}

// GetToolUsageByModel returns tool usage grouped by model.
func (r *statsRepository) GetToolUsageByModel(ctx context.Context, filter repository.StatsFilter, toolName string, limit int) ([]*repository.ToolUsageModelBreakdown, error) {
	var query string
	var args []interface{}

	// When toolName is "unknown", match events where toolName is NULL, empty, or "unknown"
	// This corresponds to the normalization done in GetToolUsageStats
	isUnknown := toolName == "unknown"

	if r.db.Dialect() == DialectPostgres {
		if isUnknown {
			args = []interface{}{filter.Window.Start, filter.Window.End, limit}
			query = `
				SELECT
					COALESCE(r.resolved_config->>'model', 'unknown') as model,
					COUNT(DISTINCT r.id) as run_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_call') as call_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = true) as success_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = false) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				WHERE r.created_at >= $1 AND r.created_at < $2
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND (e.data->>'toolName' IS NULL OR e.data->>'toolName' = '' OR e.data->>'toolName' = 'unknown')
				GROUP BY r.resolved_config->>'model'
				ORDER BY call_count DESC
				LIMIT $3`
		} else {
			args = []interface{}{filter.Window.Start, filter.Window.End, toolName, limit}
			query = `
				SELECT
					COALESCE(r.resolved_config->>'model', 'unknown') as model,
					COUNT(DISTINCT r.id) as run_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_call') as call_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = true) as success_count,
					COUNT(*) FILTER (WHERE e.event_type = 'tool_result' AND (e.data->>'success')::boolean = false) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				WHERE r.created_at >= $1 AND r.created_at < $2
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND e.data->>'toolName' = $3
				GROUP BY r.resolved_config->>'model'
				ORDER BY call_count DESC
				LIMIT $4`
		}
	} else {
		if isUnknown {
			args = []interface{}{filter.Window.Start, filter.Window.End, limit}
			query = `
				SELECT
					COALESCE(json_extract(r.resolved_config, '$.model'), 'unknown') as model,
					COUNT(DISTINCT r.id) as run_count,
					SUM(CASE WHEN e.event_type = 'tool_call' THEN 1 ELSE 0 END) as call_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 1 THEN 1 ELSE 0 END) as success_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 0 THEN 1 ELSE 0 END) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				WHERE r.created_at >= ? AND r.created_at < ?
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND (json_extract(e.data, '$.toolName') IS NULL OR json_extract(e.data, '$.toolName') = '' OR json_extract(e.data, '$.toolName') = 'unknown')
				GROUP BY json_extract(r.resolved_config, '$.model')
				ORDER BY call_count DESC
				LIMIT ?`
		} else {
			args = []interface{}{filter.Window.Start, filter.Window.End, toolName, limit}
			query = `
				SELECT
					COALESCE(json_extract(r.resolved_config, '$.model'), 'unknown') as model,
					COUNT(DISTINCT r.id) as run_count,
					SUM(CASE WHEN e.event_type = 'tool_call' THEN 1 ELSE 0 END) as call_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 1 THEN 1 ELSE 0 END) as success_count,
					SUM(CASE WHEN e.event_type = 'tool_result' AND json_extract(e.data, '$.success') = 0 THEN 1 ELSE 0 END) as failed_count
				FROM run_events e
				JOIN runs r ON e.run_id = r.id
				WHERE r.created_at >= ? AND r.created_at < ?
				  AND e.event_type IN ('tool_call', 'tool_result')
				  AND json_extract(e.data, '$.toolName') = ?
				GROUP BY json_extract(r.resolved_config, '$.model')
				ORDER BY call_count DESC
				LIMIT ?`
		}
	}

	query = r.db.Rebind(query)

	var rows []*repository.ToolUsageModelBreakdown
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_tool_usage_by_model", "Stats", "", err)
	}
	return rows, nil
}

// GetErrorPatterns aggregates error events.
func (r *statsRepository) GetErrorPatterns(ctx context.Context, filter repository.StatsFilter, limit int) ([]*repository.ErrorPattern, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}

	if r.db.Dialect() == DialectPostgres {
		query = `
			SELECT
				COALESCE(e.data->>'code', 'unknown') as error_code,
				COUNT(*) as count,
				MAX(e.timestamp) as last_seen,
				(array_agg(e.run_id ORDER BY e.timestamp DESC))[1] as sample_run_id
			FROM run_events e
			JOIN runs r ON e.run_id = r.id
			WHERE r.created_at >= $1 AND r.created_at < $2
			  AND e.event_type = 'error'
			GROUP BY e.data->>'code'
			ORDER BY count DESC
			LIMIT $3`
		args = append(args, limit)
	} else {
		// SQLite doesn't have array_agg, use a subquery
		query = `
			SELECT
				COALESCE(json_extract(e.data, '$.code'), 'unknown') as error_code,
				COUNT(*) as count,
				MAX(e.timestamp) as last_seen,
				(SELECT e2.run_id FROM run_events e2
				 WHERE e2.event_type = 'error'
				   AND COALESCE(json_extract(e2.data, '$.code'), 'unknown') = COALESCE(json_extract(e.data, '$.code'), 'unknown')
				 ORDER BY e2.timestamp DESC LIMIT 1) as sample_run_id
			FROM run_events e
			JOIN runs r ON e.run_id = r.id
			WHERE r.created_at >= ? AND r.created_at < ?
			  AND e.event_type = 'error'
			GROUP BY json_extract(e.data, '$.code')
			ORDER BY count DESC
			LIMIT ?`
		args = append(args, limit)
	}

	query = r.db.Rebind(query)

	var rows []*repository.ErrorPattern
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_error_patterns", "Stats", "", err)
	}
	return rows, nil
}

// GetTimeSeries returns time-bucketed data for charts.
func (r *statsRepository) GetTimeSeries(ctx context.Context, filter repository.StatsFilter, bucketDuration time.Duration) ([]*repository.TimeSeriesBucket, error) {
	var query string
	args := []interface{}{filter.Window.Start, filter.Window.End}
	bucketSQL := r.getBucketSQL(bucketDuration)

	if r.db.Dialect() == DialectPostgres {
		query = fmt.Sprintf(`
			SELECT
				%s as timestamp,
				COUNT(DISTINCT r.id) as runs_started,
				COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'complete') as runs_completed,
				COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'failed') as runs_failed,
				COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'cancelled') as runs_cancelled,
				COALESCE(SUM((e.data->>'totalCostUsd')::numeric), 0) as total_cost_usd,
				COALESCE(AVG(EXTRACT(EPOCH FROM (r.ended_at - r.started_at)) * 1000)::bigint, 0) as avg_duration_ms
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= $1 AND r.created_at < $2
			GROUP BY %s
			ORDER BY timestamp ASC`, bucketSQL, bucketSQL)
	} else {
		bucketSQLite := r.getBucketSQLite(bucketDuration)
		query = fmt.Sprintf(`
			SELECT
				%s as timestamp,
				COUNT(DISTINCT r.id) as runs_started,
				COUNT(DISTINCT CASE WHEN r.status = 'complete' THEN r.id END) as runs_completed,
				COUNT(DISTINCT CASE WHEN r.status = 'failed' THEN r.id END) as runs_failed,
				COUNT(DISTINCT CASE WHEN r.status = 'cancelled' THEN r.id END) as runs_cancelled,
				COALESCE(SUM(CAST(json_extract(e.data, '$.totalCostUsd') AS REAL)), 0) as total_cost_usd,
				COALESCE(CAST(AVG((julianday(r.ended_at) - julianday(r.started_at)) * 86400000) AS INTEGER), 0) as avg_duration_ms
			FROM runs r
			LEFT JOIN run_events e ON r.id = e.run_id AND e.event_type = 'metric'
			WHERE r.created_at >= ? AND r.created_at < ?
			GROUP BY %s
			ORDER BY timestamp ASC`, bucketSQLite, bucketSQLite)
	}

	query, args = r.appendFiltersWithAlias(query, args, filter, "r")
	query = r.db.Rebind(query)

	var rows []*repository.TimeSeriesBucket
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_time_series", "Stats", "", err)
	}
	return rows, nil
}

// getBucketSQL returns the PostgreSQL expression for time bucketing.
func (r *statsRepository) getBucketSQL(bucketDuration time.Duration) string {
	switch {
	case bucketDuration >= 24*time.Hour:
		return "date_trunc('day', r.created_at)"
	case bucketDuration >= 6*time.Hour:
		// 6-hour buckets for 7d view
		return "date_trunc('day', r.created_at) + (floor(extract(hour from r.created_at)/6) * interval '6 hours')"
	case bucketDuration >= time.Hour:
		return "date_trunc('hour', r.created_at)"
	case bucketDuration >= 15*time.Minute:
		mins := int(bucketDuration.Minutes())
		return fmt.Sprintf("date_trunc('hour', r.created_at) + (floor(extract(minute from r.created_at)/%d) * interval '%d minutes')", mins, mins)
	default:
		return "date_trunc('minute', r.created_at)"
	}
}

// getBucketSQLite returns the SQLite expression for time bucketing.
func (r *statsRepository) getBucketSQLite(bucketDuration time.Duration) string {
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

// appendFilters adds common filter conditions to a query.
func (r *statsRepository) appendFilters(query string, args []interface{}, filter repository.StatsFilter) (string, []interface{}) {
	return r.appendFiltersWithAlias(query, args, filter, "")
}

// appendFiltersWithAlias adds common filter conditions with table alias.
func (r *statsRepository) appendFiltersWithAlias(query string, args []interface{}, filter repository.StatsFilter, alias string) (string, []interface{}) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}

	var conditions []string

	if len(filter.RunnerTypes) > 0 {
		placeholders := make([]string, len(filter.RunnerTypes))
		for i, rt := range filter.RunnerTypes {
			placeholders[i] = "?"
			args = append(args, string(rt))
		}
		if r.db.Dialect() == DialectPostgres {
			conditions = append(conditions, fmt.Sprintf("%sresolved_config->>'runnerType' IN (%s)", prefix, strings.Join(placeholders, ",")))
		} else {
			conditions = append(conditions, fmt.Sprintf("json_extract(%sresolved_config, '$.runnerType') IN (%s)", prefix, strings.Join(placeholders, ",")))
		}
	}

	if len(filter.ProfileIDs) > 0 {
		placeholders := make([]string, len(filter.ProfileIDs))
		for i, id := range filter.ProfileIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("%sagent_profile_id IN (%s)", prefix, strings.Join(placeholders, ",")))
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
