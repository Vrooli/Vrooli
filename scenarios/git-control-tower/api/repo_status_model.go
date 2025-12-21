package main

import "time"

type RepoStatusDeps struct {
	Git     GitRunner
	RepoDir string
}

type RepoHistoryDeps struct {
	Git     GitRunner
	RepoDir string
	Limit   int
}

type RepoStatus struct {
	RepoDir   string                 `json:"repo_dir"`
	Branch    RepoBranchStatus       `json:"branch"`
	Files     RepoFilesStatus        `json:"files"`
	Scopes    map[string][]string    `json:"scopes"`
	Summary   RepoStatusSummary      `json:"summary"`
	Author    RepoAuthorStatus       `json:"author"`
	Timestamp time.Time              `json:"timestamp"`
	Raw       map[string]interface{} `json:"raw,omitempty"`
}

type RepoHistory struct {
	RepoDir   string    `json:"repo_dir"`
	Lines     []string  `json:"lines"`
	Limit     int       `json:"limit"`
	Timestamp time.Time `json:"timestamp"`
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

type RepoStatusSummary struct {
	Staged    int `json:"staged"`
	Unstaged  int `json:"unstaged"`
	Untracked int `json:"untracked"`
	Conflicts int `json:"conflicts"`
	Ignored   int `json:"ignored,omitempty"`
}
