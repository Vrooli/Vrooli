package main

// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/services/dochealth"
)

func (s *Server) handleDocsTree(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docExplorerService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation explorer service unavailable")
		return
	}
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	if scenarioName == "" {
		s.respondError(w, http.StatusBadRequest, "Scenario name is required")
		return
	}

	tree, err := s.docExplorerService.GetDocTree(r.Context(), scenarioName)
	if err != nil {
		respondDocExplorerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tree)
}

func respondDocExplorerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dochealth.ErrScenarioNameInvalid):
		respondWithError(w, http.StatusBadRequest, err)
	case errors.Is(err, dochealth.ErrScenarioNotFound):
		respondWithError(w, http.StatusNotFound, err)
	case errors.Is(err, dochealth.ErrScenarioRootInvalid):
		respondWithError(w, http.StatusServiceUnavailable, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
