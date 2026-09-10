package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/authz"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/vps"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

	if environment := r.URL.Query().Get("environment"); environment != "" {
		filter.Environment = &environment
	}

	deployments, err := s.repo.ListDeployments(r.Context(), filter)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to list deployments", err))
		return
	}

	// Convert to summaries for list view
	summaries := make([]domain.DeploymentSummary, len(deployments))
	for i, d := range deployments {
		summary := domain.DeploymentSummary{
			ID:              d.ID,
			Name:            d.Name,
			ScenarioID:      d.ScenarioID,
			Environment:     d.Environment,
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
			}
		}
		summary.Host = d.Target.Locator.Host
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
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidJSON, "Request body must be valid JSON").WithDetail("cause", err.Error()))
		return
	}

	// Parse and validate the manifest
	var rawManifest map[string]interface{}
	if err := json.Unmarshal(req.Manifest, &rawManifest); err != nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeManifestInvalid, "Manifest is not valid JSON").WithDetail("cause", err.Error()))
		return
	}

	normalized, issues, err := manifest.ValidateRaw(rawManifest)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to validate manifest", err))
		return
	}
	if manifest.HasBlockingIssues(issues) {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if repoRoot, err := bundle.FindRepoRootFromCWD(); err == nil {
		if version, _, versionErr := resolveScenarioVersion(repoRoot, normalized.Scenario.ID); versionErr == nil && version != "" {
			normalized.Scenario.Ref = version
		}
	}

	// Re-marshal the normalized manifest
	manifestJSON, err := json.Marshal(normalized)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to marshal normalized manifest", err))
		return
	}

	// A manifest names a scenario, an environment and a target. If that
	// identity already exists the record is updated in place; the deployment
	// ID is stable across manifest edits.
	environment := identity.NormalizeEnvironment(normalized.Environment)
	target := domain.TargetRefFromManifest(normalized)
	existing, err := s.findDeploymentForManifest(r.Context(), normalized.Scenario.ID, environment, target)
	if err != nil {
		apierrors.Write(w, err)
		return
	}

	if existing != nil {
		// Update existing deployment in-place
		existing.Manifest = manifestJSON
		existing.Environment = environment
		existing.Target = target
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
			apierrors.Write(w, err)
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
		ID:          uuid.New().String(),
		Name:        name,
		ScenarioID:  normalized.Scenario.ID,
		Environment: environment,
		Target:      target,
		Status:      domain.StatusPending,
		Manifest:    manifestJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store bundle info if provided
	if req.BundlePath != "" {
		deployment.BundlePath = &req.BundlePath
		deployment.BundleSHA256 = &req.BundleSHA256
		deployment.BundleSizeBytes = &req.BundleSizeBytes
	}

	if err := s.repo.CreateDeployment(r.Context(), deployment); err != nil {
		apierrors.Write(w, err)
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

// handleGetDeploymentReceipt returns the durable owner receipt for a
// successfully deployed VPS target. A deployment record alone is not proof
// of an external effect, so incomplete or failed records refuse the receipt.
// GET /deployments/{id}/receipt
func (s *Server) handleGetDeploymentReceipt(w http.ResponseWriter, r *http.Request) {
	_, deployment := s.FetchDeploymentOnly(w, r)
	if deployment == nil {
		return
	}
	receipt, err := s.signedDeploymentReceipt(r.Context(), deployment)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{
			Code:    "receipt_unavailable",
			Message: "Cloud deployment receipt is unavailable",
			Hint:    err.Error(),
		})
		return
	}
	// The caller states what it expects; a receipt for another release,
	// bundle or target is refused here rather than handed out to be
	// misread. The signature is verified before the receipt leaves the owner.
	query := r.URL.Query()
	expect := domain.ReceiptExpectation{DeploymentID: deployment.ID, ReleaseDigest: query.Get("release_digest"), BundleSHA256: query.Get("bundle_sha256"), TargetKey: query.Get("target_key")}
	if err := receipt.Check(expect); err != nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeReceiptInvalid, "Cloud deployment receipt does not match the requested identity").WithDetail("cause", err.Error()))
		return
	}
	if err := receipt.VerifySignature(r.Context(), s.publication().signer); err != nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeReceiptInvalid, "Cloud deployment receipt attestation is invalid").WithDetail("cause", err.Error()))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"receipt":     receipt,
		"attestation": map[string]any{"producer_ref": receipt.ProducerRef, "production_signer": s.publication().signerProduction},
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// handleRecoverDeployment executes an owner-routed recovery action and
// returns a receipt only after the owner observes the requested effect.
// Halt is the currently supported irreversible-safe action. Other recovery
// modes refuse explicitly until their owner contracts can prove compatibility.
// POST /deployments/{id}/recovery
func (s *Server) handleRecoverDeployment(w http.ResponseWriter, r *http.Request) {
	id, deployment := s.FetchDeploymentOnly(w, r)
	if deployment == nil {
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), id, authz.EffectHost); denied != nil {
		apierrors.Write(w, denied)
		return
	}
	var request cloudRecoveryHTTPRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_recovery_request", Message: "Recovery request is invalid", Hint: err.Error()})
			return
		}
	}
	if strings.TrimSpace(request.Action) == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_recovery_action", Message: "Recovery action is required"})
		return
	}
	if request.ExpectedBundleSHA != "" && (deployment.BundleSHA256 == nil || normalizeRecoverySHA(request.ExpectedBundleSHA) != normalizeRecoverySHA(*deployment.BundleSHA256)) {
		httputil.WriteAPIError(w, http.StatusPreconditionFailed, httputil.APIError{Code: "identity_mismatch", Message: "Recovery target bundle does not match the requested identity"})
		return
	}
	if request.Action != "halt" {
		s.handleCloudRepairRecovery(w, r, id, deployment, request)
		return
	}
	if request.DryRun {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"receipt": domain.CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: id, Action: request.Action, Outcome: "preview", Health: "unknown", DryRun: true}, "timestamp": time.Now().UTC().Format(time.RFC3339Nano)})
		return
	}
	if strings.TrimSpace(request.Confirmation) == "" {
		httputil.WriteAPIError(w, http.StatusPreconditionFailed, httputil.APIError{Code: "confirmation_required", Message: "Recovery execution requires explicit confirmation"})
		return
	}
	if deployment.Status == domain.StatusStopped {
		receipt := domain.CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: id, Action: request.Action, Outcome: "halted", Health: "stopped", ExternalReceipt: "scenario-to-cloud:recovery:" + id, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"receipt": receipt, "timestamp": receipt.ObservedAt})
		return
	}
	haltCtx, derr := s.loadDeploymentContext(r.Context(), id)
	if derr != nil {
		apierrors.Write(w, derr)
		return
	}
	if err := s.repo.UpdateDesiredState(r.Context(), id, domain.DesiredStopped); err != nil {
		s.log("failed to record desired state", map[string]interface{}{"error": err.Error()})
	}
	result := s.stopDeploymentOnVPS(r.Context(), haltCtx)
	if !result.OK {
		httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{Code: "recovery_failed", Message: "Cloud owner could not halt the deployment", Hint: result.Error})
		return
	}
	if err := s.repo.UpdateDeploymentStatus(r.Context(), id, domain.StatusStopped, nil, nil); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "recovery_persistence_failed", Message: "Cloud owner halted the deployment but could not persist recovery state", Hint: err.Error()})
		return
	}
	s.appendHistoryEvent(r.Context(), id, domain.HistoryEvent{Type: domain.EventStopped, Timestamp: time.Now().UTC(), Message: "Deployment halted by governed recovery", Success: boolPtr(true)})
	receipt := domain.CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: id, Action: request.Action, Outcome: "halted", Health: "stopped", ExternalReceipt: "scenario-to-cloud:recovery:" + id, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"receipt": receipt, "timestamp": receipt.ObservedAt})
}

