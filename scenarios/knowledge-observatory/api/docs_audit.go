package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Server) handleDocsAudit(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docHealthService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation health service unavailable")
		return
	}
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	if scenarioName == "" {
		s.respondError(w, http.StatusBadRequest, "Scenario name is required")
		return
	}

	result, err := s.docHealthService.AuditScenario(r.Context(), scenarioName)
	if err != nil {
		respondDocHealthError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
