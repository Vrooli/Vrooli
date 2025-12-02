package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// mergeSecretsRequest is the request body for merge-secrets endpoint.
type mergeSecretsRequest struct {
	Scenario string                 `json:"scenario"`
	Tier     string                 `json:"tier"`
	Manifest desktopBundleManifest  `json:"manifest"`
	Raw      map[string]interface{} `json:"-"`
}

// assembleBundleRequest is the request body for assemble endpoint.
type assembleBundleRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

// handleValidateBundle validates a desktop bundle manifest.
func (s *Server) handleValidateBundle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to read bundle: %v"}`, err), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, `{"error":"bundle manifest required"}`, http.StatusBadRequest)
		return
	}

	if err := validateDesktopBundleManifestBytes(body); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"bundle failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "valid",
		"schema": "desktop.v0.1",
	})
}

// handleMergeBundleSecrets merges secret plans into a bundle manifest.
func (s *Server) handleMergeBundleSecrets(w http.ResponseWriter, r *http.Request) {
	var req mergeSecretsRequest
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
	if err := validateDesktopBundleManifestBytes(rawPayload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	secretPlans, err := s.fetchBundleSecrets(r.Context(), req.Scenario, req.Tier)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
		return
	}

	manifest := req.Manifest
	if err := applyBundleSecrets(&manifest, secretPlans); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}

// handleAssembleBundle assembles a complete bundle manifest for a scenario.
func (s *Server) handleAssembleBundle(w http.ResponseWriter, r *http.Request) {
	var req assembleBundleRequest
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

	manifest, err := s.fetchSkeletonBundle(ctx, req.Scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to build bundle","details":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	if includeSecrets {
		secretPlans, err := s.fetchBundleSecrets(ctx, req.Scenario, req.Tier)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
			return
		}
		if err := applyBundleSecrets(manifest, secretPlans); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Validate assembled manifest to guarantee schema compliance before handing off.
	payload, _ := json.Marshal(manifest)
	if err := validateDesktopBundleManifestBytes(payload); err != nil {
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
