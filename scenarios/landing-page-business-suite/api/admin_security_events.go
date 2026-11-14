package main

import (
	"net/http"

	"landing-page-business-suite-api/internal/adminsecurity"
)

func (s *Server) adminSecurityEvents(w http.ResponseWriter, r *http.Request) {
	email, ok := s.sessionAdminEmail(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	}
	var store adminsecurity.Store = s.db
	if s.routedDB != nil {
		store = s.routedDB
	}
	events, err := (&adminsecurity.Repository{DB: store}).List(r.Context(), email, 50)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Unable to load security events.", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{"events": events})
}
