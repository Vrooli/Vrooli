package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"tunnel-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (repository
// unit tests via testutil/db.NewSQLite) and *database.RoutedDB (production
// main.go) satisfy it.
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

const (
	// RFC3339Nano matches the wire format and the round-trip in scanRoute.
	routeTimeFormat = time.RFC3339Nano

	routeColumns = `id, subdomain, scenario, domain, local_port, tier, lease_id, enabled, health_path, source, service_target, created_at, updated_at`

	insertRouteSQL = `
INSERT INTO routes (id, subdomain, scenario, domain, local_port, tier, lease_id, enabled, health_path, source, service_target, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	updateRouteSQL = `
UPDATE routes
SET subdomain = ?, scenario = ?, domain = ?, local_port = ?, tier = ?, lease_id = ?, enabled = ?, health_path = ?, source = ?, service_target = ?, updated_at = ?
WHERE id = ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, r Route) (Route, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = s.clock.Now().UTC()
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = r.CreatedAt
	}

	if r.Source == "" {
		r.Source = SourceScenario
	}
	_, err := s.db.ExecContext(ctx, insertRouteSQL,
		r.ID, r.Subdomain, r.Scenario, r.Domain, r.LocalPort,
		string(r.Tier), r.LeaseID, boolToInt(r.Enabled), r.HealthPath,
		string(r.Source), r.ServiceTarget,
		r.CreatedAt.Format(routeTimeFormat), r.UpdatedAt.Format(routeTimeFormat),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Route{}, ErrRouteConflict{Subdomain: r.Subdomain}
		}
		return Route{}, fmt.Errorf("insert route %q: %w", r.ID, err)
	}
	return r, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Route, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+routeColumns+" FROM routes WHERE id = ?", id)
	r, err := scanRoute(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Route{}, ErrRouteNotFound{ID: id}
	}
	if err != nil {
		return Route{}, fmt.Errorf("get route %q: %w", id, err)
	}
	return r, nil
}

func (s *sqliteRepository) GetBySubdomain(ctx context.Context, subdomain string) (Route, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+routeColumns+" FROM routes WHERE subdomain = ?", subdomain)
	r, err := scanRoute(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Route{}, ErrRouteNotFound{ID: subdomain}
	}
	if err != nil {
		return Route{}, fmt.Errorf("get route by subdomain %q: %w", subdomain, err)
	}
	return r, nil
}

func (s *sqliteRepository) List(ctx context.Context, tier Tier) ([]Route, error) {
	query := "SELECT " + routeColumns + " FROM routes"
	var args []any
	if tier != "" {
		query += " WHERE tier = ?"
		args = append(args, string(tier))
	}
	query += " ORDER BY subdomain ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		r, err := scanRoute(rows)
		if err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		routes = append(routes, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routes: %w", err)
	}
	return routes, nil
}

func (s *sqliteRepository) Update(ctx context.Context, r Route) (Route, error) {
	r.UpdatedAt = s.clock.Now().UTC()
	if r.Source == "" {
		r.Source = SourceScenario
	}
	res, err := s.db.ExecContext(ctx, updateRouteSQL,
		r.Subdomain, r.Scenario, r.Domain, r.LocalPort,
		string(r.Tier), r.LeaseID, boolToInt(r.Enabled), r.HealthPath,
		string(r.Source), r.ServiceTarget,
		r.UpdatedAt.Format(routeTimeFormat), r.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Route{}, ErrRouteConflict{Subdomain: r.Subdomain}
		}
		return Route{}, fmt.Errorf("update route %q: %w", r.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Route{}, fmt.Errorf("update route %q rows affected: %w", r.ID, err)
	}
	if n == 0 {
		return Route{}, ErrRouteNotFound{ID: r.ID}
	}
	return r, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM routes WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete route %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete route %q rows affected: %w", id, err)
	}
	return n > 0, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan
// surface so scanRoute works for both single-row and multi-row reads.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRoute(sc rowScanner) (Route, error) {
	var (
		r          Route
		tierRaw    string
		enabledRaw int
		sourceRaw  string
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&r.ID, &r.Subdomain, &r.Scenario, &r.Domain, &r.LocalPort,
		&tierRaw, &r.LeaseID, &enabledRaw, &r.HealthPath, &sourceRaw, &r.ServiceTarget, &createdRaw, &updatedRaw); err != nil {
		return Route{}, err
	}
	r.Tier = Tier(tierRaw)
	r.Enabled = enabledRaw != 0
	r.Source = RouteSource(sourceRaw)
	if r.Source == "" {
		r.Source = SourceScenario
	}
	created, err := time.Parse(routeTimeFormat, createdRaw)
	if err != nil {
		return Route{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	updated, err := time.Parse(routeTimeFormat, updatedRaw)
	if err != nil {
		return Route{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	r.CreatedAt = created
	r.UpdatedAt = updated
	return r, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint
// failure (the one-route-per-subdomain invariant). Matched on message
// text to avoid coupling the domain to a specific driver type.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT FAILED")
}
