package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// handleIdentifySecrets identifies required secrets for a profile.
func (s *Server) handleIdentifySecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	// Get profile
	scenario, _, err := s.profiles.GetScenarioAndTier(r.Context(), profileID)
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
		"timestamp":  GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGenerateSecretTemplate generates a secret template in various formats.
func (s *Server) handleGenerateSecretTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	formatStr := r.URL.Query().Get("format")
	if formatStr == "" {
		formatStr = "env"
	}

	// Get profile
	scenario, tier, err := s.profiles.GetScenarioAndTier(r.Context(), profileID)
	if errors.Is(err, ErrProfileNotFound) {
		http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"database error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Generate template using domain logic (extracted to domain_templates.go)
	format := SecretTemplateFormat(formatStr)
	params := SecretTemplateParams{
		ProfileID: profileID,
		Scenario:  scenario,
		TierName:  getTierName(tier),
	}
	template, err := GenerateSecretTemplate(format, params)
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
			"timestamp":  GetTimeProvider().Now().UTC().Format(time.RFC3339),
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

// handleValidateSecrets validates secrets for a profile.
func (s *Server) handleValidateSecrets(w http.ResponseWriter, r *http.Request) {
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
		"timestamp": GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleValidateSecret validates a single secret.
func (s *Server) handleValidateSecret(w http.ResponseWriter, r *http.Request) {
	// Mock individual secret validation
	response := map[string]interface{}{
		"status":    "pass",
		"message":   "Secret validation not yet implemented",
		"timestamp": GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleTestSecret provides a secret testing endpoint.
func (s *Server) handleTestSecret(w http.ResponseWriter, r *http.Request) {
	// Mock secret testing endpoint
	response := map[string]interface{}{
		"status":    "available",
		"message":   "Secret testing endpoint ready",
		"timestamp": GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
