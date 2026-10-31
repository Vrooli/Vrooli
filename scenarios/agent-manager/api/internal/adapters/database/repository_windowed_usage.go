package database

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/invocationreadmodel"
)

// Token and quota totals over a time window come from transcript_usage, one
// row per request, bucketed by when the request happened (P2-D6). The
// per-run lifetime counters on invocation_read_model_runs only serve per-run
// views; summing them at a run's end time puts a month of spend into the week
// the run finished.

const (
	// executedModes are the execution modes Agent Manager drives itself. A run
	// in any other mode is an operator session Agent Manager only observed.
	operatorSessionModes = "('imported','attached','interactive')"

	// windowUsageSelect aggregates the window's requests per run and model. A
	// harness session copied into several harness homes appears as several
	// transcript files holding the same requests; a request identical to one in
	// an earlier file of the same session counts once.
	windowUsageSelect = `uw AS (SELECT u.run_id AS uw_run_id, u.model AS uw_model,
		SUM(u.input_tokens) AS uw_in, SUM(u.output_tokens) AS uw_out,
		SUM(u.cache_read_tokens) AS uw_cr, SUM(u.cache_creation_tokens) AS uw_cc, COUNT(*) AS uw_requests
		FROM transcript_usage u LEFT JOIN transcript_files f ON f.file_key = u.file_key
		WHERE u.observed_at >= ? AND u.observed_at < ? AND NOT EXISTS (
			SELECT 1 FROM transcript_usage v JOIN transcript_files g ON g.file_key = v.file_key
			WHERE v.observed_at = u.observed_at AND f.sidechain = 0 AND g.sidechain = 0 AND f.session_id <> ''
			AND g.session_id = f.session_id AND g.harness = f.harness AND v.file_key < u.file_key
			AND v.turn_index = u.turn_index AND v.input_tokens = u.input_tokens AND v.output_tokens = u.output_tokens
			AND v.cache_read_tokens = u.cache_read_tokens AND v.cache_creation_tokens = u.cache_creation_tokens)
		GROUP BY u.run_id, u.model)`

	windowVolume = "(uw_in + uw_out + uw_cr + uw_cc)"
	// windowShare is the part of a run's lifetime tokens requested in the
	// window, used to apportion lifetime charges to it.
	windowShare = "MIN(1.0, " + windowVolume + " * 1.0 / NULLIF(invocation_read_model_runs.total_tokens, 0))"
	metered     = "COALESCE(json_extract(r.billing_snapshot, '$.mode'), '') = 'metered'"
)

// windowedUsageCTE returns the usage CTE for the filter's window and its
// arguments. A filter without a window spans all time.
func windowedUsageCTE(filter invocationreadmodel.Filter) (string, []any) {
	from, to := "0000", "9999"
	if filter.From != nil {
		from = filter.From.UTC().Format(time.RFC3339Nano)
	}
	if filter.To != nil {
		to = filter.To.UTC().Format(time.RFC3339Nano)
	}
	return windowUsageSelect, []any{from, to}
}

// runScopeClause restricts invocation_read_model_runs to a scope. An
// unspecified scope counts Agent Manager runs for run statistics and
// everything for token totals (tokens true).
func runScopeClause(scope invocationreadmodel.RunScope, tokens bool) string {
	const runs = "invocation_read_model_runs.run_id"
	executed := runs + " NOT IN (SELECT id FROM runs WHERE execution_mode IN " + operatorSessionModes + ")"
	// An observed session with no usage is not a run, and a session copy is
	// not a second one.
	operator := "(" + runs + " IN (SELECT id FROM runs WHERE execution_mode IN " + operatorSessionModes + ") AND invocation_read_model_runs.total_tokens > 0" +
		" AND NOT EXISTS (SELECT 1 FROM transcript_files f JOIN transcript_files g ON g.harness = f.harness AND g.session_id = f.session_id" +
		" AND g.sidechain = 0 AND g.file_key < f.file_key AND g.run_id <> '' WHERE f.run_id = " + runs + " AND f.sidechain = 0 AND f.session_id <> ''))"
	switch scope {
	case invocationreadmodel.ScopeAll:
		return "(" + executed + " OR " + operator + ")"
	case invocationreadmodel.ScopeOperatorSessions:
		return operator
	case invocationreadmodel.ScopeAgentManager:
		return executed
	}
	if tokens {
		return ""
	}
	return executed
}

