package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
)

// ReceiptProducerRef is the producer identity stamped on every receipt this
// service emits. Consumers compare it exactly; it is never caller-supplied.
const ReceiptProducerRef = "scenario-to-cloud"

// CloudDeploymentReceipt is the owner-produced proof that a VPS deployment
// reached the requested destination and passed the deployer's health checks.
// It contains coordinates and artifact identity, never credentials.
type CloudDeploymentReceipt struct {
	SchemaVersion      int    `json:"schema_version"`
	DeploymentID       string `json:"deployment_id"`
	ScenarioID         string `json:"scenario_id"`
	ScenarioVersion    string `json:"scenario_version"`
	TargetKind         string `json:"target_kind"`
	DestinationID      string `json:"destination_id"`
	DestinationHost    string `json:"destination_host"`
	DestinationWorkdir string `json:"destination_workdir"`
	DestinationDomain  string `json:"destination_domain"`
	BundleSHA256       string `json:"bundle_sha256"`
	// ReleaseDigest and ConfigurationDigest bind the receipt to the immutable
	// release artifact set when the bundle belongs to a stored release.
	ReleaseDigest       string `json:"release_digest,omitempty"`
	ConfigurationDigest string `json:"configuration_digest,omitempty"`
	// TargetKey is the deployment's target identity (machine:<id> / host:<host>).
	TargetKey string `json:"target_key"`
	// OperationID names the run that produced the deployed state.
	OperationID     string `json:"operation_id,omitempty"`
	Outcome         string `json:"outcome"`
	Health          string `json:"health"`
	ExternalReceipt string `json:"external_receipt"`
	ObservedAt      string `json:"observed_at"`
	// ProducerRef is the emitting service principal.
	ProducerRef string `json:"producer_ref"`
	// Signature is the producer attestation over CanonicalJSON.
	Signature *receiptsigning.SignatureEnvelope `json:"signature,omitempty"`
}

// ReceiptExpectation is what a consumer already knows about the receipt it
// asked for. Empty fields are not compared; ProducerRef always is.
type ReceiptExpectation struct {
	DeploymentID  string
	ReleaseDigest string
	BundleSHA256  string
	TargetKey     string
	// MaxAge bounds observed_at against Now; zero disables the check.
	MaxAge time.Duration
	Now    time.Time
}

// Check refuses a forged, mismatched or stale receipt. It is the single
// rule both the cloud read handler and Deployment Manager apply.
func (r CloudDeploymentReceipt) Check(expect ReceiptExpectation) error {
	if !r.Valid() {
		return fmt.Errorf("receipt is incomplete or does not describe a healthy deployed effect")
	}
	if r.ProducerRef != ReceiptProducerRef {
		return fmt.Errorf("receipt producer_ref %q is not the cloud service principal", r.ProducerRef)
	}
	if expect.DeploymentID != "" && r.DeploymentID != expect.DeploymentID {
		return fmt.Errorf("receipt is bound to deployment %q, expected %q", r.DeploymentID, expect.DeploymentID)
	}
	if expect.ReleaseDigest != "" && !strings.EqualFold(strings.TrimPrefix(r.ReleaseDigest, "sha256:"), strings.TrimPrefix(expect.ReleaseDigest, "sha256:")) {
		return fmt.Errorf("receipt is bound to release %q, expected %q", r.ReleaseDigest, expect.ReleaseDigest)
	}
	if expect.BundleSHA256 != "" && !strings.EqualFold(r.BundleSHA256, expect.BundleSHA256) {
		return fmt.Errorf("receipt is bound to bundle %q, expected %q", r.BundleSHA256, expect.BundleSHA256)
	}
	if expect.TargetKey != "" && r.TargetKey != expect.TargetKey {
		return fmt.Errorf("receipt is bound to target %q, expected %q", r.TargetKey, expect.TargetKey)
	}
	if expect.MaxAge > 0 {
		observed, err := time.Parse(time.RFC3339Nano, r.ObservedAt)
		if err != nil {
			return fmt.Errorf("receipt observed_at %q is not RFC 3339", r.ObservedAt)
		}
		now := expect.Now
		if now.IsZero() {
			now = time.Now()
		}
		if now.Sub(observed) > expect.MaxAge {
			return fmt.Errorf("receipt observed_at %s is older than the %s policy", r.ObservedAt, expect.MaxAge)
		}
	}
	return nil
}

// CanonicalJSON is the signing payload: every field but the signature, keys
// sorted.
func (r CloudDeploymentReceipt) CanonicalJSON() ([]byte, error) {
	r.Signature = nil
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return json.Marshal(generic)
}

