package main

import "time"

type Evidence struct {
	Type   string `json:"type"`            // command|file|path|note
	Ref    string `json:"ref,omitempty"`   // e.g. module dir, filepath, command
	Detail string `json:"detail,omitempty"` // extra context
}

type Finding struct {
	Level    string     `json:"level"` // error|warn|info
	Message  string     `json:"message"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

type RuleResult struct {
	RuleID     string    `json:"rule_id"`
	Passed     bool      `json:"passed"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Findings   []Finding `json:"findings,omitempty"`
}

type RunRequest struct {
	Scope string `json:"scope,omitempty"` // reserved (repo|scenario); default repo
}

type RunResponse struct {
	RepoRoot string       `json:"repo_root"`
	Results  []RuleResult `json:"results"`
}

