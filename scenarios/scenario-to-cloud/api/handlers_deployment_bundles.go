package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/ssh"

	"github.com/gorilla/mux"
)

// handleListDeploymentVPSBundles lists VPS bundle cache entries for a deployment's target.
// GET /api/v1/deployments/{id}/bundles/vps
func (s *Server) handleListDeploymentVPSBundles(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if s.deploymentRepo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "repo_not_configured",
			Message: "Deployment repository is not configured",
		})
		return
	}
	dep, err := s.deploymentRepo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}
	if dep == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}
	if dep.Manifest == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_manifest",
			Message: "Deployment has no stored manifest",
		})
		return
	}

	var manifest domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &manifest); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "invalid_manifest",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	if manifest.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "not_vps",
			Message: "Deployment target is not a VPS",
		})
		return
	}

	cfg := ssh.ConfigFromManifest(manifest)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if !bundle.ValidateVPSBundleRequest(w, cfg.Host, cfg.KeyPath, manifest.Target.VPS.Workdir) {
		return
	}

	bundles, totalSize, err := bundle.ListVPSBundles(ctx, s.sshRunner, cfg, manifest.Target.VPS.Workdir)
	if err != nil {
		httputil.WriteJSON(w, http.StatusBadGateway, domain.VPSBundleListResponse{
			OK:        false,
			Bundles:   []domain.VPSBundleInfo{},
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, domain.VPSBundleListResponse{
		OK:             true,
		Bundles:        bundles,
		TotalSizeBytes: totalSize,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGCDeploymentVPSBundles garbage-collects VPS bundle cache entries for a deployment's target.
// POST /api/v1/deployments/{id}/bundles/vps/gc
func (s *Server) handleGCDeploymentVPSBundles(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if s.deploymentRepo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "repo_not_configured",
			Message: "Deployment repository is not configured",
		})
		return
	}
	dep, err := s.deploymentRepo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}
	if dep == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}
	if dep.Manifest == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_manifest",
			Message: "Deployment has no stored manifest",
		})
		return
	}

	var manifest domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &manifest); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "invalid_manifest",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}
	if manifest.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "not_vps",
			Message: "Deployment target is not a VPS",
		})
		return
	}

	var req domain.VPSBundleGCRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	if req.KeepLatest <= 0 {
		req.KeepLatest = bundle.DefaultVPSBundleKeepLatest
	}
	if req.ScenarioID == "" {
		req.ScenarioID = manifest.Scenario.ID
	}
	// Protect the bundle currently associated with this deployment (if recorded).
	if dep.BundleSHA256 != nil && *dep.BundleSHA256 != "" {
		req.ProtectSHA256 = append(req.ProtectSHA256, *dep.BundleSHA256)
	}

	cfg := ssh.ConfigFromManifest(manifest)
	if !bundle.ValidateVPSBundleRequest(w, cfg.Host, cfg.KeyPath, manifest.Target.VPS.Workdir) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	resp := bundle.GCVPSBundles(ctx, s.sshRunner, cfg, manifest.Target.VPS.Workdir, req)
	httputil.WriteJSON(w, http.StatusOK, resp)
}
