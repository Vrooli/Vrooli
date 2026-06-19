package exposure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"tunnel-manager/internal/clock"

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

const (
	leaseTimeFormat = time.RFC3339Nano

	leaseColumns = `id, scenario, requested_by, created_at, expires_at, extended_count, status`

	insertLeaseSQL = `
INSERT INTO leases (id, scenario, requested_by, created_at, expires_at, extended_count, status)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

	updateLeaseSQL = `
UPDATE leases
SET scenario = ?, requested_by = ?, expires_at = ?, extended_count = ?, status = ?
WHERE id = ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, l Lease) (Lease, error) {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = s.clock.Now().UTC()
	}
	if l.Status == "" {
		l.Status = LeaseActive
	}
	_, err := s.db.ExecContext(ctx, insertLeaseSQL,
		l.ID, l.Scenario, l.RequestedBy,
		l.CreatedAt.Format(leaseTimeFormat), l.ExpiresAt.Format(leaseTimeFormat),
		l.ExtendedCount, string(l.Status),
	)
	if err != nil {
		return Lease{}, fmt.Errorf("insert lease %q: %w", l.ID, err)
	}
	return l, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Lease, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+leaseColumns+" FROM leases WHERE id = ?", id)
	l, err := scanLease(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Lease{}, ErrLeaseNotFound{ID: id}
	}
	if err != nil {
		return Lease{}, fmt.Errorf("get lease %q: %w", id, err)
	}
	return l, nil
}

func (s *sqliteRepository) ActiveForScenario(ctx context.Context, scenario string) (Lease, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT "+leaseColumns+" FROM leases WHERE scenario = ? AND status = ? ORDER BY created_at DESC LIMIT 1",
		scenario, string(LeaseActive))
	l, err := scanLease(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Lease{}, ErrLeaseNotFound{ID: scenario}
	}
	if err != nil {
		return Lease{}, fmt.Errorf("active lease for %q: %w", scenario, err)
	}
	return l, nil
}

func (s *sqliteRepository) Update(ctx context.Context, l Lease) (Lease, error) {
	res, err := s.db.ExecContext(ctx, updateLeaseSQL,
		l.Scenario, l.RequestedBy, l.ExpiresAt.Format(leaseTimeFormat),
		l.ExtendedCount, string(l.Status), l.ID,
	)
	if err != nil {
		return Lease{}, fmt.Errorf("update lease %q: %w", l.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Lease{}, fmt.Errorf("update lease %q rows affected: %w", l.ID, err)
	}
	if n == 0 {
		return Lease{}, ErrLeaseNotFound{ID: l.ID}
	}
	return l, nil
}

func (s *sqliteRepository) List(ctx context.Context, status LeaseStatus) ([]Lease, error) {
	query := "SELECT " + leaseColumns + " FROM leases"
	var args []any
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY created_at DESC, id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list leases: %w", err)
	}
	defer rows.Close()

	var leases []Lease
	for rows.Next() {
		l, err := scanLease(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lease: %w", err)
		}
		leases = append(leases, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leases: %w", err)
	}
	return leases, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLease(sc rowScanner) (Lease, error) {
	var (
		l          Lease
		statusRaw  string
		createdRaw string
		expiresRaw string
	)
	if err := sc.Scan(&l.ID, &l.Scenario, &l.RequestedBy, &createdRaw, &expiresRaw, &l.ExtendedCount, &statusRaw); err != nil {
		return Lease{}, err
	}
	l.Status = LeaseStatus(statusRaw)
	created, err := time.Parse(leaseTimeFormat, createdRaw)
	if err != nil {
		return Lease{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	expires, err := time.Parse(leaseTimeFormat, expiresRaw)
	if err != nil {
		return Lease{}, fmt.Errorf("parse expires_at %q: %w", expiresRaw, err)
	}
	l.CreatedAt = created
	l.ExpiresAt = expires
	return l, nil
}
