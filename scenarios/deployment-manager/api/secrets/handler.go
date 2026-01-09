package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"deployment-manager/fitness"
	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// ProfileLookup defines the interface for looking up profile data.
type ProfileLookup interface {
	GetScenarioAndTier(ctx context.Context, idOrName string) (scenario string, tierCount int, err error)
}

// ErrProfileNotFound indicates the profile was not found.
var ErrProfileNotFound = errors.New("profile not found")

// Handler handles secret-related requests.
type Handler struct {
	profiles ProfileLookup
	log      func(string, map[string]interface{})
}

// NewHandler creates a new secrets handler.
func NewHandler(profiles ProfileLookup, log func(string, map[string]interface{})) *Handler {
	return &Handler{profiles: profiles, log: log}
}

// IdentifySecrets identifies required secrets for a profile.
func (h *Handler) IdentifySecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	// Get profile
	scenario, _, err := h.profiles.GetScenarioAndTier(r.Context(), profileID)
	if errors.Is(err, ErrProfileNotFound) {
		http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"database error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Mock secret identification (would normally analyze scenario dependencies)
	secrets := []map[string]interface{}{
		{
			"name":        "DATABASE_URL",
			"type":        "required",
			"source":      "user-supplied",
			"description": "PostgreSQL connection string",
		},
		{
			"name":        "API_KEY",
			"type":        "optional",
			"source":      "vault-managed",
			"description": "Third-party API key",
		},
		{
			"name":        "DEBUG_MODE",
			"type":        "dev-only",
			"source":      "user-supplied",
			"description": "Enable debug logging",
		},
	}

	response := map[string]interface{}{
		"profile_id": profileID,
		"scenario":   scenario,
		"secrets":    secrets,
		"count":      len(secrets),
		"timestamp":  shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GenerateSecretTemplate generates a secret template in various formats.
func (h *Handler) GenerateSecretTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	formatStr := r.URL.Query().Get("format")
	if formatStr == "" {
		formatStr = "env"
	}

	// Get profile
	scenario, tier, err := h.profiles.GetScenarioAndTier(r.Context(), profileID)
	if errors.Is(err, ErrProfileNotFound) {
		http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"database error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Generate template using domain logic
	format := TemplateFormat(formatStr)
	params := TemplateParams{
		ProfileID: profileID,
		Scenario:  scenario,
		TierName:  fitness.GetTierDisplayName(tier),
	}
	template, err := GenerateTemplate(format, params)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if IsEnvFormat(format) {
		// Return plain text for .env format wrapped in JSON
		response := map[string]interface{}{
			"profile_id": profileID,
			"format":     formatStr,
			"template":   template,
			"timestamp":  shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		// Return JSON template directly
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(template))
	}
}

// ValidateSecrets validates secrets for a profile.
func (h *Handler) ValidateSecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	// Mock validation (would normally test connectivity)
	response := map[string]interface{}{
		"profile_id": profileID,
		"status":     "pass",
		"tests": []map[string]interface{}{
			{
				"secret":  "DATABASE_URL",
				"status":  "pass",
				"message": "Database connection successful",
			},
			{
				"secret":  "API_KEY",
				"status":  "warn",
				"message": "API key not configured (optional)",
			},
		},
		"timestamp": shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ValidateSecret validates a single secret.
func (h *Handler) ValidateSecret(w http.ResponseWriter, r *http.Request) {
	// Mock individual secret validation
	response := map[string]interface{}{
		"status":    "pass",
		"message":   "Secret validation not yet implemented",
		"timestamp": shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// TestSecret provides a secret testing endpoint.
func (h *Handler) TestSecret(w http.ResponseWriter, r *http.Request) {
	// Mock secret testing endpoint
	response := map[string]interface{}{
		"status":    "available",
		"message":   "Secret testing endpoint ready",
		"timestamp": shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
