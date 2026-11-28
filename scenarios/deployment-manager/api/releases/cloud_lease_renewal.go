package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"deployment-manager/shared"
	"github.com/gorilla/mux"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	"google.golang.org/protobuf/encoding/protojson"
)

const cloudLeaseRenewalDueWindow = 30 * time.Minute
const cloudLeaseRenewalTerm = 30 * 24 * time.Hour

func cloudLeaseRenewalDueStanding(operationID string, request durableDeployRequest, now time.Time) *CloudLeaseRenewalReceipt {
	a := request.CloudAcquireReceipt
	if a == nil || validateCloudAcquireReceipt(operationID, request) != nil {
		return nil
	}
	source := &instancev1.GetAcquireReceiptResponse{}
	if protojson.Unmarshal(a.SourceReceipt, source) != nil || source.GetInstance() == nil || source.GetInstance().GetExpiresAt() == nil || source.GetSpendReceipt() == nil {
		return nil
	}
	expires := source.GetInstance().GetExpiresAt().AsTime().UTC()
	if expires.Sub(now) > cloudLeaseRenewalDueWindow {
		return nil
	}
	state, reason := "due", "Compute Manager lease is due for renewal; no renewal request was made"
	if !expires.After(now) {
		state, reason = "blocked", "Compute Manager lease is expired; renewal standing is blocked pending owner reconciliation"
	}
	spend := source.GetSpendReceipt()
	quoteFingerprint := renewalQuoteFingerprint(spend)
	if quoteFingerprint == "" {
		return nil
	}
	return &CloudLeaseRenewalReceipt{SchemaVersion: 1, OperationID: operationID, ReleaseID: request.ReleaseID, ProfileID: request.ProfileID, RenewalKey: "dm-lease-renew:" + operationID, AcquisitionReceiptFingerprint: a.SourceFingerprint, InstanceID: a.InstanceID, Provider: source.GetInstance().GetProvider(), ProviderInstanceID: source.GetInstance().GetProviderInstanceId(), OldExpiresAt: expires, QuoteKey: spend.GetIdempotencyKey(), QuoteFingerprint: quoteFingerprint, AuthorizationFact: "only original acquisition admission is present; no renewal quote or authorization is present", State: state, Reason: reason, PersistedAt: now.UTC()}
}

