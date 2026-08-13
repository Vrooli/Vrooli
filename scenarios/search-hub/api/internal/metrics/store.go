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
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
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
	AutoRoutedExternal      bool
	Escalated               bool
	LatencyMs               int64
	RoutingMode             string
	EligibleProviderCount   int
	SelectedProviderCount   int
	SelectedLeafCount       int
	WidenedLeafCount        int
	FanoutWidthBoundReached bool
	WithheldExternalCount   int
	QueuedProviderCount     int
	ClassifierLatencyMs     int64
	ResolverLatencyMs       int64
	ResolverCacheHits       int64
	ResolverCacheMisses     int64
	FanoutLatencyMs         int64
	RerankLatencyMs         int64
	RerankCandidateCount    int
	RerankerLeg             string
	ResponseDegradeReason   string
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
	ActiveRerankerLeg  string
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
	WindowFrom                time.Time
	WindowTo                  time.Time
	SampleCount               int64
	MinimumSampleCount        int64
	SampleSufficient          bool
	RecentSampleCount         int64
	RecentLatencyP50Ms        int64
	RecentLatencyP95Ms        int64
	ProviderUsage             []ProviderUsage
	ResolverCacheHits         int64
	ResolverCacheMisses       int64
	ResolverCacheHitRate      float64
	RoutingBuckets            []RoutingBucket
}

// RoutingBucket is the fleet-scale read model. Its bucket is derived from the
// selected fan-out, so automatic routing and explicit-all can be compared at
// the same fleet size without retaining raw query text.
type RoutingBucket struct {
	RoutingMode     string
	FanoutBucket    string
	Queries         int64
	DegradedQueries int64
	LatencyP50Ms    int64
	LatencyP95Ms    int64
	ClassifierP95Ms int64
	ResolverP95Ms   int64
	FanoutP95Ms     int64
	RerankP95Ms     int64
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
	clock schedule.Clock
}

