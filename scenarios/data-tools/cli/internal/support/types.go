package support

import "time"

// Resource mirrors the shape returned by /api/v1/resources. The API returns
// untyped bags for config, so keep that field flexible.
type Resource struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
}

// Execution is one entry from /api/v1/executions. The API stores input/output
// as opaque JSONB; keep them as raw maps.
type Execution struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id,omitempty"`
	Status     string                 `json:"status,omitempty"`
	StartedAt  string                 `json:"started_at,omitempty"`
	FinishedAt string                 `json:"finished_at,omitempty"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
}
