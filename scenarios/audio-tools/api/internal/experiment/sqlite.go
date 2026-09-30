package experiment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the sqliteRepository depends on.
// Both *sql.DB and *database.RoutedDB satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production experiment Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	_, _ = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const experimentTimeFormat = time.RFC3339Nano

const (
	insertExperimentSQL = `
INSERT INTO experiments (id, name, status, recipe_json, created_at, started_at, finished_at, error, result_ref, machine_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	updateExperimentSQL = `
UPDATE experiments
SET name = ?, status = ?, recipe_json = ?, started_at = ?, finished_at = ?, error = ?, result_ref = ?, machine_json = ?
WHERE id = ?
`
	selectExperimentColumns = `id, name, status, recipe_json, created_at, started_at, finished_at, error, result_ref, machine_json`
	insertRunSQL            = `
INSERT INTO experiment_runs (id, experiment_id, strategy, condition_json, created_at)
VALUES (?, ?, ?, ?, ?)
`
	selectRunColumns               = `id, experiment_id, strategy, condition_json, created_at`
	insertQualificationEvidenceSQL = `
INSERT INTO qualification_evidence (id, engine_id, model_id, strategy, policy_profile, kind, fault_profile, passed, artifact_ref, notes, machine_json, observed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	selectQualificationEvidenceColumns = `id, engine_id, model_id, strategy, policy_profile, kind, fault_profile, passed, artifact_ref, notes, machine_json, observed_at`
)

func (s *sqliteRepository) CreateExperiment(ctx context.Context, exp Experiment) (Experiment, error) {
	exp = s.withExperimentDefaults(exp)
	if _, err := s.db.ExecContext(ctx, insertExperimentSQL,
		exp.ID, exp.Name, string(exp.Status), string(exp.RecipeJSON), exp.CreatedAt.Format(experimentTimeFormat),
		formatTimePtr(exp.StartedAt), formatTimePtr(exp.FinishedAt), exp.Error, exp.ResultRef, string(exp.MachineJSON),
	); err != nil {
		return Experiment{}, fmt.Errorf("experiment: insert %q: %w", exp.ID, err)
	}
	return exp, nil
}

func (s *sqliteRepository) GetExperiment(ctx context.Context, id string) (Experiment, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+selectExperimentColumns+" FROM experiments WHERE id = ?", id)
	exp, err := scanExperiment(row)
	if err == sql.ErrNoRows {
		return Experiment{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return Experiment{}, fmt.Errorf("experiment: get %q: %w", id, err)
	}
	return exp, nil
}

func (s *sqliteRepository) UpdateExperiment(ctx context.Context, exp Experiment) error {
	exp = s.withExperimentDefaults(exp)
	res, err := s.updateExperiment(ctx, s.db, exp)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("experiment: update %q rows: %w", exp.ID, err)
	}
	if n == 0 {
		return ErrNotFound{ID: exp.ID}
	}
	return nil
}

func (s *sqliteRepository) CompleteSucceeded(ctx context.Context, exp Experiment, runs []Run) error {
	exp = s.withExperimentDefaults(exp)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("experiment: begin complete %q: %w", exp.ID, err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, run := range runs {
		run.ExperimentID = exp.ID
		if _, err := s.createRun(ctx, tx, run); err != nil {
			return err
		}
	}
	res, err := s.updateExperiment(ctx, tx, exp)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("experiment: update %q rows: %w", exp.ID, err)
	}
	if n == 0 {
		return ErrNotFound{ID: exp.ID}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("experiment: commit complete %q: %w", exp.ID, err)
	}
	return nil
}

