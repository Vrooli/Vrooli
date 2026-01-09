// Package main provides HTTP handlers for deployment manifest generation.
//
// These handlers expose the deployment manifest API consumed by:
//   - scenario-to-desktop for bundle packaging
//   - deployment-manager for deployment orchestration
//   - UI clients for deployment readiness visualization
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// DeploymentHandlers provides HTTP handlers for deployment manifest operations.
type DeploymentHandlers struct {
	builder *ManifestBuilder
}

// NewDeploymentHandlers creates deployment handlers with the given ManifestBuilder.
func NewDeploymentHandlers(builder *ManifestBuilder) *DeploymentHandlers {
	return &DeploymentHandlers{builder: builder}
}

// RegisterRoutes mounts deployment manifest endpoints on the router.
func (h *DeploymentHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/secrets", h.DeploymentSecrets).Methods("POST")
	router.HandleFunc("/secrets/{scenario}", h.DeploymentSecretsGet).Methods("GET")
	router.HandleFunc("/readiness", h.DeploymentReadiness).Methods("POST")
}

// DeploymentSecrets generates a deployment manifest from a POST request body.
//
// Request body:
//
//	{
//	  "scenario": "picker-wheel",
//	  "tier": "tier-2-desktop",
//	  "resources": ["postgres", "redis"],  // optional filter
//	  "include_optional": false            // optional, default false
//	}
//
// Response: DeploymentManifest JSON
func (h *DeploymentHandlers) DeploymentSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	manifest, err := h.builder.Build(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// DeploymentSecretsGet generates a deployment manifest from query parameters.
// This provides a simpler interface for bundle consumers.
//
// URL: GET /api/v1/deployment/secrets/{scenario}
//
// Query parameters:
//   - tier: deployment tier (default: "tier-2-desktop")
//   - resources: comma-separated resource filter (optional)
//   - include_optional: boolean (default: false)
//
// Example: GET /api/v1/deployment/secrets/picker-wheel?tier=tier-2-desktop&resources=postgres,redis
func (h *DeploymentHandlers) DeploymentSecretsGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := strings.TrimSpace(vars["scenario"])
	if scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()

	// Parse tier with default
	tier := strings.TrimSpace(query.Get("tier"))
	if tier == "" {
		tier = "tier-2-desktop"
	}

	// Parse include_optional flag
	includeOptional := false
	if raw := query.Get("include_optional"); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			http.Error(w, "include_optional must be a boolean", http.StatusBadRequest)
			return
		}
		includeOptional = val
	}

	// Parse resources filter
	var resources []string
	if rawResources := query.Get("resources"); rawResources != "" {
		for _, res := range strings.Split(rawResources, ",") {
			res = strings.TrimSpace(res)
			if res != "" {
				resources = append(resources, res)
			}
		}
	}

	req := DeploymentManifestRequest{
		Scenario:        scenario,
		Tier:            tier,
		Resources:       resources,
		IncludeOptional: includeOptional,
	}

	manifest, err := h.builder.Build(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

// DeploymentReadiness returns a lightweight readiness snapshot.
// This provides only the summary without the full manifest payload,
// useful for quick health checks and UI status displays.
//
// Request body: same as DeploymentSecrets
//
// Response:
//
//	{
//	  "scenario": "picker-wheel",
//	  "tier": "tier-2-desktop",
//	  "resources": [...],
//	  "summary": {...},
//	  "generated_at": "2024-01-15T10:30:00Z"
//	}
func (h *DeploymentHandlers) DeploymentReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	manifest, err := h.builder.Build(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate readiness snapshot: %v", err), http.StatusBadRequest)
		return
	}

	resp := struct {
		Scenario    string            `json:"scenario"`
		Tier        string            `json:"tier"`
		Resources   []string          `json:"resources"`
		Summary     DeploymentSummary `json:"summary"`
		GeneratedAt time.Time         `json:"generated_at"`
	}{
		Scenario:    manifest.Scenario,
		Tier:        manifest.Tier,
		Resources:   manifest.Resources,
		Summary:     manifest.Summary,
		GeneratedAt: manifest.GeneratedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
