package coverage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// snapTimeFormat matches the notes domain (RFC3339Nano sorts lexicographically
// in time order for a fixed zone, so a string range filter is correct).
const snapTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the snapshot repository depends on
// (declared at the consumer per seam-discovery). Both *sql.DB (tests) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteSnapshots struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteSnapshotRepository constructs the production SnapshotRepository.
func NewSQLiteSnapshotRepository(db SQLExecutor, clk schedule.Clock) SnapshotRepository {
	return &sqliteSnapshots{db: db, clock: clk}
}

var _ SnapshotRepository = (*sqliteSnapshots)(nil)

const (
	insertSnapshotSQL = `INSERT INTO coverage_snapshots (computed_at, payload) VALUES (?, ?)`
	// Keep only the newest few snapshots; the cache needs just the latest.
	pruneSnapshotsSQL = `
DELETE FROM coverage_snapshots
WHERE id NOT IN (SELECT id FROM coverage_snapshots ORDER BY computed_at DESC, id DESC LIMIT 5)`
	latestSnapshotSQL = `SELECT computed_at, payload FROM coverage_snapshots ORDER BY computed_at DESC, id DESC LIMIT 1`
)

func (s *sqliteSnapshots) Save(ctx context.Context, status Status) error {
	payload, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	at := status.ComputedAt
	if at.IsZero() {
		at = s.clock.Now().UTC()
	}
	if _, err := s.db.ExecContext(ctx, insertSnapshotSQL, at.Format(snapTimeFormat), string(payload)); err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, pruneSnapshotsSQL) // best-effort prune
	return nil
}

func (s *sqliteSnapshots) Latest(ctx context.Context, ttl time.Duration, now time.Time) (Status, bool) {
	var (
		atRaw   string
		payload string
	)
	err := s.db.QueryRowContext(ctx, latestSnapshotSQL).Scan(&atRaw, &payload)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return Status{}, false
	}
	at, err := time.Parse(snapTimeFormat, atRaw)
	if err != nil {
		return Status{}, false
	}
	if now.Sub(at) > ttl {
		return Status{}, false
	}
	var status Status
	if err := json.Unmarshal([]byte(payload), &status); err != nil {
		return Status{}, false
	}
	return status, true
}
