package releases

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ErrCloudAcquireAmbiguous = fmt.Errorf("Compute Manager acquire outcome is ambiguous")
var ErrCloudEnrollmentAmbiguous = fmt.Errorf("Compute Manager enrollment outcome is ambiguous")
var ErrCloudEnrollmentPending = fmt.Errorf("Compute Manager enrollment is still pending")
var ErrCloudEnrollmentInProgress = fmt.Errorf("Compute Manager enrollment is in progress")

const (
	CloudA2ProfileID       = "web-console/staging"
	CloudA2Provider        = "hetzner"
	CloudA2Size            = "cx23"
	CloudA2CapCurrency     = "USD"
	CloudA2CapMonthlyMinor = int64(709)
	CloudA2CapTaxBasis     = "pre-tax"
	CloudA2LifetimeSeconds = int64(3600)
)

// CloudAcquirePlan is the immutable DM-owned projection of the approved
// profile policy. It is snapshotted before Compute Manager is contacted and is
// restored on replay instead of rebuilding effect arguments from defaults.
type CloudAcquirePlan struct {
	ProfileID             string `json:"profile_id"`
	Provider              string `json:"provider"`
	Region                string `json:"region"`
	Size                  string `json:"size"`
	LifetimeSeconds       int64  `json:"lifetime_seconds"`
	AcquireIdempotencyKey string `json:"acquire_idempotency_key"`
	QuoteIdempotencyKey   string `json:"quote_idempotency_key"`
	CapCurrency           string `json:"cap_currency"`
	CapMonthlyMinor       int64  `json:"cap_monthly_minor"`
	CapTaxBasis           string `json:"cap_tax_basis"`
}

func canonicalCloudAcquirePlan(profileID, region, key string) (CloudAcquirePlan, error) {
	profileID = strings.TrimSpace(profileID)
	region = strings.ToLower(strings.TrimSpace(region))
	key = strings.TrimSpace(key)
	if profileID != CloudA2ProfileID || (region != "nbg1" && region != "fsn1") || key == "" {
		return CloudAcquirePlan{}, fmt.Errorf("profile, approved A2 region, and stable acquisition key are required")
	}
	return CloudAcquirePlan{
		ProfileID: profileID, Provider: CloudA2Provider, Region: region, Size: CloudA2Size, LifetimeSeconds: CloudA2LifetimeSeconds,
		AcquireIdempotencyKey: key, QuoteIdempotencyKey: key,
		CapCurrency: CloudA2CapCurrency, CapMonthlyMinor: CloudA2CapMonthlyMinor, CapTaxBasis: CloudA2CapTaxBasis,
	}, nil
}

// A2CloudAcquirePlan returns the exact immutable acquisition policy used by
// DM's approved staging profile. Compute Manager consumers use the same value
// when validating provider request and readback facts.
func A2CloudAcquirePlan(region, key string) (CloudAcquirePlan, error) {
	return canonicalCloudAcquirePlan(CloudA2ProfileID, region, key)
}

