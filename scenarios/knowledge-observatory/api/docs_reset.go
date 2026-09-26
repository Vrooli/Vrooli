package main

// DOC: docs/reference/api-endpoints.md#documentation-reset
import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/doclogs"
)

type DocResetRequest struct {
	DocType        string `json:"doc_type"`
	MaxAgeDays     int    `json:"max_age_days,omitempty"`
	KeepMinEntries int    `json:"keep_min_entries,omitempty"`
	Preview        bool   `json:"preview,omitempty"`
}

type DocResetResponse struct {
	ScenarioName   string   `json:"scenario_name"`
	DocType        string   `json:"doc_type"`
	Preview        bool     `json:"preview"`
	RemovedCount   int      `json:"removed_count"`
	KeptCount      int      `json:"kept_count"`
	RemovedEntries []string `json:"removed_entries,omitempty"`
	NewContent     string   `json:"new_content"`
}

func (s *Server) handleDocsReset(w http.ResponseWriter, r *http.Request) {
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

	var req DocResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.MaxAgeDays < 0 || req.KeepMinEntries < 0 {
		s.respondError(w, http.StatusBadRequest, "max_age_days and keep_min_entries must be non-negative")
		return
	}

	result, docType, err := s.docHealthService.ResetScenarioDoc(r.Context(), scenarioName, req.DocType, doclogs.ResetConfig{
		MaxAgeDays:     req.MaxAgeDays,
		KeepMinEntries: req.KeepMinEntries,
		PreviewMode:    req.Preview,
	})
	if err != nil {
		if strings.Contains(err.Error(), "reset is not supported") {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondDocHealthError(w, err)
		return
	}

	s.logDocAccess(r.Context(), scenarioName, docType, "reset")

	response := DocResetResponse{
		ScenarioName:   scenarioName,
		DocType:        docType,
		Preview:        req.Preview,
		RemovedCount:   result.RemovedCount,
		KeptCount:      result.KeptCount,
		RemovedEntries: result.RemovedEntries,
		NewContent:     result.NewContent,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
