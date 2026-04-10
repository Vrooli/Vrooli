package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
	"lifestyle-dashboard/repository"
)

// GetScoreConfig returns the score configuration for all domains.
// GET /api/v1/score/config
// [REQ:LD-SCORE-CALC] Retrieves configurable domain weights.
func (h *Handler) GetScoreConfig(w http.ResponseWriter, r *http.Request) {
	if h.ScoreConfig == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "score config service not configured"))
		return
	}

	weights, err := h.ScoreConfig.GetWeights(r.Context())
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to get score configuration"))
		return
	}

	WriteJSON(w, http.StatusOK, domain.ScoreConfigResponse{
		Weights:       weights,
		DefaultWeight: "medium",
	})
}

// GetDomainWeight returns the weight configuration for a specific domain.
// GET /api/v1/score/config/{domain}
// [REQ:LD-SCORE-CALC] Retrieves single domain weight.
func (h *Handler) GetDomainWeight(w http.ResponseWriter, r *http.Request) {
	if h.ScoreConfig == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "score config service not configured"))
		return
	}

	// Extract domain from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/score/config/")
	domainName := strings.TrimSuffix(path, "/")

	if domainName == "" {
		WriteAPIError(w, errors.NewValidationError(errors.CodeMissingField, "Domain name is required").WithDetails("field", "domain"))
		return
	}

	weight, err := h.ScoreConfig.GetWeight(r.Context(), domainName)
	if repository.IsNotFound(err) {
		WriteAPIError(w, errors.NewNotFoundError(errors.CodeDomainNotFound, "domain", domainName))
		return
	}
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to get domain weight"))
		return
	}

	WriteJSON(w, http.StatusOK, weight)
}

// UpdateDomainWeight updates the weight for a specific domain.
// PUT /api/v1/score/config/{domain}
// [REQ:LD-SCORE-CALC] Updates domain weight.
func (h *Handler) UpdateDomainWeight(w http.ResponseWriter, r *http.Request) {
	if h.ScoreConfig == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "score config service not configured"))
		return
	}

	// Extract domain from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/score/config/")
	domainName := strings.TrimSuffix(path, "/")

	if domainName == "" {
		WriteAPIError(w, errors.NewValidationError(errors.CodeMissingField, "Domain name is required").WithDetails("field", "domain"))
		return
	}

	var req domain.UpdateWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, errors.ErrInvalidJSON)
		return
	}

	// Validate weight value
	validWeights := []string{"high", "medium", "low", "none"}
	isValid := false
	for _, valid := range validWeights {
		if req.Weight == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		WriteAPIError(w, errors.NewValidationError(errors.CodeInvalidField, "Invalid weight value: "+req.Weight).WithDetails("valid_values", validWeights))
		return
	}

	err := h.ScoreConfig.SetWeight(r.Context(), domainName, req.Weight)
	if repository.IsNotFound(err) {
		WriteAPIError(w, errors.NewNotFoundError(errors.CodeDomainNotFound, "domain", domainName))
		return
	}
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to update domain weight"))
		return
	}

	// Return updated weight
	weight, err := h.ScoreConfig.GetWeight(r.Context(), domainName)
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Weight updated but failed to retrieve"))
		return
	}

	WriteJSON(w, http.StatusOK, weight)
}
