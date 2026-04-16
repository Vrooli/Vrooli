package support

import "time"

// MCPEndpoint is one entry in the /mcp/endpoints scan response.
type MCPEndpoint struct {
	ID              string    `json:"id,omitempty"`
	ScenarioName    string    `json:"name"`
	MCPPort         int       `json:"port,omitempty"`
	Status          string    `json:"status,omitempty"`
	HasMCP          bool      `json:"hasMCP"`
	Tools           []string  `json:"tools,omitempty"`
	Confidence      string    `json:"confidence,omitempty"`
	LastHealthCheck time.Time `json:"lastHealthCheck,omitempty"`
}

// MCPEndpointsResponse wraps the scan output with a summary.
type MCPEndpointsResponse struct {
	Scenarios []MCPEndpoint  `json:"scenarios"`
	Summary   map[string]int `json:"summary"`
}

// MCPRegistryEndpoint is one endpoint in the MCP registry.
type MCPRegistryEndpoint struct {
	Name        string `json:"name"`
	Transport   string `json:"transport,omitempty"`
	URL         string `json:"url,omitempty"`
	ManifestURL string `json:"manifest_url,omitempty"`
}

// MCPRegistry is the response from /mcp/registry.
type MCPRegistry struct {
	Version   string                `json:"version"`
	Endpoints []MCPRegistryEndpoint `json:"endpoints"`
}

// MCPAddResponse is the response from POST /mcp/add.
type MCPAddResponse struct {
	Success        bool   `json:"success"`
	AgentSessionID string `json:"agent_session_id"`
	EstimatedTime  int    `json:"estimated_time,omitempty"`
}

// MCPSession is the response from /mcp/sessions/{id}.
type MCPSession struct {
	ID           string     `json:"id"`
	ScenarioName string     `json:"scenario_name"`
	Status       string     `json:"status"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Logs         string     `json:"logs,omitempty"`
}

// DocMetadata is one entry in /docs.
type DocMetadata struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Category     string    `json:"category,omitempty"`
	Summary      string    `json:"summary,omitempty"`
	RelativePath string    `json:"relativePath"`
	Source       string    `json:"source,omitempty"`
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
}

// DocListResponse is the response from /docs.
type DocListResponse struct {
	Docs  []DocMetadata `json:"docs"`
	Count int           `json:"count"`
}

// DocContent is the response from /docs/content.
type DocContent struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Category     string    `json:"category,omitempty"`
	Summary      string    `json:"summary,omitempty"`
	RelativePath string    `json:"relativePath"`
	Source       string    `json:"source,omitempty"`
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
	Content      string    `json:"content"`
}
