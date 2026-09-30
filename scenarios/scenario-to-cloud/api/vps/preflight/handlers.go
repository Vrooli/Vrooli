// DOC: docs/reference/api-endpoints.md#preflight — preflight and fix endpoint documentation
package preflight

import (
	"context"
	"net/http"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/reach"
)

// HandlerDeps contains dependencies for preflight handlers.
type HandlerDeps struct {
	// Reach is the transport every host observation goes through. Preflight
	// runs before a deployment record exists, so the target is derived from
	// the manifest locator and reached with the operator's ambient identity
	// (or the enrolled Bridge node).
	Reach      reach.Reach
	DNSService dns.Service
	// ValidateManifest is a function that validates and normalizes a manifest.
	// Returns normalized manifest and validation issues.
	ValidateManifest func(manifest domain.CloudManifest) (domain.CloudManifest, []domain.ValidationIssue)
	// HasBlockingIssues checks if issues contain blocking errors.
	HasBlockingIssues func(issues []domain.ValidationIssue) bool
}

// HandleRequirements returns canonical VPS requirement metadata.
// GET /api/v1/preflight/requirements
func HandleRequirements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, BuildRequirementsResponse())
	}
}

// HandlePreflight creates a handler for running VPS preflight checks.
// POST /api/v1/preflight
func HandlePreflight(deps HandlerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manifest, err := httputil.DecodeJSON[domain.CloudManifest](r.Body, 1<<20)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Request body must be valid JSON",
				Hint:    err.Error(),
			})
			return
		}

		normalized, issues := deps.ValidateManifest(manifest)
		if deps.HasBlockingIssues(issues) {
			httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
				Valid:     false,
				Issues:    issues,
				Manifest:  normalized,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		resp := Run(ctx, normalized, deps.DNSService, deps.Reach, domain.TargetRefFromManifest(normalized), RunOptions{})
		resp.Issues = issues
		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

// HostRequest names a target that has no deployment record yet: the locator
// only. There is no key reference on the wire; the transport authenticates
// with the credential binding of a bound deployment or the operator's
// ambient identity.
type HostRequest struct {
	Host string `json:"host"`
	Port int    `json:"port,omitempty"`
	User string `json:"user,omitempty"`
}

// FirewallFixRequest is the request body for opening firewall ports via UFW.
type FirewallFixRequest struct {
	HostRequest
	Ports []int `json:"ports,omitempty"`
}

// FirewallFixResponse is the response from opening firewall ports.
type FirewallFixResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Ports     []int  `json:"ports"`
	Status    string `json:"status,omitempty"`
	Timestamp string `json:"timestamp"`
}

// StopScenarioProcessesRequest is the request body for stopping stale scenario processes.
type StopScenarioProcessesRequest struct {
	HostRequest
	Workdir    string `json:"workdir"`
	ScenarioID string `json:"scenario_id,omitempty"` // If empty, stops all vrooli processes
}

// StopScenarioProcessesResponse is the response from stopping scenario processes.
type StopScenarioProcessesResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"` // "stop_scenario" or "stop_all"
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// writeStopScenarioResponse is a helper to reduce response construction boilerplate.
func writeStopScenarioResponse(w http.ResponseWriter, ok bool, action, message, output string) {
	httputil.WriteJSON(w, http.StatusOK, StopScenarioProcessesResponse{
		OK:        ok,
		Action:    action,
		Message:   message,
		Output:    output,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