// handleDeleteDeployment removes a deployment record, optionally stopping it first and cleaning up bundles.
func (s *Server) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	stopOnVPS := r.URL.Query().Get("stop") == "true"
	cleanupBundles := r.URL.Query().Get("cleanup") == "true"

	id, deployment := s.FetchDeploymentOnly(w, r)
	if deployment == nil {
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), id, authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
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
		stopResult := domain.VPSDeployResult{OK: false, Error: "deployment context unavailable"}
		if dctx, derr := s.loadDeploymentContext(r.Context(), id); derr == nil {
			stopResult = s.stopDeploymentOnVPS(r.Context(), dctx)
		}
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

	// 2. Retire the deployment's target releases through the owner. The
	// owner keeps the active and previous releases; retirement of a running
	// workload is the retirement plan's job, not a record deletion's.
	if manifest.Target.VPS != nil && s.reach != nil {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		target := deployment.Target
		if target.IsZero() {
			target = domain.TargetRefFromManifest(manifest)
		}
		resp := bundle.GCTargetReleases(ctx, s.reach, target, deployment.ID, manifest.Scenario.ID, domain.VPSBundleGCRequest{ScenarioID: manifest.Scenario.ID, KeepLatest: 1})
		if !resp.OK {
			s.log("failed to prune target releases", map[string]interface{}{"deployment_id": deployment.ID, "error": resp.Error})
		} else {
			s.log("pruned target releases", map[string]interface{}{"deployment_id": deployment.ID, "deleted": resp.DeletedCount})
		}
	}
}