// WindowedUsage totals the requests observed in the filter's window.
func (r *invocationReadModelRepository) WindowedUsage(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.UsageTotals, error) {
	cte, args := windowedUsageCTE(filter)
	where, whereArgs := runWhereClauses(filter, false, true)
	cost := func(column string) string {
		return "COALESCE(SUM(CASE WHEN " + metered + " THEN invocation_read_model_runs." + column + " * " + windowShare + " ELSE 0 END), 0)"
	}
	query := "WITH " + cte + ` SELECT COALESCE(SUM(uw_requests), 0), COUNT(DISTINCT uw_run_id),
		COALESCE(SUM(uw_in), 0), COALESCE(SUM(uw_out), 0), COALESCE(SUM(uw_cr), 0), COALESCE(SUM(uw_cc), 0),
		` + cost("total_cost_usd") + `, ` + cost("input_cost_usd") + `, ` + cost("output_cost_usd") + `, ` + cost("cache_read_cost_usd") + `, ` + cost("cache_creation_cost_usd") + `,
		` + cost("total_charge_micro_usd") + `
		FROM uw LEFT JOIN invocation_read_model_runs ON invocation_read_model_runs.run_id = uw.uw_run_id
		LEFT JOIN runs r ON r.id = uw.uw_run_id` + where
	var totals invocationreadmodel.UsageTotals
	var charge float64
	if err := r.db.QueryRowContext(ctx, query, append(args, whereArgs...)...).Scan(&totals.Requests, &totals.Runs, &totals.InputTokens, &totals.OutputTokens, &totals.CacheReadTokens, &totals.CacheCreationTokens,
		&totals.MeteredCostUSD, &totals.InputCostUSD, &totals.OutputCostUSD, &totals.CacheReadCostUSD, &totals.CacheCreationCostUSD, &charge); err != nil {
		return invocationreadmodel.UsageTotals{}, err
	}
	totals.TotalTokens = totals.InputTokens + totals.OutputTokens + totals.CacheReadTokens + totals.CacheCreationTokens
	totals.MeteredChargeMicroUSD = int64(charge + 0.5)
	return totals, nil
}

// runTokenBucketBounds defines the fixed per-run token histogram. Ranges are
// inclusive on min and exclusive on max; a zero max is unbounded. They are
// product judgment, not per-query tuning, so the same buckets answer the same
// question across time windows.
var runTokenBucketBounds = []struct {
	label string
	min   int64
	max   int64
}{
	{"<100K", 0, 100000},
	{"100K-500K", 100000, 500000},
	{"500K-1M", 500000, 1000000},
	{"1M-5M", 1000000, 5000000},
	{"5M-25M", 5000000, 25000000},
	{"25M+", 25000000, 0},
}

