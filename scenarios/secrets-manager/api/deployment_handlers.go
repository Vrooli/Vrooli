package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type DeploymentHandlers struct{}

func NewDeploymentHandlers() *DeploymentHandlers {
	return &DeploymentHandlers{}
}

func (s *APIServer) deploymentSecretsHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.deployment.DeploymentSecrets(w, r)
}

func (s *APIServer) deploymentSecretsGetHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.deployment.DeploymentSecretsGet(w, r)
}

// deploymentSecretsHandler generates deployment manifests for specific tiers
func (h *DeploymentHandlers) DeploymentSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	manifest, err := generateDeploymentManifest(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// deploymentSecretsGetHandler exposes a simple GET form for bundle consumers (scenario-to-*, deployment-manager).
// Example: GET /api/v1/deployment/secrets/picker-wheel?tier=tier-2-desktop&resources=postgres,redis&include_optional=false
func (h *DeploymentHandlers) DeploymentSecretsGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := strings.TrimSpace(vars["scenario"])
	if scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	tier := strings.TrimSpace(query.Get("tier"))
	if tier == "" {
		tier = "tier-2-desktop"
	}
	includeOptional := false
	if raw := query.Get("include_optional"); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			http.Error(w, "include_optional must be a boolean", http.StatusBadRequest)
			return
		}
		includeOptional = val
	}
	resources := []string{}
	if rawResources := query.Get("resources"); rawResources != "" {
		for _, r := range strings.Split(rawResources, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				resources = append(resources, r)
			}
		}
	}

	req := DeploymentManifestRequest{
		Scenario:        scenario,
		Tier:            tier,
		Resources:       resources,
		IncludeOptional: includeOptional,
	}

	manifest, err := generateDeploymentManifest(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}