// Sign attaches the producer attestation.
func (r *CloudDeploymentReceipt) Sign(ctx context.Context, signer receiptsigning.ReceiptSigner) error {
	if signer == nil {
		return fmt.Errorf("receipt signer is unavailable")
	}
	canonical, err := r.CanonicalJSON()
	if err != nil {
		return err
	}
	envelope, err := signer.Sign(ctx, receiptsigning.PurposeCloudEvidenceReceipt, canonical)
	if err != nil {
		return err
	}
	r.Signature = &envelope
	return nil
}

// VerifySignature checks the attestation; an unsigned receipt is refused.
func (r CloudDeploymentReceipt) VerifySignature(ctx context.Context, signer receiptsigning.ReceiptSigner) error {
	if r.Signature == nil {
		return fmt.Errorf("receipt carries no producer signature")
	}
	if signer == nil {
		return fmt.Errorf("receipt signer is unavailable")
	}
	if r.Signature.Purpose != receiptsigning.PurposeCloudEvidenceReceipt {
		return fmt.Errorf("receipt signature purpose %q is not %q", r.Signature.Purpose, receiptsigning.PurposeCloudEvidenceReceipt)
	}
	canonical, err := r.CanonicalJSON()
	if err != nil {
		return err
	}
	return signer.Verify(ctx, *r.Signature, canonical)
}

// CloudRecoveryReceipt is the owner-produced proof of a recovery effect.
// A preview is deliberately represented as an unavailable effect so callers
// cannot mistake planning for a remote mutation.
type CloudRecoveryReceipt struct {
	SchemaVersion   int    `json:"schema_version"`
	DeploymentID    string `json:"deployment_id"`
	OperationID     string `json:"operation_id,omitempty"`
	Action          string `json:"action"`
	Outcome         string `json:"outcome"`
	Health          string `json:"health"`
	BundleSHA256    string `json:"bundle_sha256,omitempty"`
	ExternalReceipt string `json:"external_receipt"`
	ObservedAt      string `json:"observed_at"`
	DryRun          bool   `json:"dry_run"`
	Error           string `json:"error,omitempty"`
	// PreviewRef is the dry-run reference an execution must present.
	PreviewRef string `json:"preview_ref,omitempty"`
	// ReviewKey, ReleaseDigest and RouteKind are the governance binding and
	// the owner route the recovery was bound to.
	ReviewKey     string `json:"review_key,omitempty"`
	ReleaseDigest string `json:"release_digest,omitempty"`
	RouteKind     string `json:"route_kind,omitempty"`
}

// Recovery routes. The cloud-target route is selected when the deployment's
// target binding carries a Bridge machine identity (native control plane on
// the target); the bundle pipeline is the documented fallback.
const (
	RecoveryRouteBundlePipeline    = "bundle_pipeline"
	RecoveryRouteCloudTargetVerb   = "cloud_target_release_rollback"
	RecoveryRouteCloudTargetRepair = "cloud_target_release_activate"
)

// RecoveryBinding ties a repair operation to the governed publication it
// acts on, the preview the operator reviewed and the owner route chosen.
type RecoveryBinding struct {
	ReviewRef     string `json:"review_ref"`
	PublicationID string `json:"publication_id,omitempty"`
	ReleaseDigest string `json:"release_digest,omitempty"`
	PreviewRef    string `json:"preview_ref"`
	RouteKind     string `json:"route_kind"`
}

func (r CloudRecoveryReceipt) Valid() bool {
	validOutcome := r.Outcome == "halted" || r.Outcome == "rolled_back" || r.Outcome == "forward_repaired"
	validHealth := r.Health == "stopped" || r.Health == "healthy"
	if r.Action != "halt" && (r.OperationID == "" || r.BundleSHA256 == "") {
		return false
	}
	return r.SchemaVersion == 1 && r.DeploymentID != "" && r.Action != "" &&
		validOutcome && validHealth && r.ExternalReceipt != "" &&
		r.ObservedAt != "" && !r.DryRun
}

