package main

// DOC: docs/reference/api-endpoints.md#documentation-health
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/docschema"
	"knowledge-observatory/internal/services/dochealth"
)

type ScenarioDocHealthResponse struct {
	ScenarioName  string                   `json:"scenario_name"`
	HealthScore   float64                  `json:"health_score"`
	TotalDocs     int                      `json:"total_docs"`
	MisplacedDocs []docschema.MisplacedDoc `json:"misplaced_docs"`
	MissingDocs   []string                 `json:"missing_docs"`
	ExtraDocs     []string                 `json:"extra_docs"`
	Warnings      []DocWarning             `json:"warnings"`
	CanAutoFix    bool                     `json:"can_auto_fix"`
}

type DocWarning struct {
	Type         string `json:"type"`
	Message      string `json:"message"`
	ExpectedPath string `json:"expected_path,omitempty"`
	Severity     string `json:"severity"`
}

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

func (s *Server) handleDocsHealth(w http.ResponseWriter, r *http.Request) {
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

	result, err := s.docHealthService.ValidateScenario(r.Context(), scenarioName)
	if err != nil {
		respondDocHealthError(w, err)
		return
	}

	missing := make([]string, 0, len(result.Validation.MissingDocs))
	for _, dt := range result.Validation.MissingDocs {
		missing = append(missing, string(dt))
	}

	warnings := buildDocWarnings(result.Validation)

	response := ScenarioDocHealthResponse{
		ScenarioName:  result.Validation.ScenarioName,
		HealthScore:   result.Validation.HealthScore,
		TotalDocs:     result.TotalDocs,
		MisplacedDocs: result.Validation.MisplacedDocs,
		MissingDocs:   missing,
		ExtraDocs:     result.Validation.ExtraDocs,
		Warnings:      warnings,
		CanAutoFix:    s.docHealingService != nil && result.Validation.HealthScore < 1,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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

	docType, err := docschema.ParseDocType(req.DocType)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if docType != docschema.DocTypeProblems && docType != docschema.DocTypeProgress {
		s.respondError(w, http.StatusBadRequest, "reset is only supported for problems and progress docs")
		return
	}

	result, err := s.docHealthService.ResetScenarioDoc(r.Context(), scenarioName, docschema.ResetConfig{
		DocType:        docType,
		MaxAgeDays:     req.MaxAgeDays,
		KeepMinEntries: req.KeepMinEntries,
		PreviewMode:    req.Preview,
	})
	if err != nil {
		respondDocHealthError(w, err)
		return
	}

	response := DocResetResponse{
		ScenarioName:   scenarioName,
		DocType:        string(docType),
		Preview:        req.Preview,
		RemovedCount:   result.RemovedCount,
		KeptCount:      result.KeptCount,
		RemovedEntries: result.RemovedEntries,
		NewContent:     result.NewContent,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func buildDocWarnings(result *docschema.ValidationResult) []DocWarning {
	warnings := make([]DocWarning, 0, len(result.MisplacedDocs)+len(result.MissingDocs)+len(result.ExtraDocs))
	for _, misplaced := range result.MisplacedDocs {
		warnings = append(warnings, DocWarning{
			Type:         "misplaced",
			Message:      "Documentation file is in the wrong location",
			ExpectedPath: misplaced.ExpectedPath,
			Severity:     misplaced.Severity,
		})
	}
	for _, missing := range result.MissingDocs {
		warnings = append(warnings, DocWarning{
			Type:     "missing",
			Message:  "Documentation file is missing",
			Severity: missingDocSeverity(missing),
		})
	}
	for _, extra := range result.ExtraDocs {
		warnings = append(warnings, DocWarning{
			Type:     "extra",
			Message:  fmt.Sprintf("Documentation file is outside the standard layout: %s", extra),
			Severity: "info",
		})
	}
	return warnings
}

func missingDocSeverity(docType docschema.DocType) string {
	if docType == docschema.DocTypeReadme {
		return "error"
	}
	return "warning"
}

func respondDocHealthError(w http.ResponseWriter, err error) {
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

func respondWithError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
