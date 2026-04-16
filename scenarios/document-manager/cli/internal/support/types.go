package support

import "time"

// Application mirrors the shape returned by /api/applications.
type Application struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	RepositoryURL     string    `json:"repository_url"`
	DocumentationPath string    `json:"documentation_path"`
	HealthScore       float64   `json:"health_score"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	AgentCount        int       `json:"agent_count,omitempty"`
	Status            string    `json:"status,omitempty"`
}

// Agent mirrors the shape returned by /api/agents.
type Agent struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Type                 string    `json:"type"`
	ApplicationID        string    `json:"application_id"`
	Configuration        string    `json:"configuration"`
	ScheduleCron         string    `json:"schedule_cron"`
	AutoApplyThreshold   float64   `json:"auto_apply_threshold"`
	LastPerformanceScore float64   `json:"last_performance_score"`
	Enabled              bool      `json:"enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	ApplicationName      string    `json:"application_name,omitempty"`
	Status               string    `json:"status,omitempty"`
}

// QueueItem mirrors the shape returned by /api/queue.
type QueueItem struct {
	ID              string    `json:"id"`
	AgentID         string    `json:"agent_id"`
	ApplicationID   string    `json:"application_id"`
	Type            string    `json:"type"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	AgentName       string    `json:"agent_name,omitempty"`
	ApplicationName string    `json:"application_name,omitempty"`
}

// BatchQueueResponse mirrors the response from /api/queue/batch.
type BatchQueueResponse struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
	Total     int      `json:"total"`
}

// SystemStatus mirrors /api/system/*-status responses.
type SystemStatus struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

// SearchResult mirrors one entry in /api/search results.
type SearchResult struct {
	ID              string                 `json:"id"`
	Score           float64                `json:"score"`
	DocumentID      string                 `json:"document_id,omitempty"`
	Content         string                 `json:"content,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ApplicationName string                 `json:"application_name,omitempty"`
}

// SearchResponse mirrors the envelope returned by /api/search.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Total   int            `json:"total"`
}

// IndexResponse mirrors the envelope returned by /api/index.
type IndexResponse struct {
	Indexed int      `json:"indexed"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}
