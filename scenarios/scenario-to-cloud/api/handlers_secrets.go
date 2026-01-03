package main

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// handleGetSecrets proxies to secrets-manager to fetch deployment secrets for a scenario.
// This provides the UI with information about what secrets are needed before deployment.
//
// GET /api/v1/secrets/{scenario}?tier={tier}&resources={resources}
//
// Query parameters:
//   - tier: deployment tier (default: tier-4-saas)
//   - resources: comma-separated list of resources to include
//
// Response:
//
//	{
//	  "secrets": {
//	    "bundle_secrets": [...],
//	    "summary": {...}
//	  }
//	}
func (s *Server) handleGetSecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := strings.TrimSpace(vars["scenario"])
	if scenario == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_scenario",
			Message: "Scenario ID is required",
		})
		return
	}

	// Parse query parameters
	tier := strings.TrimSpace(r.URL.Query().Get("tier"))
	if tier == "" {
		tier = DefaultDeploymentTier
	}

	resourcesParam := strings.TrimSpace(r.URL.Query().Get("resources"))
	var resources []string
	if resourcesParam != "" {
		for _, res := range strings.Split(resourcesParam, ",") {
			res = strings.TrimSpace(res)
			if res != "" {
				resources = append(resources, res)
			}
		}
	}

	// Fetch from secrets-manager (uses dynamic service discovery)
	resp, err := s.secretsFetcher.FetchBundleSecrets(r.Context(), scenario, tier, resources)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, APIError{
			Code:    "secrets_fetch_failed",
			Message: "Failed to fetch secrets from secrets-manager",
			Hint:    err.Error(),
		})
		return
	}

	// Build manifest secrets from response
	secrets := BuildManifestSecrets(resp)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"secrets": secrets,
	})
}
