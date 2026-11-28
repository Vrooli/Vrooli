package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	s2cdeploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	s2cdeploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
)

//go:embed staging_profile.json
var stagingProfileJSON []byte

type stagingProfileView struct {
	SchemaVersion string   `json:"schema_version"`
	ID            string   `json:"id"`
	Scenario      string   `json:"scenario"`
	Environment   string   `json:"environment"`
	Domain        string   `json:"domain"`
	Provider      string   `json:"provider"`
	Offer         string   `json:"offer"`
	Regions       []string `json:"regions"`
	CapCurrency   string   `json:"cap_currency"`
	CapMinorUnits int64    `json:"cap_minor_units"`
	Approval      string   `json:"approval"`
	HealthPath    string   `json:"health_path"`
}

type stagingReadiness struct {
	State   string   `json:"state"`
	Reasons []string `json:"reasons"`
}

type stagingAdmissionFacts struct {
	QuoteReference    string
	InstanceReference string
	Currency          string
	MonthlyMinorUnits int64
}

func evaluateStagingAdmission(facts stagingAdmissionFacts, currency string, capMinorUnits int64) stagingReadiness {
	reasons := make([]string, 0, 3)
	if strings.TrimSpace(facts.QuoteReference) == "" {
		reasons = append(reasons, "Compute Manager quote and spend admission are not attached")
	}
	if strings.TrimSpace(facts.InstanceReference) == "" {
		reasons = append(reasons, "Compute Manager instance reference is not attached")
	}
	if facts.QuoteReference != "" && facts.Currency != currency {
		reasons = append(reasons, "Compute Manager quote currency does not match the approved cap")
	}
	if facts.QuoteReference != "" && facts.Currency == currency && facts.MonthlyMinorUnits > capMinorUnits {
		reasons = append(reasons, "quoted monthly cost exceeds the approved A2 cap")
	}
	state := "ready"
	if len(reasons) > 0 {
		state = "blocked"
	}
	return stagingReadiness{State: state, Reasons: reasons}
}

type stagingOperationView struct {
	OperationID                   string                 `json:"operation_id"`
	ReleaseID                     string                 `json:"release_id"`
	ProfileID                     string                 `json:"profile_id"`
	Status                        string                 `json:"status"`
	ActiveStage                   string                 `json:"active_stage,omitempty"`
	Error                         string                 `json:"error,omitempty"`
	UpdatedAt                     string                 `json:"updated_at,omitempty"`
	CloudSpendQuoteIdempotencyKey string                 `json:"cloud_spend_quote_idempotency_key,omitempty"`
	CloudComputeManagerInstanceID string                 `json:"cloud_compute_manager_instance_id,omitempty"`
	CloudStep                     *stagingCloudStepView  `json:"cloud_step,omitempty"`
	ResumeAvailable               bool                   `json:"resume_available"`
	ResumeReason                  string                 `json:"resume_reason,omitempty"`
	TLS                           stagingTLSView         `json:"tls"`
	Certificate                   stagingCertificateView `json:"certificate"`
}

type stagingTLSView struct {
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
	SampledAt string `json:"sampled_at,omitempty"`
}

