package main

import "time"

// ViewMode specifies how to display the file content
type ViewMode string

const (
	// ViewModeDiff shows only the diff hunks with context (default)
	ViewModeDiff ViewMode = "diff"
	// ViewModeFullDiff shows the full file with changes highlighted
	ViewModeFullDiff ViewMode = "full_diff"
	// ViewModeSource shows just the file content without diff highlighting
	ViewModeSource ViewMode = "source"
)

// LineChange indicates what type of change occurred on a line
type LineChange string

const (
	LineChangeNone     LineChange = ""
	LineChangeAdded    LineChange = "added"
	LineChangeDeleted  LineChange = "deleted"
	LineChangeModified LineChange = "modified"
)

// AnnotatedLine represents a single line with its change status
type AnnotatedLine struct {
	Number     int        `json:"number"`               // Line number in current file (0 for deleted lines)
	Content    string     `json:"content"`              // The line content
	Change     LineChange `json:"change,omitempty"`     // Type of change
	OldNumber  int        `json:"old_number,omitempty"` // Line number in old file (for deleted lines)
}

// DiffRequest specifies what diff to retrieve
type DiffRequest struct {
	Path      string   `json:"path"`
	Staged    bool     `json:"staged"`
	Untracked bool     `json:"untracked"`
	Base      string   `json:"base,omitempty"`
	Commit    string   `json:"commit,omitempty"` // View diff for a specific commit (history mode)
	Mode      ViewMode `json:"mode,omitempty"`   // View mode: diff, full_diff, or source
}

// DiffResponse contains the diff output and metadata
type DiffResponse struct {
	RepoDir        string          `json:"repo_dir"`
	Path           string          `json:"path,omitempty"`
	Staged         bool            `json:"staged"`
	Untracked      bool            `json:"untracked"`
	Base           string          `json:"base,omitempty"`
	HasDiff        bool            `json:"has_diff"`
	Hunks          []DiffHunk      `json:"hunks,omitempty"`
	Stats          DiffStats       `json:"stats"`
	Raw            string          `json:"raw,omitempty"`
	FullContent    string          `json:"full_content,omitempty"`
	AnnotatedLines []AnnotatedLine `json:"annotated_lines,omitempty"` // Full file with line-level change info
	Mode           ViewMode        `json:"mode,omitempty"`            // The view mode used
	Timestamp      time.Time       `json:"timestamp"`
}

// DiffHunk represents a single hunk in a diff
type DiffHunk struct {
	OldStart int      `json:"old_start"`
	OldCount int      `json:"old_count"`
	NewStart int      `json:"new_start"`
	NewCount int      `json:"new_count"`
	Header   string   `json:"header"`
	Lines    []string `json:"lines"`
}

// DiffStats summarizes the diff changes
type DiffStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}
