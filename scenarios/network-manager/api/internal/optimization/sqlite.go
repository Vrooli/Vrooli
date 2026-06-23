package optimization

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) SaveRun(ctx context.Context, run Run) (Run, error) {
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = run.CreatedAt
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO optimization_runs (id, status, scoring_profile, baseline_snapshot_id, recommendation, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, run.ID, run.Status, run.ScoringProfile, run.BaselineSnapshotID, run.Recommendation, formatTime(run.CreatedAt), formatTime(run.UpdatedAt)); err != nil {
		return Run{}, fmt.Errorf("save optimization run %q: %w", run.ID, err)
	}
	return run, nil
}

func (r *sqliteRepository) GetRun(ctx context.Context, id string) (Run, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, status, scoring_profile, baseline_snapshot_id, recommendation, created_at, updated_at
FROM optimization_runs
WHERE id = ?
`, id)
	run, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, err
	}
	run.Candidates, err = r.candidates(ctx, run.ID)
	if err != nil {
		return Run{}, err
	}
	return run, nil
}

func (r *sqliteRepository) UpdateRun(ctx context.Context, run Run) (Run, error) {
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE optimization_runs
SET status = ?, scoring_profile = ?, baseline_snapshot_id = ?, recommendation = ?, updated_at = ?
WHERE id = ?
`, run.Status, run.ScoringProfile, run.BaselineSnapshotID, run.Recommendation, formatTime(run.UpdatedAt), run.ID)
	if err != nil {
		return Run{}, fmt.Errorf("update optimization run %q: %w", run.ID, err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return Run{}, ErrNotFound
	}
	return run, nil
}

func (r *sqliteRepository) SaveCandidate(ctx context.Context, c Candidate) (Candidate, error) {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	evidenceJSON, err := encodeStrings(c.Evidence)
	if err != nil {
		return Candidate{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO optimization_candidates (
  id, run_id, description, status, score, evidence_json, approval_required,
  rollback_supported, rollback_handle, baseline_snapshot_id, candidate_snapshot_id,
  after_snapshot_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, c.ID, c.RunID, c.Description, c.Status, c.Score, evidenceJSON, boolInt(c.ApprovalRequired), boolInt(c.RollbackSupported), c.RollbackHandle, c.BaselineSnapshotID, c.CandidateSnapshotID, c.AfterSnapshotID, formatTime(c.CreatedAt), formatTime(c.UpdatedAt)); err != nil {
		return Candidate{}, fmt.Errorf("save optimization candidate %q: %w", c.ID, err)
	}
	return c, nil
}

func (r *sqliteRepository) UpdateCandidate(ctx context.Context, c Candidate) (Candidate, error) {
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now().UTC()
	}
	evidenceJSON, err := encodeStrings(c.Evidence)
	if err != nil {
		return Candidate{}, err
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE optimization_candidates
SET description = ?, status = ?, score = ?, evidence_json = ?, approval_required = ?,
    rollback_supported = ?, rollback_handle = ?, baseline_snapshot_id = ?,
    candidate_snapshot_id = ?, after_snapshot_id = ?, updated_at = ?
WHERE id = ? AND run_id = ?
`, c.Description, c.Status, c.Score, evidenceJSON, boolInt(c.ApprovalRequired), boolInt(c.RollbackSupported), c.RollbackHandle, c.BaselineSnapshotID, c.CandidateSnapshotID, c.AfterSnapshotID, formatTime(c.UpdatedAt), c.ID, c.RunID)
	if err != nil {
		return Candidate{}, fmt.Errorf("update optimization candidate %q: %w", c.ID, err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return Candidate{}, ErrCandidateNotFound
	}
	return c, nil
}

func (r *sqliteRepository) SaveApproval(ctx context.Context, approval ApprovalRecord) (ApprovalRecord, error) {
	if approval.CreatedAt.IsZero() {
		approval.CreatedAt = time.Now().UTC()
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO optimization_approval_records (id, run_id, candidate_id, approved, note, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`, approval.ID, approval.RunID, approval.CandidateID, boolInt(approval.Approved), approval.Note, formatTime(approval.CreatedAt)); err != nil {
		return ApprovalRecord{}, fmt.Errorf("save optimization approval %q: %w", approval.ID, err)
	}
	return approval, nil
}

func (r *sqliteRepository) SaveRollback(ctx context.Context, rollback RollbackRecord) (RollbackRecord, error) {
	if rollback.CreatedAt.IsZero() {
		rollback.CreatedAt = time.Now().UTC()
	}
	detailsJSON, err := encodeStrings(rollback.Details)
	if err != nil {
		return RollbackRecord{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO optimization_rollback_records (id, run_id, candidate_id, status, details_json, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`, rollback.ID, rollback.RunID, rollback.CandidateID, rollback.Status, detailsJSON, formatTime(rollback.CreatedAt)); err != nil {
		return RollbackRecord{}, fmt.Errorf("save optimization rollback %q: %w", rollback.ID, err)
	}
	return rollback, nil
}

func (r *sqliteRepository) candidates(ctx context.Context, runID string) ([]Candidate, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, run_id, description, status, score, evidence_json, approval_required,
       rollback_supported, rollback_handle, baseline_snapshot_id, candidate_snapshot_id,
       after_snapshot_id, created_at, updated_at
FROM optimization_candidates
WHERE run_id = ?
ORDER BY created_at ASC, id ASC
`, runID)
	if err != nil {
		return nil, fmt.Errorf("list optimization candidates: %w", err)
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate optimization candidates: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(row rowScanner) (Run, error) {
	var run Run
	var createdAt, updatedAt string
	if err := row.Scan(&run.ID, &run.Status, &run.ScoringProfile, &run.BaselineSnapshotID, &run.Recommendation, &createdAt, &updatedAt); err != nil {
		return Run{}, err
	}
	var err error
	run.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Run{}, fmt.Errorf("parse optimization run created_at: %w", err)
	}
	run.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return Run{}, fmt.Errorf("parse optimization run updated_at: %w", err)
	}
	return run, nil
}

func scanCandidate(row rowScanner) (Candidate, error) {
	var c Candidate
	var evidenceJSON, createdAt, updatedAt string
	var approvalRequired, rollbackSupported int
	if err := row.Scan(&c.ID, &c.RunID, &c.Description, &c.Status, &c.Score, &evidenceJSON, &approvalRequired, &rollbackSupported, &c.RollbackHandle, &c.BaselineSnapshotID, &c.CandidateSnapshotID, &c.AfterSnapshotID, &createdAt, &updatedAt); err != nil {
		return Candidate{}, err
	}
	var err error
	c.Evidence, err = decodeStrings(evidenceJSON)
	if err != nil {
		return Candidate{}, err
	}
	c.ApprovalRequired = approvalRequired == 1
	c.RollbackSupported = rollbackSupported == 1
	c.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Candidate{}, fmt.Errorf("parse optimization candidate created_at: %w", err)
	}
	c.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return Candidate{}, fmt.Errorf("parse optimization candidate updated_at: %w", err)
	}
	return c, nil
}

func encodeStrings(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	b, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode string list: %w", err)
	}
	return string(b), nil
}

func decodeStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode string list: %w", err)
	}
	return values, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatTime(value time.Time) string {
	return value.UTC().Format(TimeFormat)
}