type stagingCertificateView struct {
	State         string `json:"state"`
	NotAfter      string `json:"not_after,omitempty"`
	SampledAt     string `json:"sampled_at,omitempty"`
	Renewal       string `json:"renewal"`
	RenewalReason string `json:"renewal_reason,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// certificateFromReceipt only presents lifecycle data persisted by DM in the
// verification receipt. It deliberately does not join the independent S2C edge
// observation or accept values from the browser.
func certificateFromReceipt(raw, operationID, stepOperationID, expectedDeploymentID string) stagingCertificateView {
	blocked := stagingCertificateView{State: "unavailable", Renewal: "blocked", Reason: "Deployment Manager verification receipt has no valid certificate lifecycle sample"}
	var receipt struct {
		DeploymentID               string    `json:"deployment_id"`
		ObservedReleaseDigest      string    `json:"observed_release_digest"`
		ObservedAt                 time.Time `json:"observed_at"`
		CertificateLifecycleState  string    `json:"certificate_lifecycle_state"`
		CertificateLifecycleReason string    `json:"certificate_lifecycle_reason"`
		CertificateNotAfter        time.Time `json:"certificate_not_after"`
		CertificateObservedAt      time.Time `json:"certificate_observed_at"`
		CertificateRenewalState    string    `json:"certificate_renewal_state"`
		CertificateEvidenceSHA256  string    `json:"certificate_evidence_sha256"`
	}
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &receipt) != nil {
		return blocked
	}
	state := strings.ToLower(strings.TrimSpace(receipt.CertificateLifecycleState))
	renewal := strings.ToLower(strings.TrimSpace(receipt.CertificateRenewalState))
	now := time.Now().UTC()
	if strings.TrimSpace(operationID) == "" || stepOperationID != operationID || strings.TrimSpace(expectedDeploymentID) == "" || receipt.DeploymentID != expectedDeploymentID {
		blocked.Reason = "Deployment Manager certificate sample identity does not match the staging operation and deployment"
		return blocked
	}
	if state == "blocked" && strings.TrimSpace(receipt.CertificateLifecycleReason) != "" {
		blocked.Reason = strings.TrimSpace(receipt.CertificateLifecycleReason)
	}
	if (state != "valid" && state != "expired") || receipt.CertificateNotAfter.IsZero() || receipt.CertificateObservedAt.IsZero() || receipt.ObservedAt.IsZero() ||
		!receipt.ObservedAt.Equal(receipt.CertificateObservedAt) || receipt.CertificateObservedAt.After(now.Add(time.Minute)) || now.Sub(receipt.CertificateObservedAt) > 120*time.Second ||
		!validSHA256(receipt.ObservedReleaseDigest) || !validSHA256(receipt.CertificateEvidenceSHA256) ||
		renewal != "blocked" || (state == "valid") != receipt.CertificateNotAfter.After(receipt.CertificateObservedAt) {
		return blocked
	}
	return stagingCertificateView{State: state, NotAfter: receipt.CertificateNotAfter.UTC().Format(time.RFC3339),
		SampledAt: receipt.CertificateObservedAt.UTC().Format(time.RFC3339), Renewal: renewal,
		RenewalReason: strings.TrimSpace(receipt.CertificateLifecycleReason)}
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == 32
}

func tlsFromReceipt(raw, operationID, stepOperationID, expectedDeploymentID string) stagingTLSView {
	blocked := stagingTLSView{State: "unavailable", Reason: "Deployment Manager verification receipt has no current TLS observation"}
	var sample struct {
		DeploymentID          string    `json:"deployment_id"`
		ObservedReleaseDigest string    `json:"observed_release_digest"`
		ObservedAt            time.Time `json:"observed_at"`
		TLSStatus             string    `json:"tls_status"`
		TLSReason             string    `json:"tls_reason"`
		TLSEvidenceSHA256     string    `json:"tls_evidence_sha256"`
	}
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &sample) != nil {
		return blocked
	}
	if strings.TrimSpace(operationID) == "" || stepOperationID != operationID || strings.TrimSpace(expectedDeploymentID) == "" || sample.DeploymentID != expectedDeploymentID {
		blocked.Reason = "Deployment Manager TLS sample identity does not match the staging operation and deployment"
		return blocked
	}
	now := time.Now().UTC()
	status := strings.ToLower(strings.TrimSpace(sample.TLSStatus))
	if sample.ObservedAt.IsZero() || sample.ObservedAt.After(now.Add(time.Minute)) || now.Sub(sample.ObservedAt) > 120*time.Second ||
		!validSHA256(sample.ObservedReleaseDigest) || !validSHA256(sample.TLSEvidenceSHA256) ||
		(status != "healthy" && status != "degraded" && status != "unhealthy" && status != "unknown" && status != "not_applicable" && status != "ambiguous") {
		blocked.Reason = "Deployment Manager TLS observation is missing, stale, malformed or unsupported"
		return blocked
	}
	return stagingTLSView{State: status, Reason: strings.TrimSpace(sample.TLSReason), SampledAt: sample.ObservedAt.UTC().Format(time.RFC3339)}
}

type stagingCloudStepView struct {
	State                    string `json:"state,omitempty"`
	Reason                   string `json:"reason,omitempty"`
	DeploymentID             string `json:"deployment_id,omitempty"`
	OwnerOperationID         string `json:"owner_operation_id,omitempty"`
	RequestKey               string `json:"request_key,omitempty"`
	RequestDigest            string `json:"request_digest,omitempty"`
	ReceiptJSON              string `json:"receipt_json,omitempty"`
	VerificationReceiptJSON  string `json:"verification_receipt_json,omitempty"`
	RecoveryIdempotencyKey   string `json:"recovery_idempotency_key,omitempty"`
	RecoveryOwnerOperationID string `json:"recovery_owner_operation_id,omitempty"`
	RecoveryState            string `json:"recovery_state,omitempty"`
	RecoveryPreviewRef       string `json:"recovery_preview_ref,omitempty"`
	RecoveryReceiptJSON      string `json:"recovery_receipt_json,omitempty"`
	RecoveryReason           string `json:"recovery_reason,omitempty"`
	UpdatedAt                string `json:"updated_at,omitempty"`
}

func stagingOperationFromProto(op *releasesv1.ReleaseOperation) stagingOperationView {
	if op == nil {
		return stagingOperationView{}
	}
	view := stagingOperationView{OperationID: op.GetOperationId(), ReleaseID: op.GetReleaseId(), ProfileID: op.GetProfileId(), Status: op.GetStatus().String(),
		ActiveStage: op.GetActiveStage(), Error: op.GetError(), CloudSpendQuoteIdempotencyKey: op.GetCloudSpendQuoteIdempotencyKey(),
		CloudComputeManagerInstanceID: op.GetCloudComputeManagerInstanceId()}
	view.ResumeAvailable, view.ResumeReason = stagingResumeStanding(op)
	if updatedAt := op.GetUpdatedAt(); updatedAt != nil {
		view.UpdatedAt = updatedAt.AsTime().UTC().Format(time.RFC3339Nano)
	}
	if step := op.GetCloudStep(); step != nil {
		cloudStep := &stagingCloudStepView{State: step.GetState(), Reason: step.GetReason(), DeploymentID: step.GetDeploymentId(),
			OwnerOperationID: step.GetOwnerOperationId(), RequestKey: step.GetRequestKey(), RequestDigest: step.GetRequestDigest(),
			ReceiptJSON: step.GetReceiptJson(), VerificationReceiptJSON: step.GetVerificationReceiptJson(),
			RecoveryIdempotencyKey: step.GetRecoveryIdempotencyKey(), RecoveryOwnerOperationID: step.GetRecoveryOwnerOperationId(),
			RecoveryState: step.GetRecoveryState(), RecoveryPreviewRef: step.GetRecoveryPreviewRef(),
			RecoveryReceiptJSON: step.GetRecoveryReceiptJson(), RecoveryReason: step.GetRecoveryReason()}
		if updatedAt := step.GetUpdatedAt(); updatedAt != nil {
			cloudStep.UpdatedAt = updatedAt.AsTime().UTC().Format(time.RFC3339Nano)
		}
		view.CloudStep = cloudStep
		view.TLS = tlsFromReceipt(step.GetVerificationReceiptJson(), op.GetOperationId(), step.GetOperationId(), step.GetDeploymentId())
		view.Certificate = certificateFromReceipt(step.GetVerificationReceiptJson(), op.GetOperationId(), step.GetOperationId(), step.GetDeploymentId())
	} else {
		view.TLS = tlsFromReceipt("", op.GetOperationId(), "", "")
		view.Certificate = certificateFromReceipt("", op.GetOperationId(), "", "")
	}
	return view
}

func stagingResumeStanding(op *releasesv1.ReleaseOperation) (bool, string) {
	if op == nil {
		return false, "operation read-back is missing"
	}
	if op.GetProfileId() != "web-console/staging" {
		return false, "operation belongs to a different deployment profile"
	}
	if status := op.GetStatus(); status != releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED && status != releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS {
		return false, "only failed or ambiguous operations can be resumed"
	}
	step := op.GetCloudStep()
	if step == nil {
		return false, "durable cloud-step standing is missing"
	}
	switch step.GetState() {
	case "intent", "deployment_ready", "operation_admitted", "reconciliation_required", "receipt_verified":
		return true, ""
	default:
		return false, "cloud step is beyond the safe resume boundary or has no resumable receipt"
	}
}

type deploymentStagingFacade struct {
	resolveDM  func(context.Context) (string, error)
	resolveS2C func(context.Context) (string, error)
	client     *http.Client
}

func newDeploymentStagingFacade(resolveDM func(context.Context) (string, error), resolveS2C ...func(context.Context) (string, error)) *deploymentStagingFacade {
	facade := &deploymentStagingFacade{resolveDM: resolveDM, client: &http.Client{Timeout: 15 * time.Second}}
	if len(resolveS2C) > 0 {
		facade.resolveS2C = resolveS2C[0]
	}
	return facade
}

func (f *deploymentStagingFacade) dmClient(r *http.Request) (releasesconnect.ReleasesServiceClient, error) {
	base, err := f.resolveDM(r.Context())
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("Deployment Manager is unavailable")
	}
	return releasesconnect.NewReleasesServiceClient(f.identityClient(r), strings.TrimRight(base, "/")), nil
}

func (f *deploymentStagingFacade) identityClient(r *http.Request) *http.Client {
	return &http.Client{Timeout: f.client.Timeout, Transport: &identityForwardingTransport{
		base: http.DefaultTransport, authorization: r.Header.Get("Authorization"),
		deviceID: r.Header.Get("X-Vrooli-Device-Id"), accessAssertion: r.Header.Get("Cf-Access-Jwt-Assertion"),
		requestID: r.Header.Get("X-Request-ID"),
	}}
}

func (f *deploymentStagingFacade) s2cClient(r *http.Request) (s2cdeploymentsconnect.DeploymentsServiceClient, error) {
	if f.resolveS2C == nil {
		return nil, fmt.Errorf("Scenario-to-Cloud is unavailable")
	}
	base, err := f.resolveS2C(r.Context())
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("Scenario-to-Cloud is unavailable")
	}
	return s2cdeploymentsconnect.NewDeploymentsServiceClient(f.identityClient(r), strings.TrimRight(base, "/")), nil
}

type stagingRecoveryIdentity struct {
	OperationID           string
	DeploymentID          string
	CurrentReleaseDigest  string
	CurrentBundleSHA256   string
	LastGoodReleaseDigest string
	ReviewRef             string
	IdempotencyKey        string
}

type stagingRecoveryPreview struct {
	DeploymentID              string `json:"deployment_id"`
	Action                    string `json:"action"`
	CurrentBundleSHA256       string `json:"current_bundle_sha256"`
	CurrentReleaseDigest      string `json:"current_release_digest"`
	LastGoodReleaseDigest     string `json:"last_good_release_digest"`
	RepairBundleSHA256        string `json:"repair_bundle_sha256"`
	ReviewRef                 string `json:"review_ref"`
	DataCompatibility         string `json:"data_compatibility"`
	DataCompatibilityDeclared bool   `json:"data_compatibility_declared"`
	RouteKind                 string `json:"route_kind"`
	ImpactSummary             string `json:"impact_summary"`
	PreviewRef                string `json:"preview_ref"`
	IdempotencyKey            string `json:"idempotency_key"`
}

type stagingRecoveryView struct {
	State                 string `json:"state"`
	Reason                string `json:"reason,omitempty"`
	OperationID           string `json:"operation_id"`
	DeploymentID          string `json:"deployment_id"`
	CurrentReleaseDigest  string `json:"current_release_digest"`
	CurrentBundleSHA256   string `json:"current_bundle_sha256"`
	LastGoodReleaseDigest string `json:"last_good_release_digest"`
	LastGoodBundleSHA256  string `json:"last_good_bundle_sha256,omitempty"`
	ReviewRef             string `json:"review_ref"`
	DataCompatibility     string `json:"data_compatibility"`
	PreviewRef            string `json:"preview_ref,omitempty"`
	RouteKind             string `json:"route_kind,omitempty"`
	ImpactSummary         string `json:"impact_summary,omitempty"`
	RecoveryOperationID   string `json:"recovery_operation_id,omitempty"`
	RecoveryStatus        string `json:"recovery_status,omitempty"`
	RecoveryError         string `json:"recovery_error,omitempty"`
	RecoveryReceiptJSON   string `json:"recovery_receipt_json,omitempty"`
}

type stagingRecoveryConfirmation struct {
	ConfirmedPreviewRef string `json:"confirmed_preview_ref"`
	Confirmed           bool   `json:"confirmed"`
}

type stagingCompatibilityAcknowledgement struct {
	CompatibilityConfirmed bool `json:"compatibility_confirmed"`
}

type stagingRecoveryOwner struct {
	DeploymentID          string
	OperationID           string
	IdempotencyKey        string
	Action                string
	Status                string
	ExpectedCurrentBundle string
	LastGoodRelease       string
	ReviewRef             string
	PreviewRef            string
	DataCompatibility     string
	ReceiptJSON           string
	Error                 string
}

func requireStagingHuman(w http.ResponseWriter, r *http.Request) bool {
	principal, err := authn.RequireIndependentHuman(r.Context())
	if err != nil || !principal.IsVerified() || !principal.IsHuman() || (!principal.ExpiresAt.IsZero() && !time.Now().Before(principal.ExpiresAt)) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "a verified human session is required for staging recovery"})
		return false
	}
	return true
}

func stagingRecoveryIdentityFromOperation(op *releasesv1.ReleaseOperation) (stagingRecoveryIdentity, error) {
	if op == nil || op.GetProfileId() != "web-console/staging" {
		return stagingRecoveryIdentity{}, fmt.Errorf("operation is outside the server-owned staging profile")
	}
	step := op.GetCloudStep()
	if step == nil || step.GetState() != "recovery_decision_required" {
		return stagingRecoveryIdentity{}, fmt.Errorf("the stored verification does not require a recovery decision")
	}
	operationID := strings.TrimSpace(op.GetOperationId())
	deploymentID := strings.TrimSpace(step.GetDeploymentId())
	key := strings.TrimSpace(step.GetRecoveryIdempotencyKey())
	if operationID == "" || deploymentID == "" || key == "" || key != "dm-recovery:"+operationID {
		return stagingRecoveryIdentity{}, fmt.Errorf("the durable recovery identity is missing or does not match the DM operation")
	}
	var verification struct {
		Healthy      *bool  `json:"healthy"`
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal([]byte(step.GetVerificationReceiptJson()), &verification); err != nil || verification.Healthy == nil || *verification.Healthy || verification.DeploymentID != deploymentID {
		return stagingRecoveryIdentity{}, fmt.Errorf("the stored verification is not a failed observation for this deployment")
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
	if err := json.Unmarshal([]byte(step.GetReceiptJson()), &deploymentReceipt); err != nil {
		return stagingRecoveryIdentity{}, fmt.Errorf("the stored Scenario-to-Cloud publication receipt is unreadable")
	}
	receipt := deploymentReceipt.Receipt
	publication := deploymentReceipt.Publication
	if receipt.DeploymentID != deploymentID || strings.TrimSpace(receipt.BundleSHA256) == "" || publication.State != "published" || strings.TrimSpace(publication.PredecessorReleaseDigest) == "" || strings.TrimSpace(publication.ReviewRef) == "" {
		return stagingRecoveryIdentity{}, fmt.Errorf("the stored publication receipt lacks a matching current, predecessor, or review identity")
	}
	currentRelease := publication.ActivatedReleaseDigest
	if currentRelease == "" {
		currentRelease = publication.ReleaseDigest
	}
	if currentRelease == "" {
		currentRelease = receipt.ReleaseDigest
	}
	if strings.TrimSpace(currentRelease) == "" {
		return stagingRecoveryIdentity{}, fmt.Errorf("the stored publication receipt has no current release digest")
	}
	return stagingRecoveryIdentity{OperationID: operationID, DeploymentID: deploymentID, CurrentReleaseDigest: currentRelease,
		CurrentBundleSHA256: receipt.BundleSHA256, LastGoodReleaseDigest: publication.PredecessorReleaseDigest,
		ReviewRef: publication.ReviewRef, IdempotencyKey: key}, nil
}

func stagingRecoveryViewForIdentity(identity stagingRecoveryIdentity) stagingRecoveryView {
	return stagingRecoveryView{State: "awaiting_preview", OperationID: identity.OperationID, DeploymentID: identity.DeploymentID,
		CurrentReleaseDigest: identity.CurrentReleaseDigest, CurrentBundleSHA256: identity.CurrentBundleSHA256,
		LastGoodReleaseDigest: identity.LastGoodReleaseDigest, ReviewRef: identity.ReviewRef, DataCompatibility: "compatible"}
}

func (f *deploymentStagingFacade) loadRecoveryIdentity(r *http.Request, operationID string) (*releasesv1.ReleaseOperation, stagingRecoveryIdentity, error) {
	dm, err := f.dmClient(r)
	if err != nil {
		return nil, stagingRecoveryIdentity{}, err
	}
	standing, err := dm.GetOperation(r.Context(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: operationID}))
	if err != nil {
		return nil, stagingRecoveryIdentity{}, err
	}
	op := standing.Msg.GetOperation()
	if op == nil || op.GetOperationId() != operationID {
		return nil, stagingRecoveryIdentity{}, connect.NewError(connect.CodeNotFound, nil)
	}
	identity, err := stagingRecoveryIdentityFromOperation(op)
	return op, identity, err
}

func (f *deploymentStagingFacade) findRecoveryOwner(r *http.Request, identity stagingRecoveryIdentity, step *releasesv1.ReleaseCloudStep) (*stagingRecoveryOwner, error) {
	s2c, err := f.s2cClient(r)
	if err != nil {
		return nil, err
	}
	var response *connect.Response[s2cdeploymentsv1.DeploymentRecoveryOperation]
	if operationID := strings.TrimSpace(step.GetRecoveryOwnerOperationId()); operationID != "" {
		response, err = s2c.GetDeploymentRecoveryOperation(r.Context(), connect.NewRequest(&s2cdeploymentsv1.GetDeploymentRecoveryOperationRequest{DeploymentId: identity.DeploymentID, OperationId: operationID}))
	} else {
		response, err = s2c.FindDeploymentRecoveryOperation(r.Context(), connect.NewRequest(&s2cdeploymentsv1.FindDeploymentRecoveryOperationRequest{DeploymentId: identity.DeploymentID, IdempotencyKey: identity.IdempotencyKey}))
	}
	if err != nil {
		return nil, err
	}
	owner := response.Msg
	if owner == nil {
		return nil, connect.NewError(connect.CodeInternal, nil)
	}
	if owner.GetDeploymentId() != identity.DeploymentID || owner.GetIdempotencyKey() != identity.IdempotencyKey || owner.GetAction() != "rollback" || owner.GetExpectedCurrentBundleSha256() != identity.CurrentBundleSHA256 || owner.GetLastGoodReleaseDigest() != identity.LastGoodReleaseDigest || owner.GetReviewRef() != identity.ReviewRef || owner.GetDataCompatibility() != "compatible" || strings.TrimSpace(owner.GetOperationId()) == "" {
		return nil, fmt.Errorf("Scenario-to-Cloud recovery readback does not match the stored DM release identity")
	}
	return &stagingRecoveryOwner{DeploymentID: owner.GetDeploymentId(), OperationID: owner.GetOperationId(), IdempotencyKey: owner.GetIdempotencyKey(),
		Action: owner.GetAction(), Status: owner.GetStatus(), ExpectedCurrentBundle: owner.GetExpectedCurrentBundleSha256(),
		LastGoodRelease: owner.GetLastGoodReleaseDigest(), ReviewRef: owner.GetReviewRef(), PreviewRef: owner.GetPreviewRef(),
		DataCompatibility: owner.GetDataCompatibility(), ReceiptJSON: owner.GetReceiptJson(), Error: owner.GetError()}, nil
}

func (f *deploymentStagingFacade) canonicalRecoveryPreview(r *http.Request, identity stagingRecoveryIdentity) (stagingRecoveryPreview, error) {
	s2c, err := f.s2cClient(r)
	if err != nil {
		return stagingRecoveryPreview{}, err
	}
	response, err := s2c.PreviewDeploymentRecovery(r.Context(), connect.NewRequest(&s2cdeploymentsv1.PreviewDeploymentRecoveryRequest{
		DeploymentId: identity.DeploymentID, Action: "rollback",
		ExpectedCurrentBundleSha256:  identity.CurrentBundleSHA256,
		ExpectedCurrentReleaseDigest: identity.CurrentReleaseDigest,
		LastGoodReleaseDigest:        identity.LastGoodReleaseDigest,
		ReviewRef:                    identity.ReviewRef, DataCompatibility: "compatible",
		IdempotencyKey: identity.IdempotencyKey,
	}))
	if err != nil {
		return stagingRecoveryPreview{}, fmt.Errorf("Scenario-to-Cloud refused the recovery preview: %w", err)
	}
	msg := response.Msg
	preview := stagingRecoveryPreview{DeploymentID: msg.GetDeploymentId(), Action: msg.GetAction(), CurrentBundleSHA256: msg.GetCurrentBundleSha256(),
		CurrentReleaseDigest: msg.GetCurrentReleaseDigest(), LastGoodReleaseDigest: msg.GetLastGoodReleaseDigest(), RepairBundleSHA256: msg.GetRepairBundleSha256(),
		ReviewRef: msg.GetReviewRef(), DataCompatibility: msg.GetDataCompatibility(), DataCompatibilityDeclared: msg.GetDataCompatibilityDeclared(),
		RouteKind: msg.GetRouteKind(), ImpactSummary: msg.GetImpactSummary(), PreviewRef: msg.GetPreviewRef(), IdempotencyKey: msg.GetIdempotencyKey()}
	if preview.DeploymentID != identity.DeploymentID || preview.Action != "rollback" || preview.CurrentBundleSHA256 != identity.CurrentBundleSHA256 || preview.CurrentReleaseDigest != identity.CurrentReleaseDigest || preview.LastGoodReleaseDigest != identity.LastGoodReleaseDigest || strings.TrimSpace(preview.RepairBundleSHA256) == "" || preview.ReviewRef != identity.ReviewRef || preview.DataCompatibility != "compatible" || !preview.DataCompatibilityDeclared || preview.IdempotencyKey != identity.IdempotencyKey || strings.TrimSpace(preview.PreviewRef) == "" || strings.TrimSpace(preview.RouteKind) == "" || strings.TrimSpace(preview.ImpactSummary) == "" {
		return stagingRecoveryPreview{}, fmt.Errorf("Scenario-to-Cloud preview does not match the stored recovery identity")
	}
	return preview, nil
}

func stagingRecoveryViewForOwner(identity stagingRecoveryIdentity, owner *stagingRecoveryOwner) stagingRecoveryView {
	state := "pending"
	if owner.Status == "succeeded" || owner.Status == "failed" || owner.Status == "ambiguous" {
		state = owner.Status
	}
	return stagingRecoveryView{State: state, OperationID: identity.OperationID, DeploymentID: identity.DeploymentID,
		CurrentReleaseDigest: identity.CurrentReleaseDigest, CurrentBundleSHA256: identity.CurrentBundleSHA256,
		LastGoodReleaseDigest: identity.LastGoodReleaseDigest, ReviewRef: identity.ReviewRef, DataCompatibility: owner.DataCompatibility,
		PreviewRef: owner.PreviewRef, RecoveryOperationID: owner.OperationID, RecoveryStatus: owner.Status,
		RecoveryError: owner.Error, RecoveryReceiptJSON: owner.ReceiptJSON}
}

func sameRecoveryDigest(left, right string) bool {
	normalize := func(value string) string {
		value = strings.TrimSpace(strings.ToLower(value))
		return strings.TrimPrefix(value, "sha256:")
	}
	left, right = normalize(left), normalize(right)
	return left != "" && left == right
}

func recoveryOwnerMatchesIdentity(owner *s2cdeploymentsv1.DeploymentRecoveryOperation, identity stagingRecoveryIdentity, previewRef string) bool {
	return owner != nil && owner.GetDeploymentId() == identity.DeploymentID && owner.GetIdempotencyKey() == identity.IdempotencyKey &&
		owner.GetOperationId() != "" && owner.GetAction() == "rollback" && sameRecoveryDigest(owner.GetExpectedCurrentBundleSha256(), identity.CurrentBundleSHA256) &&
		sameRecoveryDigest(owner.GetLastGoodReleaseDigest(), identity.LastGoodReleaseDigest) && owner.GetReviewRef() == identity.ReviewRef &&
		owner.GetDataCompatibility() == "compatible" && (previewRef == "" || owner.GetPreviewRef() == previewRef)
}

func (f *deploymentStagingFacade) readRecoveryOwner(r *http.Request, identity stagingRecoveryIdentity, step *releasesv1.ReleaseCloudStep) (*s2cdeploymentsv1.DeploymentRecoveryOperation, error) {
	s2c, err := f.s2cClient(r)
	if err != nil {
		return nil, err
	}
	var response *connect.Response[s2cdeploymentsv1.DeploymentRecoveryOperation]
	if operationID := strings.TrimSpace(step.GetRecoveryOwnerOperationId()); operationID != "" {
		response, err = s2c.GetDeploymentRecoveryOperation(r.Context(), connect.NewRequest(&s2cdeploymentsv1.GetDeploymentRecoveryOperationRequest{DeploymentId: identity.DeploymentID, OperationId: operationID}))
	} else {
		response, err = s2c.FindDeploymentRecoveryOperation(r.Context(), connect.NewRequest(&s2cdeploymentsv1.FindDeploymentRecoveryOperationRequest{DeploymentId: identity.DeploymentID, IdempotencyKey: identity.IdempotencyKey}))
	}
	if err != nil {
		if strings.TrimSpace(step.GetRecoveryOwnerOperationId()) == "" && connect.CodeOf(err) == connect.CodeNotFound {
			return nil, nil
		}
		return nil, err
	}
	if response == nil || !recoveryOwnerMatchesIdentity(response.Msg, identity, "") {
		return nil, fmt.Errorf("Scenario-to-Cloud recovery readback does not match the stored DM release identity")
	}
	return response.Msg, nil
}

func readStagingPathOperationID(w http.ResponseWriter, r *http.Request) string {
	id := strings.TrimSpace(mux.Vars(r)["operation_id"])
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "operation_id is required"})
	}
	return id
}

func decodeStagingRecoveryBody(w http.ResponseWriter, r *http.Request, destination any) bool {
	if r.Body == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "a recovery decision is required"})
		return false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8*1024+1))
	if err != nil || len(body) > 8*1024 {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": "recovery decision body is too large"})
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "recovery decision fields are invalid"})
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "recovery decision must contain one JSON object"})
		return false
	}
	return true
}

func writeStagingRecoveryStanding(w http.ResponseWriter, status int, view stagingRecoveryView) {
	writeJSON(w, status, map[string]any{"recovery": view})
}

func recoveryOperationNotFound(err error) bool {
	return connect.CodeOf(err) == connect.CodeNotFound
}

func (f *deploymentStagingFacade) handleProfile(w http.ResponseWriter, _ *http.Request) {
	var raw struct {
		ID                         string `json:"id"`
		Scenario                   string `json:"scenario"`
		Environment                string `json:"environment"`
		DeploymentManagerProfileID string `json:"deployment_manager_profile_id"`
		Target                     struct {
			Provider   string   `json:"provider"`
			Offer      string   `json:"offer"`
			Regions    []string `json:"regions"`
			MonthlyCap struct {
				Currency   string `json:"currency"`
				MinorUnits int64  `json:"minor_units"`
			} `json:"monthly_cap"`
			Approval          string  `json:"approval"`
			InstanceReference *string `json:"instance_reference"`
			QuoteReference    *string `json:"quote_reference"`
		} `json:"target"`
		Edge struct {
			Domain string `json:"domain"`
		} `json:"edge"`
		Health struct {
			Path string `json:"path"`
		} `json:"health"`
	}
	if err := json.Unmarshal(stagingProfileJSON, &raw); err != nil {
		http.Error(w, `{"error":"staging profile is invalid"}`, http.StatusInternalServerError)
		return
	}
	facts := stagingAdmissionFacts{}
	if raw.Target.QuoteReference != nil {
		facts.QuoteReference = *raw.Target.QuoteReference
	}
	if raw.Target.InstanceReference != nil {
		facts.InstanceReference = *raw.Target.InstanceReference
	}
	readiness := evaluateStagingAdmission(facts, raw.Target.MonthlyCap.Currency, raw.Target.MonthlyCap.MinorUnits)
	profile := stagingProfileView{SchemaVersion: "vrooli.deployment-profile/v1", ID: raw.ID, Scenario: raw.Scenario,
		Environment: raw.Environment, Domain: raw.Edge.Domain, Provider: raw.Target.Provider, Offer: raw.Target.Offer,
		Regions: raw.Target.Regions, CapCurrency: raw.Target.MonthlyCap.Currency, CapMinorUnits: raw.Target.MonthlyCap.MinorUnits,
		Approval: raw.Target.Approval, HealthPath: raw.Health.Path}
	writeJSON(w, http.StatusOK, map[string]any{"profile": profile, "readiness": readiness,
		"deployment_manager_profile_id": raw.DeploymentManagerProfileID})
}

// handleStart is intentionally fail-closed until the profile has a complete,
// server-owned candidate/review/artifact contract in addition to the live
// quote and target. The browser cannot provide these identities. Returning
// the missing gates here avoids sending a partial request to DM or creating a
// release that could later be mistaken for a live staging attempt.
func (f *deploymentStagingFacade) handleStart(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"state":      "blocked",
		"profile_id": "web-console/staging",
		"reasons": []string{
			"Compute Manager quote and spend admission are not attached",
			"running Compute Manager instance reference is not attached",
			"approved candidate, artifact digest, destination revision, authorization epoch, and readiness review are not attached",
			"release commit, version, cloud bundle, and complete deployment manifest are not attached",
		},
		"deployment_manager_called": false,
	})
}

func (f *deploymentStagingFacade) handleResume(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["operation_id"])
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "operation_id is required"})
		return
	}
	client, err := f.dmClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Deployment Manager is unavailable"})
		return
	}
	standing, err := client.GetOperation(r.Context(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: id}))
	if err != nil {
		writeDeploymentError(w, err)
		return
	}
	op := standing.Msg.GetOperation()
	if op == nil || op.GetProfileId() != "web-console/staging" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "staging operation was not found"})
		return
	}
	if ok, reason := stagingResumeStanding(op); !ok {
		writeJSON(w, http.StatusConflict, map[string]any{"state": "blocked", "reason": reason, "operation": stagingOperationFromProto(op)})
		return
	}
	resumed, err := client.ResumeCloudOperation(r.Context(), connect.NewRequest(&releasesv1.ResumeCloudOperationRequest{OperationId: id}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeFailedPrecondition || connect.CodeOf(err) == connect.CodeAlreadyExists {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Deployment Manager refused resume because the operation is no longer eligible"})
			return
		}
		writeDeploymentError(w, err)
		return
	}
	if resumed.Msg.GetOperation() == nil || resumed.Msg.GetOperation().GetProfileId() != "web-console/staging" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Deployment Manager did not return the resumed staging operation"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"operation": stagingOperationFromProto(resumed.Msg.GetOperation())})
}

func (f *deploymentStagingFacade) handleRecoveryStatus(w http.ResponseWriter, r *http.Request) {
	if !requireStagingHuman(w, r) {
		return
	}
	operationID := readStagingPathOperationID(w, r)
	if operationID == "" {
		return
	}
	op, identity, err := f.loadRecoveryIdentity(r, operationID)
	if err != nil {
		if recoveryOperationNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "staging deployment operation was not found"})
			return
		}
		if _, ok := err.(*connect.Error); ok {
			writeDeploymentError(w, err)
			return
		}
		writeStagingRecoveryStanding(w, http.StatusConflict, stagingRecoveryView{State: "unavailable", OperationID: operationID, Reason: err.Error()})
		return
	}
	owner, err := f.readRecoveryOwner(r, identity, op.GetCloudStep())
	if err != nil {
		writeS2CRecoveryError(w, err)
		return
	}
	if owner == nil {
		writeStagingRecoveryStanding(w, http.StatusOK, stagingRecoveryViewForIdentity(identity))
		return
	}
	writeStagingRecoveryStanding(w, http.StatusOK, stagingRecoveryViewForOwner(identity, &stagingRecoveryOwner{
		DeploymentID: owner.GetDeploymentId(), OperationID: owner.GetOperationId(), IdempotencyKey: owner.GetIdempotencyKey(),
		Action: owner.GetAction(), Status: owner.GetStatus(), ExpectedCurrentBundle: owner.GetExpectedCurrentBundleSha256(),
		LastGoodRelease: owner.GetLastGoodReleaseDigest(), ReviewRef: owner.GetReviewRef(), PreviewRef: owner.GetPreviewRef(),
		DataCompatibility: owner.GetDataCompatibility(), ReceiptJSON: owner.GetReceiptJson(), Error: owner.GetError(),
	}))
}

func (f *deploymentStagingFacade) handleRecoveryPreview(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) || !requireStagingHuman(w, r) {
		return
	}
	var acknowledgement stagingCompatibilityAcknowledgement
	if !decodeStagingRecoveryBody(w, r, &acknowledgement) {
		return
	}
	if !acknowledgement.CompatibilityConfirmed {
		writeStagingRecoveryStanding(w, http.StatusPreconditionRequired, stagingRecoveryView{State: "blocked", Reason: "confirm the stored workload data is compatible before requesting rollback preview"})
		return
	}
	operationID := readStagingPathOperationID(w, r)
	if operationID == "" {
		return
	}
	op, identity, err := f.loadRecoveryIdentity(r, operationID)
	if err != nil {
		writeRecoveryIdentityError(w, operationID, err)
		return
	}
	owner, err := f.readRecoveryOwner(r, identity, op.GetCloudStep())
	if err != nil {
		writeS2CRecoveryError(w, err)
		return
	}
	if owner != nil {
		writeStagingRecoveryStanding(w, http.StatusOK, stagingRecoveryViewForOwner(identity, &stagingRecoveryOwner{
			DeploymentID: owner.GetDeploymentId(), OperationID: owner.GetOperationId(), IdempotencyKey: owner.GetIdempotencyKey(),
			Action: owner.GetAction(), Status: owner.GetStatus(), ExpectedCurrentBundle: owner.GetExpectedCurrentBundleSha256(),
			LastGoodRelease: owner.GetLastGoodReleaseDigest(), ReviewRef: owner.GetReviewRef(), PreviewRef: owner.GetPreviewRef(),
			DataCompatibility: owner.GetDataCompatibility(), ReceiptJSON: owner.GetReceiptJson(), Error: owner.GetError(),
		}))
		return
	}
	preview, err := f.canonicalRecoveryPreview(r, identity)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Scenario-to-Cloud could not verify a current rollback preview", "reason": err.Error()})
		return
	}
	view := stagingRecoveryViewForIdentity(identity)
	view.State, view.PreviewRef, view.RouteKind, view.LastGoodBundleSHA256, view.ImpactSummary = "preview_ready", preview.PreviewRef, preview.RouteKind, preview.RepairBundleSHA256, preview.ImpactSummary
	writeStagingRecoveryStanding(w, http.StatusOK, view)
}

func (f *deploymentStagingFacade) handleRecoveryConfirm(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) || !requireStagingHuman(w, r) {
		return
	}
	var confirmation stagingRecoveryConfirmation
	if !decodeStagingRecoveryBody(w, r, &confirmation) {
		return
	}
	if !confirmation.Confirmed || strings.TrimSpace(confirmation.ConfirmedPreviewRef) == "" {
		writeJSON(w, http.StatusPreconditionRequired, map[string]any{"error": "explicit confirmation of the reviewed recovery preview is required"})
		return
	}
	operationID := readStagingPathOperationID(w, r)
	if operationID == "" {
		return
	}
	op, identity, err := f.loadRecoveryIdentity(r, operationID)
	if err != nil {
		writeRecoveryIdentityError(w, operationID, err)
		return
	}
	owner, err := f.readRecoveryOwner(r, identity, op.GetCloudStep())
	if err != nil {
		writeS2CRecoveryError(w, err)
		return
	}
	if owner != nil {
		writeStagingRecoveryStanding(w, http.StatusAccepted, stagingRecoveryViewForOwner(identity, &stagingRecoveryOwner{
			DeploymentID: owner.GetDeploymentId(), OperationID: owner.GetOperationId(), IdempotencyKey: owner.GetIdempotencyKey(),
			Action: owner.GetAction(), Status: owner.GetStatus(), ExpectedCurrentBundle: owner.GetExpectedCurrentBundleSha256(),
			LastGoodRelease: owner.GetLastGoodReleaseDigest(), ReviewRef: owner.GetReviewRef(), PreviewRef: owner.GetPreviewRef(),
			DataCompatibility: owner.GetDataCompatibility(), ReceiptJSON: owner.GetReceiptJson(), Error: owner.GetError(),
		}))
		return
	}
	preview, err := f.canonicalRecoveryPreview(r, identity)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Scenario-to-Cloud could not verify a current rollback preview", "reason": err.Error()})
		return
	}
	if confirmation.ConfirmedPreviewRef != preview.PreviewRef {
		view := stagingRecoveryViewForIdentity(identity)
		view.State, view.Reason, view.PreviewRef, view.RouteKind, view.LastGoodBundleSHA256, view.ImpactSummary = "stale_preview", "the deployment or release changed; review this new canonical preview before confirming", preview.PreviewRef, preview.RouteKind, preview.RepairBundleSHA256, preview.ImpactSummary
		writeStagingRecoveryStanding(w, http.StatusConflict, view)
		return
	}
	s2c, err := f.s2cClient(r)
	if err != nil {
		writeS2CRecoveryError(w, err)
		return
	}
	started, err := s2c.StartDeploymentRecovery(r.Context(), connect.NewRequest(&s2cdeploymentsv1.StartDeploymentRecoveryRequest{
		DeploymentId: identity.DeploymentID, Action: "rollback", ExpectedCurrentBundleSha256: identity.CurrentBundleSHA256,
		LastGoodReleaseDigest: identity.LastGoodReleaseDigest, ReviewRef: identity.ReviewRef,
		PreviewRef: preview.PreviewRef, DataCompatibility: "compatible", IdempotencyKey: identity.IdempotencyKey,
	}))
	if err != nil {
		// The owner's stable key reconciles a response lost after admission. A
		// retry always reads first and never invents a second recovery identity.
		owner, readErr := f.readRecoveryOwner(r, identity, op.GetCloudStep())
		if readErr == nil && owner != nil {
			writeStagingRecoveryStanding(w, http.StatusAccepted, stagingRecoveryViewForOwner(identity, &stagingRecoveryOwner{
				DeploymentID: owner.GetDeploymentId(), OperationID: owner.GetOperationId(), IdempotencyKey: owner.GetIdempotencyKey(),
				Action: owner.GetAction(), Status: owner.GetStatus(), ExpectedCurrentBundle: owner.GetExpectedCurrentBundleSha256(),
				LastGoodRelease: owner.GetLastGoodReleaseDigest(), ReviewRef: owner.GetReviewRef(), PreviewRef: owner.GetPreviewRef(),
				DataCompatibility: owner.GetDataCompatibility(), ReceiptJSON: owner.GetReceiptJson(), Error: owner.GetError(),
			}))
			return
		}
		writeS2CRecoveryError(w, err)
		return
	}
	if started == nil || !recoveryOwnerMatchesIdentity(started.Msg, identity, preview.PreviewRef) {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Scenario-to-Cloud returned a recovery operation with mismatched identity; refresh durable recovery status"})
		return
	}
	view := stagingRecoveryViewForIdentity(identity)
	view.State, view.PreviewRef, view.RouteKind, view.LastGoodBundleSHA256 = "pending", started.Msg.GetPreviewRef(), preview.RouteKind, preview.RepairBundleSHA256
	view.RecoveryOperationID, view.RecoveryStatus, view.RecoveryError, view.RecoveryReceiptJSON = started.Msg.GetOperationId(), started.Msg.GetStatus(), started.Msg.GetError(), started.Msg.GetReceiptJson()
	writeStagingRecoveryStanding(w, http.StatusAccepted, view)
}

func writeRecoveryIdentityError(w http.ResponseWriter, operationID string, err error) {
	if recoveryOperationNotFound(err) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "staging deployment operation was not found"})
		return
	}
	if _, ok := err.(*connect.Error); ok {
		writeDeploymentError(w, err)
		return
	}
	writeStagingRecoveryStanding(w, http.StatusConflict, stagingRecoveryView{State: "unavailable", OperationID: operationID, Reason: err.Error()})
}

func writeS2CRecoveryError(w http.ResponseWriter, err error) {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Scenario-to-Cloud did not authorize this human for the target recovery action"})
	case connect.CodeNotFound:
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Scenario-to-Cloud recovery identity could not be read back"})
	case connect.CodeUnavailable:
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Scenario-to-Cloud is unavailable"})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Scenario-to-Cloud recovery request was refused", "reason": err.Error()})
	}
}

func (f *deploymentStagingFacade) handleOperations(w http.ResponseWriter, r *http.Request) {
	client, err := f.dmClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Deployment Manager is unavailable"})
		return
	}
	list, err := client.List(r.Context(), connect.NewRequest(&releasesv1.ListReleasesRequest{ProfileId: "web-console/staging", Limit: 20}))
	if err != nil {
		writeDeploymentError(w, err)
		return
	}
	operations := make([]stagingOperationView, 0, len(list.Msg.GetReleases()))
	for _, release := range list.Msg.GetReleases() {
		if release == nil || release.GetReleaseId() == "" {
			continue
		}
		operationID := "release-op-" + release.GetReleaseId()
		response, getErr := client.GetOperation(r.Context(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: operationID}))
		if getErr != nil {
			writeDeploymentError(w, getErr)
			return
		}
		op := response.Msg.GetOperation()
		if op == nil || op.GetProfileId() != "web-console/staging" || op.GetOperationId() != operationID || op.GetReleaseId() != release.GetReleaseId() {
			continue
		}
		operations = append(operations, stagingOperationFromProto(op))
	}
	writeJSON(w, http.StatusOK, map[string]any{"operations": operations})
}

func (f *deploymentStagingFacade) handleOperation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["operation_id"])
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "operation_id is required"})
		return
	}
	client, err := f.dmClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Deployment Manager is unavailable"})
		return
	}
	response, err := client.GetOperation(r.Context(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: id}))
	if err != nil {
		writeDeploymentError(w, err)
		return
	}
	op := response.Msg.GetOperation()
	if op == nil || op.GetProfileId() != "web-console/staging" || op.GetOperationId() != id || id != "release-op-"+op.GetReleaseId() {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "deployment operation not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"operation": stagingOperationFromProto(op)})
}

func writeDeploymentError(w http.ResponseWriter, err error) {
	switch connect.CodeOf(err) {
	case connect.CodeNotFound:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "deployment operation or profile was not found"})
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authorized to read deployment status"})
	case connect.CodeUnavailable:
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Deployment Manager is unavailable"})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "could not read Deployment Manager status"})
	}
}
