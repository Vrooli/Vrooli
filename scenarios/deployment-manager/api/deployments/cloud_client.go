package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/receiptsigning"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"
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
	Name            string
	Manifest        json.RawMessage
	BundlePath      string
	BundleSHA256    string
	BundleSizeBytes int64
	RunPreflight    bool
	// ExpectedReleaseDigest binds the receipt to one immutable release when
	// the caller knows it; empty accepts whichever release the bundle maps to.
	ExpectedReleaseDigest string
	// Review is the exact readiness identity DM governs the release under.
	// scenario-to-cloud evaluates its capability profile against it and
	// reports coverage back; DM remains the only approver.
	Review *CloudReviewIdentity
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
	httpClient   *http.Client
	baseURL      string
	log          func(string, map[string]interface{})
	pollInterval time.Duration
	// receiptVerifier is the independent consumer-side trust root for
	// scenario-to-cloud receipts. Direct struct construction is retained for
	// lightweight tests and legacy callers; production constructors require
	// verification and fail closed when the verifier is unavailable.
	receiptVerifier           receiptsigning.ReceiptSigner
	receiptVerificationStrict bool
}

const cloudDeploymentPollInterval = time.Second
const cloudReceiptSigningIdentity = "scenario-to-cloud/receipt-signing"

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
	baseURL, err := discovery.ResolveScenarioURLDefault(context.Background(), "scenario-to-cloud")
	if err != nil {
		return nil, fmt.Errorf("resolve scenario-to-cloud URL: %w", err)
	}
	return &HTTPCloudHealthClient{
		httpClient:                &http.Client{Timeout: 30 * time.Second},
		baseURL:                   baseURL,
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
	var created struct {
		Deployment struct {
			ID string `json:"id"`
		} `json:"deployment"`
	}
	if err := c.cloudJSON(ctx, http.MethodPost, "/api/v1/deployments", createPayload, http.StatusCreated, &created); err != nil {
		return nil, fmt.Errorf("create cloud deployment: %w", err)
	}
	if strings.TrimSpace(created.Deployment.ID) == "" {
		return nil, fmt.Errorf("create cloud deployment returned no deployment id")
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := c.cloudJSON(ctx, http.MethodPost, "/api/v1/deployments/"+url.PathEscape(created.Deployment.ID)+"/execute", map[string]interface{}{
		"run_preflight": request.RunPreflight,
	}, http.StatusAccepted, &started); err != nil {
		return nil, fmt.Errorf("start cloud deployment %s: %w", created.Deployment.ID, err)
	}

	pollInterval := c.pollInterval
	if pollInterval <= 0 {
		pollInterval = cloudDeploymentPollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		status, err := c.cloudDeploymentStatus(ctx, created.Deployment.ID)
		if err != nil {
			return nil, err
		}
		switch status.Status {
		case "failed", "stopped":
			return nil, fmt.Errorf("cloud deployment %s ended %s: %s", created.Deployment.ID, status.Status, status.ErrorMessage)
		case "deployed":
			var envelope struct {
				Receipt CloudDeploymentReceipt `json:"receipt"`
			}
			receiptPath := "/api/v1/deployments/" + url.PathEscape(created.Deployment.ID) + "/receipt"
			if expected := strings.TrimSpace(request.ExpectedReleaseDigest); expected != "" {
				receiptPath += "?release_digest=" + url.QueryEscape(expected)
			}
			if err := c.cloudJSON(ctx, http.MethodGet, receiptPath, nil, http.StatusOK, &envelope); err != nil {
				return nil, fmt.Errorf("read cloud deployment receipt %s: %w", created.Deployment.ID, err)
			}
			if err := validateCloudReceipt(envelope.Receipt, created.Deployment.ID, request); err != nil {
				return nil, err
			}
			if c.receiptVerificationStrict {
				if c.receiptVerifier == nil {
					return nil, fmt.Errorf("cloud deployment %s receipt verifier is unavailable", created.Deployment.ID)
				}
				if err := envelope.Receipt.VerifySignature(ctx, c.receiptVerifier); err != nil {
					return nil, fmt.Errorf("cloud deployment %s receipt signature is invalid: %w", created.Deployment.ID, err)
				}
			}
			// The receipt's health field is derived from the deployment
			// record's status; only the typed observation proves the
			// target is running the delivered release now.
			health, err := c.CheckDeploymentHealth(ctx, DeploymentHealthRequest{
				DeploymentID:          created.Deployment.ID,
				ExpectedReleaseDigest: envelope.Receipt.BundleSHA256,
			})
			if err != nil {
				return nil, fmt.Errorf("observe cloud deployment %s health: %w", created.Deployment.ID, err)
			}
			if !health.Healthy {
				return nil, fmt.Errorf("cloud deployment %s receipt is not backed by a healthy current observation: %s", created.Deployment.ID, health.Explain())
			}
			receipt := envelope.Receipt
			receipt.Evidence, receipt.Publication = c.cloudEvidence(ctx, created.Deployment.ID, receipt.ReleaseDigest)
			return &receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for cloud deployment %s: %w", created.Deployment.ID, ctx.Err())
		case <-ticker.C:
		}
	}
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
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
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
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if out != nil && len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
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