func cloneCloudAcquirePlan(value *CloudAcquirePlan) *CloudAcquirePlan {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func validateCloudAcquirePlan(plan *CloudAcquirePlan, profileID, region, acquireKey, quoteKey string) error {
	if plan == nil {
		return fmt.Errorf("durable Compute Manager acquisition plan is missing")
	}
	want, err := canonicalCloudAcquirePlan(profileID, region, acquireKey)
	if err != nil {
		return err
	}
	if *plan != want || strings.TrimSpace(quoteKey) != want.QuoteIdempotencyKey {
		return fmt.Errorf("durable Compute Manager acquisition plan differs from the approved profile policy or stored quote key")
	}
	return nil
}

type releaseCloudTarget struct {
	Target struct {
		Type string `json:"type"`
		VPS  struct {
			ComputeManagerInstanceID string `json:"compute_manager_instance_id"`
		} `json:"vps"`
	} `json:"target"`
}

// validateCloudComputeRequest refuses a cloud release before durable inserts
// or orchestrator work unless the request binds an admitted Compute Manager
// quote and exact manager-owned instance into its S2C manifest.
func (h *Handler) validateCloudComputeRequest(ctx context.Context, profileID string, request StartRequest) error {
	if request.CloudAcquireCompute {
		return validateCloudAcquireRequest(profileID, request)
	}
	if strings.TrimSpace(request.CloudAcquireRegion) != "" {
		return fmt.Errorf("cloud acquire region is only valid with cloud_acquire_compute")
	}
	if len(bytes.TrimSpace(request.CloudManifest)) == 0 {
		if request.CloudSpendQuoteIdempotencyKey != "" || request.CloudComputeManagerInstanceID != "" {
			return fmt.Errorf("Compute Manager quote and instance references require a cloud deployment manifest")
		}
		return nil
	}
	if profileID != "web-console/staging" {
		return fmt.Errorf("cloud release profile %q has no configured quote and spend ceiling policy", profileID)
	}
	quoteKey := strings.TrimSpace(request.CloudSpendQuoteIdempotencyKey)
	instanceID := strings.TrimSpace(request.CloudComputeManagerInstanceID)
	if quoteKey == "" {
		return fmt.Errorf("Compute Manager spend quote/admission reference is required before cloud deployment")
	}
	if instanceID == "" {
		return fmt.Errorf("Compute Manager instance reference is required before cloud deployment")
	}
	var manifest releaseCloudTarget
	if err := json.Unmarshal(request.CloudManifest, &manifest); err != nil {
		return fmt.Errorf("cloud deployment manifest is invalid: %w", err)
	}
	if strings.TrimSpace(manifest.Target.VPS.ComputeManagerInstanceID) != instanceID {
		return fmt.Errorf("cloud manifest Compute Manager instance reference does not match the durable operation reference")
	}
	if h.cloudComputeAdmission == nil {
		return fmt.Errorf("Compute Manager quote and target verification is unavailable; cloud deployment remains blocked")
	}
	if err := h.cloudComputeAdmission.VerifyCloudComputeAdmission(ctx, quoteKey, instanceID); err != nil {
		return fmt.Errorf("Compute Manager quote or instance is not ready: %w", err)
	}
	return nil
}

func validateCloudAcquireRequest(profileID string, request StartRequest) error {
	if profileID != "web-console/staging" {
		return fmt.Errorf("cloud acquire profile %q has no configured spend policy", profileID)
	}
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return fmt.Errorf("cloud acquire requires a stable release idempotency key")
	}
	region := strings.ToLower(strings.TrimSpace(request.CloudAcquireRegion))
	if region != "nbg1" && region != "fsn1" {
		return fmt.Errorf("A2 acquire permits only NBG1 or FSN1")
	}
	if request.CloudComputeManagerInstanceID != "" || request.CloudSpendQuoteIdempotencyKey != "" {
		return fmt.Errorf("acquire intent cannot be combined with an existing quote or instance reference")
	}
	if len(bytes.TrimSpace(request.CloudManifest)) == 0 {
		return fmt.Errorf("cloud acquire requires a deployment manifest to receive the exact Compute Manager target")
	}
	var manifest struct {
		releaseCloudTarget
		Environment string `json:"environment"`
		Scenario    struct {
			ID string `json:"id"`
		} `json:"scenario"`
		Edge struct {
			Domain string `json:"domain"`
		} `json:"edge"`
	}
	if err := json.Unmarshal(request.CloudManifest, &manifest); err != nil {
		return fmt.Errorf("cloud deployment manifest is invalid: %w", err)
	}
	if strings.TrimSpace(manifest.Scenario.ID) != "web-console" || strings.TrimSpace(manifest.Environment) != "staging" || strings.TrimSpace(manifest.Edge.Domain) != "staging.vrooli.com" {
		return fmt.Errorf("A2 acquire manifest must target Web Console staging at staging.vrooli.com")
	}
	if strings.TrimSpace(manifest.Target.Type) != "vps" {
		return fmt.Errorf("A2 acquire manifest must declare a VPS target")
	}
	if strings.TrimSpace(manifest.Target.VPS.ComputeManagerInstanceID) != "" {
		return fmt.Errorf("cloud acquire manifest must not preselect a Compute Manager instance")
	}
	return nil
}

