package fleet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"vrooli-bridge/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests) and *database.RoutedDB (production) satisfy
// it, so production participates in per-request routing without the test
// fixture wrapping its handle.
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

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// rolloutTimeFormat sorts lexicographically in time order for a fixed zone, so a
// string order comparison is a correct newest-first filter.
const rolloutTimeFormat = time.RFC3339Nano

const (
	insertRolloutSQL = `
INSERT INTO rollouts (id, target_revision, status, total_nodes, dispatched, skipped, failed, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

	insertResultSQL = `
INSERT OR IGNORE INTO rollout_results (rollout_id, node_id, disposition, op_id, detail)
VALUES (?, ?, ?, ?, ?)
`

	selectRolloutColumns = `
SELECT id, target_revision, status, total_nodes, dispatched, skipped, failed, created_at
FROM rollouts
`

	selectRolloutByIDSQL = selectRolloutColumns + `WHERE id = ?`

	selectResultsSQL = `
SELECT node_id, disposition, op_id, detail
FROM rollout_results
WHERE rollout_id = ?
ORDER BY node_id ASC
`
)

func (s *sqliteRepository) Create(ctx context.Context, rollout Rollout, results []NodeResult) (Rollout, error) {
	if rollout.ID == "" {
		rollout.ID = uuid.NewString()
	}
	if rollout.CreatedAt.IsZero() {
		rollout.CreatedAt = s.clock.Now().UTC()
	}
	if _, err := s.db.ExecContext(ctx, insertRolloutSQL,
		rollout.ID, rollout.TargetRevision, int(rollout.Status), rollout.TotalNodes,
		rollout.Dispatched, rollout.Skipped, rollout.Failed, rollout.CreatedAt.Format(rolloutTimeFormat),
	); err != nil {
		return Rollout{}, fmt.Errorf("insert rollout %q: %w", rollout.ID, err)
	}
	for _, r := range results {
		if _, err := s.db.ExecContext(ctx, insertResultSQL,
			rollout.ID, r.NodeID, int(r.Disposition), r.OpID, r.Detail,
		); err != nil {
			return Rollout{}, fmt.Errorf("insert rollout result for node %q: %w", r.NodeID, err)
		}
	}
	return rollout, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Rollout, error) {
	row := s.db.QueryRowContext(ctx, selectRolloutByIDSQL, id)
	rollout, err := scanRollout(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Rollout{}, ErrRolloutNotFound{ID: id}
	}
	if err != nil {
		return Rollout{}, fmt.Errorf("get rollout %q: %w", id, err)
	}
	return rollout, nil
}

func (s *sqliteRepository) Results(ctx context.Context, rolloutID string) ([]NodeResult, error) {
	rows, err := s.db.QueryContext(ctx, selectResultsSQL, rolloutID)
	if err != nil {
		return nil, fmt.Errorf("list results for rollout %q: %w", rolloutID, err)
	}
	defer rows.Close()

	var out []NodeResult
	for rows.Next() {
		var (
			r           NodeResult
			disposition int
		)
		if err := rows.Scan(&r.NodeID, &disposition, &r.OpID, &r.Detail); err != nil {
			return nil, fmt.Errorf("scan rollout result: %w", err)
		}
		r.Disposition = NodeDisposition(disposition)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rollout results: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Rollout, error) {
	query := selectRolloutColumns + `ORDER BY created_at DESC, id DESC`
	args := make([]any, 0, 1)
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list rollouts: %w", err)
	}
	defer rows.Close()

	var out []Rollout
	for rows.Next() {
		rollout, err := scanRollout(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rollout: %w", err)
		}
		out = append(out, rollout)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rollouts: %w", err)
	}
	return out, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRollout(sc rowScanner) (Rollout, error) {
	var (
		r          Rollout
		status     int
		createdRaw string
	)
	if err := sc.Scan(&r.ID, &r.TargetRevision, &status, &r.TotalNodes,
		&r.Dispatched, &r.Skipped, &r.Failed, &createdRaw); err != nil {
		return Rollout{}, err
	}
	r.Status = RolloutStatus(status)
	created, err := time.Parse(rolloutTimeFormat, createdRaw)
	if err != nil {
		return Rollout{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	r.CreatedAt = created
	return r, nil
}