// handleExecuteDeployment admits a durable operation for the full plan and
// hands it to the operation owner. The response carries the operation id
// clients wait on (GET /operations/{id}/wait); the SSE progress stream stays
// available keyed by operation_id. Execution ownership is the durable
// record, never this request or a goroutine: an owner restart resumes from
// step receipts. Equal request keys replay the same operation.
func (s *Server) handleExecuteDeployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	req := decodeExecuteRequest(r, s, "execute")

	dep, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to get deployment", err))
		return
	}
	if dep == nil {
		apierrors.Write(w, deploymentNotFound(id))
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), id, authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
		return
	}

	// Parse manifest before attempting to start (fail fast on invalid manifest)
	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to parse deployment manifest", err))
		return
	}

	// When ForceBundleBuild is requested, refresh the manifest BEFORE validation
	// This ensures the manifest reflects current scenario state and passes validation
	if req.ForceBundleBuild {
		refreshed, err := s.orchestrator.RefreshManifest(r.Context(), m)
		if err != nil {
			s.log("manifest refresh failed in handler", map[string]interface{}{
				"deployment_id": id,
				"error":         err.Error(),
			})
			// Continue with original manifest - validation may still pass
		} else {
			m = refreshed
			if repoRoot, repoErr := bundle.FindRepoRootFromCWD(); repoErr == nil {
				if version, _, versionErr := resolveScenarioVersion(repoRoot, m.Scenario.ID); versionErr == nil && version != "" {
					m.Scenario.Ref = version
				}
			}
			// Persist refreshed manifest to database
			manifestJSON, marshalErr := json.Marshal(m)
			if marshalErr != nil {
				s.log("failed to marshal refreshed manifest", map[string]interface{}{
					"deployment_id": id,
					"error":         marshalErr.Error(),
				})
			} else if updateErr := s.repo.UpdateDeploymentManifest(r.Context(), id, manifestJSON); updateErr != nil {
				s.log("failed to persist refreshed manifest", map[string]interface{}{
					"deployment_id": id,
					"error":         updateErr.Error(),
				})
			} else {
				s.log("manifest refreshed before validation", map[string]interface{}{
					"deployment_id": id,
				})
			}
		}
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

	// The release archive is a cloud-local input of the plan: build it now
	// (or rebuild when forced) so the admitted plan pins a real digest.
	if s.orchestrator != nil && (req.ForceBundleBuild || dep.BundlePath == nil || strings.TrimSpace(*dep.BundlePath) == "") {
		if _, err := s.orchestrator.EnsureBundle(r.Context(), id, normalized, dep.BundlePath, req.ForceBundleBuild); err != nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "Bundle build failed").WithDetail("cause", err.Error()).WithDetail("step", "bundle_build"))
			return
		}
	}

	compiled, cerr := s.compileDeploymentPlan(r.Context(), id, execplan.ScopeFull)
	if cerr != nil {
		apierrors.Write(w, cerr)
		return
	}
	switch compiled.Plan.Outcome {
	case execplan.OutcomeNeedsInput:
		apierrors.Write(w, execplan.NeedsInputError(compiled.Plan.Handoff))
		return
	case execplan.OutcomeNoOp:
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"state": "no_op", "plan_digest": compiled.PlanDigest, "deployment": dep, "timestamp": time.Now().UTC().Format(time.RFC3339)})
		return
	}
	requestKey := strings.TrimSpace(req.RequestKey)
	if requestKey == "" {
		requestKey = "execute:" + uuid.New().String()
	}
	if err := s.repo.UpdateDesiredState(r.Context(), id, domain.DesiredRunning); err != nil {
		s.log("failed to record desired state", map[string]interface{}{"error": err.Error()})
	}
	op, created, aerr := s.admitAndSubmit(r, id, requestKey, compiled, operations.ExecuteOptions{
		RunPreflight: req.RunPreflight, ForceBundleBuild: req.ForceBundleBuild, ProvidedSecrets: req.ProvidedSecrets,
	})
	if aerr != nil {
		apierrors.Write(w, aerr)
		return
	}
	s.log("deployment operation admitted", map[string]interface{}{
		"deployment_id": id, "operation_id": op.ID, "request_key": requestKey, "replayed": !created, "plan_digest": op.PlanDigest,
	})
	writeOperationAccepted(w, dep, op, !created, "Operation admitted. Wait on /operations/{id}/wait or subscribe to /deployments/{id}/progress?operation_id=<id>.")
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
	result := vps.RunInspect(ctx, dctx.Manifest, opts, s.proberFor(dctx))

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
	if denied := s.authz.RequireEffect(r.Context(), dctx.ID, authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
		return
	}

	// The intent is recorded first so observation never restarts the
	// workload while the stop is in flight or after it.
	if err := s.repo.UpdateDesiredState(r.Context(), dctx.ID, domain.DesiredStopped); err != nil {
		s.log("failed to record desired state", map[string]interface{}{"error": err.Error()})
	}
	result := s.stopDeploymentOnVPS(r.Context(), dctx)

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