// NewSQLiteStore constructs the production Store. db is the connection pool
// opened in main.go; clk supplies created_at timestamps so tests advance time
// deterministically.
func NewSQLiteStore(db SQLExecutor, clk schedule.Clock) Store {
	return &sqliteStore{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Store = (*sqliteStore)(nil)

const telemetryTimeFormat = time.RFC3339Nano

// MinimumStableSamples is the minimum evidence set for reporting latency
// percentiles as stable measurements.
const MinimumStableSamples int64 = 10

const RecentLatencySampleLimit int64 = 10

// Migrate applies guarded SQLite migrations that must run before EnsureSchemas'
// drift check compares declared columns against an existing table.
func Migrate(ctx context.Context, db SQLExecutor) error {
	for _, col := range []struct {
		name string
		ddl  string
	}{
		{name: "routing_mode", ddl: "ALTER TABLE query_telemetry ADD COLUMN routing_mode TEXT NOT NULL DEFAULT ''"},
		{name: "eligible_provider_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN eligible_provider_count INTEGER NOT NULL DEFAULT 0"},
		{name: "selected_provider_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN selected_provider_count INTEGER NOT NULL DEFAULT 0"},
		{name: "selected_leaf_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN selected_leaf_count INTEGER NOT NULL DEFAULT 0"},
		{name: "widened_leaf_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN widened_leaf_count INTEGER NOT NULL DEFAULT 0"},
		{name: "fanout_width_bound_reached", ddl: "ALTER TABLE query_telemetry ADD COLUMN fanout_width_bound_reached INTEGER NOT NULL DEFAULT 0"},
		{name: "withheld_external_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN withheld_external_count INTEGER NOT NULL DEFAULT 0"},
		{name: "queued_provider_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN queued_provider_count INTEGER NOT NULL DEFAULT 0"},
		{name: "classifier_latency_ms", ddl: "ALTER TABLE query_telemetry ADD COLUMN classifier_latency_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "resolver_latency_ms", ddl: "ALTER TABLE query_telemetry ADD COLUMN resolver_latency_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "resolver_cache_hits", ddl: "ALTER TABLE query_telemetry ADD COLUMN resolver_cache_hits INTEGER NOT NULL DEFAULT 0"},
		{name: "resolver_cache_misses", ddl: "ALTER TABLE query_telemetry ADD COLUMN resolver_cache_misses INTEGER NOT NULL DEFAULT 0"},
		{name: "fanout_latency_ms", ddl: "ALTER TABLE query_telemetry ADD COLUMN fanout_latency_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "rerank_latency_ms", ddl: "ALTER TABLE query_telemetry ADD COLUMN rerank_latency_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "rerank_candidate_count", ddl: "ALTER TABLE query_telemetry ADD COLUMN rerank_candidate_count INTEGER NOT NULL DEFAULT 0"},
		{name: "response_degrade_reason", ddl: "ALTER TABLE query_telemetry ADD COLUMN response_degrade_reason TEXT NOT NULL DEFAULT ''"},
		{name: "latency_ms", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN latency_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "degraded", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN degraded INTEGER NOT NULL DEFAULT 0"},
		{name: "degrade_reason", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN degrade_reason TEXT NOT NULL DEFAULT ''"},
		{name: "reranker_leg", ddl: "ALTER TABLE query_telemetry_provider ADD COLUMN reranker_leg TEXT NOT NULL DEFAULT ''"},
	} {
		table := "query_telemetry_provider"
		if strings.Contains(col.ddl, "ALTER TABLE query_telemetry ADD") {
			table = "query_telemetry"
		}
		exists, err := columnExists(ctx, db, table, col.name)
		if err != nil {
			return fmt.Errorf("inspect %s.%s: %w", table, col.name, err)
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
			return fmt.Errorf("migrate %s.%s: %w", table, col.name, err)
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
INSERT INTO query_telemetry (query_hash, routed_types, result_count, zero_result, degraded, reranked, auto_routed_external, escalated, latency_ms, routing_mode, eligible_provider_count, selected_provider_count, selected_leaf_count, widened_leaf_count, fanout_width_bound_reached, withheld_external_count, queued_provider_count, classifier_latency_ms, resolver_latency_ms, resolver_cache_hits, resolver_cache_misses, fanout_latency_ms, rerank_latency_ms, rerank_candidate_count, response_degrade_reason, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.QueryHash, strings.Join(sample.RoutedTypes, ","), sample.ResultCount, zero,
		boolToInt(sample.Degraded), boolToInt(sample.Reranked),
		boolToInt(sample.AutoRoutedExternal), boolToInt(sample.Escalated), sample.LatencyMs,
		sample.RoutingMode, sample.EligibleProviderCount, sample.SelectedProviderCount,
		sample.SelectedLeafCount, sample.WidenedLeafCount, boolToInt(sample.FanoutWidthBoundReached),
		sample.WithheldExternalCount, sample.QueuedProviderCount, sample.ClassifierLatencyMs,
		sample.ResolverLatencyMs, sample.ResolverCacheHits, sample.ResolverCacheMisses,
		sample.FanoutLatencyMs, sample.RerankLatencyMs,
		sample.RerankCandidateCount, sample.ResponseDegradeReason, now)
	if err != nil {
		return fmt.Errorf("insert query_telemetry: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("query_telemetry last insert id: %w", err)
	}

	for pid, provider := range sample.ProviderResults {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO query_telemetry_provider (query_id, provider_id, hit_count, latency_ms, degraded, degrade_reason, reranker_leg)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, pid, provider.HitCount, provider.LatencyMs, boolToInt(provider.Degraded), provider.DegradeReason, sample.RerankerLeg); err != nil {
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
	out.WindowFrom = filter.windowFrom
	out.WindowTo = filter.windowTo
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
	cacheRow := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(resolver_cache_hits), 0), COALESCE(SUM(resolver_cache_misses), 0)
FROM query_telemetry`+filter.queryWhere, filter.queryArgs...)
	if err := cacheRow.Scan(&out.ResolverCacheHits, &out.ResolverCacheMisses); err != nil {
		return nil, fmt.Errorf("aggregate resolver cache telemetry: %w", err)
	}
	cacheLookups := out.ResolverCacheHits + out.ResolverCacheMisses
	if cacheLookups > 0 {
		out.ResolverCacheHitRate = float64(out.ResolverCacheHits) / float64(cacheLookups)
	}

	out.SampleCount = out.TotalQueries
	out.MinimumSampleCount = MinimumStableSamples
	out.SampleSufficient = out.SampleCount >= out.MinimumSampleCount
	p50, p95, err := s.latencyPercentiles(ctx, out.TotalQueries, filter.queryWhere, filter.queryArgs)
	if err != nil {
		return nil, err
	}
	if out.SampleSufficient {
		out.LatencyP50Ms, out.LatencyP95Ms = p50, p95
	}
	out.RecentSampleCount, out.RecentLatencyP50Ms, out.RecentLatencyP95Ms, err = s.recentLatencyPercentiles(ctx, filter)
	if err != nil {
		return nil, err
	}

	usage, err := s.providerUsage(ctx, filter)
	if err != nil {
		return nil, err
	}
	out.ProviderUsage = usage
	buckets, err := s.routingBuckets(ctx, filter)
	if err != nil {
		return nil, err
	}
	out.RoutingBuckets = buckets
	return out, nil
}

func fanoutBucket(n int) string {
	switch {
	case n <= 1:
		return "1"
	case n <= 6:
		return "2-6"
	case n <= 12:
		return "7-12"
	case n <= 24:
		return "13-24"
	default:
		return "25+"
	}
}

func (s *sqliteStore) routingBuckets(ctx context.Context, filter telemetryFilter) ([]RoutingBucket, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT routing_mode, selected_provider_count, degraded, latency_ms, classifier_latency_ms, resolver_latency_ms, fanout_latency_ms, rerank_latency_ms FROM query_telemetry`+filter.queryWhere, filter.queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query routing buckets: %w", err)
	}
	defer rows.Close()
	type values struct {
		latency, classifier, resolver, fanout, rerank []int64
		degraded                                      int64
	}
	grouped := map[string]*values{}
	for rows.Next() {
		var mode string
		var selected, degraded int
		var latency, classifier, resolver, fanout, rerank int64
		if err := rows.Scan(&mode, &selected, &degraded, &latency, &classifier, &resolver, &fanout, &rerank); err != nil {
			return nil, fmt.Errorf("scan routing bucket: %w", err)
		}
		if mode == "" {
			mode = "unknown"
		}
		key := mode + "\x00" + fanoutBucket(selected)
		v := grouped[key]
		if v == nil {
			v = &values{}
			grouped[key] = v
		}
		v.latency = append(v.latency, latency)
		v.classifier = append(v.classifier, classifier)
		v.resolver = append(v.resolver, resolver)
		v.fanout = append(v.fanout, fanout)
		v.rerank = append(v.rerank, rerank)
		v.degraded += int64(degraded)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routing buckets: %w", err)
	}
	out := make([]RoutingBucket, 0, len(grouped))
	for key, v := range grouped {
		parts := strings.SplitN(key, "\x00", 2)
		out = append(out, RoutingBucket{
			RoutingMode:     parts[0],
			FanoutBucket:    parts[1],
			Queries:         int64(len(v.latency)),
			DegradedQueries: v.degraded,
			LatencyP50Ms:    percentile(v.latency, 0.50),
			LatencyP95Ms:    percentile(v.latency, 0.95),
			ClassifierP95Ms: percentile(v.classifier, 0.95),
			ResolverP95Ms:   percentile(v.resolver, 0.95),
			FanoutP95Ms:     percentile(v.fanout, 0.95),
			RerankP95Ms:     percentile(v.rerank, 0.95),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RoutingMode != out[j].RoutingMode {
			return out[i].RoutingMode < out[j].RoutingMode
		}
		return out[i].FanoutBucket < out[j].FanoutBucket
	})
	return out, nil
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	values = append([]int64(nil), values...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	idx := int(float64(len(values))*p+0.999999) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

type telemetryFilter struct {
	queryWhere       string
	queryArgs        []any
	providerJoin     string
	providerWhere    string
	providerAndWhere string
	providerArgs     []any
	windowFrom       time.Time
	windowTo         time.Time
}

// daysFilter builds the optional "created_at >= ?" filter. A non-positive
// windowDays means all-time. The cutoff is computed off the store's clock so
// tests are deterministic.
func (s *sqliteStore) daysFilter(windowDays int) telemetryFilter {
	if windowDays <= 0 {
		return telemetryFilter{windowTo: s.clock.Now().UTC()}
	}
	now := s.clock.Now().UTC()
	from := now.AddDate(0, 0, -windowDays)
	cutoff := from.Format(telemetryTimeFormat)
	return telemetryFilter{
		queryWhere:       " WHERE created_at >= ?",
		queryArgs:        []any{cutoff},
		providerJoin:     ` JOIN query_telemetry t ON t.id = p.query_id`,
		providerWhere:    ` WHERE t.created_at >= ?`,
		providerAndWhere: ` AND t.created_at >= ?`,
		providerArgs:     []any{cutoff},
		windowFrom:       from,
		windowTo:         now,
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
		windowFrom:       from.UTC(),
		windowTo:         to.UTC(),
	}
}

func (s *sqliteStore) recentLatencyPercentiles(ctx context.Context, filter telemetryFilter) (int64, int64, int64, error) {
	q := `SELECT latency_ms FROM query_telemetry` + filter.queryWhere + ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args := append(append([]any{}, filter.queryArgs...), RecentLatencySampleLimit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("query recent latency samples: %w", err)
	}
	defer rows.Close()
	values := make([]int64, 0, RecentLatencySampleLimit)
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return 0, 0, 0, fmt.Errorf("scan recent latency sample: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("iterate recent latency samples: %w", err)
	}
	count := int64(len(values))
	if count < MinimumStableSamples {
		return count, 0, 0, nil
	}
	return count, percentile(values, 0.50), percentile(values, 0.95), nil
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
		leg, err := s.latestRerankerLeg(ctx, out[i].ProviderID, filter)
		if err != nil {
			return nil, err
		}
		out[i].ActiveRerankerLeg = leg
	}
	return out, nil
}

func (s *sqliteStore) latestRerankerLeg(ctx context.Context, providerID string, filter telemetryFilter) (string, error) {
	q := `SELECT p.reranker_leg FROM query_telemetry_provider p JOIN query_telemetry t ON t.id = p.query_id`
	args := []any{providerID}
	where := ` WHERE p.provider_id = ?`
	if filter.providerJoin != "" {
		where += filter.providerAndWhere
		args = append(args, filter.providerArgs...)
	}
	q += where + ` ORDER BY t.created_at DESC, t.id DESC LIMIT 1`
	var leg string
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&leg); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("latest reranker leg %s: %w", providerID, err)
	}
	return leg, nil
}

func (s *sqliteStore) providerLatencyPercentiles(ctx context.Context, providerID string, total int64, filter telemetryFilter) (int64, int64, error) {
	if total < MinimumStableSamples {
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
