package main

import "time"

// BranchInfo represents a local or remote branch summary.
type BranchInfo struct {
	Name         string    `json:"name"`
	Upstream     string    `json:"upstream,omitempty"`
	OID          string    `json:"oid,omitempty"`
	LastCommitAt time.Time `json:"last_commit_at,omitempty"`
	Ahead        int       `json:"ahead,omitempty"`
	Behind       int       `json:"behind,omitempty"`
	IsCurrent    bool      `json:"is_current,omitempty"`
}

// RepoBranchesResponse returns branch data for the repository.
type RepoBranchesResponse struct {
	Current   string       `json:"current"`
	Locals    []BranchInfo `json:"locals"`
	Remotes   []BranchInfo `json:"remotes"`
	Timestamp time.Time    `json:"timestamp"`
}

// BranchWarning provides structured warnings for branch operations.
type BranchWarning struct {
	Message              string             `json:"message"`
	RequiresConfirmation bool               `json:"requires_confirmation,omitempty"`
	RequiresTracking     bool               `json:"requires_tracking,omitempty"`
	RequiresFetch        bool               `json:"requires_fetch,omitempty"`
	DirtySummary         *RepoStatusSummary `json:"dirty_summary,omitempty"`
}

// CreateBranchRequest creates a new branch.
type CreateBranchRequest struct {
	Name       string `json:"name"`
	From       string `json:"from,omitempty"`
	Checkout   bool   `json:"checkout,omitempty"`
	AllowDirty bool   `json:"allow_dirty,omitempty"`
}

// BranchCreateResponse represents the create branch result.
type BranchCreateResponse struct {
	Success          bool           `json:"success"`
	Branch           *BranchInfo    `json:"branch,omitempty"`
	Warning          *BranchWarning `json:"warning,omitempty"`
	Error            string         `json:"error,omitempty"`
	ValidationErrors []string       `json:"validation_errors,omitempty"`
	Timestamp        time.Time      `json:"timestamp"`
}

// SwitchBranchRequest switches branches.
type SwitchBranchRequest struct {
	Name        string `json:"name"`
	AllowDirty  bool   `json:"allow_dirty,omitempty"`
	TrackRemote bool   `json:"track_remote,omitempty"`
}

// BranchSwitchResponse represents the switch branch result.
type BranchSwitchResponse struct {
	Success   bool           `json:"success"`
	Branch    *BranchInfo    `json:"branch,omitempty"`
	Warning   *BranchWarning `json:"warning,omitempty"`
	Error     string         `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// PublishBranchRequest publishes a branch to the remote.
type PublishBranchRequest struct {
	Remote      string `json:"remote,omitempty"`
	Branch      string `json:"branch,omitempty"`
	SetUpstream bool   `json:"set_upstream,omitempty"`
	Fetch       bool   `json:"fetch,omitempty"`
}

// BranchPublishResponse represents the publish branch result.
type BranchPublishResponse struct {
	Success   bool           `json:"success"`
	Remote    string         `json:"remote"`
	Branch    string         `json:"branch"`
	Warning   *BranchWarning `json:"warning,omitempty"`
	Error     string         `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}
