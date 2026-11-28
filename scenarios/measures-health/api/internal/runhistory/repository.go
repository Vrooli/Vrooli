// Package runhistory persists measures-health's own validation_run entity: one
// row per measures-coverage validation it performs. It is the substrate the
// `validation_run` measures aggregate over — measures-health dogfooding the
// capability it enforces. The write happens at the top-level ValidateScenario
// RPC (NOT inside the per-scenario fleet rollup, which would amplify writes); the
// read paths are the measure compute functions.
package runhistory

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation_runs DDL for EnsureSchemas.
func Schema() string { return schemaSQL }

// DB is the minimal query surface the repository needs. Both *sql.DB and
// *database.RoutedDB (the production test-mode-routing handle) satisfy it.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Run is one recorded measures-coverage validation (mirrors the v1/domain
// ValidationRun proto).
type Run struct {
	ID           string
	Scenario     string
	Passed       bool
	ErrorCount   int
	WarningCount int
	RanAt        time.Time
}

// Repository reads/writes the validation_runs table.
type Repository struct {
	db  DB
	now func() time.Time
}

// New constructs a Repository. now anchors the default ran_at + id; nil = time.Now.
func New(db DB, now func() time.Time) *Repository {
	if now == nil {
		now = time.Now
	}
	return &Repository{db: db, now: now}
}

// stamp formats a timestamp at second precision in UTC — the lexically-safe
// fixed-width form ran_at is stored and compared in.
func stamp(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

// Record persists one validation run. A zero RanAt defaults to now; an empty ID
// is derived from scenario + timestamp.
func (r *Repository) Record(ctx context.Context, run Run) error {
	ts := run.RanAt
	if ts.IsZero() {
		ts = r.now()
	}
	id := strings.TrimSpace(run.ID)
	if id == "" {
		id = fmt.Sprintf("%s-%d", run.Scenario, ts.UTC().UnixNano())
	}
	passed := 0
	if run.Passed {
		passed = 1
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO validation_runs (id, scenario, passed, error_count, warning_count, ran_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, run.Scenario, passed, run.ErrorCount, run.WarningCount, stamp(ts),
	)
	return err
}

// CountFailed returns the number of runs that FAILED (>=1 ERROR) in [from, to).
func (r *Repository) CountFailed(ctx context.Context, from, to time.Time) (int64, error) {
	return r.countRange(ctx, "passed = 0", from, to)
}

// CountPassing returns the number of runs that PASSED (zero ERRORs) in [from, to).
func (r *Repository) CountPassing(ctx context.Context, from, to time.Time) (int64, error) {
	return r.countRange(ctx, "passed = 1", from, to)
}

func (r *Repository) countRange(ctx context.Context, cond string, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM validation_runs WHERE `+cond+` AND ran_at >= ? AND ran_at < ?`,
		stamp(from), stamp(to),
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}
