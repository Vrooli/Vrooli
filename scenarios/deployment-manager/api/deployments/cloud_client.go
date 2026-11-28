package deployments

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"deployment-manager/releases"
	"deployment-manager/shared"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/receiptsigning"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"
	s2cdeploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	s2cdeploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
)

// CloudHealthClient checks the health of one exact cloud deployment for the
// orchestrator. The deploy pipeline uses it to refuse promotion when the
// deployment is not proven healthy, current and running the expected release.
// Health interpretation lives in cloud_health.go; nothing else in DM may
// decide "healthy" from a cloud response.
type CloudHealthClient interface {
	CheckDeploymentHealth(ctx context.Context, request DeploymentHealthRequest) (*CloudHealthResult, error)
}

// CloudDeploymentRequest is the deployment intent DM passes to
// scenario-to-cloud. Secrets are intentionally absent: scenario-to-cloud and
// its credential owner resolve those values at the deployment boundary.
type CloudDeploymentRequest struct {
	RequestKey           string
	ExistingDeploymentID string
	ExistingOperationID  string
	OnProgress           func(CloudStepProgress) error
	Name                 string
	Manifest             json.RawMessage
	BundlePath           string
	BundleSHA256         string
	BundleSizeBytes      int64
	RunPreflight         bool
	// ExpectedReleaseDigest binds the receipt to one immutable release when
	// the caller knows it; empty accepts whichever release the bundle maps to.
	ExpectedReleaseDigest string
	// Review is the exact readiness identity DM governs the release under.
	// scenario-to-cloud evaluates its capability profile against it and
	// reports coverage back; DM remains the only approver.
	Review *CloudReviewIdentity
}

// CloudStepProgress describes a durable transition in DM's S2C hand-off.
type CloudStepProgress struct {
	State                    string
	DeploymentID             string
	OwnerOperationID         string
	Reason                   string
	Receipt                  json.RawMessage
	ManagedDNSState          string
	ManagedDNSReason         string
	ManagedDNSEvidenceSHA256 string
}

// CloudReviewIdentity mirrors the readiness review identity for the cloud
// owner: source, artifact, target set, channel and policy version, plus the
// release-bound extras when the review is release-bound.
type CloudReviewIdentity struct {
	ReviewKey             string   `json:"review_key,omitempty"`
	ProfileID             string   `json:"profile_id"`
	CandidateCommit       string   `json:"candidate_commit"`
	ArtifactDigest        string   `json:"artifact_digest"`
	Targets               []string `json:"targets,omitempty"`
	Channel               string   `json:"channel"`
	PolicyVersion         int      `json:"policy_version"`
	CandidateID           string   `json:"candidate_id,omitempty"`
	DestinationRevisionID string   `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch    uint64   `json:"authorization_epoch,omitempty"`
}

// CloudEvidenceCell is one capability-profile cell as the cloud owner
// reports it. Disposition keeps the shared vocabulary (passed, failed,
// skipped, unsupported, unavailable, missing); only "passed" satisfies.
type CloudEvidenceCell struct {
	CaseID        string    `json:"case_id"`
	Lane          string    `json:"lane"`
	Disposition   string    `json:"disposition"`
	Required      bool      `json:"required"`
	RecordID      string    `json:"record_id,omitempty"`
	OperationID   string    `json:"operation_id,omitempty"`
	ObservationID string    `json:"observation_id,omitempty"`
	ReceiptRefs   []string  `json:"receipt_refs,omitempty"`
	ObservedAt    time.Time `json:"observed_at,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

// CloudEvidenceSummary is the cloud owner's per-cell evaluation of one
// release on one target. DM consumes the dispositions; it never recomputes
// them and never treats an absent summary as a pass.
type CloudEvidenceSummary struct {
	SchemaVersion string              `json:"schema_version"`
	ProfileID     string              `json:"profile_id"`
	ReleaseDigest string              `json:"release_digest"`
	TargetKey     string              `json:"target_key"`
	Cells         []CloudEvidenceCell `json:"cells"`
	RequiredCells int                 `json:"required_cells"`
	Passed        bool                `json:"passed"`
	BlockingCells []string            `json:"blocking_cells,omitempty"`
	ProducerRef   string              `json:"producer_ref"`
	EvaluatedAt   time.Time           `json:"evaluated_at"`
}

// CloudTargetReceipt is the target's own active-release pointer as the
// cloud owner observed it after activation.
type CloudTargetReceipt struct {
	ActiveRelease   string `json:"active_release"`
	PreviousRelease string `json:"previous_release,omitempty"`
	ActivatedAt     string `json:"activated_at,omitempty"`
	OperationID     string `json:"operation_id,omitempty"`
	Fence           uint64 `json:"fence,omitempty"`
	ReceiptDigest   string `json:"receipt_digest,omitempty"`
}

// CloudPublication is the cloud owner's governed publication record.
type CloudPublication struct {
	ID                       string              `json:"id"`
	RequestKey               string              `json:"request_key"`
	ReviewRef                string              `json:"review_ref,omitempty"`
	ReleaseDigest            string              `json:"release_digest"`
	State                    string              `json:"state"`
	OperationID              string              `json:"operation_id,omitempty"`
	ActivatedReleaseDigest   string              `json:"activated_release_digest,omitempty"`
	PredecessorReleaseDigest string              `json:"predecessor_release_digest,omitempty"`
	TargetKey                string              `json:"target_key"`
	TargetReceipt            *CloudTargetReceipt `json:"target_receipt,omitempty"`
}

// CloudDeploymentReceipt is the durable external-effect proof returned by
// scenario-to-cloud. A deployment record or an HTTP 202 is not sufficient.
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
	// ReleaseDigest, ConfigurationDigest, TargetKey and OperationID bind the
	// receipt to the immutable release, the exact target and the producing
	// operation. ProducerRef is the emitting service principal.
	ReleaseDigest       string          `json:"release_digest,omitempty"`
	ConfigurationDigest string          `json:"configuration_digest,omitempty"`
	TargetKey           string          `json:"target_key"`
	OperationID         string          `json:"operation_id,omitempty"`
	Outcome             string          `json:"outcome"`
	Health              string          `json:"health"`
	ExternalReceipt     string          `json:"external_receipt"`
	ObservedAt          time.Time       `json:"observed_at"`
	ProducerRef         string          `json:"producer_ref"`
	Signature           json.RawMessage `json:"signature,omitempty"`
	// Evidence and Publication are read from the owner after the receipt
	// validates. They are not part of the receipt bytes.
	Evidence    *CloudEvidenceSummary `json:"-"`
	Publication *CloudPublication     `json:"-"`
}

