package store

import (
	"database/sql"
	"fmt"

	"tunnel-manager/domain"
)

// routeColumns is the canonical column list for route queries, used in SELECT and RETURNING clauses.
const routeColumns = "id, subdomain, scenario_name, local_port, health_path, public_url, enabled, created_at, updated_at"

// scanRoute scans a single Route from the given row (sql.Row or compatible scanner).
func scanRoute(scanner interface{ Scan(dest ...any) error }) (domain.Route, error) {
	var r domain.Route
	err := scanner.Scan(&r.ID, &r.Subdomain, &r.ScenarioName, &r.LocalPort, &r.HealthPath, &r.PublicURL, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// RouteStore provides persistence operations for tunnel routes.
type RouteStore struct {
	db *sql.DB
}

func NewRouteStore(db *sql.DB) *RouteStore {
	return &RouteStore{db: db}
}

func (rs *RouteStore) List() ([]domain.Route, error) {
	rows, err := rs.db.Query(`SELECT ` + routeColumns + ` FROM routes ORDER BY subdomain`)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	defer rows.Close()

	var routes []domain.Route
	for rows.Next() {
		r, err := scanRoute(rows)
		if err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		routes = append(routes, r)
	}
	return routes, rows.Err()
}

func (rs *RouteStore) GetByID(id int) (*domain.Route, error) {
	r, err := scanRoute(rs.db.QueryRow(`SELECT `+routeColumns+` FROM routes WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get route %d: %w", id, err)
	}
	return &r, nil
}

func (rs *RouteStore) Create(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
	r, err := scanRoute(rs.db.QueryRow(
		`INSERT INTO routes (subdomain, scenario_name, local_port, health_path, public_url, enabled) VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+routeColumns,
		subdomain, scenarioName, localPort, healthPath, publicURL, enabled,
	))
	if err != nil {
		return nil, fmt.Errorf("create route: %w", err)
	}
	return &r, nil
}

func (rs *RouteStore) Update(id int, subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
	r, err := scanRoute(rs.db.QueryRow(
		`UPDATE routes SET subdomain=$1, scenario_name=$2, local_port=$3, health_path=$4, public_url=$5, enabled=$6, updated_at=NOW() WHERE id=$7 RETURNING `+routeColumns,
		subdomain, scenarioName, localPort, healthPath, publicURL, enabled, id,
	))
	if err != nil {
		return nil, fmt.Errorf("update route %d: %w", id, err)
	}
	return &r, nil
}

func (rs *RouteStore) Delete(id int) error {
	result, err := rs.db.Exec(`DELETE FROM routes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete route %d: %w", id, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound(fmt.Sprintf("route %d not found", id))
	}
	return nil
}
