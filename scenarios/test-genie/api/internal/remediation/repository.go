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
}

type SQLiteRepository struct{ db dbexec.Executor }

func NewSQLiteRepository(db dbexec.Executor) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, job Job) error {
	source, err := json.Marshal(job.Source)
	if err != nil {
		return err
	}
	selected, err := json.Marshal(job.SelectedFindingIDs)
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
	_, err = r.db.ExecContext(ctx, `INSERT INTO remediation_jobs (id, scenario_name, status, source_json, selected_finding_ids_json, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, job.ID, job.Scenario, job.Status, string(source), string(selected), job.AdditionalContext, string(attribution), string(verification), job.Failure, sqliteutil.FormatTimestamp(job.CreatedAt), sqliteutil.FormatTimestamp(job.UpdatedAt), nullableTime(job.CancelledAt))
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return ErrActiveJob
	}
	return err
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Job, error) {
	return scanJob(r.db.QueryRowContext(ctx, `SELECT id, scenario_name, status, source_json, selected_finding_ids_json, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at FROM remediation_jobs WHERE id = ?`, id))
}

func (r *SQLiteRepository) ListByScenario(ctx context.Context, scenario string, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, scenario_name, status, source_json, selected_finding_ids_json, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at FROM remediation_jobs WHERE scenario_name = ? ORDER BY created_at DESC LIMIT ?`, scenario, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *SQLiteRepository) ActiveForScenario(ctx context.Context, scenario string) (Job, error) {
	return scanJob(r.db.QueryRowContext(ctx, `SELECT id, scenario_name, status, source_json, selected_finding_ids_json, additional_context, attribution_json, verification_json, failure, created_at, updated_at, cancelled_at FROM remediation_jobs WHERE scenario_name = ? AND status IN ('created', 'running', 'agent_completed', 'verification_running') ORDER BY created_at DESC LIMIT 1`, scenario))
}

func (r *SQLiteRepository) Update(ctx context.Context, job Job) error {
	attribution, err := json.Marshal(job.Attribution)
	if err != nil {
		return err
	}
	verification, err := json.Marshal(job.Verification)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE remediation_jobs SET status = ?, attribution_json = ?, verification_json = ?, failure = ?, updated_at = ?, cancelled_at = ? WHERE id = ?`, job.Status, string(attribution), string(verification), job.Failure, sqliteutil.FormatTimestamp(job.UpdatedAt), nullableTime(job.CancelledAt), job.ID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanJob(s scanner) (Job, error) {
	var job Job
	var source, selected, attribution, verification string
	var created, updated, cancelled any
	err := s.Scan(&job.ID, &job.Scenario, &job.Status, &source, &selected, &job.AdditionalContext, &attribution, &verification, &job.Failure, &created, &updated, &cancelled)
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
	if err := json.Unmarshal([]byte(attribution), &job.Attribution); err != nil {
		return Job{}, fmt.Errorf("decode attribution: %w", err)
	}
	if err := json.Unmarshal([]byte(verification), &job.Verification); err != nil {
		return Job{}, fmt.Errorf("decode verification: %w", err)
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