// CanonicalJSON returns the exact payload scenario-to-cloud signs: the
// receipt fields with the signature removed and JSON object keys normalized.
// The wire signature remains raw JSON so this package does not couple its
// transport model to the producer's signing implementation.
func (r CloudDeploymentReceipt) CanonicalJSON() ([]byte, error) {
	r.Signature = nil
	r.Evidence, r.Publication = nil, nil
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

// VerifySignature independently verifies the producer attestation. An
// unsigned receipt is never considered an acceptable deployment proof.
func (r CloudDeploymentReceipt) VerifySignature(ctx context.Context, verifier receiptsigning.ReceiptSigner) error {
	if len(bytes.TrimSpace(r.Signature)) == 0 {
		return fmt.Errorf("receipt carries no producer signature")
	}
	if verifier == nil {
		return fmt.Errorf("receipt verifier is unavailable")
	}
	var envelope receiptsigning.SignatureEnvelope
	if err := json.Unmarshal(r.Signature, &envelope); err != nil {
		return fmt.Errorf("decode receipt signature: %w", err)
	}
	if err := envelope.Validate(); err != nil {
		return fmt.Errorf("invalid receipt signature envelope: %w", err)
	}
	if envelope.Purpose != receiptsigning.PurposeCloudEvidenceReceipt {
		return fmt.Errorf("receipt signature purpose %q is not %q", envelope.Purpose, receiptsigning.PurposeCloudEvidenceReceipt)
	}
	canonical, err := r.CanonicalJSON()
	if err != nil {
		return err
	}
	return verifier.Verify(ctx, envelope, canonical)
}

// cloudReceiptProducerRef is the only producer DM accepts a cloud receipt from.
const cloudReceiptProducerRef = "scenario-to-cloud"

// cloudReceiptMaxAge bounds a receipt's observed_at against the DM clock. A
// deployment run lasts minutes; a receipt observed hours ago describes some
// earlier effect and must be re-observed.
const cloudReceiptMaxAge = 2 * time.Hour

// CloudDeploymentClient owns only the typed handoff. scenario-to-cloud owns
// VPS setup, SSH, secrets, deployment state, recovery, and the receipt.
type CloudDeploymentClient interface {
	DeployCloud(ctx context.Context, request *CloudDeploymentRequest) (*CloudDeploymentReceipt, error)
}

type CloudRecoveryRequest struct {
	DeploymentID      string
	Action            string
	ExpectedBundleSHA string
	RepairBundleSHA   string
	DataCompatibility string
	IdempotencyKey    string
	Confirmation      string
	DryRun            bool
	// ReviewKey and ReleaseDigest bind the recovery to the governed release;
	// PreviewRef is the dry-run reference the owner issued and requires
	// before it performs the effect.
	ReviewKey     string
	ReleaseDigest string
	PreviewRef    string
}

type CloudRecoveryReceipt struct {
	SchemaVersion   int       `json:"schema_version"`
	DeploymentID    string    `json:"deployment_id"`
	OperationID     string    `json:"operation_id,omitempty"`
	Action          string    `json:"action"`
	Outcome         string    `json:"outcome"`
	Health          string    `json:"health"`
	BundleSHA256    string    `json:"bundle_sha256,omitempty"`
	ExternalReceipt string    `json:"external_receipt"`
	ObservedAt      time.Time `json:"observed_at"`
	DryRun          bool      `json:"dry_run"`
	Error           string    `json:"error,omitempty"`
	// PreviewRef is the dry-run reference; execution must present it.
	PreviewRef string `json:"preview_ref,omitempty"`
	// ReviewKey and RouteKind are the binding and route the owner recorded.
	ReviewKey string `json:"review_key,omitempty"`
	RouteKind string `json:"route_kind,omitempty"`
}

type CloudRecoveryClient interface {
	RecoverCloud(ctx context.Context, request *CloudRecoveryRequest) (*CloudRecoveryReceipt, error)
}

// HTTPCloudHealthClient calls scenario-to-cloud's health endpoint.
type HTTPCloudHealthClient struct {
	httpClient     *http.Client
	baseURL        string
	resolveBaseURL func(context.Context) (string, error)
	log            func(string, map[string]interface{})
	pollInterval   time.Duration
	// receiptVerifier is the independent consumer-side trust root for
	// scenario-to-cloud receipts. Direct struct construction is retained for
	// lightweight tests and legacy callers; production constructors require
	// verification and fail closed when the verifier is unavailable.
	receiptVerifier           receiptsigning.ReceiptSigner
	receiptVerificationStrict bool
}

const cloudDeploymentPollInterval = time.Second
const cloudReceiptSigningIdentity = "scenario-to-cloud/receipt-signing"

// ErrCloudOperationAmbiguous means S2C admission may have committed but its
// owner read-back was unavailable. Callers must leave the release resumable
// for reconciliation instead of recording an ordinary failure.
var ErrCloudOperationAmbiguous = errors.New("scenario-to-cloud operation outcome is ambiguous")

// NewHTTPCloudHealthClient constructs the default cloud health client through
// scenario discovery.
func NewHTTPCloudHealthClient(log func(string, map[string]interface{})) (*HTTPCloudHealthClient, error) {
	verifier, err := newCloudReceiptVerifier()
	if err != nil && log != nil {
		log("cloud receipt verification unavailable", map[string]interface{}{"error": err.Error()})
	}
	if configured := strings.TrimSpace(os.Getenv("SCENARIO_TO_CLOUD_URL")); configured != "" {
		client := NewHTTPCloudHealthClientAt(configured, log)
		client.receiptVerifier = verifier
		client.receiptVerificationStrict = true
		return client, nil
	}
	return &HTTPCloudHealthClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		resolveBaseURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "scenario-to-cloud")
		},
		log:                       log,
		pollInterval:              cloudDeploymentPollInterval,
		receiptVerifier:           verifier,
		receiptVerificationStrict: true,
	}, nil
}

