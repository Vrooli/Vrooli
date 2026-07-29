package remediation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/dbexec"
	"test-genie/internal/storage/sqliteutil"
)

type Repository interface {
	Create(context.Context, Job) error
	Get(context.Context, string) (Job, error)
	ListByScenario(context.Context, string, int) ([]Job, error)
	ActiveForScenario(context.Context, string) (Job, error)
	Update(context.Context, Job) error
	UpdateIfStatus(context.Context, Job, string) error
	AppendAttempt(context.Context, Attempt, string) error
	ListAttempts(context.Context, string) ([]Attempt, error)
}

type SQLiteRepository struct{ db dbexec.Executor }

func NewSQLiteRepository(db dbexec.Executor) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, job Job) error {
	if job.SourceHash == "" {
		job.SourceHash = sourceHash(job.Source)
	} else if job.SourceHash != sourceHash(job.Source) {
		return fmt.Errorf("remediation source evidence hash mismatch")
	}
	if job.SelectionHash == "" {
		job.SelectionHash = selectionHash(job.SelectedFindingIDs, job.SelectedRequirementIDs)
	} else if job.SelectionHash != selectionHash(job.SelectedFindingIDs, job.SelectedRequirementIDs) {
		return fmt.Errorf("remediation selection hash mismatch")
	}
	source, err := json.Marshal(job.Source)
	if err != nil {
		return err
	}
	selected, err := json.Marshal(job.SelectedFindingIDs)
	if err != nil {
		return err
	}
	requirements, err := json.Marshal(job.SelectedRequirementIDs)
	if err != nil {
		return err
	}
	attribution, err := json.Marshal(job.Attribution)
	if err != nil {
		return err
	}
	verification, err := json.Marshal(job.Verification)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO remediation_jobs (id, scenario_name, status, source_json, source_hash, selected_finding_ids_json, selected_requirement_ids_json, selection_hash, launch_attempt, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, job.ID, job.Scenario, job.Status, string(source), job.SourceHash, string(selected), string(requirements), job.SelectionHash, job.LaunchAttempt, job.AdditionalContext, string(attribution), string(verification), job.Failure, sqliteutil.FormatTimestamp(job.CreatedAt), sqliteutil.FormatTimestamp(job.UpdatedAt), nullableTime(job.CancelledAt))
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return ErrActiveJob
	}
	return err
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Job, error) {
	job, err := scanJob(r.db.QueryRowContext(ctx, `SELECT id, scenario_name, status, source_json, source_hash, selected_finding_ids_json, selected_requirement_ids_json, selection_hash, launch_attempt, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at FROM remediation_jobs WHERE id = ?`, id))
	if err != nil {
		return Job{}, err
	}
	return r.hydrateAttempts(ctx, job)
}

