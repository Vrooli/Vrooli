package main

import "time"

type Evidence struct {
	Type   string `json:"type"`             // command|file|path|note
	Ref    string `json:"ref,omitempty"`    // e.g. module dir, filepath, command
	Detail string `json:"detail,omitempty"` // extra context
}

type Finding struct {
	Level        string     `json:"level"` // error|warn|info
	Message      string     `json:"message"`
	Evidence     []Evidence `json:"evidence,omitempty"`
	ScenarioName string     `json:"scenario_name,omitempty"`
}

type RuleResult struct {
	RuleID     string    `json:"rule_id"`
	Passed     bool      `json:"passed"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Findings   []Finding `json:"findings,omitempty"`
	ErrorCount int       `json:"error_count"`
	WarnCount  int       `json:"warn_count"`
}

// ComputeCounts sets ErrorCount and WarnCount from the Findings slice.
func (r *RuleResult) ComputeCounts() {
	r.ErrorCount = 0
	r.WarnCount = 0
	for _, f := range r.Findings {
		switch f.Level {
		case "error":
			r.ErrorCount++
		case "warn":
			r.WarnCount++
		}
	}
}

type RunRequest struct {
	Scope         string   `json:"scope,omitempty"`          // reserved (repo|scenario); default repo
	ScenarioNames []string `json:"scenario_names,omitempty"` // optional; when set, rules should scope to these scenarios
}

type RunResponse struct {
	RepoRoot string       `json:"repo_root"`
	Results  []RuleResult `json:"results"`
	TimedOut bool         `json:"timed_out,omitempty"`
}
