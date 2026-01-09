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

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/vps"
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
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
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
			var manifest domain.CloudManifest
			if err := json.Unmarshal(d.Manifest, &manifest); err == nil {
				summary.Domain = manifest.Edge.Domain
				if manifest.Target.VPS != nil {
					summary.Host = manifest.Target.VPS.Host
				}
			}
		}
		summaries[i] = summary
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"deployments": summaries,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCreateDeployment creates a new deployment record from a manifest.
func (s *Server) handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[domain.CreateDeploymentRequest](r.Body, 2<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Parse and validate the manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_manifest",
			Message: "Manifest is not valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := manifest.ValidateAndNormalize(m)
	if manifest.HasBlockingIssues(issues) {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
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
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
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
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
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
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "update_failed",
				Message: "Failed to update existing deployment",
				Hint:    err.Error(),
			})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "create_failed",
			Message: "Failed to create deployment",
			Hint:    err.Error(),
		})
		return
	}

	s.appendHistoryEvent(r.Context(), deployment.ID, domain.HistoryEvent{
		Type:      domain.EventDeploymentCreated,
		Timestamp: time.Now().UTC(),
		Message:   "Deployment created",
		Success:   boolPtr(true),
	})

	httputil.WriteJSON(w, http.StatusCreated, map[string]interface{}{
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

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
	var manifest domain.CloudManifest
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
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "delete_failed",
			Message: "Failed to delete deployment",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"deleted":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// cleanupDeploymentBundles removes bundle files for a deployment (local + VPS).
func (s *Server) cleanupDeploymentBundles(ctx context.Context, deployment *domain.Deployment, manifest domain.CloudManifest) {
	// 1. Delete local bundle
	if deployment.BundleSHA256 != nil && *deployment.BundleSHA256 != "" {
		bundlesDir, err := bundle.GetLocalBundlesDir()
		if err == nil {
			freedBytes, err := bundle.DeleteBundle(bundlesDir, *deployment.BundleSHA256)
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
		cfg := ssh.ConfigFromManifest(manifest)
		workdir := manifest.Target.VPS.Workdir
		bundleFilename := filepath.Base(*deployment.BundlePath)
		remoteBundlePath := ssh.SafeRemoteJoin(workdir, ".vrooli/cloud/bundles", bundleFilename)

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		cmd := fmt.Sprintf("rm -f %s", ssh.QuoteSingle(remoteBundlePath))
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

	dep, err := s.repo.GetDeployment(r.Context(), id)
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

	// Parse manifest before attempting to start (fail fast on invalid manifest)
	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := manifest.ValidateAndNormalize(m)
	if manifest.HasBlockingIssues(issues) {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
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
		httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{
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
	options := deployment.ExecuteOptions{
		RunPreflight:     req.RunPreflight,
		ForceBundleBuild: req.ForceBundleBuild,
	}
	go s.orchestrator.RunPipeline(id, runID, normalized, dep.BundlePath, req.ProvidedSecrets, options)

	httputil.WriteJSON(w, http.StatusAccepted, map[string]interface{}{
		"deployment": dep,
		"run_id":     runID,
		"message":    "Deployment started. Subscribe to /deployments/{id}/progress for real-time updates.",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
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

	opts := vps.InspectOptions{TailLines: 200}
	result := vps.RunInspect(ctx, dctx.Manifest, opts, s.sshRunner)

	s.appendHistoryEvent(ctx, dctx.ID, domain.HistoryEvent{
		Type:      domain.EventInspection,
		Timestamp: time.Now().UTC(),
		Message:   "Inspection completed",
		Details:   result.Error,
		Success:   boolPtr(result.OK),
	})

	// Store the inspect result
	if resultJSON, err := json.Marshal(result); err == nil {
		if err := s.repo.UpdateDeploymentInspectResult(ctx, dctx.ID, resultJSON); err != nil {
			s.log("failed to save inspect result", map[string]interface{}{"error": err.Error()})
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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

	s.appendHistoryEvent(r.Context(), dctx.ID, domain.HistoryEvent{
		Type:      domain.EventStopped,
		Timestamp: time.Now().UTC(),
		Message:   "Deployment stop requested",
		Details:   result.Error,
		Success:   boolPtr(result.OK),
	})

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":   result.OK,
		"error":     result.Error,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// stopDeploymentOnVPS runs the stop command on the remote VPS.
func (s *Server) stopDeploymentOnVPS(ctx context.Context, m domain.CloudManifest) domain.VPSDeployResult {
	normalized, _ := manifest.ValidateAndNormalize(m)
	cfg := ssh.ConfigFromManifest(normalized)
	workdir := normalized.Target.VPS.Workdir

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario stop %s", ssh.QuoteSingle(normalized.Scenario.ID)))

	_, err := s.sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return domain.VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return domain.VPSDeployResult{OK: true, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// appendHistoryEvent persists a history event and logs failures without impacting the request.
// This is used by handlers for non-pipeline events (create, inspect, stop).
func (s *Server) appendHistoryEvent(ctx context.Context, deploymentID string, event domain.HistoryEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	recorder := s.historyRecorder
	if recorder == nil {
		recorder = s.repo
	}
	if err := recorder.AppendHistoryEvent(ctx, deploymentID, event); err != nil {
		s.log("failed to append history event", map[string]interface{}{
			"deployment_id": deploymentID,
			"type":          event.Type,
			"error":         err.Error(),
		})
	}
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
