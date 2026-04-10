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
}
