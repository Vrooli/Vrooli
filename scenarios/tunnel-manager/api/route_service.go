package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Route represents a published tunnel route mapping a subdomain to a local scenario port.
type Route struct {
	ID           int       `json:"id"`
	Subdomain    string    `json:"subdomain"`
	ScenarioName string    `json:"scenario_name"`
	LocalPort    int       `json:"local_port"`
	HealthPath   string    `json:"health_path"`
	PublicURL    string    `json:"public_url"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RouteInput is the create/update payload for routes.
type RouteInput struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	LocalPort    int    `json:"local_port"`
	HealthPath   string `json:"health_path,omitempty"`
	PublicURL    string `json:"public_url,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

// RouteService handles route manifest CRUD operations.
type RouteService struct {
	db *sql.DB
}

func NewRouteService(db *sql.DB) *RouteService {
	return &RouteService{db: db}
}

func (rs *RouteService) List() ([]Route, error) {
	rows, err := rs.db.Query(`SELECT id, subdomain, scenario_name, local_port, health_path, public_url, enabled, created_at, updated_at FROM routes ORDER BY subdomain`)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var r Route
		if err := rows.Scan(&r.ID, &r.Subdomain, &r.ScenarioName, &r.LocalPort, &r.HealthPath, &r.PublicURL, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		routes = append(routes, r)
	}
	return routes, rows.Err()
}

func (rs *RouteService) GetByID(id int) (*Route, error) {
	var r Route
	err := rs.db.QueryRow(`SELECT id, subdomain, scenario_name, local_port, health_path, public_url, enabled, created_at, updated_at FROM routes WHERE id = $1`, id).
		Scan(&r.ID, &r.Subdomain, &r.ScenarioName, &r.LocalPort, &r.HealthPath, &r.PublicURL, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get route %d: %w", id, err)
	}
	return &r, nil
}

func (rs *RouteService) Create(in RouteInput) (*Route, error) {
	if err := validateRouteInput(in, false); err != nil {
		return nil, err
	}
	healthPath := "/health"
	if in.HealthPath != "" {
		healthPath = in.HealthPath
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	var r Route
	err := rs.db.QueryRow(`INSERT INTO routes (subdomain, scenario_name, local_port, health_path, public_url, enabled) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, subdomain, scenario_name, local_port, health_path, public_url, enabled, created_at, updated_at`,
		in.Subdomain, in.ScenarioName, in.LocalPort, healthPath, in.PublicURL, enabled,
	).Scan(&r.ID, &r.Subdomain, &r.ScenarioName, &r.LocalPort, &r.HealthPath, &r.PublicURL, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create route: %w", err)
	}
	return &r, nil
}

func (rs *RouteService) Update(id int, in RouteInput) (*Route, error) {
	if err := validateRouteInput(in, true); err != nil {
		return nil, err
	}
	existing, err := rs.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	// Merge fields
	if in.Subdomain != "" {
		existing.Subdomain = in.Subdomain
	}
	if in.ScenarioName != "" {
		existing.ScenarioName = in.ScenarioName
	}
	if in.LocalPort != 0 {
		existing.LocalPort = in.LocalPort
	}
	if in.HealthPath != "" {
		existing.HealthPath = in.HealthPath
	}
	if in.PublicURL != "" {
		existing.PublicURL = in.PublicURL
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}

	var r Route
	err = rs.db.QueryRow(`UPDATE routes SET subdomain=$1, scenario_name=$2, local_port=$3, health_path=$4, public_url=$5, enabled=$6, updated_at=NOW() WHERE id=$7 RETURNING id, subdomain, scenario_name, local_port, health_path, public_url, enabled, created_at, updated_at`,
		existing.Subdomain, existing.ScenarioName, existing.LocalPort, existing.HealthPath, existing.PublicURL, existing.Enabled, id,
	).Scan(&r.ID, &r.Subdomain, &r.ScenarioName, &r.LocalPort, &r.HealthPath, &r.PublicURL, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update route %d: %w", id, err)
	}
	return &r, nil
}

func (rs *RouteService) Delete(id int) error {
	result, err := rs.db.Exec(`DELETE FROM routes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete route %d: %w", id, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("route %d not found", id)
	}
	return nil
}

func validateRouteInput(in RouteInput, isUpdate bool) error {
	if !isUpdate {
		if in.Subdomain == "" {
			return fmt.Errorf("subdomain is required")
		}
		if in.ScenarioName == "" {
			return fmt.Errorf("scenario_name is required")
		}
		if in.LocalPort == 0 {
			return fmt.Errorf("local_port is required")
		}
	}
	if in.LocalPort != 0 && (in.LocalPort < 1 || in.LocalPort > 65535) {
		return fmt.Errorf("local_port must be between 1 and 65535")
	}
	return nil
}

// --- HTTP Handlers ---

func handleListRoutes(svc *RouteService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routes, err := svc.List()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if routes == nil {
			routes = []Route{}
		}
		writeJSON(w, http.StatusOK, routes)
	}
}

func handleGetRoute(svc *RouteService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid route id")
			return
		}
		route, err := svc.GetByID(id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if route == nil {
			writeJSONError(w, http.StatusNotFound, "route not found")
			return
		}
		writeJSON(w, http.StatusOK, route)
	}
}

func handleCreateRoute(svc *RouteService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in RouteInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		route, err := svc.Create(in)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, route)
	}
}

func handleUpdateRoute(svc *RouteService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid route id")
			return
		}
		var in RouteInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		route, err := svc.Update(id, in)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		if route == nil {
			writeJSONError(w, http.StatusNotFound, "route not found")
			return
		}
		writeJSON(w, http.StatusOK, route)
	}
}

func handleDeleteRoute(svc *RouteService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid route id")
			return
		}
		if err := svc.Delete(id); err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
