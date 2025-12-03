package bundles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"deployment-manager/secrets"
)

// MergeSecretsRequest is the request body for merge-secrets endpoint.
type MergeSecretsRequest struct {
	Scenario string                 `json:"scenario"`
	Tier     string                 `json:"tier"`
	Manifest Manifest               `json:"manifest"`
	Raw      map[string]interface{} `json:"-"`
}

// AssembleBundleRequest is the request body for assemble endpoint.
type AssembleBundleRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

// Handler handles bundle-related requests.
type Handler struct {
	secretsClient *secrets.Client
	log           func(string, map[string]interface{})
}

// NewHandler creates a new bundles handler.
func NewHandler(secretsClient *secrets.Client, log func(string, map[string]interface{})) *Handler {
	return &Handler{secretsClient: secretsClient, log: log}
}

// ValidateBundle validates a desktop bundle manifest.
func (h *Handler) ValidateBundle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to read bundle: %v"}`, err), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, `{"error":"bundle manifest required"}`, http.StatusBadRequest)
		return
	}

	if err := ValidateManifestBytes(body); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"bundle failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "valid",
		"schema": "desktop.v0.1",
	})
}

// MergeBundleSecrets merges secret plans into a bundle manifest.
func (h *Handler) MergeBundleSecrets(w http.ResponseWriter, r *http.Request) {
	var req MergeSecretsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}

	// Re-validate manifest before merging.
	rawPayload, err := json.Marshal(req.Manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to marshal manifest: %v"}`, err), http.StatusBadRequest)
		return
	}
	if err := ValidateManifestBytes(rawPayload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	secretPlans, err := h.secretsClient.FetchBundleSecrets(r.Context(), req.Scenario, req.Tier)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
		return
	}

	manifest := req.Manifest
	if err := ApplyBundleSecrets(&manifest, secretPlans); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}

// AssembleBundle assembles a complete bundle manifest for a scenario.
func (h *Handler) AssembleBundle(w http.ResponseWriter, r *http.Request) {
	var req AssembleBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Scenario) == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	includeSecrets := true
	if req.IncludeSecrets != nil {
		includeSecrets = *req.IncludeSecrets
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	manifest, err := FetchSkeletonBundle(ctx, req.Scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to build bundle","details":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	if includeSecrets {
		secretPlans, err := h.secretsClient.FetchBundleSecrets(ctx, req.Scenario, req.Tier)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
			return
		}
		if err := ApplyBundleSecrets(manifest, secretPlans); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Validate assembled manifest to guarantee schema compliance before handing off.
	payload, _ := json.Marshal(manifest)
	if err := ValidateManifestBytes(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"assembled manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "assembled",
		"schema":   "desktop.v0.1",
		"manifest": manifest,
	})
}
