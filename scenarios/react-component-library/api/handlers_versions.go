package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// handleGetComponentVersions retrieves all versions of a component
func (s *Server) handleGetComponentVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentID := vars["id"]

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, component_id, version, content, changelog, created_at
		FROM component_versions
		WHERE component_id = $1
		ORDER BY created_at DESC
	`, componentID)

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query versions", err)
		return
	}
	defer rows.Close()

	versions := []ComponentVersion{}
	for rows.Next() {
		var v ComponentVersion
		err := rows.Scan(&v.ID, &v.ComponentID, &v.Version, &v.Content, &v.Changelog, &v.CreatedAt)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to scan version", err)
			return
		}
		versions = append(versions, v)
	}

	// If no versions exist, return empty array instead of null
	if versions == nil {
		versions = []ComponentVersion{}
	}

	s.respondJSON(w, http.StatusOK, versions)
}
