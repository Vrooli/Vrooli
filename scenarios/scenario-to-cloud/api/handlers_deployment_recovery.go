package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type cloudRecoveryHTTPRequest struct {
	Action            string `json:"action"`
	ExpectedBundleSHA string `json:"expected_bundle_sha256"`
	RepairBundleSHA   string `json:"repair_bundle_sha256"`
	DataCompatibility string `json:"data_compatibility"`
	IdempotencyKey    string `json:"idempotency_key"`
	Confirmation      string `json:"confirmation"`
	DryRun            bool   `json:"dry_run"`
}

// handleCloudRepairRecovery accepts rollback and forward repair only when the
// owner can identify an exact retained bundle and the caller has declared data
// compatibility. The response is a durable operation projection; the effect
// receipt is published by the operation GET after the VPS is healthy.
func (s *Server) handleCloudRepairRecovery(w http.ResponseWriter, r *http.Request, id string, deployment *domain.Deployment, request cloudRecoveryHTTPRequest) {
	if request.Action != "rollback" && request.Action != "forward_repair" {
		httputil.WriteAPIError(w, http.StatusNotImplemented, httputil.APIError{Code: "unsupported_recovery_action", Message: "Recovery action is not supported by the cloud owner", Hint: "Use halt, rollback, or forward_repair."})
		return
	}
	if strings.TrimSpace(request.DataCompatibility) != "compatible" {
		httputil.WriteAPIError(w, http.StatusPreconditionFailed, httputil.APIError{Code: "incompatible_data_state", Message: "Cloud repair requires an explicit compatible data-state qualification"})
		return
	}
	currentSHA := ""
	if deployment.BundleSHA256 != nil {
		currentSHA = normalizeRecoverySHA(*deployment.BundleSHA256)
	}
	if strings.TrimSpace(request.ExpectedBundleSHA) == "" || normalizeRecoverySHA(request.ExpectedBundleSHA) != currentSHA {
		httputil.WriteAPIError(w, http.StatusPreconditionFailed, httputil.APIError{Code: "identity_mismatch", Message: "Recovery target bundle does not match the persisted deployment identity"})
		return
	}
	repairSHA := normalizeRecoverySHA(request.RepairBundleSHA)
	if !validRecoverySHA(repairSHA) {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_repair_bundle", Message: "repair_bundle_sha256 must be a 64-character hexadecimal SHA256"})
		return
	}
	if request.Action == "rollback" && repairSHA == currentSHA {
		httputil.WriteAPIError(w, http.StatusPreconditionFailed, httputil.APIError{Code: "rollback_target_unchanged", Message: "Rollback requires a different retained predecessor bundle"})
		return
	}
	repairBundle, err := findLocalRecoveryBundle(deployment.ScenarioID, repairSHA)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusPreconditionFailed, httputil.APIError{Code: "repair_bundle_unavailable", Message: "The exact repair bundle is not retained by the cloud owner", Hint: err.Error()})
		return
	}
	if request.DryRun {
		receipt := domain.CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: id, Action: request.Action, Outcome: "preview", Health: "unknown", BundleSHA256: repairSHA, DryRun: true}
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"receipt": receipt, "timestamp": time.Now().UTC().Format(time.RFC3339Nano)})
		return
	}
	if s.orchestrator == nil {
		httputil.WriteAPIError(w, http.StatusServiceUnavailable, httputil.APIError{Code: "orchestrator_unavailable", Message: "Cloud repair owner is not ready"})
		return
	}
	if deployment.Status == domain.StatusSetupRunning || deployment.Status == domain.StatusDeploying {
		httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{Code: "deployment_in_flight", Message: "Cloud repair cannot start while deployment work is running"})
		return
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:%s:%s:%s", id, request.Action, currentSHA, repairSHA)
	}
	operation, err := s.repo.GetCloudRecoveryOperationByKey(r.Context(), id, idempotencyKey)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "recovery_lookup_failed", Message: "Cloud recovery operation lookup failed", Hint: err.Error()})
		return
	}
	if operation == nil {
		operation, err = s.repo.CreateCloudRecoveryOperation(r.Context(), &domain.RecoveryOperation{
			ID:                uuid.New().String(),
			DeploymentID:      id,
			IdempotencyKey:    idempotencyKey,
			Action:            request.Action,
			ExpectedBundleSHA: currentSHA,
			RepairBundleSHA:   repairSHA,
			RepairBundlePath:  repairBundle.Path,
			DataCompatibility: "compatible",
		})
		if err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "recovery_persist_failed", Message: "Cloud recovery operation could not be recorded", Hint: err.Error()})
			return
		}
	}
	if operation.Action != request.Action || operation.RepairBundleSHA != repairSHA || operation.ExpectedBundleSHA != currentSHA {
		httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{Code: "recovery_identity_conflict", Message: "Recovery idempotency key is bound to a different repair identity"})
		return
	}
	if operation.Status == "succeeded" || operation.Status == "failed" {
		writeCloudRecoveryOperation(w, operation)
		return
	}
	claimed, err := s.repo.ClaimCloudRecoveryOperation(r.Context(), operation.ID)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "recovery_claim_failed", Message: "Cloud recovery operation could not be claimed", Hint: err.Error()})
		return
	}
	if claimed {
		runID := "cloud-recovery:" + operation.ID
		if err := s.repo.StartDeploymentRun(r.Context(), id, runID); err != nil {
			_ = s.repo.FailCloudRecoveryOperation(r.Context(), operation.ID, err.Error())
			httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{Code: "recovery_start_failed", Message: "Cloud repair could not start", Hint: err.Error()})
			return
		}
		if err := s.repo.UpdateDeploymentBundle(r.Context(), id, repairBundle.Path, repairBundle.Sha256, repairBundle.SizeBytes); err != nil {
			_ = s.repo.UpdateDeploymentStatus(r.Context(), id, domain.StatusFailed, stringPtr(err.Error()), stringPtr("recovery_bundle"))
			_ = s.repo.FailCloudRecoveryOperation(r.Context(), operation.ID, err.Error())
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "recovery_bundle_persist_failed", Message: "Cloud repair bundle identity could not be persisted", Hint: err.Error()})
			return
		}
		manifest := domain.CloudManifest{}
		if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
			_ = s.repo.FailCloudRecoveryOperation(r.Context(), operation.ID, err.Error())
			httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{Code: "manifest_unavailable", Message: "Recovery target manifest is unavailable", Hint: err.Error()})
			return
		}
		path := repairBundle.Path
		go s.runCloudRecoveryOperation(operation.ID, id, runID, manifest, path)
	}
	operation.Status = "running"
	operation.UpdatedAt = time.Now().UTC()
	writeCloudRecoveryOperation(w, operation)
}

