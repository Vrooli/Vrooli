package main

import "time"

// DiffRequest specifies what diff to retrieve
type DiffRequest struct {
	Path   string `json:"path"`
	Staged bool   `json:"staged"`
	Base   string `json:"base,omitempty"`
}

// DiffResponse contains the diff output and metadata
type DiffResponse struct {
	RepoDir   string        `json:"repo_dir"`
	Path      string        `json:"path,omitempty"`
	Staged    bool          `json:"staged"`
	Base      string        `json:"base,omitempty"`
	HasDiff   bool          `json:"has_diff"`
	Hunks     []DiffHunk    `json:"hunks,omitempty"`
	Stats     DiffStats     `json:"stats"`
	Raw       string        `json:"raw,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
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