func (r *SQLiteRepository) ListByScenario(ctx context.Context, scenario string, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, scenario_name, status, source_json, source_hash, selected_finding_ids_json, selected_requirement_ids_json, selection_hash, launch_attempt, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at FROM remediation_jobs WHERE scenario_name = ? ORDER BY created_at DESC LIMIT ?`, scenario, limit)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range jobs {
		var hydrateErr error
		jobs[i], hydrateErr = r.hydrateAttempts(ctx, jobs[i])
		if hydrateErr != nil {
			return nil, hydrateErr
		}
	}
	return jobs, nil
}

func (r *SQLiteRepository) ActiveForScenario(ctx context.Context, scenario string) (Job, error) {
	job, err := scanJob(r.db.QueryRowContext(ctx, `SELECT id, scenario_name, status, source_json, source_hash, selected_finding_ids_json, selected_requirement_ids_json, selection_hash, launch_attempt, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at FROM remediation_jobs WHERE scenario_name = ? AND status IN ('created', 'launch_pending', 'running', 'agent_completed', 'verification_running') ORDER BY created_at DESC LIMIT 1`, scenario))
	if err != nil {
		return Job{}, err
	}
	return r.hydrateAttempts(ctx, job)
}

func (r *SQLiteRepository) Update(ctx context.Context, job Job) error {
	return r.writeJob(ctx, job, "")
}

func (r *SQLiteRepository) UpdateIfStatus(ctx context.Context, job Job, expected string) error {
	return r.writeJob(ctx, job, expected)
}

func (r *SQLiteRepository) writeJob(ctx context.Context, job Job, expected string) error {
	attribution, err := json.Marshal(job.Attribution)
	if err != nil {
		return err
	}
	verification, err := json.Marshal(job.Verification)
	if err != nil {
		return err
	}
	query := `UPDATE remediation_jobs SET status = ?, launch_attempt = ?, attribution_json = ?, verification_json = ?, failure = ?, updated_at = ?, cancelled_at = ? WHERE id = ?`
	args := []any{job.Status, job.LaunchAttempt, string(attribution), string(verification), job.Failure, sqliteutil.FormatTimestamp(job.UpdatedAt), nullableTime(job.CancelledAt), job.ID}
	if expected != "" {
		query += " AND status = ?"
		args = append(args, expected)
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		if expected != "" {
			return ErrInvalidState
		}
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) AppendAttempt(ctx context.Context, attempt Attempt, jobID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO remediation_attempts (id, job_id, kind, state, idempotency_key, role_ref, task_id, run_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, attempt.ID, jobID, attempt.Kind, attempt.State, attempt.IdempotencyKey, attempt.RoleRef, attempt.TaskID, attempt.RunID, attempt.Detail, sqliteutil.FormatTimestamp(attempt.CreatedAt))
	return err
}

func (r *SQLiteRepository) ListAttempts(ctx context.Context, jobID string) ([]Attempt, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, kind, state, idempotency_key, role_ref, task_id, run_id, detail, created_at FROM remediation_attempts WHERE job_id = ? ORDER BY created_at ASC, id ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attempts []Attempt
	for rows.Next() {
		var attempt Attempt
		var created any
		if err := rows.Scan(&attempt.ID, &attempt.Kind, &attempt.State, &attempt.IdempotencyKey, &attempt.RoleRef, &attempt.TaskID, &attempt.RunID, &attempt.Detail, &created); err != nil {
			return nil, err
		}
		if attempt.CreatedAt, err = sqliteutil.ParseTimestamp(created); err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	return attempts, rows.Err()
}

func (r *SQLiteRepository) hydrateAttempts(ctx context.Context, job Job) (Job, error) {
	attempts, err := r.ListAttempts(ctx, job.ID)
	if err != nil {
		return Job{}, err
	}
	job.Attempts = attempts
	return job, nil
}

type scanner interface{ Scan(...any) error }

func scanJob(s scanner) (Job, error) {
	var job Job
	var source, sourceHashValue, selected, requirements, selectionHashValue, attribution, verification string
	var created, updated, cancelled any
	err := s.Scan(&job.ID, &job.Scenario, &job.Status, &source, &sourceHashValue, &selected, &requirements, &selectionHashValue, &job.LaunchAttempt, &job.AdditionalContext, &attribution, &verification, &job.Failure, &created, &updated, &cancelled)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	if err := json.Unmarshal([]byte(source), &job.Source); err != nil {
		return Job{}, fmt.Errorf("decode remediation source: %w", err)
	}
	if err := json.Unmarshal([]byte(selected), &job.SelectedFindingIDs); err != nil {
		return Job{}, fmt.Errorf("decode selected findings: %w", err)
	}
	if err := json.Unmarshal([]byte(requirements), &job.SelectedRequirementIDs); err != nil {
		return Job{}, fmt.Errorf("decode selected requirements: %w", err)
	}
	if err := json.Unmarshal([]byte(attribution), &job.Attribution); err != nil {
		return Job{}, fmt.Errorf("decode attribution: %w", err)
	}
	if err := json.Unmarshal([]byte(verification), &job.Verification); err != nil {
		return Job{}, fmt.Errorf("decode verification: %w", err)
	}
	job.SourceHash = sourceHashValue
	job.SelectionHash = selectionHashValue
	if sourceHash(job.Source) != job.SourceHash {
		return Job{}, fmt.Errorf("remediation source evidence hash mismatch")
	}
	if selectionHash(job.SelectedFindingIDs, job.SelectedRequirementIDs) != job.SelectionHash {
		return Job{}, fmt.Errorf("remediation selection hash mismatch")
	}
	var parseErr error
	if job.CreatedAt, parseErr = sqliteutil.ParseTimestamp(created); parseErr != nil {
		return Job{}, parseErr
	}
	if job.UpdatedAt, parseErr = sqliteutil.ParseTimestamp(updated); parseErr != nil {
		return Job{}, parseErr
	}
	if cancelled != nil {
		job.CancelledAt, parseErr = sqliteutil.ParseTimestamp(cancelled)
		if parseErr != nil {
			return Job{}, parseErr
		}
	}
	return job, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return sqliteutil.FormatTimestamp(t)
}