func bindCloudManifestInstance(raw json.RawMessage, instanceID string) (json.RawMessage, error) {
	var manifest map[string]interface{}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("cloud deployment manifest is invalid: %w", err)
	}
	target, ok := manifest["target"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cloud deployment manifest target is missing")
	}
	vps, ok := target["vps"].(map[string]interface{})
	if !ok {
		vps = map[string]interface{}{}
		target["vps"] = vps
	}
	if existing := strings.TrimSpace(fmt.Sprint(vps["compute_manager_instance_id"])); existing != "" && existing != "<nil>" && existing != instanceID {
		return nil, fmt.Errorf("cloud manifest already binds a different Compute Manager target")
	}
	vps["compute_manager_instance_id"] = instanceID
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode cloud manifest with acquired target: %w", err)
	}
	return encoded, nil
}

func sameCloudOperationRequest(snapshot json.RawMessage, request StartRequest) bool {
	var previous durableDeployRequest
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &previous) != nil {
		return false
	}
	if previous.CloudAcquireCompute != request.CloudAcquireCompute || previous.CloudAcquireRegion != request.CloudAcquireRegion {
		return false
	}
	if request.CloudAcquireCompute {
		return strings.TrimSpace(request.CloudSpendQuoteIdempotencyKey) == "" && strings.TrimSpace(request.CloudComputeManagerInstanceID) == "" &&
			bytes.Equal(bytes.TrimSpace(previous.CloudManifest), bytes.TrimSpace(request.CloudManifest))
	}
	return previous.CloudSpendQuoteIdempotencyKey == request.CloudSpendQuoteIdempotencyKey &&
		previous.CloudComputeManagerInstanceID == request.CloudComputeManagerInstanceID &&
		bytes.Equal(bytes.TrimSpace(previous.CloudManifest), bytes.TrimSpace(request.CloudManifest))
}

func hydrateOperationCloudComputeRefs(operation *Operation) {
	if operation == nil || len(operation.RequestSnapshot) == 0 {
		return
	}
	var request durableDeployRequest
	if json.Unmarshal(operation.RequestSnapshot, &request) != nil {
		return
	}
	operation.CloudSpendQuoteIdempotencyKey = request.CloudSpendQuoteIdempotencyKey
	operation.CloudComputeManagerInstanceID = request.CloudComputeManagerInstanceID
	operation.CloudAcquireCompute = request.CloudAcquireCompute
	operation.CloudAcquireRegion = request.CloudAcquireRegion
	operation.CloudAcquireReceipt = cloneCloudAcquireReceipt(request.CloudAcquireReceipt)
	operation.CloudTeardownReceipt = cloneCloudTeardownReceipt(request.CloudTeardownReceipt)
	operation.CloudLeaseRenewalReceipt = cloneCloudLeaseRenewalReceipt(request.CloudLeaseRenewalReceipt)
	if operation.CloudLeaseRenewalReceipt == nil {
		operation.CloudLeaseRenewalReceipt = cloudLeaseRenewalDueStanding(operation.ID, request, time.Now().UTC())
	}
}

func cloneCloudAcquireReceipt(value *CloudAcquireReceipt) *CloudAcquireReceipt {
	if value == nil {
		return nil
	}
	copy := *value
	copy.SourceReceipt = append(json.RawMessage(nil), value.SourceReceipt...)
	return &copy
}

