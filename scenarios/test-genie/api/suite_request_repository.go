package main

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

// PostgresSuiteRequestRepository persists suite requests to PostgreSQL.
type PostgresSuiteRequestRepository struct {
	db *sql.DB
}

func NewPostgresSuiteRequestRepository(db *sql.DB) *PostgresSuiteRequestRepository {
	return &PostgresSuiteRequestRepository{db: db}
}

func (r *PostgresSuiteRequestRepository) Create(ctx context.Context, req *SuiteRequest) error {
	const q = `
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, notes, delegation_issue_id
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING created_at, updated_at
`
	var note sql.NullString
	if req.Notes != "" {
		note = sql.NullString{String: req.Notes, Valid: true}
	}

	var delegation sql.NullString
	if req.DelegationIssueID != nil && *req.DelegationIssueID != "" {
		delegation = sql.NullString{String: *req.DelegationIssueID, Valid: true}
	}

	return r.db.QueryRowContext(
		ctx,
		q,
		req.ID,
		req.ScenarioName,
		pq.Array(req.RequestedTypes),
		req.CoverageTarget,
		req.Priority,
		req.Status,
		note,
		delegation,
	).Scan(&req.CreatedAt, &req.UpdatedAt)
}

func (r *PostgresSuiteRequestRepository) List(ctx context.Context, limit int) ([]SuiteRequest, error) {
	const q = `
SELECT
	id,
	scenario_name,
	requested_types,
	coverage_target,
	priority,
	status,
	notes,
	delegation_issue_id,
	created_at,
	updated_at
FROM suite_requests
ORDER BY created_at DESC
LIMIT $1
`
	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suites []SuiteRequest
	for rows.Next() {
		req, err := scanSuiteRequest(rows)
		if err != nil {
			return nil, err
		}
		suites = append(suites, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suites, nil
}

func (r *PostgresSuiteRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*SuiteRequest, error) {
	const q = `
SELECT
	id,
	scenario_name,
	requested_types,
	coverage_target,
	priority,
	status,
	notes,
	delegation_issue_id,
	created_at,
	updated_at
FROM suite_requests
WHERE id = $1
`
	row := r.db.QueryRowContext(ctx, q, id)
	req, err := scanSuiteRequest(row)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *PostgresSuiteRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const q = `
UPDATE suite_requests
SET status = $1,
    updated_at = NOW()
WHERE id = $2
`
	res, err := r.db.ExecContext(ctx, q, status, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (r *PostgresSuiteRequestRepository) StatusSnapshot(ctx context.Context) (SuiteRequestSnapshot, error) {
	snapshot := SuiteRequestSnapshot{}

	countRows, err := r.db.QueryContext(ctx, `
SELECT status, COUNT(*)
FROM suite_requests
GROUP BY status
`)
	if err != nil {
		return snapshot, err
	}
	defer countRows.Close()

	for countRows.Next() {
		var status string
		var count int
		if err := countRows.Scan(&status, &count); err != nil {
			return snapshot, err
		}
		snapshot.Total += count
		switch status {
		case suiteStatusQueued:
			snapshot.Queued = count
		case suiteStatusDelegated:
			snapshot.Delegated = count
		case suiteStatusRunning:
			snapshot.Running = count
		case suiteStatusCompleted:
			snapshot.Completed = count
		case suiteStatusFailed:
			snapshot.Failed = count
		}
	}
	if err := countRows.Err(); err != nil {
		return snapshot, err
	}

	var oldest sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
SELECT created_at
FROM suite_requests
WHERE status IN ($1, $2)
ORDER BY created_at ASC
LIMIT 1
`, suiteStatusQueued, suiteStatusDelegated).Scan(&oldest); err != nil {
		if err != sql.ErrNoRows {
			return snapshot, err
		}
	}
	if oldest.Valid {
		// Normalize to UTC for stable telemetry output.
		t := oldest.Time.UTC()
		snapshot.OldestQueuedAt = &t
	}

	return snapshot, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSuiteRequest(scanner rowScanner) (SuiteRequest, error) {
	var req SuiteRequest
	var rawTypes pq.StringArray
	var note sql.NullString
	var delegation sql.NullString

	if err := scanner.Scan(
		&req.ID,
		&req.ScenarioName,
		&rawTypes,
		&req.CoverageTarget,
		&req.Priority,
		&req.Status,
		&note,
		&delegation,
		&req.CreatedAt,
		&req.UpdatedAt,
	); err != nil {
		return req, err
	}

	req.RequestedTypes = append([]string(nil), rawTypes...)
	if note.Valid {
		req.Notes = note.String
	}
	if delegation.Valid {
		req.DelegationIssueID = strPtr(delegation.String)
	}
	req.EstimatedQueueTime = estimateQueueSeconds(len(req.RequestedTypes), req.CoverageTarget)
	return req, nil
}

func strPtr(value string) *string {
	v := value
	return &v
}