// NewHTTPCloudHealthClientAt is a deterministic constructor for unit tests
// and explicitly supplied callers; production discovery remains the default.
func NewHTTPCloudHealthClientAt(baseURL string, log func(string, map[string]interface{})) *HTTPCloudHealthClient {
	return &HTTPCloudHealthClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      strings.TrimRight(baseURL, "/"),
		log:          log,
		pollInterval: cloudDeploymentPollInterval,
	}
}

// newCloudReceiptVerifier loads the same credential-authority trust root used
// by scenario-to-cloud's production receipt signer. Key material stays inside
// the authority-backed signer; it never enters this client.
func newCloudReceiptVerifier() (receiptsigning.ReceiptSigner, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	return credentialauthoritysigning.New(credentialauthoritysigning.Config{
		Identity:        credentialauthority.Identity(cloudReceiptSigningIdentity),
		Store:           authority,
		AllowedPurposes: []receiptsigning.Purpose{receiptsigning.PurposeCloudEvidenceReceipt},
	})
}

// DeployCloud creates a scenario-to-cloud deployment, starts its server-owned
// pipeline, and waits for the durable receipt. It never treats connection or
// an accepted execution request as deployment success.
func (c *HTTPCloudHealthClient) DeployCloud(ctx context.Context, request *CloudDeploymentRequest) (*CloudDeploymentReceipt, error) {
	if request == nil || len(bytes.TrimSpace(request.Manifest)) == 0 {
		return nil, fmt.Errorf("cloud deployment manifest is required")
	}
	dnsConfigured := managedDNSConfigured(request.Manifest)
	createPayload := map[string]interface{}{
		"name":              request.Name,
		"manifest":          json.RawMessage(request.Manifest),
		"bundle_path":       request.BundlePath,
		"bundle_sha256":     request.BundleSHA256,
		"bundle_size_bytes": request.BundleSizeBytes,
	}
	if request.Review != nil {
		createPayload["review"] = request.Review
	}
	deploymentID := strings.TrimSpace(request.ExistingDeploymentID)
	if deploymentID == "" {
		var created struct {
			Deployment struct {
				ID string `json:"id"`
			} `json:"deployment"`
		}
		if err := c.cloudJSON(ctx, http.MethodPost, "/api/v1/deployments", createPayload, http.StatusCreated, &created); err != nil {
			return nil, fmt.Errorf("create cloud deployment: %w", err)
		}
		deploymentID = strings.TrimSpace(created.Deployment.ID)
		if deploymentID == "" {
			return nil, fmt.Errorf("create cloud deployment returned no deployment id")
		}
		if err := c.progress(request, CloudStepProgress{State: "deployment_ready", DeploymentID: deploymentID}); err != nil {
			return nil, fmt.Errorf("persist cloud deployment identity: %w", err)
		}
	}
	ownerOperationID := strings.TrimSpace(request.ExistingOperationID)
	if ownerOperationID == "" {
		// Read by deployment first. This reconciles an accepted S2C operation
		// whose 202 response was lost, before any retry can submit work again.
		foundID, lookupErr := c.findCloudOperation(ctx, deploymentID, request.RequestKey)
		if lookupErr != nil {
			return nil, fmt.Errorf("reconcile cloud operation: %w", lookupErr)
		}
		ownerOperationID = foundID
	}
	if ownerOperationID == "" {
		var started map[string]interface{}
		err := c.cloudJSON(ctx, http.MethodPost, "/api/v1/deployments/"+url.PathEscape(deploymentID)+"/execute", map[string]interface{}{
			"run_preflight": request.RunPreflight, "request_key": request.RequestKey,
		}, http.StatusAccepted, &started)
		if err != nil {
			// The admission transaction may have committed before the response
			// was lost. Query by the same stable key before declaring ambiguity.
			foundID, lookupErr := c.findCloudOperation(ctx, deploymentID, request.RequestKey)
			if lookupErr != nil {
				_ = c.progress(request, CloudStepProgress{State: "reconciliation_required", DeploymentID: deploymentID, Reason: lookupErr.Error()})
				return nil, fmt.Errorf("%w: execute response failed (%v), owner read-back failed: %v", ErrCloudOperationAmbiguous, err, lookupErr)
			}
			ownerOperationID = foundID
			if ownerOperationID == "" {
				return nil, fmt.Errorf("start cloud deployment %s: %w", deploymentID, err)
			}
		} else if id, _ := started["operation_id"].(string); strings.TrimSpace(id) != "" {
			ownerOperationID = strings.TrimSpace(id)
		}
		if ownerOperationID == "" {
			return nil, fmt.Errorf("start cloud deployment %s returned no operation identity", deploymentID)
		}
	}
	if err := c.progress(request, CloudStepProgress{State: "operation_admitted", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID}); err != nil {
		return nil, fmt.Errorf("persist cloud operation identity: %w", err)
	}
	// A resumed DM worker must reattach to the exact S2C operation it saved,
	// not infer that operation's state from the mutable deployment summary.
	// This read is observational and happens before any further polling or
	// receipt acceptance.
	treatAsReattach := strings.TrimSpace(request.ExistingOperationID) != ""
	observeManagedDNS := func() error {
		if !dnsConfigured {
			return nil
		}
		standing, err := c.cloudOperationStanding(ctx, ownerOperationID)
		if err != nil {
			_ = c.progress(request, CloudStepProgress{State: "reconciliation_required", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID, ManagedDNSState: "ambiguous", ManagedDNSReason: "S2C managed DNS standing could not be read: " + err.Error()})
			return fmt.Errorf("%w: read managed DNS step standing: %v", ErrCloudOperationAmbiguous, err)
		}
		if standing.OperationID != ownerOperationID || standing.DeploymentID != deploymentID || standing.RequestKey != request.RequestKey {
			_ = c.progress(request, CloudStepProgress{State: "reconciliation_required", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID, ManagedDNSState: "ambiguous", ManagedDNSReason: "S2C operation identity does not match the saved deployment and request key", ManagedDNSEvidenceSHA256: managedDNSStandingDigest(standing)})
			return fmt.Errorf("%w: S2C managed DNS standing identity does not match the saved request", ErrCloudOperationAmbiguous)
		}
		projection := projectManagedDNSStanding(standing)
		if projection.EvidenceSHA256 == "" {
			projection.EvidenceSHA256 = managedDNSStandingDigest(standing)
		}
		if err := c.progress(request, CloudStepProgress{State: "operation_admitted", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID, ManagedDNSState: projection.State, ManagedDNSReason: projection.Reason, ManagedDNSEvidenceSHA256: projection.EvidenceSHA256}); err != nil {
			return fmt.Errorf("persist managed DNS read-back: %w", err)
		}
		return nil
	}
	if treatAsReattach {
		standing, err := c.cloudOperationStanding(ctx, ownerOperationID)
		if err != nil {
			_ = c.progress(request, CloudStepProgress{State: "reconciliation_required", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID, Reason: err.Error()})
			return nil, fmt.Errorf("%w: read persisted cloud operation: %v", ErrCloudOperationAmbiguous, err)
		}
		if standing.OperationID != ownerOperationID || standing.DeploymentID != deploymentID || standing.RequestKey != request.RequestKey {
			return nil, fmt.Errorf("%w: persisted cloud operation identity does not match the saved request", ErrCloudOperationAmbiguous)
		}
		if err := observeManagedDNS(); err != nil {
			return nil, err
		}
		switch standing.State {
		case "failed", "failed_recovery", "cancelled":
			return nil, fmt.Errorf("cloud operation %s ended %s: %s", ownerOperationID, standing.State, standing.ErrorMessage)
		case "admitted", "waiting_input", "running", "verifying", "reconciling", "recovering", "cancel_requested", "succeeded":
			// Keep observing through the deployment's owner-backed status and
			// receipt endpoints below. A succeeded operation still needs its
			// signed deployment receipt and health observation.
		default:
			return nil, fmt.Errorf("%w: cloud operation %s has unknown state %q", ErrCloudOperationAmbiguous, ownerOperationID, standing.State)
		}
	}

	pollInterval := c.pollInterval
	if pollInterval <= 0 {
		pollInterval = cloudDeploymentPollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if treatAsReattach {
			standing, err := c.cloudOperationStanding(ctx, ownerOperationID)
			if err != nil {
				_ = c.progress(request, CloudStepProgress{State: "reconciliation_required", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID, Reason: err.Error()})
				return nil, fmt.Errorf("%w: observe persisted cloud operation: %v", ErrCloudOperationAmbiguous, err)
			}
			if standing.OperationID != ownerOperationID || standing.DeploymentID != deploymentID || standing.RequestKey != request.RequestKey {
				return nil, fmt.Errorf("%w: observed cloud operation identity does not match the saved request", ErrCloudOperationAmbiguous)
			}
			if err := observeManagedDNS(); err != nil {
				return nil, err
			}
			switch standing.State {
			case "failed", "failed_recovery", "cancelled":
				return nil, fmt.Errorf("cloud operation %s ended %s: %s", ownerOperationID, standing.State, standing.ErrorMessage)
			case "succeeded":
				// Continue to the deployment's signed receipt and fresh health
				// read-back before DM treats the effect as complete.
			case "admitted", "waiting_input", "running", "verifying", "reconciling", "recovering", "cancel_requested":
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("wait for cloud operation %s: %w", ownerOperationID, ctx.Err())
				case <-ticker.C:
					continue
				}
			default:
				return nil, fmt.Errorf("%w: cloud operation %s has unknown state %q", ErrCloudOperationAmbiguous, ownerOperationID, standing.State)
			}
		}
		if !treatAsReattach {
			if err := observeManagedDNS(); err != nil {
				return nil, err
			}
		}
		status, err := c.cloudDeploymentStatus(ctx, deploymentID)
		if err != nil {
			return nil, err
		}
		switch status.Status {
		case "failed", "stopped":
			return nil, fmt.Errorf("cloud deployment %s ended %s: %s", deploymentID, status.Status, status.ErrorMessage)
		case "deployed":
			var envelope struct {
				Receipt CloudDeploymentReceipt `json:"receipt"`
			}
			receiptPath := "/api/v1/deployments/" + url.PathEscape(deploymentID) + "/receipt"
			if expected := strings.TrimSpace(request.ExpectedReleaseDigest); expected != "" {
				receiptPath += "?release_digest=" + url.QueryEscape(expected)
			}
			if err := c.cloudJSON(ctx, http.MethodGet, receiptPath, nil, http.StatusOK, &envelope); err != nil {
				return nil, fmt.Errorf("read cloud deployment receipt %s: %w", deploymentID, err)
			}
			if err := validateCloudReceipt(envelope.Receipt, deploymentID, request); err != nil {
				return nil, err
			}
			if c.receiptVerificationStrict {
				if c.receiptVerifier == nil {
					return nil, fmt.Errorf("cloud deployment %s receipt verifier is unavailable", deploymentID)
				}
				if err := envelope.Receipt.VerifySignature(ctx, c.receiptVerifier); err != nil {
					return nil, fmt.Errorf("cloud deployment %s receipt signature is invalid: %w", deploymentID, err)
				}
			}
			// The receipt's health field is derived from the deployment
			// record's status; only the typed observation proves the
			// target is running the delivered release now.
			health, err := c.CheckDeploymentHealth(ctx, DeploymentHealthRequest{
				DeploymentID:          deploymentID,
				ExpectedReleaseDigest: envelope.Receipt.BundleSHA256,
			})
			if err != nil {
				return nil, fmt.Errorf("observe cloud deployment %s health: %w", deploymentID, err)
			}
			if !health.Healthy {
				return nil, fmt.Errorf("cloud deployment %s receipt is not backed by a healthy current observation: %s", deploymentID, health.Explain())
			}
			receipt := envelope.Receipt
			receipt.Evidence, receipt.Publication = c.cloudEvidence(ctx, deploymentID, receipt.ReleaseDigest)
			encodedReceipt, _ := json.Marshal(struct {
				Receipt     CloudDeploymentReceipt `json:"receipt"`
				Evidence    *CloudEvidenceSummary  `json:"evidence,omitempty"`
				Publication *CloudPublication      `json:"publication,omitempty"`
			}{Receipt: receipt, Evidence: receipt.Evidence, Publication: receipt.Publication})
			if err := c.progress(request, CloudStepProgress{State: "receipt_verified", DeploymentID: deploymentID, OwnerOperationID: ownerOperationID, Receipt: encodedReceipt}); err != nil {
				return nil, fmt.Errorf("persist verified cloud receipt: %w", err)
			}
			return &receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for cloud deployment %s: %w", deploymentID, ctx.Err())
		case <-ticker.C:
		}
	}
}

