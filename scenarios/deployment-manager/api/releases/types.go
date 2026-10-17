// Package releases provides the deployment-manager-owned release lifecycle:
// a release record ties a publish attempt to a profile, commit, channel, and
// per-platform verification evidence.
package releases

import (
	"context"
	"encoding/json"
	"time"
)

// Release status values.
const (
	StatusPending      = "pending"
	StatusPublishing   = "publishing"
	StatusPublished    = "published"
	StatusFailed       = "failed"
	StatusAmbiguous    = "ambiguous"
	StatusSuperseded   = "superseded"
	StatusVerifyFailed = "verify_failed"
)

const (
	OperationQueued    = "queued"
	OperationRunning   = "running"
	OperationComplete  = "complete"
	OperationFailed    = "failed"
	OperationCanceled  = "canceled"
	OperationAmbiguous = "ambiguous"
)

// Operation is the durable server-owned standing for one release execution.
// External effects are never inferred from the HTTP request lifetime.
type Operation struct {
	ID             string `json:"operation_id"`
	ReleaseID      string `json:"release_id"`
	ProfileID      string `json:"profile_id"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	// RequestSnapshot is the exact owner request used for durable execution.
	// It is persisted for restart recovery but is intentionally not exposed in
	// the operator standing contract.
	RequestSnapshot json.RawMessage `json:"-"`
	Status          string          `json:"status"`
	ActiveStage     string          `json:"active_stage,omitempty"`
	Error           string          `json:"error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
}

// Release platform status values.
const (
	PlatformStatusPending      = "pending"
	PlatformStatusUploading    = "uploading"
	PlatformStatusPublished    = "published"
	PlatformStatusFailed       = "failed"
	PlatformStatusVerifyFailed = "verify_failed"
)

