package main

import "time"

type RepoStatusDeps struct {
	Git     GitRunner
	RepoDir string
}

type RepoStatus struct {
	RepoDir   string                 `json:"repo_dir"`
	Branch    RepoBranchStatus       `json:"branch"`
	Files     RepoFilesStatus        `json:"files"`
	Scopes    map[string][]string    `json:"scopes"`
	Summary   RepoStatusSummary      `json:"summary"`
	Timestamp time.Time              `json:"timestamp"`
	Raw       map[string]interface{} `json:"raw,omitempty"`
}

type RepoBranchStatus struct {
	Head     string `json:"head"`
	Upstream string `json:"upstream,omitempty"`
	Ahead    int    `json:"ahead,omitempty"`
	Behind   int    `json:"behind,omitempty"`
	OID      string `json:"oid,omitempty"`
}

type RepoFilesStatus struct {
	Staged    []string `json:"staged"`
	Unstaged  []string `json:"unstaged"`
	Untracked []string `json:"untracked"`
	Conflicts []string `json:"conflicts"`
	Ignored   []string `json:"ignored,omitempty"`
}

type RepoStatusSummary struct {
	Staged    int `json:"staged"`
	Unstaged  int `json:"unstaged"`
	Untracked int `json:"untracked"`
	Conflicts int `json:"conflicts"`
	Ignored   int `json:"ignored,omitempty"`
}

