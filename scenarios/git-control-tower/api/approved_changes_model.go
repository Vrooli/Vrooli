package main

// ApprovedChangesResponse is the normalized response for workspace-sandbox approved changes.
type ApprovedChangesResponse struct {
	Available        bool                 `json:"available"`
	CommittableFiles int                  `json:"committableFiles"`
	SuggestedMessage string               `json:"suggestedMessage,omitempty"`
	Files            []ApprovedChangeFile `json:"files,omitempty"`
	Warning          string               `json:"warning,omitempty"`
}

// ApprovedChangeFile summarizes a single approved change file.
type ApprovedChangeFile struct {
	RelativePath      string `json:"relativePath"`
	Status            string `json:"status"`
	SandboxID         string `json:"sandboxId,omitempty"`
	SandboxOwner      string `json:"sandboxOwner,omitempty"`
	ChangeType        string `json:"changeType,omitempty"`
	AgentManagerRunID string `json:"agentManagerRunId,omitempty"`
	ContentDigest     string `json:"contentDigest,omitempty"`
	EvidenceRevision  string `json:"evidenceRevision,omitempty"`
}

// ApprovedChangesPreviewRequest requests a commit preview for a subset of files.
type ApprovedChangesPreviewRequest struct {
	Paths []string `json:"paths"`
}

// ProvenanceResponse is the response for the provenance-by-run endpoint.
type ProvenanceResponse struct {
	Available bool                 `json:"available"`
	RunGroups []ProvenanceRunGroup `json:"runGroups"`
	Warning   string               `json:"warning,omitempty"`
}

// ProvenanceRunGroup groups pending changes by agent-manager run ID.
type ProvenanceRunGroup struct {
	RunID           string           `json:"runId"`
	SandboxID       string           `json:"sandboxId"`
	SandboxOwner    string           `json:"sandboxOwner"`
	Files           []ProvenanceFile `json:"files"`
	LatestAppliedAt string           `json:"latestAppliedAt"`
}

// ProvenanceFile represents a single file within a provenance run group.
type ProvenanceFile struct {
	FilePath     string `json:"filePath"`
	RelativePath string `json:"relativePath"`
	ChangeType   string `json:"changeType"`
	AppliedAt    string `json:"appliedAt"`
	Visibility   string `json:"visibility,omitempty"`
	ContentDigest string `json:"contentDigest,omitempty"`
	EvidenceRevision string `json:"evidenceRevision,omitempty"`
}

// ProvenanceSearchRequest is the bounded, read-only query accepted by the
// shared provenance corpus. RepoID is explicit so a federated caller cannot
// accidentally search whichever repository a UI session currently selected.
type ProvenanceSearchRequest struct {
	Query  string `json:"query"`
	RepoID int64  `json:"repoId"`
	Limit  int    `json:"limit"`
	Scope  string `json:"scope,omitempty"`
}

type ProvenanceSearchResponse struct {
	Available bool                  `json:"available"`
	Results   []ProvenanceSearchHit `json:"results"`
	Warning   string                `json:"warning,omitempty"`
}

type ProvenanceSearchHit struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Snippet          string  `json:"snippet"`
	Score            float64 `json:"score"`
	RunID            string  `json:"runId"`
	SandboxID        string  `json:"sandboxId"`
	RelativePath     string  `json:"relativePath"`
	EvidenceStanding string  `json:"evidenceStanding"`
}