func (s *Server) runCloudRecoveryOperation(operationID, deploymentID, runID string, manifest domain.CloudManifest, bundlePath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	s.orchestrator.RunPipeline(deploymentID, runID, manifest, &bundlePath, nil, deployment.ExecuteOptions{RunPreflight: false, ForceBundleBuild: false})
	deployment, err := s.repo.GetDeployment(ctx, deploymentID)
	if err != nil || deployment == nil || deployment.Status != domain.StatusDeployed || deployment.BundleSHA256 == nil {
		message := "cloud repair did not reach deployed health"
		if err != nil {
			message = err.Error()
		} else if deployment != nil && deployment.ErrorMessage != nil && *deployment.ErrorMessage != "" {
			message = *deployment.ErrorMessage
		}
		_ = s.repo.FailCloudRecoveryOperation(context.Background(), operationID, message)
		return
	}
	operation, err := s.repo.GetCloudRecoveryOperation(ctx, operationID)
	if err != nil || operation == nil {
		return
	}
	repairSHA := operation.RepairBundleSHA
	if normalizeRecoverySHA(*deployment.BundleSHA256) != repairSHA {
		_ = s.repo.FailCloudRecoveryOperation(context.Background(), operationID, "persisted deployment bundle identity does not match the repair bundle")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	outcome := "rolled_back"
	if operation.Action == "forward_repair" {
		outcome = "forward_repaired"
	}
	receipt := domain.CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: deploymentID, OperationID: operationID, Action: operation.Action, Outcome: outcome, Health: "healthy", BundleSHA256: repairSHA, ExternalReceipt: "scenario-to-cloud:recovery:" + operationID, ObservedAt: now}
	if err := s.repo.CompleteCloudRecoveryOperation(context.Background(), operationID, receipt); err != nil {
		_ = s.repo.FailCloudRecoveryOperation(context.Background(), operationID, err.Error())
	}
}

func (s *Server) handleGetCloudRecoveryOperation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	operationID := mux.Vars(r)["operation_id"]
	operation, err := s.repo.GetCloudRecoveryOperation(r.Context(), operationID)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "recovery_lookup_failed", Message: "Cloud recovery operation lookup failed", Hint: err.Error()})
		return
	}
	if operation == nil || operation.DeploymentID != id {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Code: "recovery_not_found", Message: "Cloud recovery operation was not found"})
		return
	}
	writeCloudRecoveryOperation(w, operation)
}

func writeCloudRecoveryOperation(w http.ResponseWriter, operation *domain.RecoveryOperation) {
	receipt := domain.CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: operation.DeploymentID, OperationID: operation.ID, Action: operation.Action, BundleSHA256: operation.RepairBundleSHA}
	switch operation.Status {
	case "succeeded":
		if operation.Receipt != nil {
			receipt = *operation.Receipt
		}
	case "failed":
		receipt.Outcome = "failed"
		receipt.Health = "unhealthy"
		receipt.Error = operation.ErrorMessage
	default:
		receipt.Outcome = operation.Status
		receipt.Health = "unknown"
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"receipt": receipt, "timestamp": time.Now().UTC().Format(time.RFC3339Nano)})
}

func findLocalRecoveryBundle(scenarioID, wantedSHA string) (domain.BundleInfo, error) {
	dir, err := bundle.GetLocalBundlesDir()
	if err != nil {
		return domain.BundleInfo{}, err
	}
	bundles, err := bundle.ListBundles(dir)
	if err != nil {
		return domain.BundleInfo{}, err
	}
	for _, candidate := range bundles {
		if candidate.ScenarioID == scenarioID && normalizeRecoverySHA(candidate.Sha256) == wantedSHA {
			return candidate, nil
		}
	}
	return domain.BundleInfo{}, fmt.Errorf("scenario %q bundle %s is not in the retained local bundle store", scenarioID, wantedSHA)
}

func normalizeRecoverySHA(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
}

func validRecoverySHA(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func stringPtr(value string) *string { return &value }