func (s *sqliteRepository) ListExperiments(ctx context.Context, filter ListFilter) ([]Experiment, error) {
	query := "SELECT " + selectExperimentColumns + " FROM experiments"
	var args []any
	if filter.Status != "" {
		query += " WHERE status = ?"
		args = append(args, string(filter.Status))
	}
	query += " ORDER BY created_at DESC, id DESC"
	// SQLite rejects OFFSET without a preceding LIMIT, so when the caller pages
	// (offset > 0) without a limit we emit the unbounded sentinel LIMIT -1.
	switch {
	case filter.Limit > 0:
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	case filter.Offset > 0:
		query += " LIMIT -1 OFFSET ?"
		args = append(args, filter.Offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("experiment: list: %w", err)
	}
	defer rows.Close()
	return scanExperiments(rows)
}

func (s *sqliteRepository) ListNonTerminal(ctx context.Context) ([]Experiment, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+selectExperimentColumns+" FROM experiments WHERE status IN (?, ?) ORDER BY created_at ASC",
		string(StatusQueued), string(StatusRunning),
	)
	if err != nil {
		return nil, fmt.Errorf("experiment: list non-terminal: %w", err)
	}
	defer rows.Close()
	return scanExperiments(rows)
}

func (s *sqliteRepository) DeleteExperiment(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM experiments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("experiment: delete %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("experiment: delete %q rows: %w", id, err)
	}
	if n == 0 {
		return ErrNotFound{ID: id}
	}
	return nil
}

func (s *sqliteRepository) CreateRun(ctx context.Context, run Run) (Run, error) {
	return s.createRun(ctx, s.db, run)
}

func (s *sqliteRepository) createRun(ctx context.Context, db SQLRunner, run Run) (Run, error) {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = s.clock.Now().UTC()
	}
	if len(run.ConditionJSON) == 0 {
		run.ConditionJSON = []byte("{}")
	}
	if _, err := db.ExecContext(ctx, insertRunSQL,
		run.ID, run.ExperimentID, run.Strategy, string(run.ConditionJSON), run.CreatedAt.Format(experimentTimeFormat),
	); err != nil {
		return Run{}, fmt.Errorf("experiment: insert run %q: %w", run.ID, err)
	}
	return run, nil
}

type SQLRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *sqliteRepository) updateExperiment(ctx context.Context, db SQLRunner, exp Experiment) (sql.Result, error) {
	exp = s.withExperimentDefaults(exp)
	res, err := db.ExecContext(ctx, updateExperimentSQL,
		exp.Name, string(exp.Status), string(exp.RecipeJSON), formatTimePtr(exp.StartedAt),
		formatTimePtr(exp.FinishedAt), exp.Error, exp.ResultRef, string(exp.MachineJSON), exp.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("experiment: update %q: %w", exp.ID, err)
	}
	return res, nil
}

