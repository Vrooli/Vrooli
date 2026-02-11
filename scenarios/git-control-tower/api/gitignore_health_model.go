package main

// GitignoreHealthResponse contains analysis of root .gitignore entries.
type GitignoreHealthResponse struct {
	RootEntryCount int                   `json:"root_entry_count"`
	Suggestions    []GitignoreSuggestion `json:"suggestions"`
}

// GitignoreSuggestion identifies a root .gitignore entry that could be moved.
type GitignoreSuggestion struct {
	Line          int    `json:"line"`
	Pattern       string `json:"pattern"`
	Type          string `json:"type"` // "single_group" or "cross_group"
	GroupLabel    string `json:"group_label"`
	GroupDir      string `json:"group_dir"`
	TargetPattern string `json:"target_pattern"`
	HasGitignore  bool   `json:"has_gitignore"`
}

// GitignoreMoveRequest specifies an entry to move from root to group .gitignore.
type GitignoreMoveRequest struct {
	Line          int    `json:"line"`
	Pattern       string `json:"pattern"`
	GroupDir      string `json:"group_dir"`
	TargetPattern string `json:"target_pattern"`
}

// GitignoreMoveResponse contains the result of a move operation.
type GitignoreMoveResponse struct {
	Success     bool   `json:"success"`
	RemovedFrom string `json:"removed_from,omitempty"`
	AddedTo     string `json:"added_to,omitempty"`
	Error       string `json:"error,omitempty"`
}
