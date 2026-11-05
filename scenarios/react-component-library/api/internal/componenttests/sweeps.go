package componenttests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const (
	SweepRunning  = "running"
	SweepComplete = "complete"
	SweepBlocked  = "blocked"
	SweepFailed   = "failed"
)

// Sweep is the durable state needed to resume a corpus run without treating
// an absent result as a pass. Results is keyed by library id and version.
type Sweep struct {
	ID              string
	ComponentFilter string
	IncludeClosure  bool
	StartedAt       time.Time
	CompletedAt     time.Time
	Status          string
	Results         map[string]string
	Errors          []string
}

type SweepRepository interface {
	Start(context.Context, string, bool, string) (Sweep, error)
	LatestOpen(context.Context, string) (Sweep, error)
	Save(context.Context, Sweep) error
}

type SQLiteSweepRepository struct{ db *sql.DB }

func NewSQLiteSweepRepository(db *sql.DB) *SQLiteSweepRepository {
	return &SQLiteSweepRepository{db: db}
}

func (r *SQLiteSweepRepository) Start(ctx context.Context, componentFilter string, includeClosure bool, id string) (Sweep, error) {
	if id == "" {
		id = fmt.Sprintf("cts_%d", time.Now().UTC().UnixNano())
	}
	sweep := Sweep{ID: id, ComponentFilter: componentFilter, IncludeClosure: includeClosure, StartedAt: time.Now().UTC(), Status: SweepRunning, Results: map[string]string{}}
	if err := r.Save(ctx, sweep); err != nil {
		return Sweep{}, err
	}
	return sweep, nil
}

func (r *SQLiteSweepRepository) LatestOpen(ctx context.Context, componentFilter string) (Sweep, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, component_filter, include_closure, started_at, completed_at, status, results_json, errors_json
FROM component_test_sweeps WHERE component_filter = ? AND status <> ? ORDER BY started_at DESC LIMIT 1`, componentFilter, SweepComplete)
	return scanSweep(row)
}

func (r *SQLiteSweepRepository) Save(ctx context.Context, sweep Sweep) error {
	if sweep.Results == nil {
		sweep.Results = map[string]string{}
	}
	results, err := json.Marshal(sweep.Results)
	if err != nil {
		return fmt.Errorf("encode sweep results: %w", err)
	}
	errors, err := json.Marshal(sweep.Errors)
	if err != nil {
		return fmt.Errorf("encode sweep errors: %w", err)
	}
	completed := ""
	if !sweep.CompletedAt.IsZero() {
		completed = sweep.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO component_test_sweeps
 (id, component_filter, include_closure, started_at, completed_at, status, results_json, errors_json)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT(id) DO UPDATE SET component_filter=excluded.component_filter,
 include_closure=excluded.include_closure, completed_at=excluded.completed_at,
 status=excluded.status, results_json=excluded.results_json, errors_json=excluded.errors_json`,
		sweep.ID, sweep.ComponentFilter, sweep.IncludeClosure, sweep.StartedAt.UTC().Format(time.RFC3339Nano), completed, sweep.Status, string(results), string(errors))
	return err
}

type sweepScanner interface{ Scan(...any) error }

func scanSweep(row sweepScanner) (Sweep, error) {
	var sweep Sweep
	var include int
	var started, completed, results, errors string
	if err := row.Scan(&sweep.ID, &sweep.ComponentFilter, &include, &started, &completed, &sweep.Status, &results, &errors); err != nil {
		return Sweep{}, err
	}
	sweep.IncludeClosure = include != 0
	if err := sweep.StartedAt.UnmarshalText([]byte(started)); err != nil {
		return Sweep{}, fmt.Errorf("parse sweep start: %w", err)
	}
	if completed != "" {
		if err := sweep.CompletedAt.UnmarshalText([]byte(completed)); err != nil {
			return Sweep{}, fmt.Errorf("parse sweep completion: %w", err)
		}
	}
	if err := json.Unmarshal([]byte(results), &sweep.Results); err != nil {
		return Sweep{}, fmt.Errorf("decode sweep results: %w", err)
	}
	if err := json.Unmarshal([]byte(errors), &sweep.Errors); err != nil {
		return Sweep{}, fmt.Errorf("decode sweep errors: %w", err)
	}
	return sweep, nil
}
