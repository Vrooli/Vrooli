package releases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"deployment-manager/shared"
	"github.com/gorilla/mux"
)

// ReconcileCloudRecoveryOperation observes and settles only an owner recovery
// that a human already started in Scenario-to-Cloud. It never calls Start.
func (h *Handler) ReconcileCloudRecoveryOperation(w http.ResponseWriter, r *http.Request) {
	if !h.requireReadAuthorization(w, r) {
		return
	}
	operationID := strings.TrimSpace(mux.Vars(r)["operation_id"])
	operation, err := h.repo.GetOperation(r.Context(), operationID)
	if err != nil || operation == nil || operation.ID != operationIDForRelease(operation.ReleaseID) {
		shared.JSONError(w, "canonical release operation was not found", http.StatusNotFound)
		return
	}
	if operation.Status != OperationFailed && operation.Status != OperationAmbiguous {
		shared.JSONError(w, "cloud recovery readback requires a failed or ambiguous release operation", http.StatusConflict)
		return
	}
	cloudSteps, ok := h.repo.(CloudStepRepository)
	if !ok {
		shared.JSONError(w, "durable cloud-step storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	step, err := cloudSteps.GetCloudStep(r.Context(), operationID)
	if err != nil || step == nil || step.State != "recovery_decision_required" {
		shared.JSONError(w, "release has no persisted failed-verification recovery decision", http.StatusConflict)
		return
	}
	fenced, ok := h.repo.(FencedOperationRepository)
	if !ok {
		shared.JSONError(w, "fenced release-operation ownership is unavailable", http.StatusServiceUnavailable)
		return
	}
	leaseCtx := context.WithoutCancel(r.Context())
	lease, acquired, err := fenced.AcquireOperationLease(leaseCtx, operationID, newOperationWorkerID(), 30*time.Second)
	if err != nil || !acquired {
		shared.JSONError(w, "release operation is currently owned by another worker", http.StatusConflict)
		return
	}
	leaseCtx = WithOperationLease(leaseCtx, operationID, lease.Owner, lease.Fence)
	defer func() { _ = fenced.ReleaseOperationLease(context.WithoutCancel(leaseCtx), lease) }()

	// Reload all durable inputs after taking the fence. No browser field is
	// accepted for the owner key, deployment, release or preview identity.
	operation, err = h.repo.GetOperation(leaseCtx, operationID)
	if err != nil || operation == nil {
		shared.JSONError(w, "release operation disappeared during reconciliation", http.StatusServiceUnavailable)
		return
	}
	step, err = cloudSteps.GetCloudStep(leaseCtx, operationID)
	if err != nil || step == nil {
		shared.JSONError(w, "durable cloud-step readback is unavailable", http.StatusServiceUnavailable)
		return
	}
	if step.State != "recovery_decision_required" || step.RecoveryIdempotencyKey != CloudRecoveryIdempotencyKey(operationID) {
		shared.JSONError(w, "durable recovery identity is missing or mismatched", http.StatusConflict)
		return
	}
	reader, ok := h.orch.(CloudRecoveryOperationReader)
	if !ok {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "readback_unavailable", "Scenario-to-Cloud recovery readback is not configured", OperationAmbiguous) {
			return
		}
		shared.JSONError(w, "Scenario-to-Cloud recovery readback is not configured", http.StatusServiceUnavailable)
		return
	}
	release, err := h.repo.Get(leaseCtx, operation.ReleaseID)
	if err != nil || release == nil || (release.Status != StatusFailed && release.Status != StatusAmbiguous) {
		shared.JSONError(w, "release status no longer permits recovery reconciliation", http.StatusConflict)
		return
	}
	if operation.ProfileID == "" || release.ProfileID == "" || operation.ProfileID != release.ProfileID {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "identity_mismatch", "release operation profile does not match the persisted release profile", OperationAmbiguous) {
			return
		}
		h.writeRecoveryStanding(w, r, operationID)
		return
	}
	identity, err := h.cloudRecoveryIdentity(leaseCtx, operation, step, release)
	if err != nil {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "identity_unavailable", err.Error(), OperationAmbiguous) {
			return
		}
		shared.JSONError(w, err.Error(), http.StatusConflict)
		return
	}

	var owner *CloudRecoveryOperation
	if step.RecoveryOwnerOperationID != "" {
		owner, err = reader.GetCloudRecoveryOperation(leaseCtx, step.DeploymentID, step.RecoveryOwnerOperationID)
	} else {
		owner, err = reader.FindCloudRecoveryOperation(leaseCtx, step.DeploymentID, step.RecoveryIdempotencyKey)
	}
	if err != nil {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "readback_unavailable", "S2C recovery readback failed: "+err.Error(), OperationAmbiguous) {
			return
		}
		shared.JSONError(w, "Scenario-to-Cloud recovery readback failed", http.StatusServiceUnavailable)
		return
	}
	if owner == nil {
		if step.RecoveryOwnerOperationID != "" {
			step.RecoveryState = "owner_operation_missing"
			step.RecoveryReason = "the previously attached exact S2C recovery operation is no longer readable"
			if err := cloudSteps.RecordCloudStep(leaseCtx, *step); err != nil {
				shared.JSONError(w, "could not persist missing owner operation", http.StatusServiceUnavailable)
				return
			}
			if err := fenced.UpdateOperationFenced(leaseCtx, lease, OperationAmbiguous, "recovery_owner_missing", step.RecoveryReason); err != nil {
				shared.JSONError(w, "could not persist missing owner operation standing", http.StatusServiceUnavailable)
				return
			}
			h.writeRecoveryStanding(w, r, operationID)
			return
		}
		step.RecoveryState = "awaiting_human_recovery"
		step.RecoveryReason = "no human-started recovery operation exists under the DM-owned stable key"
		if err := cloudSteps.RecordCloudStep(leaseCtx, *step); err != nil {
			shared.JSONError(w, "could not persist recovery lookup result", http.StatusServiceUnavailable)
			return
		}
		h.writeRecoveryStanding(w, r, operationID)
		return
	}
	if reason := validateCloudRecoveryOperation(*owner, identity, step); reason != "" {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "identity_mismatch", reason, OperationAmbiguous) {
			return
		}
		h.writeRecoveryStanding(w, r, operationID)
		return
	}
	if step.RecoveryOwnerOperationID != "" && step.RecoveryOwnerOperationID != owner.OperationID {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "identity_mismatch", "S2C returned a different owner operation ID than the persisted exact ID", OperationAmbiguous) {
			return
		}
		h.writeRecoveryStanding(w, r, operationID)
		return
	}
	if step.RecoveryPreviewRef != "" && step.RecoveryPreviewRef != owner.PreviewRef {
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "identity_mismatch", "S2C recovery preview identity changed after it was attached", OperationAmbiguous) {
			return
		}
		h.writeRecoveryStanding(w, r, operationID)
		return
	}
	step.RecoveryOwnerOperationID = owner.OperationID
	step.RecoveryPreviewRef = owner.PreviewRef
	step.RecoveryReason = owner.Error
	switch owner.Status {
	case "pending", "running":
		step.RecoveryState = "owner_pending"
		step.RecoveryReason = "human-started S2C rollback is still pending"
		if err := cloudSteps.RecordCloudStep(leaseCtx, *step); err != nil {
			shared.JSONError(w, "could not persist pending recovery identity", http.StatusServiceUnavailable)
			return
		}
		if err := fenced.UpdateOperationFenced(leaseCtx, lease, OperationAmbiguous, "recovery_pending", step.RecoveryReason); err != nil {
			shared.JSONError(w, "could not persist pending recovery standing", http.StatusServiceUnavailable)
			return
		}
	case "failed":
		step.RecoveryState = "owner_failed"
		if step.RecoveryReason == "" {
			step.RecoveryReason = "Scenario-to-Cloud reports the human-started rollback failed"
		}
		if err := cloudSteps.RecordCloudStep(leaseCtx, *step); err != nil {
			shared.JSONError(w, "could not persist failed recovery standing", http.StatusServiceUnavailable)
			return
		}
		if err := fenced.UpdateOperationFenced(leaseCtx, lease, OperationFailed, "recovery_failed", step.RecoveryReason); err != nil {
			shared.JSONError(w, "could not persist failed recovery operation", http.StatusServiceUnavailable)
			return
		}
	case "succeeded":
		ownerReceipt, receipt, err := validateCloudRecoveryReceipt(*owner, identity, step, release)
		if err != nil {
			if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "receipt_invalid", err.Error(), OperationAmbiguous) {
				return
			}
			h.writeRecoveryStanding(w, r, operationID)
			return
		}
		store, ok := h.repo.(RecoveryReceiptRepository)
		if !ok {
			shared.JSONError(w, "durable release recovery receipt store is unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := store.RecordRecoveryReceipt(leaseCtx, receipt); err != nil {
			if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "settlement_pending", "owner receipt is valid but DM recovery receipt settlement failed: "+err.Error(), OperationAmbiguous) {
				return
			}
			shared.JSONError(w, "could not settle owner recovery receipt", http.StatusServiceUnavailable)
			return
		}
		step.RecoveryState = "succeeded"
		step.RecoveryReason = "human-started S2C rollback completed and the exact owner receipt was recorded"
		step.RecoveryReceipt = append(json.RawMessage(nil), ownerReceipt...)
		if err := cloudSteps.RecordCloudStep(leaseCtx, *step); err != nil {
			shared.JSONError(w, "recovery receipt was recorded but the cloud-step settlement is pending reconciliation", http.StatusServiceUnavailable)
			return
		}
		if err := fenced.UpdateOperationFenced(leaseCtx, lease, OperationComplete, "recovery_settled", step.RecoveryReason); err != nil {
			shared.JSONError(w, "recovery receipt was recorded but operation settlement is pending reconciliation", http.StatusServiceUnavailable)
			return
		}
	default:
		if !h.persistCloudRecoveryStanding(w, leaseCtx, fenced, lease, cloudSteps, step, "owner_state_unknown", fmt.Sprintf("S2C recovery operation returned unsupported state %q", owner.Status), OperationAmbiguous) {
			return
		}
	}
	h.writeRecoveryStanding(w, r, operationID)
}

