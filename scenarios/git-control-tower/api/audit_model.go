package main

import "time"

// AuditOperation represents the type of operation being audited.
// [REQ:GCT-OT-P0-007] PostgreSQL audit logging
type AuditOperation string

const (
	AuditOpStage   AuditOperation = "stage"
	AuditOpUnstage AuditOperation = "unstage"
	AuditOpCommit  AuditOperation = "commit"
	AuditOpDiscard AuditOperation = "discard"
	AuditOpPush    AuditOperation = "push"
	AuditOpPull    AuditOperation = "pull"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	// ID is the unique identifier (set by database).
	ID int64 `json:"id,omitempty"`

	// Operation is the type of operation performed.
	Operation AuditOperation `json:"operation"`

	// RepoDir is the repository directory.
	RepoDir string `json:"repo_dir"`

	// Branch is the branch name at time of operation.
	Branch string `json:"branch,omitempty"`

	// Paths are the files affected (for stage/unstage).
	Paths []string `json:"paths,omitempty"`

	// CommitHash is the commit hash (for commit operations).
	CommitHash string `json:"commit_hash,omitempty"`

	// CommitMessage is the commit message (for commit operations).
	CommitMessage string `json:"commit_message,omitempty"`

	// Success indicates if the operation succeeded.
	Success bool `json:"success"`

	// Error contains error message if operation failed.
	Error string `json:"error,omitempty"`

	// Timestamp is when the operation occurred.
	Timestamp time.Time `json:"timestamp"`

	// Metadata contains additional context.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuditQueryRequest contains parameters for querying audit logs.
type AuditQueryRequest struct {
	// Operation filters by operation type.
	Operation AuditOperation `json:"operation,omitempty"`

	// Branch filters by branch name.
	Branch string `json:"branch,omitempty"`

	// Since filters entries after this time.
	Since time.Time `json:"since,omitempty"`

	// Until filters entries before this time.
	Until time.Time `json:"until,omitempty"`

	// Limit is the maximum number of entries to return.
	Limit int `json:"limit,omitempty"`

	// Offset for pagination.
	Offset int `json:"offset,omitempty"`
}

// AuditQueryResponse contains the result of an audit query.
type AuditQueryResponse struct {
	// Entries are the matching audit entries.
	Entries []AuditEntry `json:"entries"`

	// Total is the total count of matching entries (for pagination).
	Total int `json:"total"`

	// Timestamp is when the query was executed.
	Timestamp time.Time `json:"timestamp"`
}
