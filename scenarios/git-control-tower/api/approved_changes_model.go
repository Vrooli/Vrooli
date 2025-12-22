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
	RelativePath string `json:"relativePath"`
	Status       string `json:"status"`
	SandboxID    string `json:"sandboxId,omitempty"`
	SandboxOwner string `json:"sandboxOwner,omitempty"`
	ChangeType   string `json:"changeType,omitempty"`
}

// ApprovedChangesPreviewRequest requests a commit preview for a subset of files.
type ApprovedChangesPreviewRequest struct {
	Paths []string `json:"paths"`
}
