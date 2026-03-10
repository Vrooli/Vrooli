package main

// FixRequest describes which scenarios+rules to fix.
type FixRequest struct {
	ScenarioNames []string `json:"scenario_names"`
	RuleIDs       []string `json:"rule_ids,omitempty"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// FixResponse is the top-level fix result envelope.
type FixResponse struct {
	RepoRoot string      `json:"repo_root"`
	Results  []FixResult `json:"results"`
}

// FixResult records what happened for one scenario+rule pair.
type FixResult struct {
	ScenarioName string      `json:"scenario_name"`
	RuleID       string      `json:"rule_id"`
	Fixed        bool        `json:"fixed"`
	FilePath     string      `json:"file_path"`
	Changes      []FixChange `json:"changes"`
	Error        string      `json:"error,omitempty"`
}

// FixChange describes a single mutation applied (or that would be applied in dry-run).
type FixChange struct {
	Type   string `json:"type"` // generated | replaced | preserved_custom
	Detail string `json:"detail"`
}
