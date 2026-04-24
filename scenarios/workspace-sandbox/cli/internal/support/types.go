package support

import "time"

type SandboxResponse struct {
	ID            string    `json:"id"`
	ScopePath     string    `json:"scopePath"`
	ReservedPath  string    `json:"reservedPath"`
	ReservedPaths []string  `json:"reservedPaths,omitempty"`
	ProjectRoot   string    `json:"projectRoot"`
	Owner         string    `json:"owner,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	MergedDir     string    `json:"mergedDir,omitempty"`
	SizeBytes     int64     `json:"sizeBytes"`
	FileCount     int       `json:"fileCount"`
	ErrorMsg      string    `json:"errorMessage,omitempty"`
}

type ListResponse struct {
	Sandboxes  []SandboxResponse `json:"sandboxes"`
	TotalCount int               `json:"totalCount"`
}

type DiffFile struct {
	FilePath   string `json:"filePath"`
	ChangeType string `json:"changeType"`
}

type DiffResponse struct {
	SandboxID   string     `json:"sandboxId"`
	UnifiedDiff string     `json:"unifiedDiff"`
	Stats       DiffStats  `json:"stats"`
	Files       []DiffFile `json:"files"`
}

// DiffStats mirrors workspace-sandbox api/internal/types.DiffStats.
type DiffStats struct {
	FilesChanged  int   `json:"filesChanged"`
	FilesAdded    int   `json:"filesAdded"`
	FilesModified int   `json:"filesModified"`
	FilesDeleted  int   `json:"filesDeleted"`
	LinesAdded    int   `json:"linesAdded"`
	LinesRemoved  int   `json:"linesRemoved"`
	TotalBytes    int64 `json:"totalBytes"`
}

type ApprovalResponse struct {
	Success    bool   `json:"success"`
	Applied    int    `json:"applied"`
	CommitHash string `json:"commitHash,omitempty"`
	ErrorMsg   string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

type GCCollectedSandbox struct {
	ID        string    `json:"id"`
	ScopePath string    `json:"scopePath"`
	Status    string    `json:"status"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Reason    string    `json:"reason"`
}

type GCError struct {
	SandboxID string `json:"sandboxId"`
	Error     string `json:"error"`
}

type GCResult struct {
	Collected           []GCCollectedSandbox `json:"collected"`
	TotalCollected      int                  `json:"totalCollected"`
	TotalBytesReclaimed int64                `json:"totalBytesReclaimed"`
	Errors              []GCError            `json:"errors,omitempty"`
	DryRun              bool                 `json:"dryRun"`
	StartedAt           time.Time            `json:"startedAt"`
	CompletedAt         time.Time            `json:"completedAt"`
}

type ConflictCheckResponse struct {
	HasConflict         bool      `json:"hasConflict"`
	BaseCommitHash      string    `json:"baseCommitHash,omitempty"`
	CurrentHash         string    `json:"currentHash,omitempty"`
	RepoChangedFiles    []string  `json:"repoChangedFiles,omitempty"`
	SandboxChangedFiles []string  `json:"sandboxChangedFiles,omitempty"`
	ConflictingFiles    []string  `json:"conflictingFiles,omitempty"`
	CheckedAt           time.Time `json:"checkedAt"`
}

type RebaseResponse struct {
	Success          bool      `json:"success"`
	PreviousBaseHash string    `json:"previousBaseHash,omitempty"`
	NewBaseHash      string    `json:"newBaseHash,omitempty"`
	ConflictingFiles []string  `json:"conflictingFiles,omitempty"`
	RepoChangedFiles []string  `json:"repoChangedFiles,omitempty"`
	Strategy         string    `json:"strategy"`
	ErrorMsg         string    `json:"error,omitempty"`
	RebasedAt        time.Time `json:"rebasedAt"`
}

type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"`
}

type RunResponse struct {
	PID       int       `json:"pid"`
	SandboxID string    `json:"sandboxId"`
	Command   string    `json:"command"`
	Name      string    `json:"name,omitempty"`
	StartedAt time.Time `json:"startedAt,omitempty"`
}

type ProcessInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	Running   bool      `json:"running"`
	StartedAt time.Time `json:"startedAt"`
	SessionID string    `json:"sessionId,omitempty"`
}

type ProcessListResponse struct {
	Processes []ProcessInfo `json:"processes"`
	Total     int           `json:"total"`
	Running   int           `json:"running"`
}

type LogResponse struct {
	PID       int    `json:"pid"`
	SandboxID string `json:"sandboxId"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	IsActive  bool   `json:"isActive"`
	Content   string `json:"content"`
}

type LogsListResponse struct {
	Logs  []LogResponse `json:"logs"`
	Total int           `json:"total"`
}

type PendingChangesSummary struct {
	SandboxID     string    `json:"sandboxId"`
	SandboxOwner  string    `json:"sandboxOwner"`
	FileCount     int       `json:"fileCount"`
	LatestApplied time.Time `json:"latestApplied"`
}

type PendingChangesResult struct {
	Summaries  []PendingChangesSummary `json:"summaries"`
	TotalFiles int                     `json:"totalFiles"`
}

type AppliedChange struct {
	ID           string     `json:"id"`
	SandboxID    string     `json:"sandboxId"`
	SandboxOwner string     `json:"sandboxOwner"`
	FilePath     string     `json:"filePath"`
	ProjectRoot  string     `json:"projectRoot"`
	ChangeType   string     `json:"changeType"`
	AppliedAt    time.Time  `json:"appliedAt"`
	CommittedAt  *time.Time `json:"committedAt,omitempty"`
	CommitHash   string     `json:"commitHash,omitempty"`
}

type FileProvenanceResult struct {
	FilePath string          `json:"filePath"`
	Changes  []AppliedChange `json:"changes"`
}

type CommitPendingResult struct {
	Success        bool   `json:"success"`
	FilesCommitted int    `json:"filesCommitted"`
	CommitHash     string `json:"commitHash,omitempty"`
	ErrorMsg       string `json:"error,omitempty"`
}
