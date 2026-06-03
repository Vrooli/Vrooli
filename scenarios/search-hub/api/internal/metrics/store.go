// Package metrics is the search-hub metrics (telemetry) domain — the validation
// backbone (Phase 7). The router records one telemetry Sample per federated
// query; Insights aggregates that telemetry into the federation-health signals
// the plan's §6 #7 gate and PRD OT-P2 targets measure.
//
// Layering mirrors the registry domain:
//
//	routing.Router → TelemetryRecorder (records a Sample, best-effort)
//	MetricsService handler → Store.Insights (aggregates), reconciled against the
//	                         registry's ACTIVE leaves for under-utilization
//
// The store persists NO corpus content and NO vectors — only per-query
// telemetry with the query text HASHED. This is the thin-router invariant made
// concrete for the one new table Phase 7 adds.
package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"search-hub/internal/clock"
)

// SQLExecutor is the narrow database surface the store depends on. Declared at
// the consumer per seam-discovery: both *sql.DB (store tests) and
// *database.RoutedDB (production) satisfy it, so production wiring participates
// in per-request routing without forcing the test fixture to wrap its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Sample is one federated query's telemetry, recorded by the router after the
// response is built. QueryHash is the hashed query text (never the raw text).
// ProviderHits maps each provider_id the query fanned out to → that leaf's hit
// count (a degraded leaf records 0). ResultCount is the total across providers.
type Sample struct {
	QueryHash    string
	RoutedTypes  []string
	ProviderHits map[string]int
	ResultCount  int
	Degraded     bool
	Reranked     bool
	LatencyMs    int64
}

// ProviderUsage is one provider's routed/hit totals over the Insights window.
type ProviderUsage struct {
	ProviderID  string
	TimesRouted int64
	TotalHits   int64
}

// Insights is the aggregated telemetry the metrics handler reconciles against
// the registry. Latency percentiles are over the window; ProviderUsage is keyed
// by provider_id (only leaves that were routed-to appear here — the handler
// adds never-routed ACTIVE leaves as under-utilized).
type Insights struct {
	TotalQueries      int64
	ZeroResultQueries int64
	DegradedQueries   int64
	RerankedQueries   int64
	LatencyP50Ms      int64
	LatencyP95Ms      int64
	ProviderUsage     []ProviderUsage
}

// Store is the telemetry persistence seam. Production wires the SQLite-backed
// implementation; handler/router tests wire a fake.
type Store interface {
	// Record persists one query's telemetry. Best-effort by contract — callers
	// (the router's hot path) log and ignore the error rather than failing the
	// query.
	Record(ctx context.Context, s Sample) error

	// Insights aggregates telemetry over the last windowDays (0 = all-time).
	Insights(ctx context.Context, windowDays int) (*Insights, error)
}