type cloudRecoveryIdentity struct {
	DeploymentID        string
	CurrentBundleSHA256 string
	LastGoodRelease     string
	ReviewRef           string
}

func (h *Handler) cloudRecoveryIdentity(ctx context.Context, operation *Operation, step *CloudStep, release *Release) (cloudRecoveryIdentity, error) {
	if operation == nil || step == nil || release == nil || step.DeploymentID == "" || release.DeploymentID != step.DeploymentID {
		return cloudRecoveryIdentity{}, errors.New("DM release and cloud-step deployment identities do not match")
	}
	if len(step.VerificationReceipt) == 0 || !json.Valid(step.VerificationReceipt) {
		return cloudRecoveryIdentity{}, errors.New("failed verification has no durable verification receipt to bind recovery")
	}
	var health struct {
		Healthy      bool   `json:"healthy"`
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal(step.VerificationReceipt, &health); err != nil || health.Healthy || health.DeploymentID != step.DeploymentID {
		return cloudRecoveryIdentity{}, errors.New("stored verification receipt is not a failed observation for this deployment")
	}
	var deploymentReceipt struct {
		Receipt struct {
			DeploymentID  string `json:"deployment_id"`
			BundleSHA256  string `json:"bundle_sha256"`
			ReleaseDigest string `json:"release_digest"`
		} `json:"receipt"`
		Publication struct {
			ReviewRef                string `json:"review_ref"`
			ReleaseDigest            string `json:"release_digest"`
			ActivatedReleaseDigest   string `json:"activated_release_digest"`
			PredecessorReleaseDigest string `json:"predecessor_release_digest"`
			State                    string `json:"state"`
		} `json:"publication"`
	}
	if err := json.Unmarshal(step.Receipt, &deploymentReceipt); err != nil {
		return cloudRecoveryIdentity{}, fmt.Errorf("decode persisted S2C deployment receipt: %w", err)
	}
	currentDigest := deploymentReceipt.Publication.ActivatedReleaseDigest
	if currentDigest == "" {
		currentDigest = deploymentReceipt.Publication.ReleaseDigest
	}
	if currentDigest == "" {
		currentDigest = deploymentReceipt.Receipt.ReleaseDigest
	}
	publications, ok := h.repo.(PublicationReceiptRepository)
	if !ok {
		return cloudRecoveryIdentity{}, errors.New("DM publication receipt store is unavailable")
	}
	stored, err := publications.ListPublicationReceipts(ctx, release.ID)
	if err != nil {
		return cloudRecoveryIdentity{}, fmt.Errorf("read DM publication receipt: %w", err)
	}
	matched := false
	for _, candidate := range stored {
		if candidate.Producer != "scenario-to-cloud" || candidate.DestinationObject == "" || candidate.ExternalReceipt == "" || candidate.Outcome != ReceiptVerified {
			continue
		}
		if candidate.DestinationRevisionID != release.DestinationRevisionID || candidate.CandidateID != release.CandidateID || candidate.DestinationObject != deploymentReceipt.Receipt.DeploymentID || candidate.ArtifactDigest != currentDigest {
			continue
		}
		if candidate.PredecessorArtifactDigest != deploymentReceipt.Publication.PredecessorReleaseDigest || candidate.ReviewKey != deploymentReceipt.Publication.ReviewRef {
			return cloudRecoveryIdentity{}, errors.New("stored DM publication receipt does not match the S2C predecessor or review binding")
		}
		matched = true
		break
	}
	if !matched {
		return cloudRecoveryIdentity{}, errors.New("no exact trusted DM publication receipt binds the failed release to S2C publication history")
	}
	if deploymentReceipt.Receipt.DeploymentID != step.DeploymentID || deploymentReceipt.Receipt.BundleSHA256 == "" || deploymentReceipt.Publication.State != "published" || deploymentReceipt.Publication.PredecessorReleaseDigest == "" || deploymentReceipt.Publication.ReviewRef == "" {
		return cloudRecoveryIdentity{}, errors.New("persisted S2C publication lacks the exact current, last-good, or review binding")
	}
	return cloudRecoveryIdentity{DeploymentID: step.DeploymentID, CurrentBundleSHA256: deploymentReceipt.Receipt.BundleSHA256, LastGoodRelease: deploymentReceipt.Publication.PredecessorReleaseDigest, ReviewRef: deploymentReceipt.Publication.ReviewRef}, nil
}

func validateCloudRecoveryOperation(owner CloudRecoveryOperation, expected cloudRecoveryIdentity, step *CloudStep) string {
	switch {
	case owner.OperationID == "":
		return "S2C returned no recovery operation ID"
	case owner.DeploymentID != expected.DeploymentID || owner.DeploymentID != step.DeploymentID:
		return "S2C recovery belongs to a different deployment"
	case owner.IdempotencyKey != step.RecoveryIdempotencyKey:
		return "S2C recovery idempotency key does not match the DM-owned key"
	case owner.Action != "rollback":
		return "S2C recovery action is not rollback"
	case !sameSHA256(owner.ExpectedCurrentBundleSHA256, expected.CurrentBundleSHA256):
		return "S2C recovery current bundle does not match the failed DM release"
	case !sameSHA256(owner.LastGoodReleaseDigest, expected.LastGoodRelease):
		return "S2C recovery last-good release does not match the trusted DM publication predecessor"
	case owner.ReviewRef != expected.ReviewRef:
		return "S2C recovery review does not match the published DM release"
	case owner.DataCompatibility != "compatible":
		return "S2C recovery does not carry the required compatible data-state qualification"
	case strings.TrimSpace(owner.PreviewRef) == "":
		return "S2C recovery has no owner-issued preview identity"
	default:
		return ""
	}
}

func sameSHA256(left, right string) bool {
	left = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(left), "sha256:"))
	right = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(right), "sha256:"))
	return left != "" && left == right
}

