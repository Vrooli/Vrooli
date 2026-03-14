package main

// AuditorCheckRequest is the request body for starting a standards check.
type AuditorCheckRequest struct {
	Type          string   `json:"type,omitempty"`
	Standards     []string `json:"standards,omitempty"`
	ForceDisabled bool     `json:"force_disabled,omitempty"`
}

// AuditorCheckJobResponse is the initial response from starting a check.
type AuditorCheckJobResponse struct {
	JobID  string           `json:"job_id"`
	Status AuditorJobStatus `json:"status"`
}

// AuditorJobStatus tracks the progress and result of a standards check job.
type AuditorJobStatus struct {
	ID                 string              `json:"id"`
	Scenario           string              `json:"scenario"`
	ScanType           string              `json:"scan_type"`
	Status             string              `json:"status"`
	StartedAt          string              `json:"started_at"`
	CompletedAt        *string             `json:"completed_at,omitempty"`
	ElapsedSeconds     float64             `json:"elapsed_seconds"`
	TotalScenarios     int                 `json:"total_scenarios"`
	ProcessedScenarios int                 `json:"processed_scenarios"`
	ProcessedFiles     int                 `json:"processed_files"`
	TotalFiles         int                 `json:"total_files"`
	CurrentScenario    string              `json:"current_scenario,omitempty"`
	CurrentFile        string              `json:"current_file,omitempty"`
	Message            string              `json:"message,omitempty"`
	Error              string              `json:"error,omitempty"`
	Result             *AuditorCheckResult `json:"result,omitempty"`
}

// AuditorCheckResult contains the final results of a completed standards check.
type AuditorCheckResult struct {
	CheckID      string                   `json:"check_id"`
	Status       string                   `json:"status"`
	ScanType     string                   `json:"scan_type"`
	StartedAt    string                   `json:"started_at"`
	CompletedAt  string                   `json:"completed_at"`
	Duration     float64                  `json:"duration_seconds"`
	FilesScanned int                      `json:"files_scanned"`
	Violations   []AuditorViolation       `json:"violations"`
	Statistics   map[string]int           `json:"statistics"`
	Message      string                   `json:"message"`
	ScenarioName string                   `json:"scenario_name,omitempty"`
	Summary      *AuditorViolationSummary `json:"summary,omitempty"`
}

// AuditorViolation represents a single standards violation.
type AuditorViolation struct {
	ID             string         `json:"id"`
	ScenarioName   string         `json:"scenario_name"`
	Type           string         `json:"type"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path"`
	LineNumber     int            `json:"line_number"`
	CodeSnippet    string         `json:"code_snippet,omitempty"`
	Recommendation string         `json:"recommendation"`
	Standard       string         `json:"standard"`
	DiscoveredAt   string         `json:"discovered_at"`
	Source         string         `json:"source,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// AuditorRuleCount is a rule ID and its violation count.
type AuditorRuleCount struct {
	RuleID string `json:"rule_id"`
	Count  int    `json:"count"`
}

// AuditorViolationExcerpt is a shortened violation for summary display.
type AuditorViolationExcerpt struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	RuleID   string `json:"rule_id"`
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
}

// AuditorViolationSummary provides aggregate statistics about violations.
type AuditorViolationSummary struct {
	Total            int                       `json:"total"`
	BySeverity       map[string]int            `json:"by_severity"`
	ByRule           []AuditorRuleCount        `json:"by_rule"`
	HighestSeverity  string                    `json:"highest_severity"`
	TopViolations    []AuditorViolationExcerpt `json:"top_violations"`
	RecommendedSteps []string                  `json:"recommended_steps,omitempty"`
	GeneratedAt      string                    `json:"generated_at"`
}

// AuditorRule describes a rule definition.
type AuditorRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Severity    string   `json:"severity"`
	Enabled     bool     `json:"enabled"`
	Standard    string   `json:"standard"`
	Targets     []string `json:"targets"`
}

// AuditorRulesListResponse is the response from listing rules.
type AuditorRulesListResponse struct {
	Rules      map[string]AuditorRule `json:"rules"`
	Categories map[string]any         `json:"categories,omitempty"`
	Count      int                    `json:"count"`
	Total      int                    `json:"total"`
}

// AuditorFixRequest is the request body for applying fixes.
type AuditorFixRequest struct {
	ScenarioNames []string `json:"scenario_names"`
	RuleIDs       []string `json:"rule_ids"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// AuditorFixChange describes a single change made by a fix.
type AuditorFixChange struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

// AuditorFixResult records what happened for one scenario+rule pair.
type AuditorFixResult struct {
	ScenarioName string             `json:"scenario_name"`
	RuleID       string             `json:"rule_id"`
	Fixed        bool               `json:"fixed"`
	FilePath     string             `json:"file_path"`
	Changes      []AuditorFixChange `json:"changes"`
	Error        string             `json:"error,omitempty"`
}

// AuditorFixResponse is the response from applying fixes.
type AuditorFixResponse struct {
	Results        []AuditorFixResult `json:"results"`
	Count          int                `json:"count"`
	UnfixableRules []string           `json:"unfixable_rules"`
	Errors         []string           `json:"errors"`
}

// AuditorViolationsResponse is the response from listing stored violations.
type AuditorViolationsResponse struct {
	Violations []AuditorViolation `json:"violations"`
}

// AuditorRunCheckProxyRequest is the GCT proxy request for starting a check.
type AuditorRunCheckProxyRequest struct {
	ScenarioName string `json:"scenario_name"`
	CheckType    string `json:"check_type,omitempty"`
}
