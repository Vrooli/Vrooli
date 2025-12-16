package main

import "time"

// DiscardRequest specifies files to discard changes for.
type DiscardRequest struct {
	// Paths are the file paths to discard.
	Paths []string `json:"paths"`

	// Untracked indicates if the paths are untracked files (to be deleted).
	// If false, changes to tracked files will be reverted.
	Untracked bool `json:"untracked,omitempty"`
}

// DiscardResponse contains the result of a discard operation.
type DiscardResponse struct {
	// Success indicates if the operation succeeded.
	Success bool `json:"success"`

	// Discarded are the paths that were successfully discarded.
	Discarded []string `json:"discarded"`

	// Failed are the paths that failed to discard.
	Failed []string `json:"failed,omitempty"`

	// Errors are error messages for failed paths.
	Errors []string `json:"errors,omitempty"`

	// Timestamp is when the operation was performed.
	Timestamp time.Time `json:"timestamp"`
}