// handleStartDeployment starts/resumes a stopped deployment as a durable
// operation over the start-scope plan.
// POST /deployments/{id}/start
//
// Only works for deployments in 'stopped' or 'setup_complete' status.
func (s *Server) handleStartDeployment(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), dctx.ID, authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
		return
	}
	req := decodeExecuteRequest(r, s, "start")

	// Validate status - only stopped or setup_complete can be started
	if dctx.Deployment.Status != domain.StatusStopped && dctx.Deployment.Status != domain.StatusSetupComplete {
		apierrors.Write(w, apierrors.New(apierrors.CodeOperationConflict, fmt.Sprintf("Cannot start deployment with status '%s'. Must be 'stopped' or 'setup_complete'.", dctx.Deployment.Status)).
			WithDetail("status", string(dctx.Deployment.Status)))
		return
	}
	compiled, cerr := s.compileDeploymentPlan(r.Context(), dctx.ID, execplan.ScopeStart)
	if cerr != nil {
		apierrors.Write(w, cerr)
		return
	}
	if compiled.Plan.Outcome == execplan.OutcomeNeedsInput {
		apierrors.Write(w, execplan.NeedsInputError(compiled.Plan.Handoff))
		return
	}
	requestKey := strings.TrimSpace(req.RequestKey)
	if requestKey == "" {
		requestKey = "start:" + uuid.New().String()
	}
	if err := s.repo.UpdateDesiredState(r.Context(), dctx.ID, domain.DesiredRunning); err != nil {
		s.log("failed to record desired state", map[string]interface{}{"error": err.Error()})
	}
	op, created, aerr := s.admitAndSubmit(r, dctx.ID, requestKey, compiled, operations.ExecuteOptions{ProvidedSecrets: req.ProvidedSecrets})
	if aerr != nil {
		apierrors.Write(w, aerr)
		return
	}
	s.log("deployment start operation admitted", map[string]interface{}{
		"deployment_id": dctx.ID, "operation_id": op.ID, "replayed": !created,
	})
	writeOperationAccepted(w, dctx.Deployment, op, !created, "Start admitted. Wait on /operations/{id}/wait or subscribe to /deployments/{id}/progress?operation_id=<id>.")
}

// stopDeploymentOnVPS runs the stop command on the remote VPS.
// stopDeploymentOnVPS is the scoped lifecycle stop of the deployment's
// scenario through the target owner (privilegebroker process.stop.scoped):
// it releases only this deployment's demand on shared resources.
func (s *Server) stopDeploymentOnVPS(ctx context.Context, dctx *DeploymentContext) domain.VPSDeployResult {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	rt := s.executionRuntime(dctx.Manifest, dctx.ID, "")
	if _, err := managementRepair(ctx, rt, "process.stop.scoped", map[string]any{"process": map[string]any{"scenario": dctx.Manifest.Scenario.ID, "workdir": dctx.Workdir}}); err != nil {
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
