// Package trend persists and reads per-scenario performance samples (build time,
// startup, LCP, bundle size) as an additive, newest-first trend. Writes are
// never destructive. Modeled on structure-health's perf store and the
// test-genie self-health snapshots. Sample producers (the benchmark and startup
// domains) write through Insert.
package trend

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"performance-health/internal/perfsample"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the declarative DDL for the trend store (idempotent).
func Schema() string { return schemaSQL }

const timeLayout = time.RFC3339Nano

// Executor is the narrow database seam the store needs; *sql.DB satisfies it.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Sample is one persisted performance sample. It is an alias of the shared
// perfsample DTO so producer domains can emit samples through their own narrow
// writer seam without importing the trend domain (the concrete store is wired
// from the composition root). Every axis is optional; a zero value means "not
// measured this run".
type Sample = perfsample.Sample

// Store persists and reads performance samples.
type Store struct {
	db Executor
}

// NewStore binds a store to the database executor seam.
func NewStore(db Executor) *Store { return &Store{db: db} }

// Insert appends one sample (additive; never overwrites).
func (s *Store) Insert(ctx context.Context, sample Sample) error {
	if s == nil || s.db == nil {
		return errors.New("trend: nil store")
	}
	if sample.Scenario == "" {
		return errors.New("trend: scenario is required")
	}
	capturedAt := sample.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO perf_samples
			(scenario, flow, captured_at, go_build_ms, ui_build_ms, bundle_bytes, lcp_ms, cls,
			 response_end_ms, dom_interactive_ms, dom_content_loaded_ms, load_event_end_ms, navigation_type, startup_ms,
			 slowest_component, slowest_component_avg_ms, slowest_component_max_ms, drawn_fps, dropped_frame_rate,
			 long_task_total_ms, long_task_max_ms, raster_total_ms, layout_total_ms, paint_total_ms, input_event_count, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.Scenario, sample.Flow, capturedAt.UTC().Format(timeLayout), sample.GoBuildMs, sample.UIBuildMs,
		sample.BundleBytes, sample.LCPMs, sample.CLS,
		sample.ResponseEndMs, sample.DOMInteractiveMs, sample.DOMContentLoadedMs, sample.LoadEventEndMs,
		sample.NavigationType, sample.StartupMs,
		sample.SlowestComponent, sample.SlowestComponentAvgMs, sample.SlowestComponentMaxMs,
		sample.DrawnFPS, sample.DroppedFrameRate, sample.LongTaskTotalMs, sample.LongTaskMaxMs,
		sample.RasterTotalMs, sample.LayoutTotalMs, sample.PaintTotalMs, sample.InputEventCount, sample.Note,
	)
	if err != nil {
		return fmt.Errorf("trend: insert sample: %w", err)
	}
	return nil
}

