package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// handleGetAdoptions retrieves all adoption records
func (s *Server) handleGetAdoptions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, component_id, component_library_id, scenario_name, adopted_path,
		       version, status, created_at, updated_at
		FROM adoption_records
		ORDER BY created_at DESC
	`)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query adoptions", err)
		return
	}
	defer rows.Close()

	adoptions := []AdoptionRecord{}
	for rows.Next() {
		var a AdoptionRecord
		err := rows.Scan(
			&a.ID, &a.ComponentID, &a.ComponentLibraryID, &a.ScenarioName,
			&a.AdoptedPath, &a.Version, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to scan adoption", err)
			return
		}
		adoptions = append(adoptions, a)
	}

	s.respondJSON(w, http.StatusOK, adoptions)
}

// handleCreateAdoption creates a new adoption record
func (s *Server) handleCreateAdoption(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ComponentID string `json:"componentId"`
		ScenarioName string `json:"scenarioName"`
		AdoptedPath string `json:"adoptedPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get component details to populate the adoption record
	var componentLibraryID, version string
	err := s.db.QueryRowContext(r.Context(), `
		SELECT library_id, version FROM components WHERE id = $1
	`, input.ComponentID).Scan(&componentLibraryID, &version)

	if err == sql.ErrNoRows {
		s.respondError(w, http.StatusNotFound, "Component not found", nil)
		return
	}
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query component", err)
		return
	}

	var a AdoptionRecord
	err = s.db.QueryRowContext(r.Context(), `
		INSERT INTO adoption_records (component_id, component_library_id, scenario_name,
		                             adopted_path, version, status)
		VALUES ($1, $2, $3, $4, $5, 'current')
		ON CONFLICT (component_id, scenario_name, adopted_path)
		DO UPDATE SET version = EXCLUDED.version, status = EXCLUDED.status, updated_at = NOW()
		RETURNING id, component_id, component_library_id, scenario_name, adopted_path,
		          version, status, created_at, updated_at
	`,
		input.ComponentID, componentLibraryID, input.ScenarioName, input.AdoptedPath, version,
	).Scan(
		&a.ID, &a.ComponentID, &a.ComponentLibraryID, &a.ScenarioName,
		&a.AdoptedPath, &a.Version, &a.Status, &a.CreatedAt, &a.UpdatedAt,
	)

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create adoption", err)
		return
	}

	s.respondJSON(w, http.StatusCreated, a)
}
