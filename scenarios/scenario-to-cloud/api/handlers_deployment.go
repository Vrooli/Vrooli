package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
)

// handleListDeployments returns all deployment records.
func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	filter := domain.ListFilter{}

	// Parse optional query params
	if status := r.URL.Query().Get("status"); status != "" {
		st := domain.DeploymentStatus(status)
		filter.Status = &st
	}
	if scenarioID := r.URL.Query().Get("scenario_id"); scenarioID != "" {
		filter.ScenarioID = &scenarioID
	}

	deployments, err := s.repo.ListDeployments(r.Context(), filter)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "list_deployments_failed",
			Message: "Failed to list deployments",
			Hint:    err.Error(),
		})
		return
	}

	// Convert to summaries for list view
	summaries := make([]domain.DeploymentSummary, len(deployments))
	for i, d := range deployments {
		summary := domain.DeploymentSummary{
			ID:             d.ID,
			Name:           d.Name,
			ScenarioID:     d.ScenarioID,
			Status:         d.Status,
			ErrorMessage:   d.ErrorMessage,
			CreatedAt:      d.CreatedAt,
			LastDeployedAt: d.LastDeployedAt,
		}
		// Extract domain and host from manifest
		if d.Manifest != nil {
			var manifest CloudManifest
			if err := json.Unmarshal(d.Manifest, &manifest); err == nil {
				summary.Domain = manifest.Edge.Domain
				if manifest.Target.VPS != nil {
					summary.Host = manifest.Target.VPS.Host
				}
			}
		}
		summaries[i] = summary
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployments": summaries,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCreateDeployment creates a new deployment record from a manifest.
func (s *Server) handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[domain.CreateDeploymentRequest](r.Body, 2<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Parse and validate the manifest
	var manifest CloudManifest
	if err := json.Unmarshal(req.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_manifest",
			Message: "Manifest is not valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if hasBlockingIssues(issues) {
		writeJSON(w, http.StatusUnprocessableEntity, ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Re-marshal the normalized manifest
	manifestJSON, err := json.Marshal(normalized)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "marshal_failed",
			Message: "Failed to marshal normalized manifest",
			Hint:    err.Error(),
		})
		return
	}

	// Check for existing deployment with same host+scenario (for update-in-place)
	var host string
	if normalized.Target.VPS != nil {
		host = normalized.Target.VPS.Host
	}
	existing, err := s.repo.GetDeploymentByHostAndScenario(r.Context(), host, normalized.Scenario.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "lookup_failed",
			Message: "Failed to check for existing deployment",
			Hint:    err.Error(),
		})
		return
	}

	if existing != nil {
		// Update existing deployment in-place
		existing.Manifest = manifestJSON
		existing.Status = domain.StatusPending
		existing.ErrorMessage = nil
		existing.ErrorStep = nil
		existing.UpdatedAt = time.Now()

		// Update name if provided
		if req.Name != "" {
			existing.Name = req.Name
		}

		if err := s.repo.UpdateDeployment(r.Context(), existing); err != nil {
			writeAPIError(w, http.StatusInternalServerError, APIError{
				Code:    "update_failed",
				Message: "Failed to update existing deployment",
				Hint:    err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"deployment": existing,
			"updated":    true,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Generate name if not provided
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("%s @ %s", normalized.Scenario.ID, normalized.Edge.Domain)
	}

	// Create new deployment
	now := time.Now()
	deployment := &domain.Deployment{
		ID:         uuid.New().String(),
		Name:       name,
		ScenarioID: normalized.Scenario.ID,
		Status:     domain.StatusPending,
		Manifest:   manifestJSON,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.CreateDeployment(r.Context(), deployment); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "create_failed",
			Message: "Failed to create deployment",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"deployment": deployment,
		"created":    true,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetDeployment returns a single deployment by ID.
func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment": deployment,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleDeleteDeployment removes a deployment record, optionally stopping it first.
func (s *Server) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	stopOnVPS := r.URL.Query().Get("stop") == "true"

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Optionally stop the deployment on VPS first
	if stopOnVPS {
		var manifest CloudManifest
		if err := json.Unmarshal(deployment.Manifest, &manifest); err == nil {
			stopResult := s.stopDeploymentOnVPS(r.Context(), manifest)
			if !stopResult.OK {
				// Log the error but continue with deletion
				s.log("failed to stop deployment on VPS", map[string]interface{}{
					"deployment_id": id,
					"error":         stopResult.Error,
				})
			}
		}
	}

	if err := s.repo.DeleteDeployment(r.Context(), id); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "delete_failed",
			Message: "Failed to delete deployment",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleExecuteDeployment runs the full deployment pipeline: bundle build, VPS setup, VPS deploy.
func (s *Server) handleExecuteDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if hasBlockingIssues(issues) {
		writeJSON(w, http.StatusUnprocessableEntity, ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	// Step 1: Build bundle (if not already built)
	var bundlePath string
	if deployment.BundlePath != nil && *deployment.BundlePath != "" {
		bundlePath = *deployment.BundlePath
	} else {
		repoRoot, err := FindRepoRootFromCWD()
		if err != nil {
			setDeploymentError(s.repo, ctx, id, "bundle_build", err.Error())
			writeAPIError(w, http.StatusInternalServerError, APIError{
				Code:    "repo_root_not_found",
				Message: "Unable to locate Vrooli repo root",
				Hint:    err.Error(),
			})
			return
		}

		outDir := repoRoot + "/scenarios/scenario-to-cloud/coverage/bundles"
		artifact, err := BuildMiniVrooliBundle(repoRoot, outDir, normalized)
		if err != nil {
			setDeploymentError(s.repo, ctx, id, "bundle_build", err.Error())
			writeAPIError(w, http.StatusInternalServerError, APIError{
				Code:    "bundle_build_failed",
				Message: "Failed to build bundle",
				Hint:    err.Error(),
			})
			return
		}

		bundlePath = artifact.Path
		if err := s.repo.UpdateDeploymentBundle(ctx, id, artifact.Path, artifact.Sha256, artifact.SizeBytes); err != nil {
			s.log("failed to update bundle info", map[string]interface{}{"error": err.Error()})
		}
	}

	// Step 2: VPS Setup
	if err := s.repo.UpdateDeploymentStatus(ctx, id, domain.StatusSetupRunning, nil, nil); err != nil {
		s.log("failed to update status", map[string]interface{}{"error": err.Error()})
	}

	setupResult := RunVPSSetup(ctx, normalized, bundlePath, ExecSSHRunner{}, ExecSCPRunner{})
	setupJSON, _ := json.Marshal(setupResult)
	if err := s.repo.UpdateDeploymentSetupResult(ctx, id, setupJSON); err != nil {
		s.log("failed to save setup result", map[string]interface{}{"error": err.Error()})
	}

	if !setupResult.OK {
		setDeploymentError(s.repo, ctx, id, "vps_setup", setupResult.Error)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"deployment": deployment,
			"step":       "setup",
			"success":    false,
			"error":      setupResult.Error,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Step 3: VPS Deploy
	if err := s.repo.UpdateDeploymentStatus(ctx, id, domain.StatusDeploying, nil, nil); err != nil {
		s.log("failed to update status", map[string]interface{}{"error": err.Error()})
	}

	deployResult := RunVPSDeploy(ctx, normalized, ExecSSHRunner{})
	deployJSON, _ := json.Marshal(deployResult)
	if err := s.repo.UpdateDeploymentDeployResult(ctx, id, deployJSON, deployResult.OK); err != nil {
		s.log("failed to save deploy result", map[string]interface{}{"error": err.Error()})
	}

	if !deployResult.OK {
		setDeploymentError(s.repo, ctx, id, "vps_deploy", deployResult.Error)
	}

	// Fetch updated deployment
	deployment, _ = s.repo.GetDeployment(ctx, id)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment": deployment,
		"success":    deployResult.OK,
		"error":      deployResult.Error,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleInspectDeployment fetches status and logs from the deployed VPS.
func (s *Server) handleInspectDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	opts := VPSInspectOptions{
		TailLines: 200,
	}

	result := RunVPSInspect(ctx, normalized, opts, ExecSSHRunner{})
	resultJSON, _ := json.Marshal(result)

	// Store the inspect result
	if err := s.repo.UpdateDeploymentInspectResult(ctx, id, resultJSON); err != nil {
		s.log("failed to save inspect result", map[string]interface{}{"error": err.Error()})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleStopDeployment stops the scenario on the VPS.
func (s *Server) handleStopDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	result := s.stopDeploymentOnVPS(r.Context(), manifest)

	if result.OK {
		if err := s.repo.UpdateDeploymentStatus(r.Context(), id, domain.StatusStopped, nil, nil); err != nil {
			s.log("failed to update status", map[string]interface{}{"error": err.Error()})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   result.OK,
		"error":     result.Error,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// stopDeploymentOnVPS runs the stop command on the remote VPS.
func (s *Server) stopDeploymentOnVPS(ctx context.Context, manifest CloudManifest) VPSDeployResult {
	normalized, _ := ValidateAndNormalizeManifest(manifest)
	cfg := sshConfigFromManifest(normalized)
	workdir := normalized.Target.VPS.Workdir

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	sshRunner := ExecSSHRunner{}
	cmd := fmt.Sprintf("cd %s && vrooli scenario stop %s", shellQuoteSingle(workdir), shellQuoteSingle(normalized.Scenario.ID))

	_, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return VPSDeployResult{OK: true, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// setDeploymentError is a helper to set error status on a deployment.
func setDeploymentError(repo interface {
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMsg, errorStep *string) error
}, ctx context.Context, id, step, errMsg string) {
	_ = repo.UpdateDeploymentStatus(ctx, id, domain.StatusFailed, &errMsg, &step)
}
