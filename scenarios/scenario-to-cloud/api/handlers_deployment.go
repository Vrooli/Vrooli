package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
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
			ID:              d.ID,
			Name:            d.Name,
			ScenarioID:      d.ScenarioID,
			Status:          d.Status,
			ErrorMessage:    d.ErrorMessage,
			ProgressStep:    d.ProgressStep,
			ProgressPercent: d.ProgressPercent,
			CreatedAt:       d.CreatedAt,
			LastDeployedAt:  d.LastDeployedAt,
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

		// Update bundle info if provided
		if req.BundlePath != "" {
			existing.BundlePath = &req.BundlePath
			existing.BundleSHA256 = &req.BundleSHA256
			existing.BundleSizeBytes = &req.BundleSizeBytes
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

	// Store bundle info if provided
	if req.BundlePath != "" {
		deployment.BundlePath = &req.BundlePath
		deployment.BundleSHA256 = &req.BundleSHA256
		deployment.BundleSizeBytes = &req.BundleSizeBytes
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
	_, deployment := s.FetchDeploymentOnly(w, r)
	if deployment == nil {
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment": deployment,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleDeleteDeployment removes a deployment record, optionally stopping it first and cleaning up bundles.
func (s *Server) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	stopOnVPS := r.URL.Query().Get("stop") == "true"
	cleanupBundles := r.URL.Query().Get("cleanup") == "true"

	id, deployment := s.FetchDeploymentOnly(w, r)
	if deployment == nil {
		return
	}

	// Optionally stop the deployment on VPS first
	var manifest CloudManifest
	if stopOnVPS || cleanupBundles {
		if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
			s.log("failed to unmarshal manifest", map[string]interface{}{
				"deployment_id": id,
				"error":         err.Error(),
			})
		}
	}

	if stopOnVPS && manifest.Target.VPS != nil {
		stopResult := s.stopDeploymentOnVPS(r.Context(), manifest)
		if !stopResult.OK {
			// Log the error but continue with deletion
			s.log("failed to stop deployment on VPS", map[string]interface{}{
				"deployment_id": id,
				"error":         stopResult.Error,
			})
		}
	}

	// Optionally clean up bundle files (local + VPS)
	if cleanupBundles && deployment.BundleSHA256 != nil && *deployment.BundleSHA256 != "" {
		// Check if other deployments use this bundle
		refCount, err := s.repo.CountDeploymentsByBundleSHA256(r.Context(), *deployment.BundleSHA256)
		if err != nil {
			s.log("failed to check bundle references", map[string]interface{}{
				"sha256": *deployment.BundleSHA256,
				"error":  err.Error(),
			})
		} else if refCount <= 1 {
			// Safe to delete - only this deployment uses this bundle
			s.cleanupDeploymentBundles(r.Context(), deployment, manifest)
		} else {
			s.log("skipping bundle cleanup - bundle used by other deployments", map[string]interface{}{
				"sha256":    *deployment.BundleSHA256,
				"ref_count": refCount,
			})
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

// cleanupDeploymentBundles removes bundle files for a deployment (local + VPS).
func (s *Server) cleanupDeploymentBundles(ctx context.Context, deployment *domain.Deployment, manifest CloudManifest) {
	// 1. Delete local bundle
	if deployment.BundleSHA256 != nil && *deployment.BundleSHA256 != "" {
		repoRoot, err := FindRepoRootFromCWD()
		if err == nil {
			bundlesDir := repoRoot + "/scenarios/scenario-to-cloud/coverage/bundles"
			freedBytes, err := DeleteBundle(bundlesDir, *deployment.BundleSHA256)
			if err != nil {
				s.log("failed to delete local bundle", map[string]interface{}{
					"sha256": *deployment.BundleSHA256,
					"error":  err.Error(),
				})
			} else if freedBytes > 0 {
				s.log("deleted local bundle", map[string]interface{}{
					"sha256":      *deployment.BundleSHA256,
					"freed_bytes": freedBytes,
				})
			}
		}
	}

	// 2. Delete VPS bundle
	if manifest.Target.VPS != nil && deployment.BundlePath != nil && *deployment.BundlePath != "" {
		cfg := sshConfigFromManifest(manifest)
		workdir := manifest.Target.VPS.Workdir
		bundleFilename := filepath.Base(*deployment.BundlePath)
		remoteBundlePath := safeRemoteJoin(workdir, ".vrooli/cloud/bundles", bundleFilename)

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		cmd := fmt.Sprintf("rm -f %s", shellQuoteSingle(remoteBundlePath))
		if _, err := s.sshRunner.Run(ctx, cfg, cmd); err != nil {
			s.log("failed to delete VPS bundle", map[string]interface{}{
				"path":  remoteBundlePath,
				"error": err.Error(),
			})
		} else {
			s.log("deleted VPS bundle", map[string]interface{}{
				"path": remoteBundlePath,
			})
		}
	}
}

// handleExecuteDeployment starts the deployment pipeline in the background.
// Returns immediately with the deployment info. Clients can subscribe to
// /deployments/{id}/progress for real-time progress updates via SSE.
//
// Idempotency: Uses atomic status transition via StartDeploymentRun to prevent
// duplicate concurrent executions. The run_id allows tracking which execution
// produced which results.
func (s *Server) handleExecuteDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Parse optional request body for provided secrets
	var req domain.ExecuteDeploymentRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Non-fatal: if body can't be parsed, just use empty secrets
			s.log("failed to parse execute request body", map[string]interface{}{"error": err.Error()})
		}
	}

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

	// Parse manifest before attempting to start (fail fast on invalid manifest)
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

	// Generate a unique run_id for this execution
	runID := uuid.New().String()

	// Atomic status transition: prevents race conditions where two requests
	// could both pass a status check and start duplicate deployments.
	// StartDeploymentRun only succeeds if status is NOT already running.
	if err := s.repo.StartDeploymentRun(r.Context(), id, runID); err != nil {
		writeAPIError(w, http.StatusConflict, APIError{
			Code:    "already_running",
			Message: "Deployment is already in progress or not found",
			Hint:    err.Error(),
		})
		return
	}

	s.log("deployment run started", map[string]interface{}{
		"deployment_id": id,
		"run_id":        runID,
	})

	// Start deployment in background with the run_id for tracking
	go s.runDeploymentPipeline(id, runID, normalized, deployment.BundlePath, req.ProvidedSecrets)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"deployment": deployment,
		"run_id":     runID,
		"message":    "Deployment started. Subscribe to /deployments/{id}/progress for real-time updates.",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// runDeploymentPipeline executes the full deployment with progress tracking.
// The runID parameter uniquely identifies this execution for idempotency tracking.
func (s *Server) runDeploymentPipeline(id, runID string, manifest CloudManifest, existingBundlePath *string, providedSecrets map[string]string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	progress := 0.0

	// Helper to emit and persist progress
	emitProgress := func(eventType, step, stepTitle string, pct float64, errMsg string) {
		event := ProgressEvent{
			Type:      eventType,
			Step:      step,
			StepTitle: stepTitle,
			Progress:  pct,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		if errMsg != "" {
			event.Error = errMsg
		}

		// Broadcast to SSE clients
		s.progressHub.Broadcast(id, event)

		// Persist to database for reconnection
		if err := s.repo.UpdateDeploymentProgress(ctx, id, step, pct); err != nil {
			s.log("failed to persist progress", map[string]interface{}{"error": err.Error()})
		}
	}

	// Helper for errors (used for bundle_build step which isn't in VPS runners)
	emitError := func(step, stepTitle, errMsg string) {
		event := ProgressEvent{
			Type:      "deployment_error",
			Step:      step,
			StepTitle: stepTitle,
			Progress:  progress,
			Error:     errMsg,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		s.progressHub.Broadcast(id, event)
	}

	// Log the run_id for traceability
	s.log("deployment pipeline started", map[string]interface{}{
		"deployment_id": id,
		"run_id":        runID,
		"scenario_id":   manifest.Scenario.ID,
	})

	// Fetch and validate secrets
	if err := s.ensureSecretsAvailable(ctx, &manifest, providedSecrets, id, emitError); err != nil {
		return // Error already logged and emitted
	}

	// Step 1: Build bundle (if not already built)
	emitProgress("step_started", "bundle_build", "Building bundle", progress, "")

	bundlePath, err := s.ensureBundleBuilt(ctx, manifest, existingBundlePath, id, emitError)
	if err != nil {
		return // Error already logged and emitted
	}

	progress += StepWeights["bundle_build"]
	emitProgress("step_completed", "bundle_build", "Building bundle", progress, "")

	// Step 2: VPS Setup
	if err := s.repo.UpdateDeploymentStatus(ctx, id, domain.StatusSetupRunning, nil, nil); err != nil {
		s.log("failed to update status", map[string]interface{}{"error": err.Error()})
	}

	setupResult := RunVPSSetupWithProgress(ctx, manifest, bundlePath, s.sshRunner, s.scpRunner, s.progressHub, s.repo, id, &progress)
	setupJSON, _ := json.Marshal(setupResult)
	if err := s.repo.UpdateDeploymentSetupResult(ctx, id, setupJSON); err != nil {
		s.log("failed to save setup result", map[string]interface{}{"error": err.Error()})
	}

	if !setupResult.OK {
		// VPS runner already emitted deployment_error event with correct step
		failedStep := setupResult.FailedStep
		if failedStep == "" {
			failedStep = "vps_setup"
		}
		setDeploymentError(s.repo, ctx, id, failedStep, setupResult.Error)
		return
	}

	// Step 3: VPS Deploy
	if err := s.repo.UpdateDeploymentStatus(ctx, id, domain.StatusDeploying, nil, nil); err != nil {
		s.log("failed to update status", map[string]interface{}{"error": err.Error()})
	}

	deployResult := RunVPSDeployWithProgress(ctx, manifest, s.sshRunner, s.secretsGenerator, s.progressHub, s.repo, id, &progress)
	deployJSON, _ := json.Marshal(deployResult)
	if err := s.repo.UpdateDeploymentDeployResult(ctx, id, deployJSON, deployResult.OK); err != nil {
		s.log("failed to save deploy result", map[string]interface{}{"error": err.Error()})
	}

	if !deployResult.OK {
		// VPS runner already emitted deployment_error event with correct step
		failedStep := deployResult.FailedStep
		if failedStep == "" {
			failedStep = "vps_deploy"
		}
		setDeploymentError(s.repo, ctx, id, failedStep, deployResult.Error)
		return
	}

	// Success!
	s.progressHub.Broadcast(id, ProgressEvent{
		Type:      "completed",
		Progress:  100,
		Message:   "Deployment successful",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleInspectDeployment fetches status and logs from the deployed VPS.
func (s *Server) handleInspectDeployment(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	opts := VPSInspectOptions{TailLines: 200}
	result := RunVPSInspect(ctx, dctx.Manifest, opts, s.sshRunner)

	// Store the inspect result
	if resultJSON, err := json.Marshal(result); err == nil {
		if err := s.repo.UpdateDeploymentInspectResult(ctx, dctx.ID, resultJSON); err != nil {
			s.log("failed to save inspect result", map[string]interface{}{"error": err.Error()})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleStopDeployment stops the scenario on the VPS.
func (s *Server) handleStopDeployment(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}

	result := s.stopDeploymentOnVPS(r.Context(), dctx.Manifest)

	if result.OK {
		if err := s.repo.UpdateDeploymentStatus(r.Context(), dctx.ID, domain.StatusStopped, nil, nil); err != nil {
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

	cmd := vrooliCommand(workdir, fmt.Sprintf("vrooli scenario stop %s", shellQuoteSingle(normalized.Scenario.ID)))

	_, err := s.sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return VPSDeployResult{OK: true, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// setDeploymentError is a helper to set error status on a deployment.
func setDeploymentError(repo interface {
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMsg, errorStep *string) error
}, ctx context.Context, id, step, errMsg string,
) {
	_ = repo.UpdateDeploymentStatus(ctx, id, domain.StatusFailed, &errMsg, &step)
}

// ensureSecretsAvailable fetches secrets from secrets-manager and validates user_prompt secrets.
// Returns an error if secrets cannot be fetched or validated (error already logged and emitted).
func (s *Server) ensureSecretsAvailable(
	ctx context.Context,
	manifest *CloudManifest,
	providedSecrets map[string]string,
	deploymentID string,
	emitError func(step, stepTitle, errMsg string),
) error {
	// Fetch secrets from secrets-manager BEFORE building bundle
	if manifest.Secrets == nil {
		secretsCtx, secretsCancel := context.WithTimeout(ctx, 30*time.Second)
		secretsResp, err := s.secretsFetcher.FetchBundleSecrets(
			secretsCtx,
			manifest.Scenario.ID,
			DefaultDeploymentTier,
			manifest.Dependencies.Resources,
		)
		secretsCancel()

		if err != nil {
			s.log("secrets-manager fetch failed", map[string]interface{}{
				"scenario_id": manifest.Scenario.ID,
				"error":       err.Error(),
			})
			errMsg := fmt.Sprintf("secrets-manager unavailable: %v", err)
			setDeploymentError(s.repo, ctx, deploymentID, "secrets_fetch", errMsg)
			emitError("secrets_fetch", "Fetching secrets", err.Error())
			return err
		}

		manifest.Secrets = BuildManifestSecrets(secretsResp)
		s.log("fetched secrets manifest", map[string]interface{}{
			"scenario_id":   manifest.Scenario.ID,
			"total_secrets": len(secretsResp.BundleSecrets),
		})
	}

	// Validate user_prompt secrets
	if providedSecrets == nil {
		providedSecrets = make(map[string]string)
	}
	if missing, err := ValidateUserPromptSecrets(*manifest, providedSecrets); err != nil {
		s.log("missing required user_prompt secrets", map[string]interface{}{
			"scenario_id": manifest.Scenario.ID,
			"missing":     missing,
		})
		setDeploymentError(s.repo, ctx, deploymentID, "secrets_validate", err.Error())
		emitError("secrets_validate", "Validating secrets", err.Error())
		return err
	}

	return nil
}

// ensureBundleBuilt builds a new bundle or returns the existing bundle path.
// Returns the bundle path or an error (error already logged and emitted).
func (s *Server) ensureBundleBuilt(
	ctx context.Context,
	manifest CloudManifest,
	existingBundlePath *string,
	deploymentID string,
	emitError func(step, stepTitle, errMsg string),
) (string, error) {
	// Use existing bundle if provided
	if existingBundlePath != nil && *existingBundlePath != "" {
		return *existingBundlePath, nil
	}

	// Get bundle output directory
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		setDeploymentError(s.repo, ctx, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building bundle", err.Error())
		return "", err
	}

	outDir, err := GetLocalBundlesDir()
	if err != nil {
		setDeploymentError(s.repo, ctx, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building bundle", err.Error())
		return "", err
	}

	// Clean up old bundles (keep 3 newest)
	s.cleanupOldBundles(outDir, manifest.Scenario.ID)

	// Build the bundle
	artifact, err := BuildMiniVrooliBundle(repoRoot, outDir, manifest)
	if err != nil {
		setDeploymentError(s.repo, ctx, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building bundle", err.Error())
		return "", err
	}

	// Update database with bundle info
	if err := s.repo.UpdateDeploymentBundle(ctx, deploymentID, artifact.Path, artifact.Sha256, artifact.SizeBytes); err != nil {
		s.log("failed to update bundle info", map[string]interface{}{"error": err.Error()})
	}

	return artifact.Path, nil
}

// cleanupOldBundles removes old bundles for a scenario, keeping the newest N.
func (s *Server) cleanupOldBundles(bundlesDir, scenarioID string) {
	const retentionCount = 3
	deleted, _, err := DeleteBundlesForScenario(bundlesDir, scenarioID, retentionCount)
	if err != nil {
		s.log("bundle cleanup warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		return
	}
	if len(deleted) > 0 {
		s.log("cleaned old bundles", map[string]interface{}{
			"scenario_id": scenarioID,
			"count":       len(deleted),
		})
	}
}
