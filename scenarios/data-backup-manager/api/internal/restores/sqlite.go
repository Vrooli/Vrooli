package restores

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO restores
			(id, target_id, destination_id, snapshot_id, mode, status, location, checksum,
			 last_verified_at, requested_at, finished_at, error, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.TargetID, r.DestinationID, r.SnapshotID,
		string(r.Mode), string(r.Status),
		r.Location, r.Checksum,
		formatTime(r.LastVerifiedAt), r.RequestedAt.UTC().Format(restoreTimeFormat),
		formatTime(r.FinishedAt), r.Error, formatTime(r.UpdatedAt),
	)
	if err != nil {
		return Restore{}, fmt.Errorf("insert restore: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) UpdateRestoreStatus(ctx context.Context, id string, status RestoreStatus) error {
	now := formatTime(s.clock.Now().UTC())
	if _, err := s.db.ExecContext(ctx,
		`UPDATE restores SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), now, id,
	); err != nil {
		return fmt.Errorf("update restore status %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) FinishRestore(ctx context.Context, id string, status RestoreStatus, checksum string, lastVerifiedAt, finishedAt time.Time, errMsg string) error {
	now := formatTime(s.clock.Now().UTC())
	if _, err := s.db.ExecContext(ctx,
		`UPDATE restores SET status = ?, checksum = ?, last_verified_at = ?, finished_at = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(status), checksum, formatTime(lastVerifiedAt), formatTime(finishedAt), errMsg, now, id,
	); err != nil {
		return fmt.Errorf("finish restore %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) ListNonTerminalRestores(ctx context.Context) ([]Restore, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, target_id, destination_id, snapshot_id, mode, status, location, checksum,
		        last_verified_at, requested_at, finished_at, error, updated_at
		 FROM restores WHERE status IN (?, ?, ?) ORDER BY requested_at ASC, id ASC`,
		string(RestoreRequested), string(RestoreRestoring), string(RestoreVerifying))
	if err != nil {
		return nil, fmt.Errorf("list non-terminal restores: %w", err)
	}
	defer rows.Close()
	var out []Restore
	for rows.Next() {
		r, err := scanRestore(rows)
		if err != nil {
			return nil, fmt.Errorf("scan non-terminal restore: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate non-terminal restores: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) GetRestore(ctx context.Context, id string) (Restore, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, target_id, destination_id, snapshot_id, mode, status, location, checksum,
		        last_verified_at, requested_at, finished_at, error, updated_at
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
	         last_verified_at, requested_at, finished_at, error, updated_at`
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

func (s *sqliteRepository) LastVerifiedByTarget(ctx context.Context, targetIDs []string) ([]VerifiedStatus, error) {
	// Newest verified-first; the first row seen per target is its latest verify.
	query := `
SELECT target_id, last_verified_at, snapshot_id
FROM restores
WHERE status = ? AND mode = ? %s
ORDER BY last_verified_at DESC, id DESC`
	where := ""
	args := []any{string(RestoreVerified), string(ModeVerify)}
	if len(targetIDs) > 0 {
		where = "AND target_id IN (" + placeholders(len(targetIDs)) + ")"
		for _, id := range targetIDs {
			args = append(args, id)
		}
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(query, where), args...)
	if err != nil {
		return nil, fmt.Errorf("last verified by target: %w", err)
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	var out []VerifiedStatus
	for rows.Next() {
		var targetID, verifiedRaw, snapshotID string
		if err := rows.Scan(&targetID, &verifiedRaw, &snapshotID); err != nil {
			return nil, fmt.Errorf("scan verified status: %w", err)
		}
		if _, ok := seen[targetID]; ok {
			continue // first row per target is the latest verify
		}
		seen[targetID] = struct{}{}
		out = append(out, VerifiedStatus{
			TargetID:       targetID,
			LastVerifiedAt: parseTime(verifiedRaw),
			SnapshotID:     snapshotID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verified statuses: %w", err)
	}
	return out, nil
}

// placeholders returns "?,?,…" with n placeholders for an IN clause.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRestore(sc rowScanner) (Restore, error) {
	var (
		r                                                      Restore
		mode, status                                           string
		lastVerifiedRaw, requestedRaw, finishedRaw, updatedRaw string
	)
	if err := sc.Scan(
		&r.ID, &r.TargetID, &r.DestinationID, &r.SnapshotID,
		&mode, &status,
		&r.Location, &r.Checksum,
		&lastVerifiedRaw, &requestedRaw, &finishedRaw, &r.Error, &updatedRaw,
	); err != nil {
		return Restore{}, err
	}
	r.Mode = RestoreMode(mode)
	r.Status = RestoreStatus(status)
	r.LastVerifiedAt = parseTime(lastVerifiedRaw)
	r.RequestedAt = parseTime(requestedRaw)
	r.FinishedAt = parseTime(finishedRaw)
	r.UpdatedAt = parseTime(updatedRaw)
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
