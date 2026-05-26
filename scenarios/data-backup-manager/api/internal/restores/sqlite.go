package restores

import (
	"context"
	"database/sql"
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

const restoreTimeFormat = time.RFC3339Nano

func (s *sqliteRepository) CreateRestore(ctx context.Context, r Restore) (Restore, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.RequestedAt.IsZero() {
		r.RequestedAt = s.clock.Now().UTC()
	}
	if r.Status == "" {
		r.Status = RestoreRequested
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO restores
			(id, target_id, destination_id, snapshot_id, mode, status, location, checksum,
			 last_verified_at, requested_at, finished_at, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.TargetID, r.DestinationID, r.SnapshotID,
		string(r.Mode), string(r.Status),
		r.Location, r.Checksum,
		formatTime(r.LastVerifiedAt), r.RequestedAt.UTC().Format(restoreTimeFormat),
		formatTime(r.FinishedAt), r.Error,
	)
	if err != nil {
		return Restore{}, fmt.Errorf("insert restore: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) GetRestore(ctx context.Context, id string) (Restore, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, target_id, destination_id, snapshot_id, mode, status, location, checksum,
		        last_verified_at, requested_at, finished_at, error
		 FROM restores WHERE id = ?`, id)
	r, err := scanRestore(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Restore{}, ErrRestoreNotFound{ID: id}
	}
	if err != nil {
		return Restore{}, fmt.Errorf("get restore %q: %w", id, err)
	}
	return r, nil
}

func (s *sqliteRepository) ListRestores(ctx context.Context, targetID string, limit int) ([]Restore, error) {
	if limit <= 0 {
		return nil, nil
	}
	var (
		rows *sql.Rows
		err  error
	)
	cols := `id, target_id, destination_id, snapshot_id, mode, status, location, checksum,
	         last_verified_at, requested_at, finished_at, error`
	if targetID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT `+cols+` FROM restores WHERE target_id = ? ORDER BY requested_at DESC, id DESC LIMIT ?`,
			targetID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT `+cols+` FROM restores ORDER BY requested_at DESC, id DESC LIMIT ?`,
			limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list restores: %w", err)
	}
	defer rows.Close()
	var out []Restore
	for rows.Next() {
		r, err := scanRestore(rows)
		if err != nil {
			return nil, fmt.Errorf("scan restore: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate restores: %w", err)
	}
	return out, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRestore(sc rowScanner) (Restore, error) {
	var (
		r                                          Restore
		mode, status                               string
		lastVerifiedRaw, requestedRaw, finishedRaw string
	)
	if err := sc.Scan(
		&r.ID, &r.TargetID, &r.DestinationID, &r.SnapshotID,
		&mode, &status,
		&r.Location, &r.Checksum,
		&lastVerifiedRaw, &requestedRaw, &finishedRaw, &r.Error,
	); err != nil {
		return Restore{}, err
	}
	r.Mode = RestoreMode(mode)
	r.Status = RestoreStatus(status)
	r.LastVerifiedAt = parseTime(lastVerifiedRaw)
	r.RequestedAt = parseTime(requestedRaw)
	r.FinishedAt = parseTime(finishedRaw)
	return r, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(restoreTimeFormat)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(restoreTimeFormat, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
