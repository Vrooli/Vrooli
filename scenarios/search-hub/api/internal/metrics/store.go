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
// ProviderResults maps each provider_id the query fanned out to → that leaf's
// telemetry. ResultCount is the total across providers.
type Sample struct {
	QueryHash       string
	RoutedTypes     []string
	ProviderResults map[string]ProviderResult
	ResultCount     int
	Degraded        bool
	Reranked        bool
	// AutoRoutedExternal / Escalated record the OT-P2-002 auto-external decisions
	// so the Insights surface can report the auto-routed-external and escalation
	// rates for validation.
	AutoRoutedExternal bool
	Escalated          bool
	LatencyMs          int64
}

// ProviderResult is one provider leg's persisted telemetry.
type ProviderResult struct {
	HitCount      int
	LatencyMs     int64
	Degraded      bool
	DegradeReason string
}

// ProviderDegradationReason is one provider's reason-count bucket over the
// Insights window.
type ProviderDegradationReason struct {
	Reason string
	Count  int64
}

// ProviderUsage is one provider's routed/hit/health totals over the Insights window.
type ProviderUsage struct {
	ProviderID         string
	TimesRouted        int64
	TotalHits          int64
	LatencyP50Ms       int64
	LatencyP95Ms       int64
	DegradedCount      int64
	DegradationRate    float64
	DegradationReasons []ProviderDegradationReason
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
	// AutoRoutedExternalQueries / EscalatedQueries are the OT-P2-002 validation
	// counts: how many queries in the window folded an external provider in via
	// the web-shaped path, and how many escalated to external on an empty project
	// corpus. Both stay 0 while the opt-in flag is off.
	AutoRoutedExternalQueries int64
	EscalatedQueries          int64
	LatencyP50Ms              int64
	LatencyP95Ms              int64
	ProviderUsage             []ProviderUsage
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

	// InsightsRange aggregates telemetry in the exact [from, to) range used by
	// declared measures after resolving a canonical time_window param.
	InsightsRange(ctx context.Context, from, to time.Time) (*Insights, error)
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

// Migrate applies guarded SQLite migrations that must run before EnsureSchemas'
// drift check compares declared columns against an existing table.
func Migrate(ctx context.Context, db SQLExecutor) error {
	for _, col := range []struct {
		name string
		ddl  string
	}{
		{name: "latency_ms", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN latency_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "degraded", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN degraded INTEGER NOT NULL DEFAULT 0"},
		{name: "degrade_reason", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN degrade_reason TEXT NOT NULL DEFAULT ''"},
	} {
		exists, err := columnExists(ctx, db, "query_telemetry_provider", col.name)
		if err != nil {
			return fmt.Errorf("inspect query_telemetry_provider.%s: %w", col.name, err)
		}
		if exists {
			continue
		}
		if _, err := db.ExecContext(ctx, col.ddl); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				continue
			}
			return fmt.Errorf("migrate query_telemetry_provider.%s: %w", col.name, err)
		}
	}
	return nil
}

func columnExists(ctx context.Context, db SQLExecutor, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func (s *sqliteStore) Record(ctx context.Context, sample Sample) error {
	now := s.clock.Now().UTC().Format(telemetryTimeFormat)
	zero := 0
	if sample.ResultCount == 0 {
		zero = 1
	}

	res, err := s.db.ExecContext(ctx, `
INSERT INTO query_telemetry (query_hash, routed_types, result_count, zero_result, degraded, reranked, auto_routed_external, escalated, latency_ms, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.QueryHash, strings.Join(sample.RoutedTypes, ","), sample.ResultCount, zero,
		boolToInt(sample.Degraded), boolToInt(sample.Reranked),
		boolToInt(sample.AutoRoutedExternal), boolToInt(sample.Escalated), sample.LatencyMs, now)
	if err != nil {
		return fmt.Errorf("insert query_telemetry: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("query_telemetry last insert id: %w", err)
	}

	for pid, provider := range sample.ProviderResults {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO query_telemetry_provider (query_id, provider_id, hit_count, latency_ms, degraded, degrade_reason)
VALUES (?, ?, ?, ?, ?, ?)`,
			id, pid, provider.HitCount, provider.LatencyMs, boolToInt(provider.Degraded), provider.DegradeReason); err != nil {
			return fmt.Errorf("insert query_telemetry_provider %q: %w", pid, err)
		}
	}
	return nil
}

func (s *sqliteStore) Insights(ctx context.Context, windowDays int) (*Insights, error) {
	filter := s.daysFilter(windowDays)
	return s.insights(ctx, filter)
}

func (s *sqliteStore) InsightsRange(ctx context.Context, from, to time.Time) (*Insights, error) {
	return s.insights(ctx, s.rangeFilter(from, to))
}

func (s *sqliteStore) insights(ctx context.Context, filter telemetryFilter) (*Insights, error) {
	out := &Insights{}
	row := s.db.QueryRowContext(ctx, `
SELECT
  COUNT(*),
  COALESCE(SUM(zero_result), 0),
  COALESCE(SUM(degraded), 0),
  COALESCE(SUM(reranked), 0),
  COALESCE(SUM(auto_routed_external), 0),
  COALESCE(SUM(escalated), 0)
FROM query_telemetry`+filter.queryWhere, filter.queryArgs...)
	if err := row.Scan(&out.TotalQueries, &out.ZeroResultQueries, &out.DegradedQueries, &out.RerankedQueries,
		&out.AutoRoutedExternalQueries, &out.EscalatedQueries); err != nil {
		return nil, fmt.Errorf("aggregate query_telemetry: %w", err)
	}

	p50, p95, err := s.latencyPercentiles(ctx, out.TotalQueries, filter.queryWhere, filter.queryArgs)
	if err != nil {
		return nil, err
	}
	out.LatencyP50Ms, out.LatencyP95Ms = p50, p95

	usage, err := s.providerUsage(ctx, filter)
	if err != nil {
		return nil, err
	}
	out.ProviderUsage = usage
	return out, nil
}

type telemetryFilter struct {
	queryWhere       string
	queryArgs        []any
	providerJoin     string
	providerWhere    string
	providerAndWhere string
	providerArgs     []any
}

// daysFilter builds the optional "created_at >= ?" filter. A non-positive
// windowDays means all-time. The cutoff is computed off the store's clock so
// tests are deterministic.
func (s *sqliteStore) daysFilter(windowDays int) telemetryFilter {
	if windowDays <= 0 {
		return telemetryFilter{}
	}
	cutoff := s.clock.Now().UTC().AddDate(0, 0, -windowDays).Format(telemetryTimeFormat)
	return telemetryFilter{
		queryWhere:       " WHERE created_at >= ?",
		queryArgs:        []any{cutoff},
		providerJoin:     ` JOIN query_telemetry t ON t.id = p.query_id`,
		providerWhere:    ` WHERE t.created_at >= ?`,
		providerAndWhere: ` AND t.created_at >= ?`,
		providerArgs:     []any{cutoff},
	}
}

func (s *sqliteStore) rangeFilter(from, to time.Time) telemetryFilter {
	fromText := from.UTC().Format(telemetryTimeFormat)
	toText := to.UTC().Format(telemetryTimeFormat)
	return telemetryFilter{
		queryWhere:       " WHERE created_at >= ? AND created_at < ?",
		queryArgs:        []any{fromText, toText},
		providerJoin:     ` JOIN query_telemetry t ON t.id = p.query_id`,
		providerWhere:    ` WHERE t.created_at >= ? AND t.created_at < ?`,
		providerAndWhere: ` AND t.created_at >= ? AND t.created_at < ?`,
		providerArgs:     []any{fromText, toText},
	}
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

// providerUsage returns per-provider routed counts, hit totals, latency
// percentiles, and degradation buckets over the window, ordered by provider_id.
func (s *sqliteStore) providerUsage(ctx context.Context, filter telemetryFilter) ([]ProviderUsage, error) {
	q := `
SELECT
  p.provider_id,
  COUNT(*) AS times_routed,
  COALESCE(SUM(p.hit_count), 0) AS total_hits,
  COALESCE(SUM(p.degraded), 0) AS degraded_count
FROM query_telemetry_provider p` + filter.providerJoin + filter.providerWhere + `
GROUP BY p.provider_id
ORDER BY p.provider_id ASC`

	rows, err := s.db.QueryContext(ctx, q, filter.providerArgs...)
	if err != nil {
		return nil, fmt.Errorf("aggregate provider usage: %w", err)
	}
	defer rows.Close()

	out := make([]ProviderUsage, 0)
	for rows.Next() {
		var u ProviderUsage
		if err := rows.Scan(&u.ProviderID, &u.TimesRouted, &u.TotalHits, &u.DegradedCount); err != nil {
			return nil, fmt.Errorf("scan provider usage: %w", err)
		}
		u.DegradationRate = rate(u.DegradedCount, u.TimesRouted)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider usage: %w", err)
	}
	for i := range out {
		p50, p95, err := s.providerLatencyPercentiles(ctx, out[i].ProviderID, out[i].TimesRouted, filter)
		if err != nil {
			return nil, err
		}
		out[i].LatencyP50Ms, out[i].LatencyP95Ms = p50, p95
		reasons, err := s.providerDegradationReasons(ctx, out[i].ProviderID, filter)
		if err != nil {
			return nil, err
		}
		out[i].DegradationReasons = reasons
	}
	return out, nil
}

func (s *sqliteStore) providerLatencyPercentiles(ctx context.Context, providerID string, total int64, filter telemetryFilter) (int64, int64, error) {
	if total == 0 {
		return 0, 0, nil
	}
	p50, err := s.providerPercentileAt(ctx, providerID, total, 0.50, filter)
	if err != nil {
		return 0, 0, err
	}
	p95, err := s.providerPercentileAt(ctx, providerID, total, 0.95, filter)
	if err != nil {
		return 0, 0, err
	}
	return p50, p95, nil
}

func (s *sqliteStore) providerPercentileAt(ctx context.Context, providerID string, total int64, p float64, filter telemetryFilter) (int64, error) {
	idx := int64(float64(total)*p+0.999999) - 1
	if idx < 0 {
		idx = 0
	}
	if idx > total-1 {
		idx = total - 1
	}

	q := `SELECT p.latency_ms FROM query_telemetry_provider p`
	args := []any{providerID}
	where := ` WHERE p.provider_id = ?`
	if filter.providerJoin != "" {
		q += ` JOIN query_telemetry t ON t.id = p.query_id`
		where += filter.providerAndWhere
		args = append(args, filter.providerArgs...)
	}
	q += where + ` ORDER BY p.latency_ms ASC LIMIT 1 OFFSET ?`
	args = append(args, idx)

	var v int64
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&v); err != nil {
		return 0, fmt.Errorf("provider %s latency percentile %.2f: %w", providerID, p, err)
	}
	return v, nil
}

func (s *sqliteStore) providerDegradationReasons(ctx context.Context, providerID string, filter telemetryFilter) ([]ProviderDegradationReason, error) {
	q := `
SELECT p.degrade_reason, COUNT(*) AS count
FROM query_telemetry_provider p`
	args := []any{providerID}
	where := ` WHERE p.provider_id = ? AND p.degraded = 1 AND p.degrade_reason <> ''`
	if filter.providerJoin != "" {
		q += ` JOIN query_telemetry t ON t.id = p.query_id`
		where += filter.providerAndWhere
		args = append(args, filter.providerArgs...)
	}
	q += where + `
GROUP BY p.degrade_reason
ORDER BY count DESC, p.degrade_reason ASC`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate provider degradation reasons %s: %w", providerID, err)
	}
	defer rows.Close()

	out := make([]ProviderDegradationReason, 0)
	for rows.Next() {
		var r ProviderDegradationReason
		if err := rows.Scan(&r.Reason, &r.Count); err != nil {
			return nil, fmt.Errorf("scan provider degradation reason %s: %w", providerID, err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider degradation reasons %s: %w", providerID, err)
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func rate(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