func (h *Handler) RequestCloudLeaseRenewal(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuthorization(w, r) {
		return
	}
	operationID := strings.TrimSpace(mux.Vars(r)["operation_id"])
	if operationID == "" {
		shared.JSONError(w, "canonical release operation ID is required", http.StatusBadRequest)
		return
	}
	op, err := h.repo.GetOperation(r.Context(), operationID)
	if err != nil || op == nil {
		shared.JSONError(w, "release operation was not found", http.StatusNotFound)
		return
	}
	if op.ID != operationIDForRelease(op.ReleaseID) {
		shared.JSONError(w, "release operation identity is not canonical", http.StatusConflict)
		return
	}
	fenced, ok := h.repo.(FencedOperationRepository)
	if !ok {
		shared.JSONError(w, "fenced release-operation ownership is unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx := context.WithoutCancel(r.Context())
	lease, acquired, err := fenced.AcquireOperationLease(ctx, operationID, newOperationWorkerID(), 30*time.Minute)
	if err != nil {
		shared.JSONError(w, "could not acquire release-operation ownership", http.StatusServiceUnavailable)
		return
	}
	if !acquired {
		shared.JSONError(w, "another release worker still owns this operation", http.StatusConflict)
		return
	}
	ctx = WithOperationLease(ctx, operationID, lease.Owner, lease.Fence)
	defer fenced.ReleaseOperationLease(context.WithoutCancel(ctx), lease)
	op, err = h.repo.GetOperation(ctx, operationID)
	if err != nil || op == nil || op.ID != operationIDForRelease(op.ReleaseID) {
		shared.JSONError(w, "release operation changed", http.StatusConflict)
		return
	}
	var request durableDeployRequest
	if len(op.RequestSnapshot) == 0 || json.Unmarshal(op.RequestSnapshot, &request) != nil || request.ReleaseID != op.ReleaseID || request.ProfileID != op.ProfileID {
		shared.JSONError(w, "durable release target is unavailable", http.StatusConflict)
		return
	}
	receipt := request.CloudLeaseRenewalReceipt
	if receipt == nil {
		var buildErr error
		receipt, buildErr = pendingCloudLeaseRenewal(operationID, request, time.Now().UTC())
		if buildErr != nil {
			shared.JSONError(w, "lease renewal remains blocked before owner request: "+buildErr.Error(), http.StatusConflict)
			return
		}
		if !saveCloudLeaseRenewal(ctx, h, operationID, &request, op, receipt, w) {
			return
		}
	}
	if err := validateCloudLeaseRenewalReceipt(operationID, request); err != nil {
		shared.JSONError(w, "persisted lease renewal standing is invalid", http.StatusConflict)
		return
	}
	if receipt.State == "blocked" && receipt.OwnerReceipt != nil {
		hydrateOperationCloudComputeRefs(op)
		shared.JSONOK(w, op)
		return
	}
	// This route only creates and reads the operation-bound phone review. The
	// acquisition quote is not a quote for an incremental renewal term, so the
	// review remains blocked until a future quote/cap authority and separate
	// verified-human approval receipt are available. In particular, do not call
	// Compute Manager's renewal API from this review action.
	if receipt.State != "blocked" {
		receipt.State = "blocked"
		receipt.Reason = "No fresh incremental renewal quote and cap are stored; expiry is unchanged and human approval is unavailable"
	}
	if !saveCloudLeaseRenewal(ctx, h, operationID, &request, op, receipt, w) {
		return
	}
	hydrateOperationCloudComputeRefs(op)
	shared.JSONOK(w, op)
}

func saveCloudLeaseRenewal(ctx context.Context, h *Handler, operationID string, request *durableDeployRequest, op *Operation, receipt *CloudLeaseRenewalReceipt, w http.ResponseWriter) bool {
	request.CloudLeaseRenewalReceipt = receipt
	if err := validateCloudLeaseRenewalReceipt(operationID, *request); err != nil {
		shared.JSONError(w, "lease renewal receipt is invalid: "+err.Error(), http.StatusConflict)
		return false
	}
	snapshot, err := json.Marshal(request)
	if err != nil {
		shared.JSONError(w, "could not encode lease renewal receipt", http.StatusInternalServerError)
		return false
	}
	snapshots, ok := h.repo.(OperationSnapshotRepository)
	if !ok {
		shared.JSONError(w, "fenced operation snapshot storage is unavailable", http.StatusServiceUnavailable)
		return false
	}
	if err := snapshots.UpdateOperationSnapshot(ctx, operationID, snapshot); err != nil {
		shared.JSONError(w, "could not persist lease renewal receipt safely: "+err.Error(), http.StatusConflict)
		return false
	}
	op.RequestSnapshot = append(json.RawMessage(nil), snapshot...)
	op.CloudLeaseRenewalReceipt = cloneCloudLeaseRenewalReceipt(receipt)
	return true
}

func pendingCloudLeaseRenewal(operationID string, request durableDeployRequest, now time.Time) (*CloudLeaseRenewalReceipt, error) {
	a := request.CloudAcquireReceipt
	if a == nil || validateCloudAcquireReceipt(operationID, request) != nil {
		return nil, errMissingLeaseAcquire
	}
	source := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(a.SourceReceipt, source); err != nil {
		return nil, err
	}
	cm, spend := source.GetInstance(), source.GetSpendReceipt()
	if cm == nil || cm.GetExpiresAt() == nil || cm.GetId() != a.InstanceID || spend == nil || spend.GetIdempotencyKey() != a.AcquireKey || spend.GetAdmissionDecision() != "approved" || spend.GetQuoteState() != "approved" {
		return nil, errMissingLeaseAcquire
	}
	quoteFingerprint := renewalQuoteFingerprint(spend)
	if quoteFingerprint == "" {
		return nil, errMissingLeaseAcquire
	}
	oldExpiry := cm.GetExpiresAt().AsTime().UTC()
	requested := oldExpiry.Add(cloudLeaseRenewalTerm)
	state, reason := "pending", "durable Compute Manager renewal intent recorded; owner receipt is required before expiry can change"
	if !oldExpiry.After(now) {
		state, reason = "blocked", "lease is expired; a new expiry cannot be asserted without exact Compute Manager renewal evidence"
	}
	return &CloudLeaseRenewalReceipt{SchemaVersion: 1, OperationID: operationID, ReleaseID: request.ReleaseID, ProfileID: request.ProfileID, RenewalKey: "dm-lease-renew:" + operationID, AcquisitionReceiptFingerprint: a.SourceFingerprint, InstanceID: a.InstanceID, Provider: cm.GetProvider(), ProviderInstanceID: cm.GetProviderInstanceId(), OldExpiresAt: oldExpiry, RequestedUntil: requested, QuoteKey: spend.GetIdempotencyKey(), QuoteFingerprint: quoteFingerprint, AuthorizationFact: "only original acquisition admission is present; no renewal quote or authorization is present", State: state, Reason: reason, PersistedAt: now.UTC()}, nil
}

func renewalOwnerMatches(owner *instancev1.RenewalReceipt, dm *CloudLeaseRenewalReceipt) bool {
	if owner == nil {
		return false
	}
	switch owner.GetState() {
	case instancev1.RenewalState_RENEWAL_STATE_PENDING, instancev1.RenewalState_RENEWAL_STATE_RENEWED, instancev1.RenewalState_RENEWAL_STATE_REFUSED, instancev1.RenewalState_RENEWAL_STATE_FAILED, instancev1.RenewalState_RENEWAL_STATE_AMBIGUOUS:
	default:
		return false
	}
	return owner.GetIdempotencyKey() == dm.RenewalKey && owner.GetOwner() == "deployment-manager" && owner.GetInstanceId() == dm.InstanceID && owner.GetProvider() == dm.Provider && owner.GetProviderInstanceId() == dm.ProviderInstanceID && owner.GetPriorExpiresAt() != nil && owner.GetPriorExpiresAt().AsTime().Equal(dm.OldExpiresAt) && owner.GetRequestedUntil() != nil && owner.GetRequestedUntil().AsTime().Equal(dm.RequestedUntil) && owner.GetSpendIdempotencyKey() == dm.QuoteKey && owner.GetQuoteFingerprint() == dm.QuoteFingerprint && owner.GetReason() != "" && owner.GetCreatedAt() != nil && owner.GetUpdatedAt() != nil
}

func validateCloudLeaseRenewalReceipt(operationID string, request durableDeployRequest) error {
	r, a := request.CloudLeaseRenewalReceipt, request.CloudAcquireReceipt
	if r == nil {
		return nil
	}
	if a == nil || r.SchemaVersion != 1 || r.OperationID != operationID || r.ReleaseID != request.ReleaseID || r.ProfileID != request.ProfileID || r.RenewalKey != "dm-lease-renew:"+operationID || r.AcquisitionReceiptFingerprint != a.SourceFingerprint || r.InstanceID != a.InstanceID || r.Provider == "" || r.ProviderInstanceID == "" || r.OldExpiresAt.IsZero() || !r.RequestedUntil.Equal(r.OldExpiresAt.Add(cloudLeaseRenewalTerm)) || r.QuoteKey != a.AcquireKey || r.QuoteFingerprint == "" || r.AuthorizationFact == "" || (r.State != "due" && r.State != "pending" && r.State != "renewed" && r.State != "blocked") || r.Reason == "" || r.PersistedAt.IsZero() {
		return errMissingLeaseAcquire
	}
	source := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(a.SourceReceipt, source); err != nil {
		return err
	}
	cm, spend := source.GetInstance(), source.GetSpendReceipt()
	if cm == nil || cm.GetExpiresAt() == nil || cm.GetProvider() != r.Provider || cm.GetProviderInstanceId() != r.ProviderInstanceID || !cm.GetExpiresAt().AsTime().Equal(r.OldExpiresAt) || spend == nil || spend.GetIdempotencyKey() != r.QuoteKey || renewalQuoteFingerprint(spend) != r.QuoteFingerprint || spend.GetAdmissionDecision() != "approved" {
		return errMissingLeaseAcquire
	}
	if r.State == "renewed" && (r.NewExpiresAt.IsZero() || !r.NewExpiresAt.Equal(r.RequestedUntil) || r.OwnerReceipt == nil) {
		return errMissingLeaseAcquire
	}
	if r.State != "renewed" && !r.NewExpiresAt.IsZero() {
		return errMissingLeaseAcquire
	}
	if r.OwnerReceipt != nil {
		owner := &instancev1.RenewalReceipt{}
		if protojson.Unmarshal(r.OwnerReceipt, owner) != nil || !renewalOwnerMatches(owner, r) {
			return errMissingLeaseAcquire
		}
	}
	return nil
}

func renewalQuoteFingerprint(s *instancev1.SpendReceipt) string {
	if s == nil || s.GetObservedAt() == nil || s.GetExpiresAt() == nil {
		return ""
	}
	preimage := struct {
		SpendKey              string `json:"spend_idempotency_key"`
		Provider              string `json:"provider"`
		OfferID               string `json:"offer_id"`
		Region                string `json:"region"`
		Size                  string `json:"size"`
		Term                  string `json:"term"`
		Currency              string `json:"currency"`
		ServerMinor           int64  `json:"server_minor_per_month"`
		IPv4Minor             int64  `json:"ipv4_minor_per_month"`
		TotalMinor            int64  `json:"total_minor_per_month"`
		TaxBasis              string `json:"tax_basis"`
		SourceURL             string `json:"source_url"`
		SourceVersion         string `json:"source_version"`
		ObservedAt            string `json:"observed_at"`
		QuoteExpiresAt        string `json:"quote_expires_at"`
		QuoteState            string `json:"quote_state"`
		QuoteReason           string `json:"quote_reason"`
		CapCurrency           string `json:"cap_currency"`
		CapMonthlyMinor       int64  `json:"cap_monthly_minor"`
		CapTaxBasis           string `json:"cap_tax_basis"`
		Decision              string `json:"admission_decision"`
		AdmissionReason       string `json:"admission_reason"`
		ProvenanceFingerprint string `json:"price_provenance_fingerprint"`
	}{s.GetIdempotencyKey(), s.GetProvider(), s.GetOfferId(), s.GetRegion(), s.GetSize(), s.GetTerm(), s.GetCurrency(), s.GetServerMinorPerMonth(), s.GetIpv4MinorPerMonth(), s.GetTotalMinorPerMonth(), s.GetTaxBasis(), s.GetSourceUrl(), s.GetSourceVersion(), s.GetObservedAt().AsTime().UTC().Format(time.RFC3339Nano), s.GetExpiresAt().AsTime().UTC().Format(time.RFC3339Nano), s.GetQuoteState(), s.GetQuoteReason(), s.GetCapCurrency(), s.GetCapMonthlyMinor(), s.GetCapTaxBasis(), s.GetAdmissionDecision(), s.GetAdmissionReason(), s.GetPriceProvenanceFingerprint()}
	encoded, _ := json.Marshal(preimage)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

var errMissingLeaseAcquire = &leaseStandingError{}

type leaseStandingError struct{}

func (*leaseStandingError) Error() string {
	return "exact acquisition expiry and approved cost facts are unavailable"
}
