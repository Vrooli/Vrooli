package validation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"workflow-health/internal/execution"
	"workflow-health/internal/workflows"
)

type catalogResponse struct {
	Scenario string           `json:"scenario"`
	Journeys []catalogJourney `json:"journeys"`
}

type catalogJourney struct {
	JourneyID            string                  `json:"journey_id"`
	DisplayName          string                  `json:"display_name"`
	SourcePath           string                  `json:"source_path"`
	ExecutionMode        string                  `json:"execution_mode,omitempty"`
	Required             bool                    `json:"required"`
	Category             string                  `json:"category"`
	Requirements         []string                `json:"requirements,omitempty"`
	EstimatedDurationSec int                     `json:"estimated_duration_seconds,omitempty"`
	Safety               workflows.SafetyProfile `json:"safety"`
}

// CatalogHTTP exposes workflow-health's normalized provider catalog without
// exposing BAS internals or asking downstream consumers to parse workflow
// files. It is read-only; execution remains on the durable provider contract.
func (h *connectHandler) CatalogHTTP(w http.ResponseWriter, r *http.Request) {
	scenario := strings.TrimSpace(r.URL.Query().Get("scenario"))
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if scenario == "" && path == "" {
		writeCatalogError(w, http.StatusBadRequest, "scenario or path is required")
		return
	}
	report, err := h.run(r.Context(), scenario, path, execution.Options{})
	if err != nil {
		writeCatalogError(w, http.StatusBadRequest, err.Error())
		return
	}
	response := catalogResponse{Scenario: report.Scenario}
	if report.Catalog != nil {
		for _, asset := range report.Catalog.Cases {
			response.Journeys = append(response.Journeys, catalogJourney{
				JourneyID: asset.ID, DisplayName: firstCatalogName(asset.WorkflowAsset), SourcePath: asset.Path,
				ExecutionMode: firstCatalogExecutionMode(asset.WorkflowAsset), Required: strings.EqualFold(asset.Labels["required"], "true"),
				Category: catalogCategory(asset.Labels), Requirements: requirementIDs(asset.Requirements),
				EstimatedDurationSec: estimatedSeconds(asset.Labels), Safety: asset.Safety,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func catalogCategory(labels map[string]string) string {
	for _, key := range []string{"category", "validation_category", "suite"} {
		if category := strings.TrimSpace(labels[key]); category != "" {
			return category
		}
	}
	return "existing-bas-case"
}

func firstCatalogName(asset workflows.WorkflowAsset) string {
	if strings.TrimSpace(asset.Name) != "" {
		return asset.Name
	}
	return asset.ID
}

func firstCatalogExecutionMode(asset workflows.WorkflowAsset) string {
	if strings.TrimSpace(asset.ExecutionMode) != "" {
		return asset.ExecutionMode
	}
	return asset.Safety.ExecutionMode
}

func requirementIDs(requirements []workflows.RequirementLink) []string {
	ids := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		if strings.TrimSpace(requirement.ID) != "" {
			ids = append(ids, requirement.ID)
		}
	}
	return ids
}

func estimatedSeconds(labels map[string]string) int {
	for _, key := range []string{"estimated_duration_seconds", "duration_seconds"} {
		if value, err := strconv.Atoi(strings.TrimSpace(labels[key])); err == nil && value > 0 {
			return value
		}
	}
	return 0
}

func writeCatalogError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
