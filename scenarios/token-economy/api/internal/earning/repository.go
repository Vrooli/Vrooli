// Package earning owns the inbound earning-adapter contract.
package earning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository interface {
	GetByDedup(context.Context, string, string) (Submission, error)
	Store(context.Context, Submission) (Submission, bool, error)
	List(context.Context) ([]Submission, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) GetByDedup(ctx context.Context, adapterIdentity, dedupKey string) (Submission, error) {
	return readSubmission(ctx, r.db, adapterIdentity, dedupKey)
}

func (r *sqliteRepository) Store(ctx context.Context, submission Submission) (Submission, bool, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO earning_submissions (
			id, adapter_identity, dedup_key, payload_summary, grant_id, actor_identity, submitted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		submission.ID, submission.AdapterIdentity, submission.DedupKey, submission.PayloadSummary,
		submission.GrantID, submission.ActorIdentity, submission.SubmittedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Submission{}, false, fmt.Errorf("store earning submission: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Submission{}, false, fmt.Errorf("read earning insertion result: %w", err)
	}
	if rows == 1 {
		return submission, true, nil
	}
	existing, err := readSubmission(ctx, r.db, submission.AdapterIdentity, submission.DedupKey)
	return existing, false, err
}

func (r *sqliteRepository) List(ctx context.Context) ([]Submission, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, adapter_identity, dedup_key, payload_summary, grant_id, actor_identity, submitted_at
		FROM earning_submissions ORDER BY submitted_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list earning submissions: %w", err)
	}
	defer rows.Close()
	values := make([]Submission, 0)
	for rows.Next() {
		var submission Submission
		var submittedAt string
		if err := rows.Scan(&submission.ID, &submission.AdapterIdentity, &submission.DedupKey, &submission.PayloadSummary,
			&submission.GrantID, &submission.ActorIdentity, &submittedAt); err != nil {
			return nil, fmt.Errorf("scan earning submission: %w", err)
		}
		submission.SubmittedAt, err = time.Parse(time.RFC3339Nano, submittedAt)
		if err != nil {
			return nil, fmt.Errorf("parse earning submission time: %w", err)
		}
		values = append(values, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate earning submissions: %w", err)
	}
	return values, nil
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readSubmission(ctx context.Context, db rowQueryer, adapterIdentity, dedupKey string) (Submission, error) {
	var submission Submission
	var submittedAt string
	err := db.QueryRowContext(ctx, `
		SELECT id, adapter_identity, dedup_key, payload_summary, grant_id, actor_identity, submitted_at
		FROM earning_submissions
		WHERE adapter_identity = ? AND dedup_key = ?`, adapterIdentity, dedupKey).Scan(
		&submission.ID, &submission.AdapterIdentity, &submission.DedupKey, &submission.PayloadSummary,
		&submission.GrantID, &submission.ActorIdentity, &submittedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Submission{}, ErrSubmissionNotFound
	}
	if err != nil {
		return Submission{}, fmt.Errorf("read earning submission: %w", err)
	}
	submission.SubmittedAt, err = time.Parse(time.RFC3339Nano, submittedAt)
	if err != nil {
		return Submission{}, fmt.Errorf("parse earning submission time: %w", err)
	}
	return submission, nil
}
