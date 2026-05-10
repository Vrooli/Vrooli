package main

// DOC: docs/reference/api-endpoints.md#documentation-health
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/doclogs"
	"knowledge-observatory/internal/docvalidation"
	"knowledge-observatory/internal/services/dochealth"
)

type ScenarioDocHealthResponse struct {
	ScenarioName     string                       `json:"scenario_name"`
	SourceTemplateID string                       `json:"source_template_id"`
	ManifestPath     string                       `json:"manifest_path"`
	ManifestStatus   string                       `json:"manifest_status"`
	HealthScore      float64                      `json:"health_score"`
	TotalDocs        int                          `json:"total_docs"`
	MisplacedDocs    []docvalidation.MisplacedDoc `json:"misplaced_docs"`
	MissingDocs      []string                     `json:"missing_docs"`
	ExtraDocs        []string                     `json:"extra_docs"`
	TemporaryDocs    []string                     `json:"temporary_docs"`
	Warnings         []DocWarning                 `json:"warnings"`
	ContractFindings []DocWarning                 `json:"contract_findings,omitempty"`
	ContentIssues    []DocWarning                 `json:"content_issues,omitempty"`
	CanAutoFix       bool                         `json:"can_auto_fix"`
	FixCategory      string                       `json:"fix_category"`
}

type DocWarning struct {
	Type         string `json:"type"`
	Message      string `json:"message"`
	ExpectedPath string `json:"expected_path,omitempty"`
	Path         string `json:"path,omitempty"`
	DocType      string `json:"doc_type,omitempty"`
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
		if strings.Contains(err.Error(), "reset is not supported") {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondDocHealthError(w, err)
		return
	}

	missing := append([]string{}, result.Validation.MissingDocs...)

	warnings := buildDocWarnings(result.Validation)

	fixCategory := computeFixCategory(result.Validation.MisplacedDocs, missing, result.Validation.ExtraDocs, result.Validation.TemporaryDocs)

	response := ScenarioDocHealthResponse{
		ScenarioName:     result.Validation.ScenarioName,
		SourceTemplateID: result.Validation.SourceTemplateID,
		ManifestPath:     result.Validation.ManifestPath,
		ManifestStatus:   result.Validation.ManifestStatus,
		HealthScore:      result.Validation.HealthScore,
		TotalDocs:        result.TotalDocs,
		MisplacedDocs:    result.Validation.MisplacedDocs,
		MissingDocs:      missing,
		ExtraDocs:        result.Validation.ExtraDocs,
		TemporaryDocs:    result.Validation.TemporaryDocs,
		Warnings:         warnings,
		ContractFindings: buildContractWarnings(result.Validation),
		ContentIssues:    buildContentWarnings(result.Validation),
		CanAutoFix:       s.docHealingService != nil && result.Validation.HealthScore < 1,
		FixCategory:      fixCategory,
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

func buildDocWarnings(result *docvalidation.Result) []DocWarning {
	warnings := make([]DocWarning, 0, len(result.MisplacedDocs)+len(result.MissingDocs)+len(result.ExtraDocs))
	for _, misplaced := range result.MisplacedDocs {
		warnings = append(warnings, DocWarning{
			Type:         "misplaced",
			Message:      "Documentation file is in the wrong location",
			ExpectedPath: misplaced.ExpectedPath,
			Severity:     misplaced.Severity,
		})
	}
	missingDetails := map[string]docvalidation.MissingDoc{}
	for _, missing := range result.MissingDocDetails {
		missingDetails[missing.DocType] = missing
	}
	for _, missing := range result.MissingDocs {
		detail := missingDetails[missing]
		severity := detail.Severity
		if severity == "" {
			severity = "warning"
		}
		warnings = append(warnings, DocWarning{
			Type:     "missing",
			Message:  "Documentation file is missing",
			Path:     detail.Path,
			DocType:  missing,
			Severity: severity,
		})
	}
	for _, extra := range result.ExtraDocs {
		warnings = append(warnings, DocWarning{
			Type:     "extra",
			Message:  fmt.Sprintf("Documentation file is not registered in the documentation contract: %s", extra),
			Severity: "info",
		})
	}
	for _, temporary := range result.TemporaryDocs {
		warnings = append(warnings, DocWarning{
			Type:     "temporary",
			Message:  fmt.Sprintf("Temporary documentation artifact should be cleaned up: %s", temporary),
			Severity: "warning",
		})
	}
	return warnings
}

func buildContractWarnings(result *docvalidation.Result) []DocWarning {
	warnings := make([]DocWarning, 0, len(result.ContractFindings))
	for _, finding := range result.ContractFindings {
		warnings = append(warnings, DocWarning{
			Type:     finding.Code,
			Message:  finding.Message,
			Path:     finding.Path,
			DocType:  finding.DocType,
			Severity: finding.Severity,
		})
	}
	return warnings
}

func buildContentWarnings(result *docvalidation.Result) []DocWarning {
	warnings := make([]DocWarning, 0, len(result.ContentIssues))
	for _, issue := range result.ContentIssues {
		warnings = append(warnings, DocWarning{
			Type:     "content",
			Message:  issue.Message,
			Path:     issue.Path,
			DocType:  issue.DocType,
			Severity: issue.Severity,
		})
	}
	return warnings
}

func computeFixCategory(misplaced []docvalidation.MisplacedDoc, missing []string, extra []string, temporary []string) string {
	hasMisplaced := len(misplaced) > 0
	hasAgentIssues := len(missing) > 0 || len(extra) > 0 || len(temporary) > 0
	switch {
	case hasMisplaced && !hasAgentIssues:
		return "all_auto"
	case !hasMisplaced && hasAgentIssues:
		return "all_agent"
	case hasMisplaced && hasAgentIssues:
		return "mixed"
	default:
		return "none"
	}
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
