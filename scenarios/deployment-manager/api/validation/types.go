// Package validation provides visual validation quality gates for desktop deployments.
// It orchestrates screen-recorded smoke tests and manages approval workflows.
package validation

import "time"

// Request initiates a visual validation for a deployment profile.
type Request struct {
	ProfileID     string `json:"profile_id"`
	Platform      string `json:"platform,omitempty"`
	RecordVideo   bool   `json:"record_video"`
	DisplayWidth  int    `json:"display_width,omitempty"`
	DisplayHeight int    `json:"display_height,omitempty"`
}

// Record persists the state of a visual validation through its lifecycle.
type Record struct {
	ID              string     `json:"id"`
	ProfileID       string     `json:"profile_id"`
	DeploymentID    string     `json:"deployment_id,omitempty"`
	SmokeTestID     string     `json:"smoke_test_id"`
	Status          string     `json:"status"` // pending, recording, passed, failed, review_required
	VideoURL        string     `json:"video_url,omitempty"`
	VideoSizeBytes  int64      `json:"video_size_bytes,omitempty"`
	VideoDurationMs int64      `json:"video_duration_ms,omitempty"`
	Platform        string     `json:"platform,omitempty"`
	ReviewDecision  string     `json:"review_decision,omitempty"` // approved, rejected
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	ReviewNotes     string     `json:"review_notes,omitempty"`
	ApprovalID      string     `json:"approval_id,omitempty"` // links to deployment_approvals
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
}

// ReviewRequest contains the reviewer's decision.
type ReviewRequest struct {
	Decision string `json:"decision"` // approved, rejected
	Reviewer string `json:"reviewed_by"`
	Notes    string `json:"notes,omitempty"`
}