func validateCloudRecoveryReceipt(owner CloudRecoveryOperation, expected cloudRecoveryIdentity, step *CloudStep, release *Release) (json.RawMessage, RecoveryReceipt, error) {
	var receipt struct {
		SchemaVersion   int    `json:"schema_version"`
		DeploymentID    string `json:"deployment_id"`
		OperationID     string `json:"operation_id"`
		Action          string `json:"action"`
		Outcome         string `json:"outcome"`
		Health          string `json:"health"`
		BundleSHA256    string `json:"bundle_sha256"`
		ExternalReceipt string `json:"external_receipt"`
		ObservedAt      string `json:"observed_at"`
		DryRun          bool   `json:"dry_run"`
		PreviewRef      string `json:"preview_ref"`
		ReviewKey       string `json:"review_key"`
		ReleaseDigest   string `json:"release_digest"`
	}
	if len(owner.ReceiptJSON) == 0 || json.Unmarshal(owner.ReceiptJSON, &receipt) != nil {
		return nil, RecoveryReceipt{}, errors.New("S2C succeeded without a decodable terminal receipt")
	}
	observedAt, err := time.Parse(time.RFC3339Nano, receipt.ObservedAt)
	if err != nil || receipt.SchemaVersion != 1 || receipt.DeploymentID != expected.DeploymentID || receipt.OperationID != owner.OperationID || receipt.Action != "rollback" || receipt.Outcome != "rolled_back" || receipt.Health != "healthy" || receipt.ExternalReceipt == "" || receipt.DryRun || receipt.BundleSHA256 == "" || receipt.PreviewRef != owner.PreviewRef || receipt.ReviewKey != expected.ReviewRef || !sameSHA256(receipt.ReleaseDigest, expected.LastGoodRelease) {
		return nil, RecoveryReceipt{}, errors.New("S2C terminal rollback receipt is incomplete or mismatched")
	}
	if step.RecoveryPreviewRef != "" && step.RecoveryPreviewRef != owner.PreviewRef {
		return nil, RecoveryReceipt{}, errors.New("S2C terminal rollback preview differs from the DM attached preview")
	}
	return append(json.RawMessage(nil), owner.ReceiptJSON...), RecoveryReceipt{
		ReceiptID: fmt.Sprintf("%s:%s", release.ID, receipt.ExternalReceipt), ReleaseID: release.ID,
		CandidateID: release.CandidateID, DestinationRevisionID: release.DestinationRevisionID,
		DeploymentID: receipt.DeploymentID, Action: receipt.Action, Outcome: receipt.Outcome,
		Health: receipt.Health, BundleSHA256: receipt.BundleSHA256, ExternalReceipt: receipt.ExternalReceipt,
		ObservedAt: observedAt, DryRun: false, PreviewRef: receipt.PreviewRef,
	}, nil
}