type sqliteStore struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteStore constructs the production Store. db is the connection pool
// opened in main.go; clk supplies created_at timestamps so tests advance time
// deterministically.
func NewSQLiteStore(db SQLExecutor, clk clock.Clock) Store {
	return &sqliteStore{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Store = (*sqliteStore)(nil)

const telemetryTimeFormat = time.RFC3339Nano

func (s *sqliteStore) Record(ctx context.Context, sample Sample) error {
	now := s.clock.Now().UTC().Format(telemetryTimeFormat)
	zero := 0
	if sample.ResultCount == 0 {
		zero = 1
	}

	res, err := s.db.ExecContext(ctx, `
INSERT INTO query_telemetry (query_hash, routed_types, result_count, zero_result, degraded, reranked, latency_ms, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.QueryHash, strings.Join(sample.RoutedTypes, ","), sample.ResultCount, zero,
		boolToInt(sample.Degraded), boolToInt(sample.Reranked), sample.LatencyMs, now)
	if err != nil {
		return fmt.Errorf("insert query_telemetry: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("query_telemetry last insert id: %w", err)
	}

	for pid, hits := range sample.ProviderHits {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO query_telemetry_provider (query_id, provider_id, hit_count)
VALUES (?, ?, ?)`, id, pid, hits); err != nil {
			return fmt.Errorf("insert query_telemetry_provider %q: %w", pid, err)
		}
	}
	return nil
}

func (s *sqliteStore) Insights(ctx context.Context, windowDays int) (*Insights, error) {
	whereClause, args := s.windowClause(windowDays)

	out := &Insights{}
	row := s.db.QueryRowContext(ctx, `
SELECT
  COUNT(*),
  COALESCE(SUM(zero_result), 0),
  COALESCE(SUM(degraded), 0),
  COALESCE(SUM(reranked), 0)
FROM query_telemetry`+whereClause, args...)
	if err := row.Scan(&out.TotalQueries, &out.ZeroResultQueries, &out.DegradedQueries, &out.RerankedQueries); err != nil {
		return nil, fmt.Errorf("aggregate query_telemetry: %w", err)
	}

	p50, p95, err := s.latencyPercentiles(ctx, out.TotalQueries, whereClause, args)
	if err != nil {
		return nil, err
	}
	out.LatencyP50Ms, out.LatencyP95Ms = p50, p95

	usage, err := s.providerUsage(ctx, windowDays)
	if err != nil {
		return nil, err
	}
	out.ProviderUsage = usage
	return out, nil
}

// windowClause builds the optional "WHERE created_at >= ?" filter. A
// non-positive windowDays means all-time (no filter). The cutoff is computed
// off the store's clock so tests are deterministic.
func (s *sqliteStore) windowClause(windowDays int) (string, []any) {
	if windowDays <= 0 {
		return "", nil
	}
	cutoff := s.clock.Now().UTC().AddDate(0, 0, -windowDays).Format(telemetryTimeFormat)
	return " WHERE created_at >= ?", []any{cutoff}
}

// latencyPercentiles returns the p50 and p95 latency over the window using
// SQLite's ORDER BY + OFFSET (nearest-rank method) so it never loads every row
// into memory. Returns 0,0 when there is no telemetry.
func (s *sqliteStore) latencyPercentiles(ctx context.Context, total int64, whereClause string, args []any) (p50, p95 int64, err error) {
	if total == 0 {
		return 0, 0, nil
	}
	p50, err = s.percentileAt(ctx, total, 0.50, whereClause, args)
	if err != nil {
		return 0, 0, err
	}
	p95, err = s.percentileAt(ctx, total, 0.95, whereClause, args)
	if err != nil {
		return 0, 0, err
	}
	return p50, p95, nil
}

// percentileAt returns the latency at the nearest-rank percentile p (0..1) over
// `total` rows. Rank index is ceil(p*total)-1, clamped to [0,total-1].
func (s *sqliteStore) percentileAt(ctx context.Context, total int64, p float64, whereClause string, args []any) (int64, error) {
	idx := int64(float64(total)*p+0.999999) - 1 // ceil-ish then 0-based
	if idx < 0 {
		idx = 0
	}
	if idx > total-1 {
		idx = total - 1
	}
	q := `SELECT latency_ms FROM query_telemetry` + whereClause + ` ORDER BY latency_ms ASC LIMIT 1 OFFSET ?`
	qArgs := append(append([]any{}, args...), idx)
	var v int64
	if err := s.db.QueryRowContext(ctx, q, qArgs...).Scan(&v); err != nil {
		return 0, fmt.Errorf("latency percentile %.2f: %w", p, err)
	}
	return v, nil
}

// providerUsage returns per-provider routed counts + summed hits over the
// window, ordered by provider_id.
func (s *sqliteStore) providerUsage(ctx context.Context, windowDays int) ([]ProviderUsage, error) {
	var (
		joinWindow string
		args       []any
	)
	if windowDays > 0 {
		cutoff := s.clock.Now().UTC().AddDate(0, 0, -windowDays).Format(telemetryTimeFormat)
		joinWindow = ` JOIN query_telemetry t ON t.id = p.query_id WHERE t.created_at >= ?`
		args = []any{cutoff}
	}
	q := `
SELECT p.provider_id, COUNT(*) AS times_routed, COALESCE(SUM(p.hit_count), 0) AS total_hits
FROM query_telemetry_provider p` + joinWindow + `
GROUP BY p.provider_id
ORDER BY p.provider_id ASC`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate provider usage: %w", err)
	}
	defer rows.Close()

	out := make([]ProviderUsage, 0)
	for rows.Next() {
		var u ProviderUsage
		if err := rows.Scan(&u.ProviderID, &u.TimesRouted, &u.TotalHits); err != nil {
			return nil, fmt.Errorf("scan provider usage: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider usage: %w", err)
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
