package support

import (
	"encoding/json"
	"time"
)

// Graph mirrors the API Graph shape returned by /api/v1/graphs/:id and
// embedded in /api/v1/graphs list payloads.
type Graph struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Data        json.RawMessage        `json:"data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Version     int                    `json:"version,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Permissions *GraphPermissions      `json:"permissions,omitempty"`
}

// GraphPermissions mirrors the API's access-control struct for a graph.
type GraphPermissions struct {
	Public       bool     `json:"public"`
	AllowedUsers []string `json:"allowed_users,omitempty"`
	Editors      []string `json:"editors,omitempty"`
}

// Plugin mirrors /api/v1/plugins entries.
type Plugin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category,omitempty"`
	Description string                 `json:"description,omitempty"`
	Formats     []string               `json:"formats,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ListResponse is the standard list envelope used by graphs/plugins endpoints:
// {data: [...], total: N, limit: "...", offset: "..."}.
type ListResponse struct {
	Data   json.RawMessage `json:"data"`
	Total  int             `json:"total"`
	Limit  string          `json:"limit,omitempty"`
	Offset string          `json:"offset,omitempty"`
}

// ValidationResult mirrors the POST /api/v1/graphs/:id/validate response.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ExportResponse mirrors POST /api/v1/graphs/:id/export.
type ExportResponse struct {
	Format   string `json:"format"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
	MimeType string `json:"mime_type"`
}

// DashboardStats mirrors GET /api/v1/stats.
type DashboardStats struct {
	TotalGraphs      int `json:"totalGraphs"`
	ConversionsToday int `json:"conversionsToday"`
	ActiveUsers      int `json:"activeUsers"`
}