func cloneCloudTeardownReceipt(value *CloudTeardownReceipt) *CloudTeardownReceipt {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneCloudLeaseRenewalReceipt(value *CloudLeaseRenewalReceipt) *CloudLeaseRenewalReceipt {
	if value == nil {
		return nil
	}
	copy := *value
	copy.OwnerReceipt = append(json.RawMessage(nil), value.OwnerReceipt...)
	return &copy
}

func pendingCloudTeardown(operationID string, request durableDeployRequest, persistedAt time.Time) (*CloudTeardownReceipt, error) {
	if request.CloudAcquireReceipt == nil {
		return nil, fmt.Errorf("exact typed Compute Manager acquisition receipt is unavailable")
	}
	if err := validateCloudAcquireReceipt(operationID, request); err != nil {
		return nil, fmt.Errorf("exact acquisition target cannot authorize teardown: %w", err)
	}
	source := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(request.CloudAcquireReceipt.SourceReceipt, source); err != nil {
		return nil, fmt.Errorf("decode exact acquisition source for teardown: %w", err)
	}
	instance := source.GetInstance()
	if instance == nil || strings.TrimSpace(instance.GetProvider()) == "" || strings.TrimSpace(instance.GetProviderInstanceId()) == "" {
		return nil, fmt.Errorf("Compute Manager receipt has no exact provider-owned target identity")
	}
	return &CloudTeardownReceipt{
		SchemaVersion: 1, OperationID: operationID, ReleaseID: request.ReleaseID, ProfileID: request.ProfileID,
		TeardownKey: "dm-release:" + operationID, AcquisitionReceiptFingerprint: request.CloudAcquireReceipt.SourceFingerprint,
		InstanceID: request.CloudAcquireReceipt.InstanceID, Provider: instance.GetProvider(), ProviderInstanceID: instance.GetProviderInstanceId(),
		TargetManifestSHA256: request.CloudAcquireReceipt.TargetManifestSHA256, State: "pending", Reason: "release intent persisted; awaiting exact Compute Manager release receipt", PersistedAt: persistedAt.UTC(),
	}, nil
}

func validateCloudTeardownReceipt(operationID string, request durableDeployRequest) error {
	receipt := request.CloudTeardownReceipt
	if receipt == nil {
		return nil
	}
	acquire := request.CloudAcquireReceipt
	if acquire == nil || receipt.SchemaVersion != 1 || receipt.OperationID != operationID || receipt.ReleaseID != request.ReleaseID || receipt.ProfileID != request.ProfileID || receipt.TeardownKey != "dm-release:"+operationID || receipt.AcquisitionReceiptFingerprint != acquire.SourceFingerprint || receipt.InstanceID != acquire.InstanceID || receipt.TargetManifestSHA256 != acquire.TargetManifestSHA256 || (receipt.State != "pending" && receipt.State != "released" && receipt.State != "refused" && receipt.State != "failed" && receipt.State != "ambiguous") || strings.TrimSpace(receipt.Reason) == "" || receipt.PersistedAt.IsZero() {
		return fmt.Errorf("cloud teardown receipt does not bind the exact operation and acquisition receipt")
	}
	source := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(acquire.SourceReceipt, source); err != nil {
		return fmt.Errorf("decode acquisition source in teardown receipt: %w", err)
	}
	instance := source.GetInstance()
	if instance == nil || receipt.Provider != instance.GetProvider() || receipt.ProviderInstanceID != instance.GetProviderInstanceId() || receipt.ProviderInstanceID == "" {
		return fmt.Errorf("cloud teardown receipt changed the provider-owned target identity")
	}
	if len(receipt.OwnerReceipt) > 0 {
		owner := &instancev1.ReleaseReceipt{}
		if err := protojson.Unmarshal(receipt.OwnerReceipt, owner); err != nil || owner.GetIdempotencyKey() != receipt.TeardownKey || owner.GetOwner() != instance.GetTenant() || owner.GetInstanceId() != receipt.InstanceID || owner.GetProvider() != receipt.Provider || owner.GetProviderInstanceId() != receipt.ProviderInstanceID || owner.GetAction() != "release" || owner.GetState() == instancev1.ReleaseState_RELEASE_STATE_UNSPECIFIED {
			return fmt.Errorf("CM release receipt is absent, pending, or mismatched")
		}
		want := map[string]string{"pending": "RELEASE_STATE_PENDING", "released": "RELEASE_STATE_RELEASED", "refused": "RELEASE_STATE_REFUSED", "failed": "RELEASE_STATE_FAILED", "ambiguous": "RELEASE_STATE_AMBIGUOUS"}[receipt.State]
		if want == "" || owner.GetState().String() != want {
			return fmt.Errorf("DM teardown state does not match exact CM receipt state")
		}
	} else if receipt.State != "pending" && receipt.State != "ambiguous" {
		return fmt.Errorf("terminal teardown state requires the exact CM owner receipt")
	}
	return nil
}

func newCloudAcquireReceipt(operationID string, request DeployRequest, sourceRaw json.RawMessage, persistedAt time.Time) (*CloudAcquireReceipt, error) {
	source := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(sourceRaw, source); err != nil {
		return nil, fmt.Errorf("decode generated Compute Manager acquire receipt: %w", err)
	}
	if strings.TrimSpace(operationID) == "" || request.ReleaseID == "" || request.ProfileID == "" || request.CloudAcquireIdempotencyKey == "" || request.CloudComputeManagerInstanceID == "" {
		return nil, fmt.Errorf("DM operation, release, profile, acquire key and exact CM instance are required")
	}
	if source.GetState() != "acquired" || source.GetInstance() == nil || source.GetSpendReceipt() == nil {
		return nil, fmt.Errorf("generated CM acquire readback is not an acquired instance with a spend receipt")
	}
	instance, spend := source.GetInstance(), source.GetSpendReceipt()
	plan := request.CloudAcquirePlan
	if request.CloudAcquireCompute {
		if err := validateCloudAcquirePlan(plan, request.ProfileID, request.CloudAcquireRegion, request.CloudAcquireIdempotencyKey, request.CloudSpendQuoteIdempotencyKey); err != nil {
			return nil, err
		}
		if instance.GetProvider() != plan.Provider || instance.GetRegion() != plan.Region || instance.GetSize() != plan.Size || spend.GetCapCurrency() != plan.CapCurrency || spend.GetCapMonthlyMinor() != plan.CapMonthlyMinor || spend.GetCapTaxBasis() != plan.CapTaxBasis {
			return nil, fmt.Errorf("generated CM acquisition receipt differs from the snapshotted provider, offer, region, or cap")
		}
	} else if plan != nil || instance.GetProvider() == "" || spend.GetProvider() != instance.GetProvider() || spend.GetRegion() != instance.GetRegion() || spend.GetSize() != instance.GetSize() {
		return nil, fmt.Errorf("generated CM receipt does not bind the selected source, offer and target")
	}
	if instance.GetId() != request.CloudComputeManagerInstanceID || instance.GetTenant() != "deployment-manager" {
		return nil, fmt.Errorf("generated CM receipt instance/owner does not match the DM target")
	}
	if spend.GetIdempotencyKey() != request.CloudAcquireIdempotencyKey || spend.GetInstanceId() != instance.GetId() || spend.GetAdmissionDecision() != "approved" || spend.GetQuoteState() != "approved" {
		return nil, fmt.Errorf("generated CM spend receipt does not bind the admitted acquire key and exact instance")
	}
	if request.CloudSpendQuoteIdempotencyKey != request.CloudAcquireIdempotencyKey {
		return nil, fmt.Errorf("DM quote reference does not match the generated CM acquire receipt key")
	}
	if persistedAt.IsZero() {
		return nil, fmt.Errorf("DM receipt persistence time is required")
	}
	manifestDigest := sha256.Sum256(bytes.TrimSpace(request.CloudManifest))
	wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("fingerprint generated CM acquire receipt: %w", err)
	}
	fingerprint := sha256.Sum256(wire)
	return &CloudAcquireReceipt{
		SchemaVersion: 1, OperationID: operationID, ReleaseID: request.ReleaseID, ProfileID: request.ProfileID,
		AcquireKey: request.CloudAcquireIdempotencyKey, InstanceID: instance.GetId(),
		TargetManifestSHA256: "sha256:" + hex.EncodeToString(manifestDigest[:]),
		SourceFingerprint:    "sha256:" + hex.EncodeToString(fingerprint[:]), PersistedAt: persistedAt.UTC(),
		SourceReceipt: append(json.RawMessage(nil), sourceRaw...),
	}, nil
}

