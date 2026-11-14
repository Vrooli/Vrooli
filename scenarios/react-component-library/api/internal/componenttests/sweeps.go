package componenttests

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	SweepRunning  SweepStatus = "running"
	SweepComplete SweepStatus = "complete"
	SweepBlocked  SweepStatus = "blocked"
	SweepFailed   SweepStatus = "failed"
)

type SweepStatus string

func (s SweepStatus) Valid() bool {
	switch s {
	case SweepRunning, SweepComplete, SweepBlocked, SweepFailed:
		return true
	default:
		return false
	}
}

func (s SweepStatus) CanTransitionTo(next SweepStatus) bool {
	if !s.Valid() || !next.Valid() {
		return false
	}
	if s == next {
		return true
	}
	switch s {
	case SweepRunning:
		return next == SweepComplete || next == SweepBlocked || next == SweepFailed
	case SweepBlocked, SweepFailed:
		return next == SweepRunning
	default:
		return false
	}
}

type ErrInvalidSweepTransition struct {
	Current SweepStatus
	Next    SweepStatus
}

func (e ErrInvalidSweepTransition) Error() string {
	return fmt.Sprintf("invalid component test sweep transition %q -> %q", e.Current, e.Next)
}

// Sweep is the durable state needed to resume a corpus run without treating
// an absent result as a pass. Results is keyed by library id and version.
type Sweep struct {
	ID              string
	ComponentFilter string
	IncludeClosure  bool
	StartedAt       time.Time
	CompletedAt     time.Time
	Status          SweepStatus
	Results         map[string]string
	Errors          []string
}

func (s *Sweep) TransitionStatus(next SweepStatus) error {
	if s == nil || !s.Status.CanTransitionTo(next) {
		current := SweepStatus("<nil>")
		if s != nil {
			current = s.Status
		}
		return ErrInvalidSweepTransition{Current: current, Next: next}
	}
	s.Status = next
	return nil
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
	if sweep.ID == "" || !sweep.Status.Valid() {
		return ErrInvalidSweepTransition{Current: "<new>", Next: sweep.Status}
	}
	if sweep.Status == SweepRunning && !sweep.CompletedAt.IsZero() {
		return ErrInvalidSweepTransition{Current: sweep.Status, Next: sweep.Status}
	}
	if sweep.Results == nil {
		sweep.Results = map[string]string{}
	}
	results, err := json.Marshal(sweep.Results)
	if err != nil {
		return fmt.Errorf("encode sweep results: %w", err)
	}
	errorsJSON, err := json.Marshal(sweep.Errors)
	if err != nil {
		return fmt.Errorf("encode sweep errors: %w", err)
	}
	completed := ""
	if !sweep.CompletedAt.IsZero() {
		completed = sweep.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var currentStatus, currentResults, currentErrors string
	err = tx.QueryRowContext(ctx, `SELECT status, results_json, errors_json FROM component_test_sweeps WHERE id = ?`, sweep.ID).Scan(&currentStatus, &currentResults, &currentErrors)
	if errors.Is(err, sql.ErrNoRows) {
		if sweep.Status != SweepRunning {
			return ErrInvalidSweepTransition{Current: "<new>", Next: sweep.Status}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO component_test_sweeps
 (id, component_filter, include_closure, started_at, completed_at, status, results_json, errors_json)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, sweep.ID, sweep.ComponentFilter, sweep.IncludeClosure, sweep.StartedAt.UTC().Format(time.RFC3339Nano), completed, sweep.Status, string(results), string(errorsJSON))
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	current := SweepStatus(currentStatus)
	if !current.CanTransitionTo(sweep.Status) {
		return ErrInvalidSweepTransition{Current: current, Next: sweep.Status}
	}
	result, err := tx.ExecContext(ctx, `UPDATE component_test_sweeps SET component_filter=?, include_closure=?, completed_at=?, status=?, results_json=?, errors_json=? WHERE id=? AND status=? AND results_json=? AND errors_json=?`, sweep.ComponentFilter, sweep.IncludeClosure, completed, sweep.Status, string(results), string(errorsJSON), sweep.ID, currentStatus, currentResults, currentErrors)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("component test sweep %q changed during save", sweep.ID)
	}
	return tx.Commit()
}

type sweepScanner interface{ Scan(...any) error }

func scanSweep(row sweepScanner) (Sweep, error) {
	var sweep Sweep
	var include int
	var started, completed, results, errors string
	if err := row.Scan(&sweep.ID, &sweep.ComponentFilter, &include, &started, &completed, &sweep.Status, &results, &errors); err != nil {
		return Sweep{}, err
	}
	if !sweep.Status.Valid() {
		return Sweep{}, fmt.Errorf("unknown component test sweep status %q", sweep.Status)
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
