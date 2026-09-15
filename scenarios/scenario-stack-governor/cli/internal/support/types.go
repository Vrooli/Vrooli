package support

import "time"

type RuleDefinition struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	WhyImportant   string `json:"why_important"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	DefaultEnabled bool   `json:"default_enabled"`
	Fixable        bool   `json:"fixable"`
}

type Rule struct {
	RuleDefinition
	Enabled bool `json:"enabled"`
}

type RulesConfig struct {
	Version      string          `json:"version"`
	EnabledRules map[string]bool `json:"enabled_rules"`
}

type RulesResponse struct {
	Rules  []Rule      `json:"rules"`
	Config RulesConfig `json:"config"`
}

type ScenariosResponse struct {
	Scenarios []string `json:"scenarios"`
}

type Evidence struct {
	Type   string `json:"type"`
	Ref    string `json:"ref,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type Finding struct {
	Level        string     `json:"level"`
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

type RunRequest struct {
	ScenarioNames []string `json:"scenario_names,omitempty"`
}

type RunResponse struct {
	RepoRoot string       `json:"repo_root"`
	Results  []RuleResult `json:"results"`
	TimedOut bool         `json:"timed_out,omitempty"`
}

type FixRequest struct {
	ScenarioNames []string `json:"scenario_names"`
	RuleIDs       []string `json:"rule_ids,omitempty"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

type FixResponse struct {
	RepoRoot string      `json:"repo_root"`
	Results  []FixResult `json:"results"`
}

type FixResult struct {
	ScenarioName string      `json:"scenario_name"`
	RuleID       string      `json:"rule_id"`
	Fixed        bool        `json:"fixed"`
	FilePath     string      `json:"file_path"`
	Changes      []FixChange `json:"changes"`
	Diff         *FileDiff   `json:"diff,omitempty"`
	Error        string      `json:"error,omitempty"`
}

type FileDiff struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

type FixChange struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}
