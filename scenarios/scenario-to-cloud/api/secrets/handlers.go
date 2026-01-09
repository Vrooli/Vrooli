package secrets

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"scenario-to-cloud/internal/httputil"
)

// HandleGetSecrets returns an HTTP handler that proxies to secrets-manager
// to fetch deployment secrets for a scenario.
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
func HandleGetSecrets(fetcher Fetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		scenario := strings.TrimSpace(vars["scenario"])
		if scenario == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
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
		resp, err := fetcher.FetchBundleSecrets(r.Context(), scenario, tier, resources)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadGateway, httputil.APIError{
				Code:    "secrets_fetch_failed",
				Message: "Failed to fetch secrets from secrets-manager",
				Hint:    err.Error(),
			})
			return
		}

		// Build manifest secrets from response
		secrets := BuildManifestSecrets(resp)

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"secrets": secrets,
		})
	}
}
