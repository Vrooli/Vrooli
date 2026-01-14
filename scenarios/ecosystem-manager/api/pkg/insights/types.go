package insights

import "time"

// InsightReport represents an analysis of execution patterns and suggested improvements
type InsightReport struct {
	ID             string              `json:"id"`
	TaskID         string              `json:"task_id"`
	GeneratedAt    time.Time           `json:"generated_at"`
	AnalysisWindow AnalysisWindow      `json:"analysis_window"`
	ExecutionCount int                 `json:"execution_count"`
	Patterns       []Pattern           `json:"patterns"`
	Suggestions    []Suggestion        `json:"suggestions"`
	Statistics     ExecutionStatistics `json:"statistics"`
	GeneratedBy    string              `json:"generated_by"` // task_id of insight-generator
}

// AnalysisWindow describes what executions were analyzed
type AnalysisWindow struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Limit        int       `json:"limit"`         // How many executions analyzed
	StatusFilter string    `json:"status_filter"` // e.g., "failed,timeout"
}

// Pattern represents a recurring issue or behavior
type Pattern struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`      // failure_mode, timeout, rate_limit, stuck_state
	Frequency   int      `json:"frequency"` // How many executions exhibit this
	Severity    string   `json:"severity"`  // critical, high, medium, low
	Description string   `json:"description"`
	Examples    []string `json:"examples"` // Execution IDs showing this pattern
	Evidence    []string `json:"evidence"` // Specific log excerpts
}

// Suggestion represents an actionable improvement
type Suggestion struct {
	ID          string           `json:"id"`
	PatternID   string           `json:"pattern_id"` // Which pattern this addresses
	Type        string           `json:"type"`       // prompt, timeout, code, autosteer_profile
	Priority    string           `json:"priority"`   // critical, high, medium, low
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Changes     []ProposedChange `json:"changes"`
	Impact      ImpactEstimate   `json:"impact"`
	Status      string           `json:"status"` // pending, applied, rejected, superseded
	AppliedAt   *time.Time       `json:"applied_at,omitempty"`
}

// ProposedChange describes a specific change to apply
type ProposedChange struct {
	File        string `json:"file"` // Relative path from scenario root
	Type        string `json:"type"` // edit, create, config_update
	Description string `json:"description"`
	Before      string `json:"before,omitempty"`       // For edits
	After       string `json:"after,omitempty"`        // For edits
	Content     string `json:"content,omitempty"`      // For creates
	ConfigPath  string `json:"config_path,omitempty"`  // For config updates (e.g., "phases[0].timeout")
	ConfigValue any    `json:"config_value,omitempty"` // New config value
}

// ImpactEstimate describes expected impact of a suggestion
type ImpactEstimate struct {
	SuccessRateImprovement string `json:"success_rate_improvement"` // e.g., "+15-25%"
	TimeReduction          string `json:"time_reduction,omitempty"` // e.g., "-10m avg"
	Confidence             string `json:"confidence"`               // high, medium, low
	Rationale              string `json:"rationale"`
}

// ExecutionStatistics provides aggregate stats for analyzed executions
type ExecutionStatistics struct {
	TotalExecutions      int     `json:"total_executions"`
	SuccessCount         int     `json:"success_count"`
	FailureCount         int     `json:"failure_count"`
	TimeoutCount         int     `json:"timeout_count"`
	RateLimitCount       int     `json:"rate_limit_count"`
	SuccessRate          float64 `json:"success_rate"`
	AvgDuration          string  `json:"avg_duration"`
	MedianDuration       string  `json:"median_duration"`
	MostCommonExitReason string  `json:"most_common_exit_reason"`
}

// SystemInsightReport represents system-wide analysis across all tasks
type SystemInsightReport struct {
	ID                string                   `json:"id"`
	GeneratedAt       time.Time                `json:"generated_at"`
	TimeWindow        AnalysisWindow           `json:"time_window"`
	TaskCount         int                      `json:"task_count"`
	TotalExecutions   int                      `json:"total_executions"`
	CrossTaskPatterns []CrossTaskPattern       `json:"cross_task_patterns"`
	SystemSuggestions []Suggestion             `json:"system_suggestions"`
	ByTaskType        map[string]TaskTypeStats `json:"by_task_type"`
	ByOperation       map[string]TaskTypeStats `json:"by_operation"`
}

// CrossTaskPattern represents a pattern affecting multiple tasks
type CrossTaskPattern struct {
	Pattern
	AffectedTasks []string `json:"affected_tasks"`
	TaskTypes     []string `json:"task_types"` // Which task types show this
}

// TaskTypeStats provides aggregate stats by task type or operation
type TaskTypeStats struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	AvgDuration string  `json:"avg_duration"`
	TopPattern  string  `json:"top_pattern"`
}