// RunCostDistribution computes per-run percentiles of the tokens and metered
// cost each run spent inside the window, plus a fixed token histogram. It uses
// nearest-rank, so every percentile is a real observed run.
func (r *invocationReadModelRepository) RunCostDistribution(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunCostDistribution, error) {
	cte, args := windowedUsageCTE(filter)
	where, whereArgs := runWhereClauses(filter, false, true)
	query := "WITH " + cte + ` SELECT SUM` + windowVolume + `,
		CAST(ROUND(SUM(CASE WHEN ` + metered + ` THEN invocation_read_model_runs.total_cost_usd * ` + windowShare + ` ELSE 0 END) * 1000000) AS INTEGER)
		FROM uw LEFT JOIN invocation_read_model_runs ON invocation_read_model_runs.run_id = uw.uw_run_id
		LEFT JOIN runs r ON r.id = uw.uw_run_id` + where + ` GROUP BY uw_run_id HAVING SUM` + windowVolume + ` > 0`
	rows, err := r.db.QueryxContext(ctx, query, append(args, whereArgs...)...)
	if err != nil {
		return invocationreadmodel.RunCostDistribution{}, err
	}
	defer rows.Close()
	tokens, costs := []int64{}, []int64{}
	for rows.Next() {
		var tok, costMicro int64
		if err := rows.Scan(&tok, &costMicro); err != nil {
			return invocationreadmodel.RunCostDistribution{}, err
		}
		tokens, costs = append(tokens, tok), append(costs, costMicro)
	}
	if err := rows.Err(); err != nil {
		return invocationreadmodel.RunCostDistribution{}, err
	}
	dist := invocationreadmodel.RunCostDistribution{SampleSize: int64(len(tokens)), TokenBuckets: tokenBuckets(tokens)}
	if len(tokens) == 0 {
		return dist, nil
	}
	sort.Slice(tokens, func(i, j int) bool { return tokens[i] < tokens[j] })
	sort.Slice(costs, func(i, j int) bool { return costs[i] < costs[j] })
	dist.P50Tokens, dist.P90Tokens, dist.P95Tokens, dist.P99Tokens, dist.MaxTokens = nearestRank(tokens, 0.50), nearestRank(tokens, 0.90), nearestRank(tokens, 0.95), nearestRank(tokens, 0.99), tokens[len(tokens)-1]
	dist.P50CostUSD, dist.P90CostUSD, dist.P95CostUSD, dist.P99CostUSD, dist.MaxCostUSD = float64(nearestRank(costs, 0.50))/1e6, float64(nearestRank(costs, 0.90))/1e6, float64(nearestRank(costs, 0.95))/1e6, float64(nearestRank(costs, 0.99))/1e6, float64(costs[len(costs)-1])/1e6
	return dist, nil
}

func tokenBuckets(tokens []int64) []invocationreadmodel.RunTokenBucket {
	buckets := make([]invocationreadmodel.RunTokenBucket, len(runTokenBucketBounds))
	for i, bound := range runTokenBucketBounds {
		buckets[i] = invocationreadmodel.RunTokenBucket{Label: bound.label, MinTokens: bound.min, MaxTokens: bound.max}
	}
	for _, token := range tokens {
		for i := range buckets {
			if token >= buckets[i].MinTokens && (buckets[i].MaxTokens == 0 || token < buckets[i].MaxTokens) {
				buckets[i].RunCount++
				break
			}
		}
	}
	return buckets
}

// breakdownDimension names how one dimension groups the two halves of a
// breakdown row: runs that ended in the window, and requests made in it.
type breakdownDimension struct {
	runKey, runValue, usageKey, usageValue string
}

func breakdownDimensionFor(dimension string) (breakdownDimension, bool) {
	const t = "invocation_read_model_runs."
	profileName := "COALESCE((SELECT name FROM agent_profiles WHERE id = " + t + "profile_id), " + t + "profile_id)"
	switch dimension {
	case "runner":
		return breakdownDimension{t + "runner_type", t + "runner_type", t + "runner_type", t + "runner_type"}, true
	case "model":
		// A run can change model; requests carry the model that served them.
		return breakdownDimension{t + "model", t + "model", "COALESCE(NULLIF(uw_model, ''), 'unknown')", "COALESCE(NULLIF(uw_model, ''), 'unknown')"}, true
	case "profile":
		return breakdownDimension{t + "profile_id", profileName, t + "profile_id", profileName}, true
	case "workload":
		return breakdownDimension{"COALESCE(" + t + "workload_key, '')", "COALESCE(" + t + "workload_key, '')", "COALESCE(" + t + "workload_key, '')", "COALESCE(" + t + "workload_key, '')"}, true
	case "workload_kind":
		return breakdownDimension{t + "workload_kind", t + "workload_kind", t + "workload_kind", t + "workload_kind"}, true
	}
	return breakdownDimension{}, false
}

