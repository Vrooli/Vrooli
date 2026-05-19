package queue

import (
	"context"
	"database/sql"
	"time"

	"test-genie/internal/storage/sqliteutil"

	"github.com/google/uuid"
)

// SQLiteSuiteRequestRepository persists suite requests in Test Genie's embedded
// SQLite database.
type SQLiteSuiteRequestRepository struct {
	db           *sql.DB
	clock        func() time.Time
	activeWindow time.Duration
}

func NewSQLiteSuiteRequestRepository(db *sql.DB) *SQLiteSuiteRequestRepository {
	return &SQLiteSuiteRequestRepository{
		db:           db,
		clock:        time.Now,
		activeWindow: ActiveQueueWindow(),
	}
}

func (r *SQLiteSuiteRequestRepository) Create(ctx context.Context, req *SuiteRequest) error {
	const q = `
INSERT INTO suite_requests (
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
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)`

	requestedTypes, err := sqliteutil.MarshalStringSlice(req.RequestedTypes)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now

	var note any
	if req.Notes != "" {
		note = req.Notes
	}

	var delegation any
	if req.DelegationIssueID != nil && *req.DelegationIssueID != "" {
		delegation = *req.DelegationIssueID
	}

	_, err = r.db.ExecContext(
		ctx,
		q,
		req.ID.String(),
		req.ScenarioName,
		requestedTypes,
		req.CoverageTarget,
		req.Priority,
		req.Status,
		note,
		delegation,
		sqliteutil.FormatTimestamp(req.CreatedAt),
		sqliteutil.FormatTimestamp(req.UpdatedAt),
	)
	return err
}

func (r *SQLiteSuiteRequestRepository) List(ctx context.Context, limit int) ([]SuiteRequest, error) {
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
LIMIT ?
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

func (r *SQLiteSuiteRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*SuiteRequest, error) {
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
WHERE id = ?
`
	row := r.db.QueryRowContext(ctx, q, id.String())
	req, err := scanSuiteRequest(row)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *SQLiteSuiteRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const q = `
UPDATE suite_requests
SET status = ?,
    updated_at = ?
WHERE id = ?
`
	res, err := r.db.ExecContext(ctx, q, status, sqliteutil.FormatTimestamp(time.Now().UTC()), id.String())
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (r *SQLiteSuiteRequestRepository) StatusSnapshot(ctx context.Context) (SuiteRequestSnapshot, error) {
	const q = `
SELECT
	COUNT(*) AS total,
	COALESCE(SUM(CASE WHEN status = ? AND updated_at >= ? THEN 1 ELSE 0 END), 0) AS queued,
	COALESCE(SUM(CASE WHEN status = ? AND updated_at >= ? THEN 1 ELSE 0 END), 0) AS delegated,
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS running,
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS completed,
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS failed,
	COALESCE(SUM(CASE WHEN status IN (?, ?) AND updated_at < ? THEN 1 ELSE 0 END), 0) AS stale,
	MIN(CASE WHEN status IN (?, ?) AND updated_at >= ? THEN created_at END) AS oldest_queued_at
FROM suite_requests
`

	snapshot := SuiteRequestSnapshot{}
	cutoff := sqliteutil.FormatTimestamp(r.clock().UTC().Add(-r.activeWindow))
	var oldest sql.NullString
	if err := r.db.QueryRowContext(
		ctx,
		q,
		StatusQueued, cutoff,
		StatusDelegated, cutoff,
		StatusRunning,
		StatusCompleted,
		StatusFailed,
		StatusQueued, StatusDelegated, cutoff,
		StatusQueued, StatusDelegated, cutoff,
	).Scan(
		&snapshot.Total,
		&snapshot.Queued,
		&snapshot.Delegated,
		&snapshot.Running,
		&snapshot.Completed,
		&snapshot.Failed,
		&snapshot.Stale,
		&oldest,
	); err != nil {
		return snapshot, err
	}

	if oldest.Valid {
		ts, err := sqliteutil.ParseTimestamp(oldest.String)
		if err != nil {
			return snapshot, err
		}
		snapshot.OldestQueuedAt = &ts
	}

	return snapshot, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSuiteRequest(scanner rowScanner) (SuiteRequest, error) {
	var req SuiteRequest
	var rawID string
	var rawTypes any
	var note sql.NullString
	var delegation sql.NullString
	var createdAt any
	var updatedAt any

	if err := scanner.Scan(
		&rawID,
		&req.ScenarioName,
		&rawTypes,
		&req.CoverageTarget,
		&req.Priority,
		&req.Status,
		&note,
		&delegation,
		&createdAt,
		&updatedAt,
	); err != nil {
		return req, err
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return req, err
	}
	req.ID = parsedID

	req.RequestedTypes, err = sqliteutil.UnmarshalStringSlice(rawTypes)
	if err != nil {
		return req, err
	}
	if note.Valid {
		req.Notes = note.String
	}
	if delegation.Valid {
		req.DelegationIssueID = strPtr(delegation.String)
	}
	req.CreatedAt, err = sqliteutil.ParseTimestamp(createdAt)
	if err != nil {
		return req, err
	}
	req.UpdatedAt, err = sqliteutil.ParseTimestamp(updatedAt)
	if err != nil {
		return req, err
	}
	req.EstimatedQueueTime = estimateQueueSeconds(len(req.RequestedTypes), req.CoverageTarget)
	return req, nil
}

func strPtr(value string) *string {
	v := value
	return &v
}