func (h *Handler) persistCloudRecoveryStanding(w http.ResponseWriter, ctx context.Context, fenced FencedOperationRepository, lease *OperationLease, cloudSteps CloudStepRepository, step *CloudStep, state, reason, operationStatus string) bool {
	step.RecoveryState, step.RecoveryReason = state, reason
	if err := cloudSteps.RecordCloudStep(ctx, *step); err != nil {
		shared.JSONError(w, "could not persist cloud recovery readback state", http.StatusServiceUnavailable)
		return false
	}
	if err := fenced.UpdateOperationFenced(ctx, lease, operationStatus, "recovery_"+state, reason); err != nil {
		shared.JSONError(w, "could not persist release operation recovery state", http.StatusServiceUnavailable)
		return false
	}
	return true
}

func (h *Handler) writeRecoveryStanding(w http.ResponseWriter, r *http.Request, operationID string) {
	operation, err := h.repo.GetOperation(r.Context(), operationID)
	if err != nil || operation == nil {
		shared.JSONError(w, "recovery standing was persisted but could not be read", http.StatusServiceUnavailable)
		return
	}
	if cloudSteps, ok := h.repo.(CloudStepRepository); ok {
		operation.CloudStep, _ = cloudSteps.GetCloudStep(r.Context(), operationID)
	}
	shared.JSONOK(w, operation)
}
