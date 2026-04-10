package deployments

import "time"

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
	Platforms     []PlatformGateStatus `json:"platforms"`
}

// PlatformGateStatus is the approval status for a single platform.
type PlatformGateStatus struct {
	Platform string `json:"platform"`
	Required bool   `json:"required"`
	Status   string `json:"status"` // pending, approved, rejected, stale, missing
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
