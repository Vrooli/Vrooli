package main

import "time"

// SyncStatusRequest contains the parameters for checking push/pull status.
// [REQ:GCT-OT-P0-006] Push/pull status
type SyncStatusRequest struct {
	// Fetch controls whether to fetch from remote first for accurate counts.
	// Default: false (use cached data from last fetch)
	Fetch bool `json:"fetch,omitempty"`

	// Remote is the remote name to check. Default: "origin"
	Remote string `json:"remote,omitempty"`
}

// SyncStatusResponse contains the push/pull status for the repository.
type SyncStatusResponse struct {
	// Branch is the current branch name.
	Branch string `json:"branch"`

	// Upstream is the tracking branch (e.g., "origin/main").
	Upstream string `json:"upstream,omitempty"`

	// RemoteURL is the URL of the remote repository.
	RemoteURL string `json:"remote_url,omitempty"`

	// Ahead is the number of commits ahead of upstream.
	Ahead int `json:"ahead"`

	// Behind is the number of commits behind upstream.
	Behind int `json:"behind"`

	// HasUpstream indicates if the branch tracks an upstream.
	HasUpstream bool `json:"has_upstream"`

	// CanPush indicates if push is safe and possible.
	// True when: ahead > 0 AND (behind == 0 OR force allowed) AND no uncommitted changes (or user acknowledges risk)
	CanPush bool `json:"can_push"`

	// CanPull indicates if pull is needed and safe.
	// True when: behind > 0 AND no uncommitted changes blocking the pull
	CanPull bool `json:"can_pull"`

	// NeedsPull indicates if the local branch is behind remote.
	NeedsPull bool `json:"needs_pull"`

	// NeedsPush indicates if the local branch is ahead of remote.
	NeedsPush bool `json:"needs_push"`

	// HasUncommittedChanges indicates if there are staged/unstaged changes.
	HasUncommittedChanges bool `json:"has_uncommitted_changes"`

	// SafetyWarnings contains any warnings about the sync state.
	SafetyWarnings []string `json:"safety_warnings,omitempty"`

	// Recommendations contains suggested actions.
	Recommendations []string `json:"recommendations,omitempty"`

	// Fetched indicates if a fetch was performed.
	Fetched bool `json:"fetched"`

	// FetchError contains any error from fetching (operation continues).
	FetchError string `json:"fetch_error,omitempty"`

	// Timestamp is when the status was generated.
	Timestamp time.Time `json:"timestamp"`
}
