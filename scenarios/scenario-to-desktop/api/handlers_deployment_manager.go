// Package main provides deployment-manager integration handlers.
// This file contains proxy handlers that coordinate with the deployment-manager
// scenario for bundle export and automated desktop builds.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/api-core/discovery"
)

// ============================================================================
// Types
// ============================================================================

type bundleExportRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier,omitempty"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

type bundleExportResponse struct {
	Status               string      `json:"status"`
	Schema               string      `json:"schema"`
	Manifest             interface{} `json:"manifest"`
	Checksum             string      `json:"checksum,omitempty"`
	GeneratedAt          string      `json:"generated_at,omitempty"`
	DeploymentManagerURL string      `json:"deployment_manager_url,omitempty"`
	ManifestPath         string      `json:"manifest_path,omitempty"`
}

type autoBuildProxyRequest struct {
	Scenario  string   `json:"scenario"`
	Platforms []string `json:"platforms,omitempty"`
	Targets   []string `json:"targets,omitempty"`
	DryRun    bool     `json:"dry_run,omitempty"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// exportBundleHandler proxies bundle export requests to deployment-manager.
func (s *Server) exportBundleHandler(w http.ResponseWriter, r *http.Request) {
	var req bundleExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}
	if containsParentRef(req.Scenario) || strings.ContainsAny(req.Scenario, `/\`) {
		http.Error(w, "invalid scenario name", http.StatusBadRequest)
		return
	}

	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(r.Context(), "deployment-manager")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve deployment-manager: %v", err), http.StatusBadGateway)
		return
	}

	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	if req.IncludeSecrets == nil {
		include := true
		req.IncludeSecrets = &include
	}

	client := &http.Client{Timeout: 5 * time.Minute}

	payload, err := json.Marshal(map[string]interface{}{
		"scenario":        req.Scenario,
		"tier":            req.Tier,
		"include_secrets": req.IncludeSecrets,
		"output_dir":      filepath.Join(detectVrooliRoot(), "scenarios", req.Scenario, "platforms", "electron", "bundle"),
		"stage_bundle":    true,
	})
	if err != nil {
		http.Error(w, "failed to marshal request", http.StatusInternalServerError)
		return
	}

	resp, err := client.Post(
		fmt.Sprintf("%s/api/v1/bundles/export", deploymentManagerURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("deployment-manager export failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read deployment-manager response", http.StatusBadGateway)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	var exportResp bundleExportResponse
	if err := json.Unmarshal(body, &exportResp); err != nil {
		http.Error(w, "failed to parse deployment-manager response", http.StatusBadGateway)
		return
	}
	exportResp.DeploymentManagerURL = deploymentManagerURL

	manifestPath := exportResp.ManifestPath
	if manifestPath == "" {
		var err error
		manifestPath, err = writeBundleManifest(req.Scenario, exportResp.Manifest)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to write bundle manifest: %v", err), http.StatusBadGateway)
			return
		}
	}
	exportResp.ManifestPath = manifestPath

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(exportResp)
}

// deploymentManagerAutoBuildHandler proxies auto-build requests to deployment-manager.
func (s *Server) deploymentManagerAutoBuildHandler(w http.ResponseWriter, r *http.Request) {
	var req autoBuildProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}
	if containsParentRef(req.Scenario) || strings.ContainsAny(req.Scenario, `/\`) {
		http.Error(w, "invalid scenario name", http.StatusBadRequest)
		return
	}

	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(r.Context(), "deployment-manager")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve deployment-manager: %v", err), http.StatusBadGateway)
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to marshal build request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("%s/api/v1/build/auto", deploymentManagerURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("deployment-manager build failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read deployment-manager response", http.StatusBadGateway)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// deploymentManagerAutoBuildStatusHandler proxies build status requests to deployment-manager.
func (s *Server) deploymentManagerAutoBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	buildID := mux.Vars(r)["build_id"]
	if buildID == "" {
		http.Error(w, "build_id is required", http.StatusBadRequest)
		return
	}

	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(r.Context(), "deployment-manager")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve deployment-manager: %v", err), http.StatusBadGateway)
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(
		fmt.Sprintf("%s/api/v1/build/auto/%s", deploymentManagerURL, buildID),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("deployment-manager status failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read deployment-manager response", http.StatusBadGateway)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// ============================================================================
// Helper Functions
// ============================================================================

// writeBundleManifest writes a bundle manifest to the scenario's electron platform directory.
func writeBundleManifest(scenario string, manifest interface{}) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest missing from deployment-manager response")
	}
	root := detectVrooliRoot()
	scenarioDir := filepath.Join(root, "scenarios", scenario)
	if _, err := os.Stat(scenarioDir); err != nil {
		return "", fmt.Errorf("scenario directory not found: %w", err)
	}

	outDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create bundle directory: %w", err)
	}

	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serialize manifest: %w", err)
	}

	outPath := filepath.Join(outDir, "bundle.json")
	if err := os.WriteFile(outPath, payload, 0o644); err != nil {
		return "", fmt.Errorf("write bundle.json: %w", err)
	}
	return outPath, nil
}
