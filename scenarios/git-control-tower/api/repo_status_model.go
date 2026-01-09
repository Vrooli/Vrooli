package main

import "time"

type RepoStatusDeps struct {
	Git     GitRunner
	RepoDir string
}

type RepoHistoryDeps struct {
	Git          GitRunner
	RepoDir      string
	Limit        int
	IncludeFiles bool
}

type RepoStatus struct {
	RepoDir   string                 `json:"repo_dir"`
	Branch    RepoBranchStatus       `json:"branch"`
	Files     RepoFilesStatus        `json:"files"`
	FileStats RepoFileStats          `json:"file_stats,omitempty"`
	Scopes    map[string][]string    `json:"scopes"`
	Summary   RepoStatusSummary      `json:"summary"`
	Author    RepoAuthorStatus       `json:"author"`
	Timestamp time.Time              `json:"timestamp"`
	Raw       map[string]interface{} `json:"raw,omitempty"`
}

type RepoHistory struct {
	RepoDir   string             `json:"repo_dir"`
	Lines     []string           `json:"lines"`
	Entries   []RepoHistoryEntry `json:"entries,omitempty"`
	Limit     int                `json:"limit"`
	Timestamp time.Time          `json:"timestamp"`
}

type RepoHistoryEntry struct {
	Hash    string   `json:"hash"`
	Author  string   `json:"author,omitempty"`
	Date    string   `json:"date,omitempty"`
	Subject string   `json:"subject"`
	Files   []string `json:"files"`
}

type RepoAuthorStatus struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type RepoBranchStatus struct {
	Head     string `json:"head"`
	Upstream string `json:"upstream,omitempty"`
	Ahead    int    `json:"ahead,omitempty"`
	Behind   int    `json:"behind,omitempty"`
	OID      string `json:"oid,omitempty"`
}

type RepoFilesStatus struct {
	Staged    []string          `json:"staged"`
	Unstaged  []string          `json:"unstaged"`
	Untracked []string          `json:"untracked"`
	Conflicts []string          `json:"conflicts"`
	Binary    []string          `json:"binary,omitempty"`
	Ignored   []string          `json:"ignored,omitempty"`
	Statuses  map[string]string `json:"statuses,omitempty"`
}

type RepoFileStats struct {
	Staged    map[string]DiffStats `json:"staged,omitempty"`
	Unstaged  map[string]DiffStats `json:"unstaged,omitempty"`
	Untracked map[string]DiffStats `json:"untracked,omitempty"`
}

type RepoStatusSummary struct {
	Staged    int `json:"staged"`
	Unstaged  int `json:"unstaged"`
	Untracked int `json:"untracked"`
	Conflicts int `json:"conflicts"`
	Ignored   int `json:"ignored,omitempty"`
}
