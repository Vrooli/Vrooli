package main

import "time"

// PushRequest contains parameters for pushing to remote.
type PushRequest struct {
	// Remote is the remote name. Default: "origin"
	Remote string `json:"remote,omitempty"`

	// Branch is the branch to push. Default: current branch
	Branch string `json:"branch,omitempty"`

	// SetUpstream sets the upstream tracking branch (-u flag).
	SetUpstream bool `json:"set_upstream,omitempty"`
}

// PushResponse contains the result of a push operation.
type PushResponse struct {
	// Success indicates if the push succeeded.
	Success bool `json:"success"`

	// Remote is the remote that was pushed to.
	Remote string `json:"remote"`

	// Branch is the branch that was pushed.
	Branch string `json:"branch"`

	// Error contains the error message if push failed.
	Error string `json:"error,omitempty"`

	// Timestamp is when the operation was performed.
	Timestamp time.Time `json:"timestamp"`
}

// PullRequest contains parameters for pulling from remote.
type PullRequest struct {
	// Remote is the remote name. Default: "origin"
	Remote string `json:"remote,omitempty"`

	// Branch is the branch to pull. Default: current tracking branch
	Branch string `json:"branch,omitempty"`
}

// PullResponse contains the result of a pull operation.
type PullResponse struct {
	// Success indicates if the pull succeeded.
	Success bool `json:"success"`

	// Remote is the remote that was pulled from.
	Remote string `json:"remote"`

	// Branch is the branch that was pulled.
	Branch string `json:"branch"`

	// Error contains the error message if pull failed.
	Error string `json:"error,omitempty"`

	// HasConflicts indicates if there are merge conflicts.
	HasConflicts bool `json:"has_conflicts,omitempty"`

	// Timestamp is when the operation was performed.
	Timestamp time.Time `json:"timestamp"`
}