type cloudOperationStanding struct {
	OperationID         string                        `json:"operation_id"`
	OperationIDCamel    string                        `json:"operationId"`
	DeploymentID        string                        `json:"deployment_id"`
	DeploymentIDCamel   string                        `json:"deploymentId"`
	RequestKey          string                        `json:"request_key"`
	RequestKeyCamel     string                        `json:"requestKey"`
	State               string                        `json:"state"`
	ErrorMessage        string                        `json:"error_message,omitempty"`
	StepReceipts        []cloudOperationStepReceipt   `json:"step_receipts,omitempty"`
	StepReceiptsCamel   []cloudOperationStepReceipt   `json:"stepReceipts,omitempty"`
	UnknownEffects      []cloudOperationUnknownEffect `json:"unknown_effects,omitempty"`
	UnknownEffectsCamel []cloudOperationUnknownEffect `json:"unknownEffects,omitempty"`
	ActiveStep          string                        `json:"active_step,omitempty"`
	ActiveStepCamel     string                        `json:"activeStep,omitempty"`
	Error               struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type cloudOperationStepReceipt struct {
	Step    string `json:"step"`
	Outcome string `json:"outcome"`
	Source  string `json:"source,omitempty"`
	Fence   uint64 `json:"fence,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Error   string `json:"error,omitempty"`
}

type cloudOperationUnknownEffect struct {
	Step   string `json:"step"`
	Reason string `json:"reason,omitempty"`
}

func (s *cloudOperationStanding) normalize() {
	if s.OperationID == "" {
		s.OperationID = s.OperationIDCamel
	}
	if s.DeploymentID == "" {
		s.DeploymentID = s.DeploymentIDCamel
	}
	if s.RequestKey == "" {
		s.RequestKey = s.RequestKeyCamel
	}
	if s.ActiveStep == "" {
		s.ActiveStep = s.ActiveStepCamel
	}
	if len(s.StepReceipts) == 0 {
		s.StepReceipts = s.StepReceiptsCamel
	}
	if len(s.UnknownEffects) == 0 {
		s.UnknownEffects = s.UnknownEffectsCamel
	}
}

func (c *HTTPCloudHealthClient) cloudOperationStanding(ctx context.Context, operationID string) (cloudOperationStanding, error) {
	var standing cloudOperationStanding
	path := "/vrooli.scenario_to_cloud.v1.operations.OperationsService/GetOperation"
	if err := c.cloudJSON(ctx, http.MethodPost, path, map[string]string{"operation_id": operationID}, http.StatusOK, &standing); err != nil {
		return standing, err
	}
	standing.normalize()
	if standing.ErrorMessage == "" {
		standing.ErrorMessage = standing.Error.Message
	}
	return standing, nil
}

type managedDNSProjection struct {
	State          string
	Reason         string
	EvidenceSHA256 string
}

func managedDNSConfigured(manifest json.RawMessage) bool {
	var value struct {
		Edge struct {
			ManagedDNSProfile string `json:"managed_dns_profile"`
		} `json:"edge"`
	}
	if json.Unmarshal(manifest, &value) != nil {
		return false
	}
	return strings.TrimSpace(value.Edge.ManagedDNSProfile) != ""
}

func projectManagedDNSStanding(standing cloudOperationStanding) managedDNSProjection {
	const stepID = "edge.dns.ensure"
	var receipts []cloudOperationStepReceipt
	var unknown []cloudOperationUnknownEffect
	for _, receipt := range standing.StepReceipts {
		if receipt.Step == stepID {
			receipts = append(receipts, receipt)
		}
	}
	for _, effect := range standing.UnknownEffects {
		if effect.Step == stepID {
			unknown = append(unknown, effect)
		}
	}
	if len(receipts) > 1 || (len(receipts) > 0 && len(unknown) > 0) {
		return dnsProjection("ambiguous", "S2C returned contradictory managed DNS step evidence", receipts, unknown)
	}
	if len(unknown) > 0 {
		return dnsProjection("ambiguous", "S2C could not determine whether the managed DNS effect completed", nil, unknown)
	}
	if len(receipts) == 1 {
		receipt := receipts[0]
		switch receipt.Outcome {
		case "succeeded", "unchanged":
			return dnsProjection("succeeded", "S2C committed the managed DNS step receipt", receipts, nil)
		case "failed":
			return dnsProjection("failed", nonEmptyOr(receipt.Error, "S2C committed a failed managed DNS step"), receipts, nil)
		case "unknown":
			return dnsProjection("ambiguous", nonEmptyOr(receipt.Error, "S2C marked the managed DNS outcome unknown"), receipts, nil)
		default:
			return dnsProjection("ambiguous", "S2C returned an unsupported managed DNS step outcome", receipts, nil)
		}
	}
	if standing.ActiveStep == stepID || (standing.State != "succeeded" && standing.State != "failed" && standing.State != "failed_recovery" && standing.State != "cancelled") {
		return managedDNSProjection{State: "pending", Reason: "S2C has not committed a managed DNS step receipt"}
	}
	if standing.State == "failed" || standing.State == "failed_recovery" || standing.State == "cancelled" {
		return managedDNSProjection{State: "not_reached", Reason: "S2C ended before committing a managed DNS step receipt"}
	}
	return managedDNSProjection{State: "ambiguous", Reason: "S2C operation succeeded without the configured managed DNS step receipt"}
}

func nonEmptyOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func dnsProjection(state, reason string, receipts []cloudOperationStepReceipt, unknown []cloudOperationUnknownEffect) managedDNSProjection {
	value := struct {
		Receipts []cloudOperationStepReceipt   `json:"receipts,omitempty"`
		Unknown  []cloudOperationUnknownEffect `json:"unknown_effects,omitempty"`
	}{receipts, unknown}
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return managedDNSProjection{State: state, Reason: reason, EvidenceSHA256: "sha256:" + hex.EncodeToString(digest[:])}
}

func managedDNSStandingDigest(standing cloudOperationStanding) string {
	value := struct {
		OperationID    string                        `json:"operation_id"`
		DeploymentID   string                        `json:"deployment_id"`
		RequestKey     string                        `json:"request_key"`
		State          string                        `json:"state"`
		ActiveStep     string                        `json:"active_step,omitempty"`
		StepReceipts   []cloudOperationStepReceipt   `json:"step_receipts,omitempty"`
		UnknownEffects []cloudOperationUnknownEffect `json:"unknown_effects,omitempty"`
	}{standing.OperationID, standing.DeploymentID, standing.RequestKey, standing.State, standing.ActiveStep, standing.StepReceipts, standing.UnknownEffects}
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func mergeManagedDNSReadback(oldState, oldReason, oldDigest, newState, newReason, newDigest string) (string, string, string) {
	if oldState == "" || oldState == "pending" || oldState == "ambiguous" {
		return newState, newReason, newDigest
	}
	if oldState == newState && (oldDigest == "" || oldDigest == newDigest) {
		return newState, newReason, newDigest
	}
	if (oldState == "succeeded" || oldState == "failed") && oldState == newState && oldDigest == newDigest {
		return oldState, oldReason, oldDigest
	}
	return "ambiguous", "new S2C DNS evidence conflicts with previously persisted " + oldState + " standing", newDigest
}

func (c *HTTPCloudHealthClient) progress(request *CloudDeploymentRequest, update CloudStepProgress) error {
	if request.OnProgress == nil {
		return nil
	}
	return request.OnProgress(update)
}

func (c *HTTPCloudHealthClient) findCloudOperation(ctx context.Context, deploymentID, requestKey string) (string, error) {
	if strings.TrimSpace(requestKey) == "" {
		return "", fmt.Errorf("stable request key is required")
	}
	var response struct {
		Operations []struct {
			OperationID      string `json:"operation_id"`
			OperationIDCamel string `json:"operationId"`
			RequestKey       string `json:"request_key"`
			RequestKeyCamel  string `json:"requestKey"`
		} `json:"operations"`
	}
	path := "/vrooli.scenario_to_cloud.v1.operations.OperationsService/ListDeploymentOperations"
	if err := c.cloudJSON(ctx, http.MethodPost, path, map[string]string{"deployment_id": deploymentID}, http.StatusOK, &response); err != nil {
		return "", err
	}
	for _, op := range response.Operations {
		key := op.RequestKey
		if key == "" {
			key = op.RequestKeyCamel
		}
		if key == requestKey {
			id := op.OperationID
			if id == "" {
				id = op.OperationIDCamel
			}
			if id == "" {
				return "", fmt.Errorf("matching cloud operation has no identity")
			}
			return id, nil
		}
	}
	return "", nil
}

// cloudEvidence reads the owner's per-cell dispositions and publication for
// the deployed release. Absence is returned as nil so the caller decides
// (it refuses promotion); it is never synthesised into a pass.
func (c *HTTPCloudHealthClient) cloudEvidence(ctx context.Context, deploymentID, releaseDigest string) (*CloudEvidenceSummary, *CloudPublication) {
	var summary *CloudEvidenceSummary
	if strings.TrimSpace(releaseDigest) != "" {
		var envelope struct {
			Evidence *CloudEvidenceSummary `json:"evidence"`
		}
		path := "/api/v1/deployments/" + url.PathEscape(deploymentID) + "/evidence?release_digest=" + url.QueryEscape(releaseDigest)
		if err := c.cloudJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &envelope); err == nil {
			summary = envelope.Evidence
		} else if c.log != nil {
			c.log("cloud evidence unavailable", map[string]interface{}{"deployment_id": deploymentID, "error": err.Error()})
		}
	}
	var publication *CloudPublication
	var pubEnvelope struct {
		Publication *CloudPublication `json:"publication"`
	}
	if err := c.cloudJSON(ctx, http.MethodGet, "/api/v1/deployments/"+url.PathEscape(deploymentID)+"/publication", nil, http.StatusOK, &pubEnvelope); err == nil {
		publication = pubEnvelope.Publication
	}
	return summary, publication
}

func (c *HTTPCloudHealthClient) RecoverCloud(ctx context.Context, request *CloudRecoveryRequest) (*CloudRecoveryReceipt, error) {
	if request == nil || strings.TrimSpace(request.DeploymentID) == "" || strings.TrimSpace(request.Action) == "" {
		return nil, fmt.Errorf("cloud recovery requires deployment and action identity")
	}
	payload := map[string]interface{}{
		"action":                 request.Action,
		"expected_bundle_sha256": request.ExpectedBundleSHA,
		"repair_bundle_sha256":   request.RepairBundleSHA,
		"data_compatibility":     request.DataCompatibility,
		"idempotency_key":        request.IdempotencyKey,
		"confirmation":           request.Confirmation,
		"dry_run":                request.DryRun,
		"review_ref":             request.ReviewKey,
		"release_digest":         request.ReleaseDigest,
		"preview_ref":            request.PreviewRef,
	}
	var envelope struct {
		Receipt CloudRecoveryReceipt `json:"receipt"`
	}
	if err := c.cloudJSON(ctx, http.MethodPost, "/api/v1/deployments/"+url.PathEscape(request.DeploymentID)+"/recovery", payload, http.StatusOK, &envelope); err != nil {
		return nil, fmt.Errorf("recover cloud deployment %s: %w", request.DeploymentID, err)
	}
	if envelope.Receipt.DeploymentID != request.DeploymentID || envelope.Receipt.Action != request.Action {
		return nil, fmt.Errorf("cloud recovery receipt does not match requested identity")
	}
	if request.DryRun {
		if !envelope.Receipt.DryRun || envelope.Receipt.Outcome != "preview" {
			return nil, fmt.Errorf("cloud recovery preview returned an invalid receipt")
		}
		return &envelope.Receipt, nil
	}
	if envelope.Receipt.Outcome == "pending" || envelope.Receipt.Outcome == "running" {
		if strings.TrimSpace(envelope.Receipt.OperationID) == "" {
			return nil, fmt.Errorf("cloud recovery returned an in-progress receipt without an operation identity")
		}
		return c.waitForCloudRecovery(ctx, request, envelope.Receipt.OperationID)
	}
	if envelope.Receipt.SchemaVersion != 1 || !validCloudRecoveryOutcome(request.Action, envelope.Receipt) || envelope.Receipt.ExternalReceipt == "" || envelope.Receipt.ObservedAt.IsZero() || envelope.Receipt.DryRun {
		return nil, fmt.Errorf("cloud recovery returned an incomplete effect receipt")
	}
	return &envelope.Receipt, nil
}

func (c *HTTPCloudHealthClient) waitForCloudRecovery(ctx context.Context, request *CloudRecoveryRequest, operationID string) (*CloudRecoveryReceipt, error) {
	pollInterval := c.pollInterval
	if pollInterval <= 0 {
		pollInterval = cloudDeploymentPollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		var envelope struct {
			Receipt CloudRecoveryReceipt `json:"receipt"`
		}
		path := "/api/v1/deployments/" + url.PathEscape(request.DeploymentID) + "/recovery/" + url.PathEscape(operationID)
		if err := c.cloudJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &envelope); err != nil {
			return nil, fmt.Errorf("read cloud recovery operation %s: %w", operationID, err)
		}
		if envelope.Receipt.DeploymentID != request.DeploymentID || envelope.Receipt.Action != request.Action || envelope.Receipt.OperationID != operationID {
			return nil, fmt.Errorf("cloud recovery operation returned a mismatched receipt")
		}
		switch envelope.Receipt.Outcome {
		case "pending", "running":
		case "failed":
			message := envelope.Receipt.Error
			if message == "" {
				message = "cloud recovery operation failed"
			}
			return nil, fmt.Errorf("%s", message)
		default:
			if envelope.Receipt.SchemaVersion != 1 || !validCloudRecoveryOutcome(request.Action, envelope.Receipt) || envelope.Receipt.ExternalReceipt == "" || envelope.Receipt.ObservedAt.IsZero() || envelope.Receipt.DryRun {
				return nil, fmt.Errorf("cloud recovery operation returned an incomplete effect receipt")
			}
			return &envelope.Receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for cloud recovery operation %s: %w", operationID, ctx.Err())
		case <-ticker.C:
		}
	}
}

// GetCloudRecoveryOperation reads one exact human-started S2C recovery. This
// path is read-only: DM never invokes the owner's Start RPC.
func (c *HTTPCloudHealthClient) GetCloudRecoveryOperation(ctx context.Context, deploymentID, operationID string) (*releases.CloudRecoveryOperation, error) {
	client, err := c.generatedDeploymentClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetDeploymentRecoveryOperation(ctx, connect.NewRequest(&s2cdeploymentsv1.GetDeploymentRecoveryOperationRequest{DeploymentId: deploymentID, OperationId: operationID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read S2C recovery operation %q: %w", operationID, err)
	}
	return s2cRecoveryOperationFromProto(response.Msg), nil
}

// FindCloudRecoveryOperation reconciles a human-started operation whose first
// readback was lost, using only DM's persisted stable key.
func (c *HTTPCloudHealthClient) FindCloudRecoveryOperation(ctx context.Context, deploymentID, idempotencyKey string) (*releases.CloudRecoveryOperation, error) {
	client, err := c.generatedDeploymentClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.FindDeploymentRecoveryOperation(ctx, connect.NewRequest(&s2cdeploymentsv1.FindDeploymentRecoveryOperationRequest{DeploymentId: deploymentID, IdempotencyKey: idempotencyKey}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find S2C recovery by stable key: %w", err)
	}
	return s2cRecoveryOperationFromProto(response.Msg), nil
}

func (c *HTTPCloudHealthClient) generatedDeploymentClient(ctx context.Context) (s2cdeploymentsconnect.DeploymentsServiceClient, error) {
	baseURL, err := c.baseURLFor(ctx)
	if err != nil {
		return nil, err
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return s2cdeploymentsconnect.NewDeploymentsServiceClient(httpClient, strings.TrimRight(baseURL, "/")), nil
}

func s2cRecoveryOperationFromProto(value *s2cdeploymentsv1.DeploymentRecoveryOperation) *releases.CloudRecoveryOperation {
	if value == nil {
		return nil
	}
	return &releases.CloudRecoveryOperation{
		DeploymentID: value.GetDeploymentId(), OperationID: value.GetOperationId(), IdempotencyKey: value.GetIdempotencyKey(),
		Action: value.GetAction(), Status: value.GetStatus(), ExpectedCurrentBundleSHA256: value.GetExpectedCurrentBundleSha256(),
		LastGoodReleaseDigest: value.GetLastGoodReleaseDigest(), ReviewRef: value.GetReviewRef(), PreviewRef: value.GetPreviewRef(),
		DataCompatibility: value.GetDataCompatibility(), ReceiptJSON: json.RawMessage(value.GetReceiptJson()), Error: value.GetError(),
	}
}

var _ releases.CloudRecoveryOperationReader = (*HTTPCloudHealthClient)(nil)

func validCloudRecoveryOutcome(action string, receipt CloudRecoveryReceipt) bool {
	if action != "halt" && strings.TrimSpace(receipt.BundleSHA256) == "" {
		return false
	}
	switch action {
	case "halt":
		return receipt.Outcome == "halted" && receipt.Health == "stopped"
	case "rollback":
		return receipt.Outcome == "rolled_back" && receipt.Health == "healthy"
	case "forward_repair":
		return receipt.Outcome == "forward_repaired" && receipt.Health == "healthy"
	default:
		return false
	}
}

type cloudDeploymentStatus struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

func (c *HTTPCloudHealthClient) cloudDeploymentStatus(ctx context.Context, id string) (cloudDeploymentStatus, error) {
	var envelope struct {
		Deployment cloudDeploymentStatus `json:"deployment"`
	}
	if err := c.cloudJSON(ctx, http.MethodGet, "/api/v1/deployments/"+url.PathEscape(id), nil, http.StatusOK, &envelope); err != nil {
		return cloudDeploymentStatus{}, fmt.Errorf("read cloud deployment %s: %w", id, err)
	}
	return envelope.Deployment, nil
}

func (c *HTTPCloudHealthClient) cloudJSON(ctx context.Context, method, path string, payload interface{}, expectedStatus int, out interface{}) error {
	baseURL, err := c.baseURLFor(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("read response: %w", readErr)
	}
	createReplay := expectedStatus == http.StatusCreated && resp.StatusCode == http.StatusOK
	if resp.StatusCode != expectedStatus && !createReplay {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if out != nil && len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *HTTPCloudHealthClient) baseURLFor(ctx context.Context) (string, error) {
	baseURL := strings.TrimSpace(c.baseURL)
	if c.resolveBaseURL != nil {
		resolved, err := c.resolveBaseURL(ctx)
		if err != nil {
			return "", fmt.Errorf("resolve scenario-to-cloud URL: %w", err)
		}
		baseURL = strings.TrimSpace(resolved)
	}
	validated, err := shared.ValidateServiceURL(baseURL)
	if err != nil {
		return "", fmt.Errorf("scenario-to-cloud URL: %w", err)
	}
	return validated, nil
}

// cloudReceiptNow is the DM clock used for receipt freshness (tests pin it).
var cloudReceiptNow = func() time.Time { return time.Now().UTC() }

// validateCloudReceipt refuses forged, mismatched or stale receipts: the
// producer must be the cloud service principal, the receipt must be bound to
// the deployment, the bundle and (when known) the release the caller asked
// for, must name its target, and must have been observed within policy.
func validateCloudReceipt(receipt CloudDeploymentReceipt, expectedID string, request *CloudDeploymentRequest) error {
	expectedScenario := cloudManifestScenario(request.Manifest)
	if receipt.SchemaVersion != 1 || receipt.DeploymentID != expectedID || (expectedScenario != "" && receipt.ScenarioID != expectedScenario) || strings.TrimSpace(receipt.ScenarioID) == "" || strings.TrimSpace(receipt.TargetKind) != "vps" || strings.TrimSpace(receipt.DestinationID) == "" || strings.TrimSpace(receipt.DestinationHost) == "" || strings.TrimSpace(receipt.DestinationWorkdir) == "" || strings.TrimSpace(receipt.DestinationDomain) == "" || strings.TrimSpace(receipt.BundleSHA256) == "" || (strings.TrimSpace(request.BundleSHA256) != "" && !strings.EqualFold(receipt.BundleSHA256, request.BundleSHA256)) || strings.TrimSpace(receipt.Outcome) != "deployed" || strings.TrimSpace(receipt.Health) != "healthy" || strings.TrimSpace(receipt.ExternalReceipt) == "" || receipt.ObservedAt.IsZero() {
		return fmt.Errorf("cloud deployment %s returned an incomplete or unhealthy receipt", expectedID)
	}
	if receipt.ProducerRef != cloudReceiptProducerRef {
		return fmt.Errorf("cloud deployment %s receipt producer %q is not the cloud service principal", expectedID, receipt.ProducerRef)
	}
	if strings.TrimSpace(receipt.TargetKey) == "" {
		return fmt.Errorf("cloud deployment %s receipt names no target", expectedID)
	}
	if expected := strings.TrimPrefix(strings.TrimSpace(request.ExpectedReleaseDigest), "sha256:"); expected != "" && !strings.EqualFold(strings.TrimPrefix(receipt.ReleaseDigest, "sha256:"), expected) {
		return fmt.Errorf("cloud deployment %s receipt is bound to release %q, expected %q", expectedID, receipt.ReleaseDigest, request.ExpectedReleaseDigest)
	}
	if age := cloudReceiptNow().Sub(receipt.ObservedAt.UTC()); age > cloudReceiptMaxAge {
		return fmt.Errorf("cloud deployment %s receipt observed %s ago is older than the %s policy", expectedID, age.Round(time.Second), cloudReceiptMaxAge)
	}
	return nil
}

// checkCloudEvidence is the promotion gate over the owner's evidence: every
// required cell must be passed for the exact release and target the receipt
// names. Missing evidence refuses; evidence for another release or target
// is incompatible; dispositions other than passed keep their own word.
func checkCloudEvidence(receipt *CloudDeploymentReceipt) error {
	if receipt == nil {
		return fmt.Errorf("cloud receipt is required")
	}
	summary := receipt.Evidence
	if summary == nil {
		return fmt.Errorf("cloud evidence for release %q is unavailable; promotion refused", receipt.ReleaseDigest)
	}
	if summary.ProducerRef != cloudReceiptProducerRef {
		return fmt.Errorf("cloud evidence producer %q is not the cloud service principal", summary.ProducerRef)
	}
	if receipt.ReleaseDigest != "" && !strings.EqualFold(strings.TrimPrefix(summary.ReleaseDigest, "sha256:"), strings.TrimPrefix(receipt.ReleaseDigest, "sha256:")) {
		return fmt.Errorf("cloud evidence is for release %q, receipt is for %q: incompatible", summary.ReleaseDigest, receipt.ReleaseDigest)
	}
	if summary.TargetKey != "" && receipt.TargetKey != "" && summary.TargetKey != receipt.TargetKey {
		return fmt.Errorf("cloud evidence is for target %q, receipt is for %q: incompatible", summary.TargetKey, receipt.TargetKey)
	}
	if summary.RequiredCells == 0 {
		return fmt.Errorf("cloud evidence declares no required cells; promotion refused")
	}
	if !summary.Passed || len(summary.BlockingCells) > 0 {
		blocking := summary.BlockingCells
		if len(blocking) == 0 {
			for _, cell := range summary.Cells {
				if cell.Disposition != "passed" {
					blocking = append(blocking, cell.CaseID+"/"+cell.Lane+"="+cell.Disposition)
				}
			}
		}
		return fmt.Errorf("cloud evidence has %d required cell(s) not passed: %s", len(blocking), strings.Join(blocking, ", "))
	}
	for _, cell := range summary.Cells {
		if cell.Required && cell.Disposition != "passed" {
			return fmt.Errorf("cloud evidence cell %s/%s is %s", cell.CaseID, cell.Lane, cell.Disposition)
		}
	}
	return nil
}