func validateCloudAcquireReceipt(operationID string, request durableDeployRequest) error {
	receipt := request.CloudAcquireReceipt
	if receipt == nil {
		return nil
	}
	if receipt.SchemaVersion != 1 || receipt.OperationID != operationID || receipt.ReleaseID != request.ReleaseID || receipt.ProfileID != request.ProfileID || receipt.AcquireKey == "" || receipt.AcquireKey != request.CloudAcquireIdempotencyKey || receipt.InstanceID == "" || receipt.InstanceID != request.CloudComputeManagerInstanceID || receipt.AcquireKey != request.CloudSpendQuoteIdempotencyKey || receipt.PersistedAt.IsZero() || len(bytes.TrimSpace(receipt.SourceReceipt)) == 0 {
		return fmt.Errorf("DM acquisition receipt does not bind operation, profile, acquire key, quote key and exact instance")
	}
	if request.CloudAcquireCompute {
		if err := validateCloudAcquirePlan(request.CloudAcquirePlan, request.ProfileID, request.CloudAcquireRegion, request.CloudAcquireIdempotencyKey, request.CloudSpendQuoteIdempotencyKey); err != nil {
			return err
		}
	} else if request.CloudAcquirePlan != nil {
		return fmt.Errorf("pre-existing Compute Manager target cannot carry a new-acquire plan")
	}
	manifestDigest := sha256.Sum256(bytes.TrimSpace(request.CloudManifest))
	if receipt.TargetManifestSHA256 != "sha256:"+hex.EncodeToString(manifestDigest[:]) {
		return fmt.Errorf("DM acquisition receipt target digest differs from the stored manifest")
	}
	source := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(receipt.SourceReceipt, source); err != nil {
		return fmt.Errorf("decode persisted generated CM acquisition receipt: %w", err)
	}
	instance, spend := source.GetInstance(), source.GetSpendReceipt()
	plan := request.CloudAcquirePlan
	policyMismatch := instance == nil || spend == nil
	if !policyMismatch && request.CloudAcquireCompute {
		policyMismatch = instance.GetProvider() != plan.Provider || instance.GetRegion() != plan.Region || instance.GetSize() != plan.Size || spend.GetCapCurrency() != plan.CapCurrency || spend.GetCapMonthlyMinor() != plan.CapMonthlyMinor || spend.GetCapTaxBasis() != plan.CapTaxBasis
	} else if !policyMismatch {
		policyMismatch = plan != nil || instance.GetProvider() == "" || spend.GetProvider() != instance.GetProvider() || spend.GetRegion() != instance.GetRegion() || spend.GetSize() != instance.GetSize()
	}
	if source.GetState() != "acquired" || policyMismatch || instance.GetId() != receipt.InstanceID || instance.GetTenant() != "deployment-manager" || spend.GetIdempotencyKey() != receipt.AcquireKey || spend.GetInstanceId() != receipt.InstanceID || spend.GetAdmissionDecision() != "approved" || spend.GetQuoteState() != "approved" {
		return fmt.Errorf("persisted generated CM acquisition receipt does not match its DM envelope")
	}
	wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(source)
	if err != nil {
		return fmt.Errorf("fingerprint persisted generated CM acquisition receipt: %w", err)
	}
	fingerprint := sha256.Sum256(wire)
	if receipt.SourceFingerprint != "sha256:"+hex.EncodeToString(fingerprint[:]) {
		return fmt.Errorf("persisted generated CM acquisition receipt fingerprint mismatch")
	}
	return nil
}