// Series returns the samples for one (scenario, flow) newest-first, bounded by
// limit (limit <= 0 returns the default page). flow="" selects scenario-level
// samples (build/bundle/startup/scenario-LCP); a non-empty flow selects only
// that interaction-capture journey's samples.
func (s *Store) Series(ctx context.Context, scenario, flow string, limit int) ([]Sample, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("trend: nil store")
	}
	if scenario == "" {
		return nil, errors.New("trend: scenario is required")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT scenario, flow, captured_at, go_build_ms, ui_build_ms, bundle_bytes, lcp_ms, cls,
			response_end_ms, dom_interactive_ms, dom_content_loaded_ms, load_event_end_ms, navigation_type, startup_ms,
			slowest_component, slowest_component_avg_ms, slowest_component_max_ms, drawn_fps, dropped_frame_rate,
			long_task_total_ms, long_task_max_ms, raster_total_ms, layout_total_ms, paint_total_ms, input_event_count, note
		FROM perf_samples
		WHERE scenario = ? AND flow = ?
		ORDER BY captured_at DESC, id DESC
		LIMIT ?`, scenario, flow, limit)
	if err != nil {
		return nil, fmt.Errorf("trend: query series: %w", err)
	}
	defer rows.Close()

	var out []Sample
	for rows.Next() {
		var (
			sample     Sample
			capturedAt string
		)
		if scanErr := rows.Scan(&sample.Scenario, &sample.Flow, &capturedAt, &sample.GoBuildMs, &sample.UIBuildMs,
			&sample.BundleBytes, &sample.LCPMs, &sample.CLS,
			&sample.ResponseEndMs, &sample.DOMInteractiveMs, &sample.DOMContentLoadedMs, &sample.LoadEventEndMs,
			&sample.NavigationType, &sample.StartupMs,
			&sample.SlowestComponent, &sample.SlowestComponentAvgMs, &sample.SlowestComponentMaxMs,
			&sample.DrawnFPS, &sample.DroppedFrameRate, &sample.LongTaskTotalMs, &sample.LongTaskMaxMs,
			&sample.RasterTotalMs, &sample.LayoutTotalMs, &sample.PaintTotalMs, &sample.InputEventCount, &sample.Note); scanErr != nil {
			return nil, fmt.Errorf("trend: scan sample: %w", scanErr)
		}
		if t, perr := time.Parse(timeLayout, capturedAt); perr == nil {
			sample.CapturedAt = t
		}
		out = append(out, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("trend: iterate series: %w", err)
	}
	return out, nil
}

// Latest returns the newest scenario-level (flow="") sample and whether one
// exists. It reads at most one row, so it never holds an open rows cursor across
// another query (avoiding the SQLite pool=1 nested-query deadlock).
func (s *Store) Latest(ctx context.Context, scenario string) (Sample, bool, error) {
	return s.latest(ctx, scenario, "")
}

// LatestFlow returns the newest sample tagged to a specific flow slug.
func (s *Store) LatestFlow(ctx context.Context, scenario, flow string) (Sample, bool, error) {
	return s.latest(ctx, scenario, flow)
}

func (s *Store) latest(ctx context.Context, scenario, flow string) (Sample, bool, error) {
	if s == nil || s.db == nil {
		return Sample{}, false, errors.New("trend: nil store")
	}
	if scenario == "" {
		return Sample{}, false, errors.New("trend: scenario is required")
	}
	series, err := s.Series(ctx, scenario, flow, 1)
	if err != nil {
		return Sample{}, false, err
	}
	if len(series) == 0 {
		return Sample{}, false, nil
	}
	return series[0], true, nil
}

// EnsureColumns is the idempotent additive migration for the trend store: it
// adds any perf_samples column missing from an OLDER, already-created database
// (the P4 scaffold shipped a narrower table). It must run BEFORE EnsureSchemas
// so the declared CREATE TABLE block matches the on-disk shape (SQLite silently
// no-ops a column added to CREATE TABLE IF NOT EXISTS on an existing table, so
// the api-core drift detector would otherwise fail the boot).
//
// On a fresh database (the table does not exist yet) this is a no-op: the
// subsequent CREATE TABLE creates the full, current shape. It never drops or
// rewrites data.
func EnsureColumns(ctx context.Context, db Executor) error {
	if db == nil {
		return errors.New("trend: nil executor")
	}
	existing, err := columnSet(ctx, db)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		// Table not created yet — CREATE TABLE will produce the current shape.
		return nil
	}
	additive := []struct{ name, ddl string }{
		{"slowest_component", "ALTER TABLE perf_samples ADD COLUMN slowest_component TEXT NOT NULL DEFAULT ''"},
		{"slowest_component_avg_ms", "ALTER TABLE perf_samples ADD COLUMN slowest_component_avg_ms REAL NOT NULL DEFAULT 0"},
		{"slowest_component_max_ms", "ALTER TABLE perf_samples ADD COLUMN slowest_component_max_ms REAL NOT NULL DEFAULT 0"},
		{"flow", "ALTER TABLE perf_samples ADD COLUMN flow TEXT NOT NULL DEFAULT ''"},
		{"drawn_fps", "ALTER TABLE perf_samples ADD COLUMN drawn_fps REAL NOT NULL DEFAULT 0"},
		{"dropped_frame_rate", "ALTER TABLE perf_samples ADD COLUMN dropped_frame_rate REAL NOT NULL DEFAULT 0"},
		{"long_task_total_ms", "ALTER TABLE perf_samples ADD COLUMN long_task_total_ms INTEGER NOT NULL DEFAULT 0"},
		{"long_task_max_ms", "ALTER TABLE perf_samples ADD COLUMN long_task_max_ms REAL NOT NULL DEFAULT 0"},
		{"raster_total_ms", "ALTER TABLE perf_samples ADD COLUMN raster_total_ms REAL NOT NULL DEFAULT 0"},
		{"layout_total_ms", "ALTER TABLE perf_samples ADD COLUMN layout_total_ms REAL NOT NULL DEFAULT 0"},
		{"paint_total_ms", "ALTER TABLE perf_samples ADD COLUMN paint_total_ms REAL NOT NULL DEFAULT 0"},
		{"input_event_count", "ALTER TABLE perf_samples ADD COLUMN input_event_count INTEGER NOT NULL DEFAULT 0"},
		{"cls", "ALTER TABLE perf_samples ADD COLUMN cls REAL NOT NULL DEFAULT 0"},
		{"response_end_ms", "ALTER TABLE perf_samples ADD COLUMN response_end_ms INTEGER NOT NULL DEFAULT 0"},
		{"dom_interactive_ms", "ALTER TABLE perf_samples ADD COLUMN dom_interactive_ms INTEGER NOT NULL DEFAULT 0"},
		{"dom_content_loaded_ms", "ALTER TABLE perf_samples ADD COLUMN dom_content_loaded_ms INTEGER NOT NULL DEFAULT 0"},
		{"load_event_end_ms", "ALTER TABLE perf_samples ADD COLUMN load_event_end_ms INTEGER NOT NULL DEFAULT 0"},
		{"navigation_type", "ALTER TABLE perf_samples ADD COLUMN navigation_type TEXT NOT NULL DEFAULT ''"},
	}
	for _, col := range additive {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, col.ddl); err != nil {
			return fmt.Errorf("trend: add column %s: %w", col.name, err)
		}
	}
	return nil
}

func columnSet(ctx context.Context, db Executor) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(perf_samples)`)
	if err != nil {
		return nil, fmt.Errorf("trend: read columns: %w", err)
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notnull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("trend: scan column: %w", err)
		}
		out[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("trend: iterate columns: %w", err)
	}
	return out, nil
}

// Service is the engine behind TrendService.
type Service struct {
	store *Store
}

// NewService wires a trend Service.
func NewService(store *Store) *Service { return &Service{store: store} }

// Trend returns a scenario's persisted scenario-level (flow="") samples, newest
// first.
func (s *Service) Trend(ctx context.Context, scenario string, limit int) ([]Sample, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("trend: service not wired")
	}
	return s.store.Series(ctx, scenario, "", limit)
}
