package deployments

import (
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// DeploymentApproval tracks per-platform, per-commit approval status.
// Each approval is tied to a specific git commit hash so that any code change
// automatically invalidates prior approvals (staleness detection).
type DeploymentApproval struct {
	ID            string     `json:"id"`
	ProfileID     string     `json:"profile_id"`
	GitCommitHash string     `json:"git_commit_hash"`
	Platform      string     `json:"platform"`
	Status        string     `json:"status"` // pending, approved, rejected, stale
	ApprovedBy    string     `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	ValidationID  string     `json:"validation_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Approval status constants.
const (
	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
	ApprovalStatusStale    = "stale"
)

// ApprovalDecisionRequest contains a reviewer's approve/reject decision.
type ApprovalDecisionRequest struct {
	Decision string `json:"decision"` // approved, rejected
	Reviewer string `json:"reviewer"`
	Notes    string `json:"notes,omitempty"`
}

// ReleaseGateStatus summarizes whether all required platforms are approved
// for a specific commit, gating whether a deployment can proceed.
type ReleaseGateStatus struct {
	ProfileID     string               `json:"profile_id"`
	GitCommitHash string               `json:"git_commit_hash"`
	Ready         bool                 `json:"ready"`
	Reason        string               `json:"reason"`
	Platforms     []PlatformGateStatus `json:"platforms"`
	Targets       []TargetGateStatus   `json:"targets"`
}

// PlatformGateStatus is the approval status for a single platform.
type PlatformGateStatus struct {
	Platform string `json:"platform"`
	Required bool   `json:"required"`
	Status   string `json:"status"` // pending, approved, rejected, stale, missing
}

// RequiredTarget is the profile's immutable evidence target definition. Bridge
// identifiers are intentionally omitted: they belong to an individual run.
type RequiredTarget struct {
	Ramp       string              `json:"ramp"`
	Platform   string              `json:"platform"`
	OS         string              `json:"os"`
	DeviceKind commonv1.DeviceKind `json:"device_kind"`
}

// TargetGateStatus combines evidence and the human approval state for one
// required target at one exact commit.
type TargetGateStatus struct {
	Target              RequiredTarget `json:"target"`
	EvidenceDisposition string         `json:"evidence_disposition"`
	EvidenceRunID       string         `json:"evidence_run_id,omitempty"`
	ApprovalStatus      string         `json:"approval_status"`
}

// CreateApprovalRequest is the API request body for creating an approval.
type CreateApprovalRequest struct {
	GitCommitHash string `json:"git_commit_hash"`
	Platform      string `json:"platform"`
	ValidationID  string `json:"validation_id,omitempty"`
}

// SetRequiredPlatformsRequest is the API request body for configuring required platforms.
type SetRequiredPlatformsRequest struct {
	Platforms []string `json:"platforms"`
}

type SetRequiredTargetsRequest struct {
	Targets []RequiredTarget `json:"targets"`
}
