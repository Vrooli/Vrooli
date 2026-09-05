// Package releases provides the deployment-manager-owned release lifecycle:
// a release record ties a publish attempt to a profile, commit, channel, and
// per-platform verification evidence.
package releases

import (
	"context"
	"time"
)

// Release status values.
const (
	StatusPending      = "pending"
	StatusPublishing   = "publishing"
	StatusPublished    = "published"
	StatusFailed       = "failed"
	StatusSuperseded   = "superseded"
	StatusVerifyFailed = "verify_failed"
)

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
	ID                    string             `json:"id"`
	ProfileID             string             `json:"profile_id"`
	DeploymentID          string             `json:"deployment_id,omitempty"`
	ProfileVersion        int                `json:"profile_version,omitempty"`
	GitCommitHash         string             `json:"git_commit_hash"`
	ArtifactDigest        string             `json:"artifact_digest,omitempty"`
	ReadinessReviewKey    string             `json:"readiness_review_key,omitempty"`
	ReleaseVersion        string             `json:"release_version"`
	Channel               string             `json:"channel"`
	Status                string             `json:"status"`
	ReleaseNotes          string             `json:"release_notes,omitempty"`
	ReleasedBy            string             `json:"released_by,omitempty"`
	PromotedFromReleaseID string             `json:"promoted_from_release_id,omitempty"`
	ReadinessGoalRef      string             `json:"readiness_goal_ref,omitempty"`
	ApprovedAtCommit      string             `json:"approved_at_commit,omitempty"`
	VerificationEvidence  []VerificationItem `json:"verification_evidence,omitempty"`
	Platforms             []ReleasePlatform  `json:"platforms,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	PublishedAt           *time.Time         `json:"published_at,omitempty"`
	UpdatedAt             time.Time          `json:"updated_at"`
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

	// ListByProfile returns recent releases for a profile (newest first).
	ListByProfile(ctx context.Context, profileID string, limit int) ([]*Release, error)

	// UpdateStatus transitions the release status and sets published_at when
	// moving into a terminal success state.
	UpdateStatus(ctx context.Context, releaseID, status string) error

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
	Channel            string   `json:"channel,omitempty"`
	GitCommitHash      string   `json:"git_commit_hash"`
	ArtifactDigest     string   `json:"artifact_digest"`
	ReadinessReviewKey string   `json:"readiness_review_key"`
	ReleaseVersion     string   `json:"release_version"`
	ReleaseNotes       string   `json:"release_notes,omitempty"`
	ReleasedBy         string   `json:"released_by,omitempty"`
	Platforms          []string `json:"platforms,omitempty"`
}