// Release is a canonical publish attempt for a profile at a given commit/channel.
type Release struct {
	ID                    string                `json:"id"`
	ProfileID             string                `json:"profile_id"`
	DeploymentID          string                `json:"deployment_id,omitempty"`
	ProfileVersion        int                   `json:"profile_version,omitempty"`
	GitCommitHash         string                `json:"git_commit_hash"`
	ArtifactDigest        string                `json:"artifact_digest,omitempty"`
	CandidateID           string                `json:"candidate_id,omitempty"`
	DestinationRevisionID string                `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch    uint64                `json:"authorization_epoch,omitempty"`
	IdempotencyKey        string                `json:"idempotency_key,omitempty"`
	ReadinessReviewKey    string                `json:"readiness_review_key,omitempty"`
	ReleaseVersion        string                `json:"release_version"`
	Channel               string                `json:"channel"`
	Status                string                `json:"status"`
	ReleaseNotes          string                `json:"release_notes,omitempty"`
	ReleasedBy            string                `json:"released_by,omitempty"`
	PromotedFromReleaseID string                `json:"promoted_from_release_id,omitempty"`
	ReadinessGoalRef      string                `json:"readiness_goal_ref,omitempty"`
	ApprovedAtCommit      string                `json:"approved_at_commit,omitempty"`
	VerificationEvidence  []VerificationItem    `json:"verification_evidence,omitempty"`
	Platforms             []ReleasePlatform     `json:"platforms,omitempty"`
	CreatedAt             time.Time             `json:"created_at"`
	PublishedAt           *time.Time            `json:"published_at,omitempty"`
	UpdatedAt             time.Time             `json:"updated_at"`
	RecoveryReceipts      []RecoveryReceipt     `json:"recovery_receipts,omitempty"`
	PublicationReceipts   []PublicationReceipt  `json:"publication_receipts,omitempty"`
	ClientUpdateReceipts  []ClientUpdateReceipt `json:"client_update_receipts,omitempty"`
}

// ReleaseDossier is the reviewer-facing, read-only release record. It joins
// mutable execution standing with the immutable identities and current health
// projection; missing identity records remain explicit instead of being
// silently treated as a valid release.
type ReleaseDossier struct {
	SchemaVersion int                        `json:"schema_version"`
	GeneratedAt   time.Time                  `json:"generated_at"`
	Release       *Release                   `json:"release"`
	Health        ReleaseHealth              `json:"health"`
	Candidate     *CandidateRecord           `json:"candidate,omitempty"`
	Destination   *DestinationRevisionRecord `json:"destination,omitempty"`
	Review        *ReviewRecord              `json:"review,omitempty"`
	MissingProof  []string                   `json:"missing_proof,omitempty"`
}

type RecoveryRequest struct {
	ReleaseID             string           `json:"release_id,omitempty"`
	ReviewKey             string           `json:"review_key"`
	CandidateID           string           `json:"candidate_id"`
	DestinationRevisionID string           `json:"destination_revision_id"`
	Action                string           `json:"action"`
	Confirmation          string           `json:"confirmation,omitempty"`
	DryRun                bool             `json:"dry_run,omitempty"`
	ExpectedPredecessor   int64            `json:"expected_predecessor_revision,omitempty"`
	DataCompatibility     string           `json:"data_compatibility,omitempty"`
	RepairArtifactIDs     map[string]int64 `json:"repair_artifact_ids,omitempty"`
	RepairBundleSHA256    string           `json:"repair_bundle_sha256,omitempty"`
	IdempotencyKey        string           `json:"idempotency_key,omitempty"`
	// PreviewRef is the owner's dry-run reference; a cloud owner requires the
	// reference of the preview it issued before it performs the effect.
	PreviewRef string `json:"preview_ref,omitempty"`
	ProfileID  string `json:"-"`
	Channel    string `json:"-"`
}

type RecoveryReceipt struct {
	ReceiptID             string    `json:"receipt_id"`
	ReleaseID             string    `json:"release_id"`
	CandidateID           string    `json:"candidate_id"`
	DestinationRevisionID string    `json:"destination_revision_id"`
	DeploymentID          string    `json:"deployment_id"`
	Action                string    `json:"action"`
	Outcome               string    `json:"outcome"`
	Health                string    `json:"health"`
	BundleSHA256          string    `json:"bundle_sha256,omitempty"`
	ExternalReceipt       string    `json:"external_receipt"`
	ObservedAt            time.Time `json:"observed_at"`
	DryRun                bool      `json:"dry_run"`
	// PreviewRef is returned by a dry run and must be presented to execute.
	PreviewRef string `json:"preview_ref,omitempty"`
}

func (r RecoveryReceipt) Valid() bool {
	validOutcome := r.Outcome == "halted" || r.Outcome == "withdrawn" || r.Outcome == "rolled_back" || r.Outcome == "forward_repaired"
	validHealth := r.Health == "stopped" || r.Health == "channel_restored" || r.Health == "channel_updated"
	return r.ReceiptID != "" && r.ReleaseID != "" && r.CandidateID != "" && r.DestinationRevisionID != "" && r.DeploymentID != "" && r.Action != "" && validOutcome && validHealth && r.ExternalReceipt != "" && !r.ObservedAt.IsZero() && !r.DryRun
}

type RecoveryExecutor interface {
	Recover(context.Context, *RecoveryRequest, string, string) (*RecoveryReceipt, error)
}

// RecoveryControlProvider reports the effect controls that the configured
// owner can execute for an exact destination revision. Read models use this
// seam to describe the real recovery surface without guessing from release
// status or exposing controls the owner cannot route.
type RecoveryControlProvider interface {
	RecoveryControls(context.Context, string) ([]string, error)
}

// OwnerObservation is the normalized, owner-produced standing used to
// reconcile an uncertain release. A healthy transport response is not enough;
// the observed release digest must match the exact expected owner bytes.
type OwnerObservation struct {
	DeploymentID          string
	TargetID              string
	ExpectedReleaseDigest string
	ObservedReleaseDigest string
	Healthy               bool
	ObservedAt            time.Time
	Detail                string
}

type ReleaseObservationProvider interface {
	ObserveRelease(context.Context, *Release) (*OwnerObservation, error)
}

type RecoveryReceiptRepository interface {
	RecordRecoveryReceipt(context.Context, RecoveryReceipt) error
	ListRecoveryReceipts(context.Context, string) ([]RecoveryReceipt, error)
}

// ClientUpdateReceiptRepository stores owner-produced evidence that an
// installed client relaunched the exact successor artifact. The release
// standing remains independent from this ledger so a published row cannot be
// mistaken for a successful client update.
type ClientUpdateReceiptRepository interface {
	RecordClientUpdateReceipt(context.Context, string, ClientUpdateReceipt) error
	ListClientUpdateReceipts(context.Context, string) ([]ClientUpdateReceipt, error)
}

// ReadinessRecord is the release-owned projection consumed by the deployment
// gate. Goal lifecycle remains owned by swarm-manager; commit identity remains
// owned here.
type ReadinessRecord struct {
	VerdictPresent   bool
	ReadinessGoalRef string
	ApprovedAtCommit string
	GoalClosed       bool
	Waiver           *ReadinessWaiver
}

type ReadinessApproval struct {
	Key                   string
	Scenario              string
	ProfileID             string
	CandidateCommit       string
	ArtifactDigest        string
	CandidateID           string
	DestinationRevisionID string
	AuthorizationEpoch    uint64
	EvidenceSetDigest     string
	PolicyDigest          string
	Targets               []string
	Channel               string
	PolicyVersion         int
	Status                string
	ApprovedAt            *time.Time
}

func (a ReadinessApproval) ReleaseBinding() (ReviewBinding, error) {
	return (ReviewBinding{
		CandidateID: a.CandidateID, DestinationRevisionID: a.DestinationRevisionID,
		Targets: a.Targets, Channel: a.Channel, EvidenceSetDigest: a.EvidenceSetDigest,
		PolicyDigest: a.PolicyDigest, AuthorizationEpoch: a.AuthorizationEpoch,
	}).Canonical()
}

type ReadinessWaiver struct {
	Reason string
	Actor  string
	Commit string
	At     time.Time
}

// ReleasePlatform is per-platform publish/verify state for a release.
type ReleasePlatform struct {
	ReleaseID      string     `json:"release_id"`
	Platform       string     `json:"platform"`
	Status         string     `json:"status"`
	ApprovalID     string     `json:"approval_id,omitempty"`
	LPBSArtifactID int64      `json:"lpbs_artifact_id,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// VerificationItem captures the evidence from a verify call for one platform.
type VerificationItem struct {
	Platform        string    `json:"platform"`
	Channel         string    `json:"channel"`
	ExpectedVersion string    `json:"expected_version"`
	ObservedVersion string    `json:"observed_version,omitempty"`
	SHA512Match     bool      `json:"sha512_match"`
	Match           bool      `json:"match"`
	Error           string    `json:"error,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
}

// Repository is the storage seam for releases and their per-platform rows.
type Repository interface {
	// Insert creates a release with its per-platform rows. Must run inside a
	// transaction that has acquired the profile-scoped advisory lock so that
	// concurrent callers for the same profile observe the UNIQUE constraint
	// correctly. The repository is also responsible for marking older
	// releases as superseded on successful publish.
	Insert(ctx context.Context, release *Release) error

	// Get retrieves a release by id, including its platform rows.
	Get(ctx context.Context, releaseID string) (*Release, error)
	GetByIdempotencyKey(ctx context.Context, profileID, key string) (*Release, error)
	InsertOperation(ctx context.Context, operation *Operation) error
	GetOperation(ctx context.Context, operationID string) (*Operation, error)
	ListActiveOperations(ctx context.Context) ([]*Operation, error)
	UpdateOperation(ctx context.Context, operationID, status, stage, errMsg string) error

	// ListByProfile returns recent releases for a profile (newest first).
	ListByProfile(ctx context.Context, profileID string, limit int) ([]*Release, error)

	// UpdateStatus transitions the release status and sets published_at when
	// moving into a terminal success state.
	UpdateStatus(ctx context.Context, releaseID, status string) error

	// SetDeploymentID binds an owner deployment identity to the release after
	// the owner returns a durable external-effect receipt.
	SetDeploymentID(ctx context.Context, releaseID, deploymentID string) error

	// SetVerificationEvidence persists the per-platform verify outcomes.
	SetVerificationEvidence(ctx context.Context, releaseID string, items []VerificationItem) error

	// MarkPlatformPublished updates a platform row with the LPBS artifact id
	// and flips its status to published.
	MarkPlatformPublished(ctx context.Context, releaseID, platform string, artifactID int64) error

	// MarkPlatformStatus sets only the status and optional error on a
	// platform row, without requiring an artifact id.
	MarkPlatformStatus(ctx context.Context, releaseID, platform, status, errMsg string) error

	// MarkSuperseded sets older releases for the same profile+channel to
	// superseded once a newer release reaches published.
	MarkSuperseded(ctx context.Context, profileID, channel, exceptReleaseID string) error

	// AcquireProfileLock takes a transaction-scoped advisory lock for the
	// given profile. Returns true if the lock was acquired, false if another
	// release orchestration is already in flight.
	AcquireProfileLock(ctx context.Context, profileID string) (bool, func(), error)

	// RecordReadinessWaiver records an actor-bound exception for one exact
	// profile and commit. Waivers are explicit operator evidence, not approval.
	RecordReadinessWaiver(ctx context.Context, profileID, commit, reason, actor string) error
	GetLatestReadiness(ctx context.Context, profileID string) (*ReadinessRecord, error)
}

// StartRequest is the body for POST /api/v1/profiles/{id}/releases/start.
type StartRequest struct {
	Channel               string          `json:"channel,omitempty"`
	GitCommitHash         string          `json:"git_commit_hash"`
	ArtifactDigest        string          `json:"artifact_digest"`
	CandidateID           string          `json:"candidate_id"`
	DestinationRevisionID string          `json:"destination_revision_id"`
	AuthorizationEpoch    uint64          `json:"authorization_epoch"`
	IdempotencyKey        string          `json:"idempotency_key"`
	ReadinessReviewKey    string          `json:"readiness_review_key"`
	ReleaseVersion        string          `json:"release_version"`
	ReleaseNotes          string          `json:"release_notes,omitempty"`
	ReleasedBy            string          `json:"released_by,omitempty"`
	Platforms             []string        `json:"platforms,omitempty"`
	CloudManifest         json.RawMessage `json:"cloud_manifest,omitempty"`
	CloudDeploymentName   string          `json:"cloud_deployment_name,omitempty"`
	CloudBundlePath       string          `json:"cloud_bundle_path,omitempty"`
	CloudBundleSHA256     string          `json:"cloud_bundle_sha256,omitempty"`
	CloudBundleSizeBytes  int64           `json:"cloud_bundle_size_bytes,omitempty"`
	CloudRunPreflight     bool            `json:"cloud_run_preflight,omitempty"`
}