// RunBreakdown groups the window by one dimension. Run counts, outcomes and
// run time come from runs that ended in the window; tokens and charges from
// the requests made in it. Rows are ranked by tokens.
func (r *invocationReadModelRepository) RunBreakdown(ctx context.Context, filter invocationreadmodel.Filter, dimension string, limit int) ([]invocationreadmodel.RunBreakdownRow, error) {
	dim, ok := breakdownDimensionFor(dimension)
	if !ok {
		return nil, fmt.Errorf("unsupported durable run breakdown dimension %q", dimension)
	}
	if limit <= 0 {
		limit = 20
	}
	rowsByKey := map[string]*invocationreadmodel.RunBreakdownRow{}
	where, args := invocationReadModelRunWhere(filter)
	runRows, err := r.db.QueryxContext(ctx, fmt.Sprintf(`SELECT %s, %s, COUNT(*),
		SUM(CASE WHEN status = 'complete' AND `+executedRunSQL+` THEN 1 ELSE 0 END), SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END),
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0)
		FROM invocation_read_model_runs%s GROUP BY %s`, dim.runValue, dim.runKey, where, dim.runKey), args...)
	if err != nil {
		return nil, err
	}
	defer runRows.Close()
	for runRows.Next() {
		var row invocationreadmodel.RunBreakdownRow
		if err := runRows.Scan(&row.Value, &row.Key, &row.RunCount, &row.SuccessCount, &row.FailedCount, &row.AvgDurationMS); err != nil {
			return nil, err
		}
		rowsByKey[row.Key] = &row
	}
	if err := runRows.Err(); err != nil {
		return nil, err
	}

	cte, cteArgs := windowedUsageCTE(filter)
	usageWhere, usageArgs := runWhereClauses(filter, false, true)
	cost := func(column string) string {
		return "COALESCE(SUM(CASE WHEN " + metered + " THEN invocation_read_model_runs." + column + " * " + windowShare + " ELSE 0 END), 0)"
	}
	usageRows, err := r.db.QueryxContext(ctx, "WITH "+cte+fmt.Sprintf(` SELECT COALESCE(%s, 'unknown'), COALESCE(%s, 'unknown'), COALESCE(SUM(uw_in), 0), COALESCE(SUM(uw_out), 0), COALESCE(SUM(uw_cr), 0), COALESCE(SUM(uw_cc), 0),
		%s, %s
		FROM uw LEFT JOIN invocation_read_model_runs ON invocation_read_model_runs.run_id = uw.uw_run_id LEFT JOIN runs r ON r.id = uw.uw_run_id%s GROUP BY 2`,
		dim.usageValue, dim.usageKey, cost("total_cost_usd"), cost("total_charge_micro_usd"), usageWhere), append(cteArgs, usageArgs...)...)
	if err != nil {
		return nil, err
	}
	defer usageRows.Close()
	for usageRows.Next() {
		var value, key string
		var in, out, cacheRead, cacheCreate int64
		var costUSD, charge float64
		if err := usageRows.Scan(&value, &key, &in, &out, &cacheRead, &cacheCreate, &costUSD, &charge); err != nil {
			return nil, err
		}
		row := rowsByKey[key]
		if row == nil {
			row = &invocationreadmodel.RunBreakdownRow{Value: value, Key: key}
			rowsByKey[key] = row
		}
		row.InputTokens, row.OutputTokens, row.CacheReadTokens, row.CacheCreationTokens = row.InputTokens+in, row.OutputTokens+out, row.CacheReadTokens+cacheRead, row.CacheCreationTokens+cacheCreate
		row.TotalTokens = row.InputTokens + row.OutputTokens + row.CacheReadTokens + row.CacheCreationTokens
		row.TotalCostUSD += costUSD
		row.TotalChargeMicroUSD += int64(charge + 0.5)
	}
	if err := usageRows.Err(); err != nil {
		return nil, err
	}

	out := make([]invocationreadmodel.RunBreakdownRow, 0, len(rowsByKey))
	for _, row := range rowsByKey {
		if row.RunCount > 0 {
			row.CompletionRate = float64(row.SuccessCount) / float64(row.RunCount)
		}
		if row.SuccessCount > 0 {
			row.ConsumptionPerSuccessfulCompletion = float64(row.TotalTokens) / float64(row.SuccessCount)
		}
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalTokens != out[j].TotalTokens {
			return out[i].TotalTokens > out[j].TotalTokens
		}
		if out[i].RunCount != out[j].RunCount {
			return out[i].RunCount > out[j].RunCount
		}
		return strings.Compare(out[i].Value, out[j].Value) < 0
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
