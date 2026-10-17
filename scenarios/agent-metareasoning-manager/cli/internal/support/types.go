package support

// Workflow mirrors the WorkflowMetadata shape returned by /workflows and
// /workflows/search. Nested capabilities/time fields are not used by CLI
// rendering so they stay in the free-form maps where applicable.
type Workflow struct {
	ID          string   `json:"id"`
	Platform    string   `json:"platform"`
	PlatformID  string   `json:"platform_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	UsageCount  int      `json:"usage_count"`
	LastUsed    string   `json:"last_used,omitempty"`
}

// ReasoningResultSummary mirrors one row in the /reasoning/results list
// response. The full result rows are returned as a free-form map by the
// detail endpoint.
type ReasoningResultSummary struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Confidence      float64 `json:"confidence"`
	Model           string  `json:"model"`
	ExecutionTimeMS int64   `json:"execution_time_ms"`
	Success         bool    `json:"success"`
	CreatedAt       string  `json:"created_at"`
	Error           string  `json:"error,omitempty"`
}

// ReasoningResultsPage is the envelope returned by GET /reasoning/results.
type ReasoningResultsPage struct {
	Results []ReasoningResultSummary `json:"results"`
	Count   int                      `json:"count"`
}
