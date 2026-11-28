package routing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// RouteAggregate holds the count facts for one time window, computed in a single
// SQL pass over route_events. Rates are derived from these counts by the
// measures layer; this struct never carries prompt/response content.
type RouteAggregate struct {
	Total            int64
	Succeeded        int64
	Failed           int64
	FallbackUsed     int64
	BreakerOpen      int64
	CapacityRejected int64
	InputTokens      int64
	OutputTokens     int64
	CostUSD          float64
	PricedRows       int64
	LocalServed      int64
}

type CallerAggregate struct {
	Scenario     string
	InputTokens  int64
	OutputTokens int64
	CostUSD      float64
	PricedRows   int64
}

// MetricsRepository is the read-only aggregate surface the route measures
// compute against. SQLRepository implements it; tests can inject a fake.
type MetricsRepository interface {
	Aggregate(ctx context.Context, from, to time.Time) (RouteAggregate, error)
	LatencyP95(ctx context.Context, from, to time.Time) (int64, error)
}

type CallerMetricsRepository interface {
	AggregateByCaller(ctx context.Context, from, to time.Time) ([]CallerAggregate, error)
}

// windowBound formats a time as the fixed-width second-granular prefix used to
// compare against substr(created_at,1,19). route_events.created_at is stored as
// RFC3339Nano (variable-width fraction), so comparing the first 19 characters —
// "YYYY-MM-DDTHH:MM:SS" — keeps range membership lexicographically correct at
// second granularity, which is the resolution route analytics windows need.
func windowBound(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05")
}

const aggregateSQL = `SELECT
  COUNT(*) AS total,
  COALESCE(SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END), 0) AS succeeded,
  COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed,
  COALESCE(SUM(CASE WHEN fallback_used = 1 THEN 1 ELSE 0 END), 0) AS fallback_used,
  COALESCE(SUM(CASE WHEN rejection_reason = 'provider_breaker_open' THEN 1 ELSE 0 END), 0) AS breaker_open,
  COALESCE(SUM(CASE WHEN capacity_verdict = 'insufficient_capacity' THEN 1 ELSE 0 END), 0) AS capacity_rejected
  ,COALESCE(SUM(input_tokens), 0) AS input_tokens
  ,COALESCE(SUM(output_tokens), 0) AS output_tokens
  ,COALESCE(SUM(cost_estimate), 0) AS cost_usd
  ,COALESCE(SUM(CASE WHEN cost_estimate > 0 THEN 1 ELSE 0 END), 0) AS priced_rows
  ,COALESCE(SUM(CASE WHEN selected_locality = 'local' AND status = 'succeeded' THEN 1 ELSE 0 END), 0) AS local_served
FROM route_events
WHERE substr(created_at, 1, 19) >= ? AND substr(created_at, 1, 19) < ?`

func (r *SQLRepository) Aggregate(ctx context.Context, from, to time.Time) (RouteAggregate, error) {
	if r == nil || r.db == nil {
		return RouteAggregate{}, fmt.Errorf("route evidence repository is not configured")
	}
	var agg RouteAggregate
	err := r.db.QueryRowContext(ctx, aggregateSQL, windowBound(from), windowBound(to)).Scan(
		&agg.Total,
		&agg.Succeeded,
		&agg.Failed,
		&agg.FallbackUsed,
		&agg.BreakerOpen,
		&agg.CapacityRejected,
		&agg.InputTokens,
		&agg.OutputTokens,
		&agg.CostUSD,
		&agg.PricedRows,
		&agg.LocalServed,
	)
	if err != nil {
		return RouteAggregate{}, err
	}
	return agg, nil
}

const callerAggregateSQL = `SELECT scenario, COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cost_estimate), 0), COALESCE(SUM(CASE WHEN cost_estimate > 0 THEN 1 ELSE 0 END)
FROM route_events WHERE substr(created_at, 1, 19) >= ? AND substr(created_at, 1, 19) < ? GROUP BY scenario ORDER BY scenario`

func (r *SQLRepository) AggregateByCaller(ctx context.Context, from, to time.Time) ([]CallerAggregate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("route evidence repository is not configured")
	}
	rows, err := r.db.QueryContext(ctx, callerAggregateSQL, windowBound(from), windowBound(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CallerAggregate
	for rows.Next() {
		var row CallerAggregate
		if err := rows.Scan(&row.Scenario, &row.InputTokens, &row.OutputTokens, &row.CostUSD, &row.PricedRows); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// LatencyP95 returns the 95th-percentile latency_ms over the window using a
// nearest-rank offset selection — a single SQL pass, not a list-and-filter in
// Go. An empty window returns 0.
const latencyP95SQL = `SELECT latency_ms FROM route_events
WHERE substr(created_at, 1, 19) >= ? AND substr(created_at, 1, 19) < ?
ORDER BY latency_ms
LIMIT 1 OFFSET (
  SELECT CAST(0.95 * (COUNT(*) - 1) AS INTEGER) FROM route_events
  WHERE substr(created_at, 1, 19) >= ? AND substr(created_at, 1, 19) < ?
)`

func (r *SQLRepository) LatencyP95(ctx context.Context, from, to time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("route evidence repository is not configured")
	}
	lo, hi := windowBound(from), windowBound(to)
	var p95 int64
	err := r.db.QueryRowContext(ctx, latencyP95SQL, lo, hi, lo, hi).Scan(&p95)
	if err != nil {
		// No rows in the window: nearest-rank offset selects nothing.
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return p95, nil
}
