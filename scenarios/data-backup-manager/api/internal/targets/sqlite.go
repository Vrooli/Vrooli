package targets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"data-backup-manager/internal/sources"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Both *sql.DB (repository unit tests via testutil/db.NewSQLite) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const targetTimeFormat = time.RFC3339Nano

const (
	insertTargetSQL = `
INSERT INTO targets (id, owner, name, source_kind, locator, critical, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	updateTargetSQL = `
UPDATE targets
SET source_kind = ?, locator = ?, critical = ?, updated_at = ?
WHERE id = ?
`
	selectTargetByOwnerNameSQL = `
SELECT id, owner, name, source_kind, locator, critical, created_at, updated_at
FROM targets WHERE owner = ? AND name = ?
`
	selectTargetByIDSQL = `
SELECT id, owner, name, source_kind, locator, critical, created_at, updated_at
FROM targets WHERE id = ?
`
	listTargetsSQL = `
SELECT id, owner, name, source_kind, locator, critical, created_at, updated_at
FROM targets
ORDER BY owner ASC, name ASC
LIMIT ?
`
	listTargetsByOwnerSQL = `
SELECT id, owner, name, source_kind, locator, critical, created_at, updated_at
FROM targets
WHERE owner = ?
ORDER BY name ASC
LIMIT ?
`
	deleteTargetSQL = `DELETE FROM targets WHERE owner = ? AND name = ?`
)

func (s *sqliteRepository) Create(ctx context.Context, t Target) (Target, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = t.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, insertTargetSQL,
		t.ID, t.Owner, t.Name, string(t.SourceKind), t.Locator, t.Critical,
		t.CreatedAt.Format(targetTimeFormat), t.UpdatedAt.Format(targetTimeFormat),
	)
	if err != nil {
		return Target{}, fmt.Errorf("insert target %s/%s: %w", t.Owner, t.Name, err)
	}
	return t, nil
}

func (s *sqliteRepository) Update(ctx context.Context, t Target) (Target, error) {
	t.UpdatedAt = s.clock.Now().UTC()
	_, err := s.db.ExecContext(ctx, updateTargetSQL,
		string(t.SourceKind), t.Locator, t.Critical, t.UpdatedAt.Format(targetTimeFormat), t.ID,
	)
	if err != nil {
		return Target{}, fmt.Errorf("update target %q: %w", t.ID, err)
	}
	return t, nil
}

func (s *sqliteRepository) GetByOwnerName(ctx context.Context, owner, name string) (Target, error) {
	row := s.db.QueryRowContext(ctx, selectTargetByOwnerNameSQL, owner, name)
	t, err := scanTarget(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrTargetNotFound{Owner: owner, Name: name}
	}
	if err != nil {
		return Target{}, fmt.Errorf("get target %s/%s: %w", owner, name, err)
	}
	return t, nil
}

func (s *sqliteRepository) GetByID(ctx context.Context, id string) (Target, error) {
	row := s.db.QueryRowContext(ctx, selectTargetByIDSQL, id)
	t, err := scanTarget(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrTargetNotFound{ID: id}
	}
	if err != nil {
		return Target{}, fmt.Errorf("get target %q: %w", id, err)
	}
	return t, nil
}

func (s *sqliteRepository) List(ctx context.Context, owner string, limit int) ([]Target, error) {
	if limit <= 0 {
		return nil, nil
	}
	var (
		rows *sql.Rows
		err  error
	)
	if owner != "" {
		rows, err = s.db.QueryContext(ctx, listTargetsByOwnerSQL, owner, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, listTargetsSQL, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			return nil, fmt.Errorf("scan target: %w", err)
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate targets: %w", err)
	}
	return targets, nil
}

func (s *sqliteRepository) DeleteByOwnerName(ctx context.Context, owner, name string) (bool, error) {
	res, err := s.db.ExecContext(ctx, deleteTargetSQL, owner, name)
	if err != nil {
		return false, fmt.Errorf("delete target %s/%s: %w", owner, name, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete target %s/%s rows: %w", owner, name, err)
	}
	return n > 0, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTarget(sc rowScanner) (Target, error) {
	var (
		t          Target
		kindRaw    string
		critical   bool
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&t.ID, &t.Owner, &t.Name, &kindRaw, &t.Locator, &critical, &createdRaw, &updatedRaw); err != nil {
		return Target{}, err
	}
	t.SourceKind = sources.SourceKind(kindRaw)
	t.Critical = critical
	created, err := time.Parse(targetTimeFormat, createdRaw)
	if err != nil {
		return Target{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	updated, err := time.Parse(targetTimeFormat, updatedRaw)
	if err != nil {
		return Target{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	t.CreatedAt = created
	t.UpdatedAt = updated
	return t, nil
}
