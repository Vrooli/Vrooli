package audits

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"data-backup-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the repository depends on.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const auditTimeFormat = time.RFC3339Nano

func (s *sqliteRepository) CreateAudit(ctx context.Context, a Audit) (Audit, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if a.RequestedAt.IsZero() {
		a.RequestedAt = now
	}
	if a.Status == "" {
		a.Status = AuditRequested
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audits
			(id, target_id, destination_id, snapshot_id, status, include_content_hash,
			 include_sqlite_check, restorable, live_json, snapshot_json, comparison_json,
			 snapshot_time, requested_at, finished_at, error, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TargetID, a.DestinationID, a.SnapshotID, string(a.Status),
		boolToInt(a.IncludeContentHash), boolToInt(a.IncludeSQLiteCheck), boolToInt(a.Restorable),
		marshalInventory(a.Live), marshalInventory(a.Snapshot), marshalComparison(a.Comparison),
		formatTime(a.SnapshotTime), a.RequestedAt.UTC().Format(auditTimeFormat),
		formatTime(a.FinishedAt), a.Error, formatTime(a.UpdatedAt),
	)
	if err != nil {
		return Audit{}, fmt.Errorf("insert audit: %w", err)
	}
	return a, nil
}

func (s *sqliteRepository) UpdateAuditStatus(ctx context.Context, id string, status AuditStatus) error {
	now := formatTime(s.clock.Now().UTC())
	if _, err := s.db.ExecContext(ctx,
		`UPDATE audits SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), now, id,
	); err != nil {
		return fmt.Errorf("update audit status %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) FinishAudit(ctx context.Context, a Audit) error {
	now := formatTime(s.clock.Now().UTC())
	if _, err := s.db.ExecContext(ctx,
		`UPDATE audits SET status = ?, restorable = ?, live_json = ?, snapshot_json = ?,
			comparison_json = ?, snapshot_time = ?, finished_at = ?, error = ?, updated_at = ?
		 WHERE id = ?`,
		string(a.Status), boolToInt(a.Restorable),
		marshalInventory(a.Live), marshalInventory(a.Snapshot), marshalComparison(a.Comparison),
		formatTime(a.SnapshotTime), formatTime(a.FinishedAt), a.Error, now, a.ID,
	); err != nil {
		return fmt.Errorf("finish audit %q: %w", a.ID, err)
	}
	return nil
}

func (s *sqliteRepository) ListNonTerminalAudits(ctx context.Context) ([]Audit, error) {
	rows, err := s.db.QueryContext(ctx,
		auditSelect+` WHERE status IN (?, ?) ORDER BY requested_at ASC, id ASC`,
		string(AuditRequested), string(AuditRunning))
	if err != nil {
		return nil, fmt.Errorf("list non-terminal audits: %w", err)
	}
	defer rows.Close()
	return scanAuditRows(rows)
}

func (s *sqliteRepository) GetAudit(ctx context.Context, id string) (Audit, error) {
	row := s.db.QueryRowContext(ctx, auditSelect+` WHERE id = ?`, id)
	a, err := scanAudit(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Audit{}, ErrAuditNotFound{ID: id}
	}
	if err != nil {
		return Audit{}, fmt.Errorf("get audit %q: %w", id, err)
	}
	return a, nil
}

func (s *sqliteRepository) ListAudits(ctx context.Context, targetID string, limit int) ([]Audit, error) {
	if limit <= 0 {
		return nil, nil
	}
	var (
		rows *sql.Rows
		err  error
	)
	if targetID != "" {
		rows, err = s.db.QueryContext(ctx,
			auditSelect+` WHERE target_id = ? ORDER BY requested_at DESC, id DESC LIMIT ?`,
			targetID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			auditSelect+` ORDER BY requested_at DESC, id DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list audits: %w", err)
	}
	defer rows.Close()
	return scanAuditRows(rows)
}

const auditSelect = `SELECT id, target_id, destination_id, snapshot_id, status,
	include_content_hash, include_sqlite_check, restorable, live_json, snapshot_json,
	comparison_json, snapshot_time, requested_at, finished_at, error, updated_at FROM audits`

type rowScanner interface{ Scan(dest ...any) error }

func scanAuditRows(rows *sql.Rows) ([]Audit, error) {
	var out []Audit
	for rows.Next() {
		a, err := scanAudit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audits: %w", err)
	}
	return out, nil
}

func scanAudit(sc rowScanner) (Audit, error) {
	var (
		a                                              Audit
		status                                         string
		includeContent, includeSQLite, restorable      int
		liveJSON, snapshotJSON, comparisonJSON         string
		snapshotTimeRaw, requestedRaw, finishedRaw, up string
	)
	if err := sc.Scan(
		&a.ID, &a.TargetID, &a.DestinationID, &a.SnapshotID, &status,
		&includeContent, &includeSQLite, &restorable,
		&liveJSON, &snapshotJSON, &comparisonJSON,
		&snapshotTimeRaw, &requestedRaw, &finishedRaw, &a.Error, &up,
	); err != nil {
		return Audit{}, err
	}
	a.Status = AuditStatus(status)
	a.IncludeContentHash = includeContent != 0
	a.IncludeSQLiteCheck = includeSQLite != 0
	a.Restorable = restorable != 0
	a.Live = unmarshalInventory(liveJSON)
	a.Snapshot = unmarshalInventory(snapshotJSON)
	a.Comparison = unmarshalComparison(comparisonJSON)
	a.SnapshotTime = parseTime(snapshotTimeRaw)
	a.RequestedAt = parseTime(requestedRaw)
	a.FinishedAt = parseTime(finishedRaw)
	a.UpdatedAt = parseTime(up)
	return a, nil
}

func marshalInventory(inv *InventorySummary) string {
	if inv == nil {
		return ""
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return ""
	}
	return string(b)
}

func unmarshalInventory(s string) *InventorySummary {
	if s == "" {
		return nil
	}
	var inv InventorySummary
	if err := json.Unmarshal([]byte(s), &inv); err != nil {
		return nil
	}
	return &inv
}

func marshalComparison(c *AuditComparison) string {
	if c == nil {
		return ""
	}
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(b)
}

func unmarshalComparison(s string) *AuditComparison {
	if s == "" {
		return nil
	}
	var c AuditComparison
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return nil
	}
	return &c
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(auditTimeFormat)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(auditTimeFormat, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