func (s *sqliteRepository) ListRuns(ctx context.Context, experimentID string) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+selectRunColumns+" FROM experiment_runs WHERE experiment_id = ? ORDER BY created_at ASC, id ASC",
		experimentID,
	)
	if err != nil {
		return nil, fmt.Errorf("experiment: list runs: %w", err)
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("experiment: iterate runs: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) CreateQualificationEvidence(ctx context.Context, evidence QualificationEvidence) (QualificationEvidence, error) {
	if evidence.ID == "" {
		evidence.ID = uuid.NewString()
	}
	if evidence.ObservedAt.IsZero() {
		evidence.ObservedAt = s.clock.Now().UTC()
	}
	if len(evidence.MachineJSON) == 0 {
		evidence.MachineJSON = []byte(`{}`)
	}
	if _, err := s.db.ExecContext(ctx, insertQualificationEvidenceSQL,
		evidence.ID, evidence.EngineID, evidence.ModelID, evidence.Strategy, evidence.PolicyProfile,
		evidence.Kind, evidence.FaultProfile, evidence.Passed, evidence.ArtifactRef,
		evidence.Notes, string(evidence.MachineJSON), evidence.ObservedAt.Format(experimentTimeFormat),
	); err != nil {
		return QualificationEvidence{}, fmt.Errorf("experiment: record qualification evidence %q: %w", evidence.ID, err)
	}
	return evidence, nil
}

func (s *sqliteRepository) ListQualificationEvidence(ctx context.Context, filter QualificationEvidenceFilter) ([]QualificationEvidence, error) {
	query := "SELECT " + selectQualificationEvidenceColumns + " FROM qualification_evidence"
	var clauses []string
	var args []any
	if filter.EngineID != "" {
		clauses = append(clauses, "engine_id = ?")
		args = append(args, filter.EngineID)
	}
	if filter.ModelID != "" {
		clauses = append(clauses, "model_id = ?")
		args = append(args, filter.ModelID)
	}
	if filter.Strategy != "" {
		clauses = append(clauses, "strategy = ?")
		args = append(args, filter.Strategy)
	}
	if filter.PolicyProfile != "" {
		clauses = append(clauses, "policy_profile = ?")
		args = append(args, filter.PolicyProfile)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY observed_at ASC, id ASC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("experiment: list qualification evidence: %w", err)
	}
	defer rows.Close()
	var out []QualificationEvidence
	for rows.Next() {
		evidence, scanErr := scanQualificationEvidence(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, evidence)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("experiment: iterate qualification evidence: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) withExperimentDefaults(exp Experiment) Experiment {
	if exp.ID == "" {
		exp.ID = uuid.NewString()
	}
	if exp.Status == "" {
		exp.Status = StatusQueued
	}
	if exp.CreatedAt.IsZero() {
		exp.CreatedAt = s.clock.Now().UTC()
	}
	if len(exp.RecipeJSON) == 0 {
		exp.RecipeJSON = []byte("{}")
	}
	if len(exp.MachineJSON) == 0 {
		exp.MachineJSON = []byte("{}")
	}
	return exp
}

type scanner interface {
	Scan(dest ...any) error
}

func scanExperiments(rows *sql.Rows) ([]Experiment, error) {
	var out []Experiment
	for rows.Next() {
		exp, err := scanExperiment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, exp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("experiment: iterate experiments: %w", err)
	}
	return out, nil
}

func scanExperiment(sc scanner) (Experiment, error) {
	var (
		exp                   Experiment
		status                string
		recipe, machine       string
		createdAt             string
		startedAt, finishedAt string
	)
	if err := sc.Scan(
		&exp.ID, &exp.Name, &status, &recipe, &createdAt, &startedAt, &finishedAt,
		&exp.Error, &exp.ResultRef, &machine,
	); err != nil {
		return Experiment{}, err
	}
	exp.Status = Status(status)
	exp.RecipeJSON = []byte(recipe)
	exp.MachineJSON = []byte(machine)
	created, err := time.Parse(experimentTimeFormat, createdAt)
	if err != nil {
		return Experiment{}, fmt.Errorf("experiment: parse created_at %q: %w", createdAt, err)
	}
	exp.CreatedAt = created
	exp.StartedAt = parseTimePtr(startedAt)
	exp.FinishedAt = parseTimePtr(finishedAt)
	return exp, nil
}

func scanRun(sc scanner) (Run, error) {
	var (
		run       Run
		condition string
		createdAt string
	)
	if err := sc.Scan(&run.ID, &run.ExperimentID, &run.Strategy, &condition, &createdAt); err != nil {
		return Run{}, fmt.Errorf("experiment: scan run: %w", err)
	}
	run.ConditionJSON = []byte(condition)
	created, err := time.Parse(experimentTimeFormat, createdAt)
	if err != nil {
		return Run{}, fmt.Errorf("experiment: parse run created_at %q: %w", createdAt, err)
	}
	run.CreatedAt = created
	return run, nil
}

func scanQualificationEvidence(sc scanner) (QualificationEvidence, error) {
	var evidence QualificationEvidence
	var machine, observedAt string
	if err := sc.Scan(
		&evidence.ID, &evidence.EngineID, &evidence.ModelID, &evidence.Strategy, &evidence.PolicyProfile,
		&evidence.Kind, &evidence.FaultProfile, &evidence.Passed, &evidence.ArtifactRef,
		&evidence.Notes, &machine, &observedAt,
	); err != nil {
		return QualificationEvidence{}, err
	}
	evidence.MachineJSON = []byte(machine)
	observed, err := time.Parse(experimentTimeFormat, observedAt)
	if err != nil {
		return QualificationEvidence{}, fmt.Errorf("experiment: parse qualification observed_at %q: %w", observedAt, err)
	}
	evidence.ObservedAt = observed
	return evidence, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(experimentTimeFormat)
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(experimentTimeFormat, s)
	if err != nil {
		return nil
	}
	return &t
}
