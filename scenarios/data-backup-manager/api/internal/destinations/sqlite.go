package destinations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"data-backup-manager/internal/clock"

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
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const destTimeFormat = time.RFC3339Nano

const (
	insertDestSQL = `
INSERT INTO destinations (id, name, backend_kind, location, repository_location, cap_bytes, cap_policy, encryption_algorithm, secret_ref, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	updateDestSQL = `
UPDATE destinations
SET cap_bytes = ?, cap_policy = ?, updated_at = ?
WHERE id = ?
`
	selectDestByIDSQL = `
SELECT id, name, backend_kind, location, repository_location, cap_bytes, cap_policy, encryption_algorithm, secret_ref, created_at, updated_at
FROM destinations WHERE id = ?
`
	selectDestByNameSQL = `
SELECT id, name, backend_kind, location, repository_location, cap_bytes, cap_policy, encryption_algorithm, secret_ref, created_at, updated_at
FROM destinations WHERE name = ?
`
	listDestsSQL = `
SELECT id, name, backend_kind, location, repository_location, cap_bytes, cap_policy, encryption_algorithm, secret_ref, created_at, updated_at
FROM destinations
ORDER BY name ASC
LIMIT ?
`
	deleteDestSQL = `DELETE FROM destinations WHERE id = ?`
)

func (s *sqliteRepository) Create(ctx context.Context, d Destination) (Destination, error) {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, insertDestSQL,
		d.ID, d.Name, string(d.BackendKind), d.Location, d.RepositoryLocation, d.CapBytes, string(d.CapPolicy),
		d.EncryptionAlgorithm, d.SecretRef,
		d.CreatedAt.Format(destTimeFormat), d.UpdatedAt.Format(destTimeFormat),
	)
	if err != nil {
		return Destination{}, fmt.Errorf("insert destination %q: %w", d.Name, err)
	}
	return d, nil
}

func (s *sqliteRepository) Update(ctx context.Context, d Destination) (Destination, error) {
	d.UpdatedAt = s.clock.Now().UTC()
	_, err := s.db.ExecContext(ctx, updateDestSQL,
		d.CapBytes, string(d.CapPolicy), d.UpdatedAt.Format(destTimeFormat), d.ID,
	)
	if err != nil {
		return Destination{}, fmt.Errorf("update destination %q: %w", d.ID, err)
	}
	return d, nil
}

func (s *sqliteRepository) GetByID(ctx context.Context, id string) (Destination, error) {
	row := s.db.QueryRowContext(ctx, selectDestByIDSQL, id)
	d, err := scanDest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Destination{}, ErrDestinationNotFound{ID: id}
	}
	if err != nil {
		return Destination{}, fmt.Errorf("get destination %q: %w", id, err)
	}
	return d, nil
}

func (s *sqliteRepository) GetByName(ctx context.Context, name string) (Destination, error) {
	row := s.db.QueryRowContext(ctx, selectDestByNameSQL, name)
	d, err := scanDest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Destination{}, ErrDestinationNotFound{Name: name}
	}
	if err != nil {
		return Destination{}, fmt.Errorf("get destination by name %q: %w", name, err)
	}
	return d, nil
}

func (s *sqliteRepository) List(ctx context.Context, limit int) ([]Destination, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, listDestsSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list destinations: %w", err)
	}
	defer rows.Close()

	var dests []Destination
	for rows.Next() {
		d, err := scanDest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan destination: %w", err)
		}
		dests = append(dests, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate destinations: %w", err)
	}
	return dests, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, deleteDestSQL, id)
	if err != nil {
		return false, fmt.Errorf("delete destination %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete destination %q rows: %w", id, err)
	}
	return n > 0, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDest(sc rowScanner) (Destination, error) {
	var (
		d          Destination
		backendRaw string
		policyRaw  string
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(
		&d.ID, &d.Name, &backendRaw, &d.Location, &d.RepositoryLocation, &d.CapBytes, &policyRaw,
		&d.EncryptionAlgorithm, &d.SecretRef, &createdRaw, &updatedRaw,
	); err != nil {
		return Destination{}, err
	}
	d.BackendKind = BackendKind(backendRaw)
	d.CapPolicy = CapPolicy(policyRaw)
	created, err := time.Parse(destTimeFormat, createdRaw)
	if err != nil {
		return Destination{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	updated, err := time.Parse(destTimeFormat, updatedRaw)
	if err != nil {
		return Destination{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	d.CreatedAt = created
	d.UpdatedAt = updated
	return d, nil
}