// RecoveryOperation records the owner-side durable lifecycle for a recovery
// request. The operation row is the source of truth while a repair runs; the
// receipt is emitted only after the deployment record and persisted deploy
// result agree on the repaired bundle.
type RecoveryOperation struct {
	ID                string                `json:"id"`
	DeploymentID      string                `json:"deployment_id"`
	IdempotencyKey    string                `json:"idempotency_key"`
	Action            string                `json:"action"`
	ExpectedBundleSHA string                `json:"expected_bundle_sha256,omitempty"`
	RepairBundleSHA   string                `json:"repair_bundle_sha256,omitempty"`
	RepairBundlePath  string                `json:"repair_bundle_path,omitempty"`
	DataCompatibility string                `json:"data_compatibility,omitempty"`
	Status            string                `json:"status"`
	Binding           *RecoveryBinding      `json:"binding,omitempty"`
	Receipt           *CloudRecoveryReceipt `json:"receipt,omitempty"`
	ErrorMessage      string                `json:"error_message,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	StartedAt         *time.Time            `json:"started_at,omitempty"`
	CompletedAt       *time.Time            `json:"completed_at,omitempty"`
}

// ReceiptRelease is the immutable release the deployed bundle belongs to,
// resolved by the caller from the release store (empty when the bundle was
// never built as a release).
type ReceiptRelease struct {
	ReleaseDigest       string
	ConfigurationDigest string
	// OperationID names the operation that produced the deployed state when
	// the caller can resolve it (the latest succeeded operation).
	OperationID string
}

// BuildCloudDeploymentReceipt constructs a receipt only after the persisted
// deployment record and persisted deploy result agree on a successful effect.
// observedAt is the time the record reached its deployed state; a zero value
// falls back to the record's updated_at, never to the read time.
func BuildCloudDeploymentReceipt(deployment *Deployment, observedAt time.Time, rel ReceiptRelease) (*CloudDeploymentReceipt, error) {
	if deployment == nil {
		return nil, fmt.Errorf("deployment is required")
	}
	if deployment.ID == "" || deployment.ScenarioID == "" {
		return nil, fmt.Errorf("deployment identity is incomplete")
	}
	if deployment.Status != StatusDeployed {
		return nil, fmt.Errorf("deployment %q is not deployed", deployment.ID)
	}
	if !deployment.DeployResult.Valid || len(deployment.DeployResult.Data) == 0 {
		return nil, fmt.Errorf("deployment %q has no persisted deploy result", deployment.ID)
	}
	var result VPSDeployResult
	if err := json.Unmarshal(deployment.DeployResult.Data, &result); err != nil {
		return nil, fmt.Errorf("decode deploy result: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("deployment %q deploy result is not successful", deployment.ID)
	}
	if len(deployment.Manifest) == 0 {
		return nil, fmt.Errorf("deployment %q has no manifest", deployment.ID)
	}
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		return nil, fmt.Errorf("decode deployment manifest: %w", err)
	}
	if manifest.Target.VPS == nil {
		return nil, fmt.Errorf("deployment %q has no VPS target", deployment.ID)
	}
	host := strings.TrimSpace(manifest.Target.VPS.Host)
	workdir := strings.TrimSpace(manifest.Target.VPS.Workdir)
	domain := strings.TrimSpace(manifest.Edge.Domain)
	bundleSHA := ""
	if deployment.BundleSHA256 != nil {
		bundleSHA = strings.TrimSpace(*deployment.BundleSHA256)
	}
	if host == "" || workdir == "" || domain == "" || bundleSHA == "" {
		return nil, fmt.Errorf("deployment %q is missing exact destination or bundle identity", deployment.ID)
	}
	if observedAt.IsZero() {
		observedAt = deployment.UpdatedAt
	}
	if observedAt.IsZero() {
		return nil, fmt.Errorf("deployment %q has no deployed-at time", deployment.ID)
	}
	targetKey := deployment.Target.Key()
	if targetKey == "" {
		targetKey = "host:" + host
	}
	operationID := strings.TrimSpace(rel.OperationID)
	destinationID := destinationIdentity(deployment.ScenarioID, host, workdir, domain)
	return &CloudDeploymentReceipt{
		SchemaVersion:       1,
		DeploymentID:        deployment.ID,
		ScenarioID:          deployment.ScenarioID,
		ScenarioVersion:     strings.TrimSpace(manifest.Scenario.Ref),
		TargetKind:          "vps",
		DestinationID:       destinationID,
		DestinationHost:     host,
		DestinationWorkdir:  workdir,
		DestinationDomain:   domain,
		BundleSHA256:        bundleSHA,
		ReleaseDigest:       strings.TrimSpace(rel.ReleaseDigest),
		ConfigurationDigest: strings.TrimSpace(rel.ConfigurationDigest),
		TargetKey:           targetKey,
		OperationID:         operationID,
		Outcome:             "deployed",
		Health:              "healthy",
		ExternalReceipt:     "scenario-to-cloud:" + deployment.ID,
		ObservedAt:          observedAt.UTC().Format(time.RFC3339Nano),
		ProducerRef:         ReceiptProducerRef,
	}, nil
}

func destinationIdentity(scenarioID, host, workdir, domain string) string {
	canonical := strings.Join([]string{scenarioID, host, workdir, domain}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Valid reports whether the receipt has enough owner evidence to be consumed
// as a successful cloud effect.
func (r CloudDeploymentReceipt) Valid() bool {
	return r.SchemaVersion == 1 && r.DeploymentID != "" && r.ScenarioID != "" &&
		r.TargetKind == "vps" && r.DestinationID != "" && r.DestinationHost != "" &&
		r.DestinationWorkdir != "" && r.DestinationDomain != "" && r.BundleSHA256 != "" && r.TargetKey != "" &&
		r.Outcome == "deployed" && r.Health == "healthy" && r.ExternalReceipt != "" && r.ObservedAt != "" && r.ProducerRef != ""
}
