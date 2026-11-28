package releases

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"deployment-manager/shared"
	"github.com/gorilla/mux"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	"google.golang.org/protobuf/encoding/protojson"
)

// RequestCloudTeardown persists the exact CM release intent before calling the
// owner. Retries use the same stable CM key and reconcile through exact receipt
// readback; browser input supplies only the canonical operation ID.
func (h *Handler) RequestCloudTeardown(w http.ResponseWriter, r *http.Request) {
	if !h.requireExecutionAuthorization(w, r) {
		return
	}
	operationID := strings.TrimSpace(mux.Vars(r)["operation_id"])
	if operationID == "" {
		shared.JSONError(w, "canonical release operation ID is required", http.StatusBadRequest)
		return
	}
	operation, err := h.repo.GetOperation(r.Context(), operationID)
	if err != nil || operation == nil {
		shared.JSONError(w, "release operation was not found", http.StatusNotFound)
		return
	}
	if operation.ID != operationIDForRelease(operation.ReleaseID) {
		shared.JSONError(w, "release operation identity is not canonical", http.StatusConflict)
		return
	}
	if operation.Status != OperationComplete && operation.Status != OperationCanceled {
		shared.JSONError(w, "teardown is available only for a completed or explicitly cancelled release", http.StatusConflict)
		return
	}
	fenced, ok := h.repo.(FencedOperationRepository)
	if !ok {
		shared.JSONError(w, "fenced release-operation ownership is unavailable", http.StatusServiceUnavailable)
		return
	}
	leaseCtx := context.WithoutCancel(r.Context())
	lease, acquired, err := fenced.AcquireOperationLease(leaseCtx, operationID, newOperationWorkerID(), 30*time.Minute)
	if err != nil {
		shared.JSONError(w, "could not acquire release-operation ownership", http.StatusServiceUnavailable)
		return
	}
	if !acquired {
		shared.JSONError(w, "another release worker still owns this operation", http.StatusConflict)
		return
	}
	leaseCtx = WithOperationLease(leaseCtx, operationID, lease.Owner, lease.Fence)
	defer fenced.ReleaseOperationLease(context.WithoutCancel(leaseCtx), lease)
	operation, err = h.repo.GetOperation(leaseCtx, operationID)
	if err != nil || operation == nil || operation.ID != operationIDForRelease(operation.ReleaseID) || (operation.Status != OperationComplete && operation.Status != OperationCanceled) {
		shared.JSONError(w, "release operation changed or is not teardown eligible", http.StatusConflict)
		return
	}
	var request durableDeployRequest
	if len(operation.RequestSnapshot) == 0 || json.Unmarshal(operation.RequestSnapshot, &request) != nil || request.ReleaseID != operation.ReleaseID || request.ProfileID != operation.ProfileID {
		shared.JSONError(w, "durable release target is unavailable", http.StatusConflict)
		return
	}
	receipt := request.CloudTeardownReceipt
	if receipt == nil {
		var buildErr error
		receipt, buildErr = pendingCloudTeardown(operationID, request, time.Now().UTC())
		if buildErr != nil {
			shared.JSONError(w, "teardown refused before effect: "+buildErr.Error(), http.StatusConflict)
			return
		}
		if !saveCloudTeardown(leaseCtx, h, operationID, &request, operation, receipt, w) {
			return
		}
	}
	if err := validateCloudTeardownReceipt(operationID, request); err != nil {
		shared.JSONError(w, "persisted teardown standing is invalid", http.StatusConflict)
		return
	}
	// A terminal receipt is immutable and replayed without another owner call.
	if receipt.State == "released" || receipt.State == "refused" || receipt.State == "failed" {
		hydrateOperationCloudComputeRefs(operation)
		shared.JSONOK(w, operation)
		return
	}
	if h.cloudComputeReleaser == nil {
		hydrateOperationCloudComputeRefs(operation)
		shared.JSONOK(w, operation)
		return
	}
	// Adopted VPS is passed through CM's release operation, whose owner contract
	// is detach-only; DM never calls a provider or supplies a host-destroy action.
	result, releaseErr := h.cloudComputeReleaser.ReleaseCloudCompute(leaseCtx, CloudReleaseRequest{IdempotencyKey: receipt.TeardownKey, InstanceID: receipt.InstanceID, Provider: receipt.Provider, ProviderInstanceID: receipt.ProviderInstanceID})
	if releaseErr != nil || result == nil || len(result.Receipt) == 0 {
		receipt.State = "ambiguous"
		if releaseErr != nil {
			receipt.Reason = "Compute Manager receipt readback is unavailable; release remains blocked: " + releaseErr.Error()
		} else {
			receipt.Reason = "Compute Manager returned no exact release receipt; release remains blocked"
		}
		if !saveCloudTeardown(leaseCtx, h, operationID, &request, operation, receipt, w) {
			return
		}
		hydrateOperationCloudComputeRefs(operation)
		shared.JSONOK(w, operation)
		return
	}
	owner := &instancev1.ReleaseReceipt{}
	if err := protojson.Unmarshal(result.Receipt, owner); err != nil || owner.GetIdempotencyKey() != receipt.TeardownKey || owner.GetOwner() != "deployment-manager" || owner.GetInstanceId() != receipt.InstanceID || owner.GetProvider() != receipt.Provider || owner.GetProviderInstanceId() != receipt.ProviderInstanceID || owner.GetAction() != "release" {
		receipt.State, receipt.Reason = "ambiguous", "Compute Manager returned a mismatched release receipt; release remains blocked"
		if !saveCloudTeardown(leaseCtx, h, operationID, &request, operation, receipt, w) {
			return
		}
		hydrateOperationCloudComputeRefs(operation)
		shared.JSONOK(w, operation)
		return
	}
	ownerState := map[instancev1.ReleaseState]string{instancev1.ReleaseState_RELEASE_STATE_PENDING: "pending", instancev1.ReleaseState_RELEASE_STATE_RELEASED: "released", instancev1.ReleaseState_RELEASE_STATE_REFUSED: "refused", instancev1.ReleaseState_RELEASE_STATE_FAILED: "failed", instancev1.ReleaseState_RELEASE_STATE_AMBIGUOUS: "ambiguous"}[owner.GetState()]
	if ownerState == "" {
		ownerState = "ambiguous"
	}
	receipt.State, receipt.Reason, receipt.OwnerReceipt = ownerState, owner.GetReason(), append(json.RawMessage(nil), result.Receipt...)
	if receipt.Reason == "" {
		receipt.Reason = "Compute Manager release state: " + ownerState
	}
	if err := validateCloudTeardownReceipt(operationID, request); err != nil {
		shared.JSONError(w, "Compute Manager receipt failed exact validation: "+err.Error(), http.StatusConflict)
		return
	}
	if !saveCloudTeardown(leaseCtx, h, operationID, &request, operation, receipt, w) {
		return
	}
	hydrateOperationCloudComputeRefs(operation)
	shared.JSONOK(w, operation)
}

func saveCloudTeardown(ctx context.Context, h *Handler, operationID string, request *durableDeployRequest, operation *Operation, receipt *CloudTeardownReceipt, w http.ResponseWriter) bool {
	request.CloudTeardownReceipt = receipt
	snapshot, err := json.Marshal(request)
	if err != nil {
		shared.JSONError(w, "could not encode teardown standing", http.StatusInternalServerError)
		return false
	}
	snapshots, ok := h.repo.(OperationSnapshotRepository)
	if !ok {
		shared.JSONError(w, "fenced operation snapshot storage is unavailable", http.StatusServiceUnavailable)
		return false
	}
	if err := snapshots.UpdateOperationSnapshot(ctx, operationID, snapshot); err != nil {
		shared.JSONError(w, "could not persist teardown receipt safely: "+err.Error(), http.StatusConflict)
		return false
	}
	operation.RequestSnapshot = append(json.RawMessage(nil), snapshot...)
	operation.CloudTeardownReceipt = cloneCloudTeardownReceipt(receipt)
	return true
}
