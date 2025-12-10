package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"deployment-manager/codesigning"
	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// SigningValidator validates signing configuration and prerequisites.
type SigningValidator interface {
	// GetSigningConfig retrieves the signing config for a profile.
	GetSigningConfig(ctx context.Context, profileID string) (*codesigning.SigningConfig, error)
	// ValidateConfig checks structural validity of a SigningConfig.
	ValidateConfig(config *codesigning.SigningConfig) *codesigning.ValidationResult
	// CheckPrerequisites validates tools and certificates are available.
	CheckPrerequisites(ctx context.Context, config *codesigning.SigningConfig) *codesigning.ValidationResult
}

// Handler handles deployment requests.
type Handler struct {
	log              func(string, map[string]interface{})
	signingValidator SigningValidator
}

// NewHandler creates a new deployments handler.
func NewHandler(log func(string, map[string]interface{})) *Handler {
	return &Handler{log: log}
}

// NewHandlerWithSigning creates a deployments handler with signing validation support.
func NewHandlerWithSigning(log func(string, map[string]interface{}), signingValidator SigningValidator) *Handler {
	return &Handler{
		log:              log,
		signingValidator: signingValidator,
	}
}

// Deploy initiates a deployment for a profile.
// [REQ:DM-P0-028,DM-P0-029,DM-P0-033]
func (h *Handler) Deploy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["profile_id"]

	// [REQ:DM-P0-028] Validate profile using domain logic
	validation := ValidateProfile(profileID)
	if !validation.Valid {
		status, response := FormatValidationError(validation)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate signing prerequisites if signing validator is configured
	if h.signingValidator != nil {
		signingValidation := h.validateSigning(r.Context(), profileID)
		if !signingValidation.Valid {
			status, response := FormatValidationError(signingValidation)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	deploymentID := GenerateID()

	response := map[string]interface{}{
		"deployment_id": deploymentID,
		"profile_id":    profileID,
		"status":        "queued",
		"logs_url":      fmt.Sprintf("/api/v1/deployments/%s/logs", deploymentID),
		"message":       "Deployment orchestration not yet fully implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// validateSigning checks signing prerequisites for a profile's signing configuration.
// Returns a ValidationResult with errors if signing is enabled but prerequisites are missing.
func (h *Handler) validateSigning(ctx context.Context, profileID string) ValidationResult {
	result := ValidationResult{
		ProfileID: profileID,
		Valid:     true,
	}

	// Get the signing config for the profile
	config, err := h.signingValidator.GetSigningConfig(ctx, profileID)
	if err != nil {
		// If we can't get the config, log but don't block deployment
		// (profile might not have signing configured, which is valid)
		if h.log != nil {
			h.log("debug", map[string]interface{}{
				"msg":        "no signing config found for profile",
				"profile_id": profileID,
				"error":      err.Error(),
			})
		}
		return result
	}

	// If signing is disabled, no validation needed
	if config == nil || !config.Enabled {
		return result
	}

	// Validate structural configuration
	structuralResult := h.signingValidator.ValidateConfig(config)
	if !structuralResult.Valid {
		result.Valid = false
		for _, e := range structuralResult.Errors {
			result.Errors = append(result.Errors, ValidationError{
				Kind:    ErrorSigningValidation,
				Message: fmt.Sprintf("[%s] %s", e.Platform, e.Message),
			})
		}
		result.RecommendedFix = "Fix signing configuration errors before deploying"
		return result
	}

	// Check prerequisites (tools, certificates)
	prereqResult := h.signingValidator.CheckPrerequisites(ctx, config)
	if !prereqResult.Valid {
		result.Valid = false
		for _, e := range prereqResult.Errors {
			result.Errors = append(result.Errors, ValidationError{
				Kind:    ErrorSigningValidation,
				Message: fmt.Sprintf("[%s] %s", e.Platform, e.Message),
			})
		}
		result.RecommendedFix = buildSigningRemediation(prereqResult)

		// Log certificate expiry warnings
		for _, w := range prereqResult.Warnings {
			if h.log != nil {
				h.log("warning", map[string]interface{}{
					"msg":        "signing warning",
					"profile_id": profileID,
					"platform":   w.Platform,
					"warning":    w.Message,
				})
			}
		}
	}

	return result
}

// buildSigningRemediation creates actionable remediation text for signing errors.
func buildSigningRemediation(result *codesigning.ValidationResult) string {
	var remediation string
	for _, e := range result.Errors {
		if e.Remediation != "" {
			if remediation != "" {
				remediation += "; "
			}
			remediation += e.Remediation
		}
	}
	if remediation == "" {
		remediation = "Install required signing tools or configure valid certificates"
	}
	return remediation
}

// Status returns the status of a deployment.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["deployment_id"]

	response := map[string]interface{}{
		"id":           deploymentID,
		"status":       "queued",
		"started_at":   shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
		"completed_at": nil,
		"artifacts":    []string{},
		"message":      "Deployment status tracking not yet fully implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
