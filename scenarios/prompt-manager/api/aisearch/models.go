// Package aisearch provides AI-powered semantic search functionality.
//
// DOC: docs/reference/api-endpoints.md#ai-search
package aisearch

// AISearchRequest represents a search request.
type AISearchRequest struct {
	Query       string   `json:"query"`
	Queries     []string `json:"queries,omitempty"` // multi-query: each searched independently, results merged
	Limit       int      `json:"limit,omitempty"`
	Output      string   `json:"output,omitempty"`      // "results", "combined", or "both"
	Format      string   `json:"format,omitempty"`      // "xml", "markdown", or "json" (for combined output)
	RenderLimit int      `json:"renderLimit,omitempty"` // optional override for combined output count
}

// AISearchResult represents a single search result.
type AISearchResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Folder       string   `json:"folder"`
	Tags         []string `json:"tags,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"` // 0-100 for display
}

// AISearchResponse wraps search results with metadata.
type AISearchResponse struct {
	Results     []AISearchResult `json:"results,omitempty"`
	Combined    string           `json:"combined,omitempty"`
	SkillCount  int              `json:"skillCount,omitempty"`
	TotalTokens int              `json:"totalTokens,omitempty"`
	Format      string           `json:"format,omitempty"`
	Total       int              `json:"total"`
	Query       string           `json:"query"`
	Method      string           `json:"method"` // "ai" or "text"
	Output      string           `json:"output,omitempty"`
}

// AvailabilityStatus represents the AI search system status.
type AvailabilityStatus struct {
	Available    bool   `json:"available"`
	Ollama       bool   `json:"ollama"`
	Qdrant       bool   `json:"qdrant"`
	IndexedCount int    `json:"indexedCount"`
	Message      string `json:"message,omitempty"`
}

// ReindexResponse represents the response from a reindex operation.
type ReindexResponse struct {
	Indexed int    `json:"indexed"`
	Skipped int    `json:"skipped"`
	Errors  int    `json:"errors"`
	Message string `json:"message"`
}

// ReindexStatus represents the status of a reindex job.
type ReindexStatus struct {
	Running    bool   `json:"running"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Indexed    int    `json:"indexed"`
	Skipped    int    `json:"skipped"`
	Errors     int    `json:"errors"`
	Total      int    `json:"total"`
	Message    string `json:"message,omitempty"`
	Canceled   bool   `json:"canceled,omitempty"`
	Error      string `json:"error,omitempty"`
}

// VectorPayload represents the metadata stored with each vector point.
type VectorPayload struct {
	SkillID     string   `json:"skill_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Folder      string   `json:"folder"`
	Tags        []string `json:"tags"`
	Modes       []string `json:"modes"`
}

// --- Action AI search types ---

// AIActionSearchRequest represents an action AI search request.
type AIActionSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// AIActionSearchResult represents a single Action AI search result.
type AIActionSearchResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status"`
	Owner        string   `json:"owner"`
	Command      string   `json:"command,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
}

// AIActionSearchResponse wraps Action AI search results.
type AIActionSearchResponse struct {
	Results []AIActionSearchResult `json:"results,omitempty"`
	Total   int                    `json:"total"`
	Query   string                 `json:"query"`
	Method  string                 `json:"method"`
}

// --- Agent AI search types ---

// AIAgentSearchRequest represents an agent AI search request.
type AIAgentSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// AIAgentSearchResult represents a single agent AI search result.
type AIAgentSearchResult struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status"`
	Tags         []string `json:"tags,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
}

// AIAgentSearchResponse wraps agent AI search results.
type AIAgentSearchResponse struct {
	Results []AIAgentSearchResult `json:"results,omitempty"`
	Total   int                   `json:"total"`
	Query   string                `json:"query"`
	Method  string                `json:"method"`
}

// --- Team AI search types ---

// AITeamSearchRequest represents a team AI search request.
type AITeamSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// AITeamSearchResult represents a single team AI search result.
type AITeamSearchResult struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"displayName"`
	Mission      string  `json:"mission,omitempty"`
	Enabled      bool    `json:"enabled"`
	MemberCount  int     `json:"memberCount"`
	Score        float64 `json:"score"`
	ScorePercent int     `json:"scorePercent"`
}

// AITeamSearchResponse wraps team AI search results.
type AITeamSearchResponse struct {
	Results []AITeamSearchResult `json:"results,omitempty"`
	Total   int                  `json:"total"`
	Query   string               `json:"query"`
	Method  string               `json:"method"`
}

// --- Discover types (unified topic + skill search) ---

// DiscoverRequest represents a unified topic + skill discovery request.
type DiscoverRequest struct {
	Queries    []string `json:"queries"`
	Complexity string   `json:"complexity,omitempty"` // minor|moderate|major|architectural
	Limit      int      `json:"limit,omitempty"`
	Type       string   `json:"type,omitempty"` // skill|action|all; empty preserves skill-only behavior
}

// DiscoverResult is a single discovery result with content size and source tracking.
type DiscoverResult struct {
	Type         string   `json:"type,omitempty"` // skill|action; omitted for legacy skill-only discovery
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
	Source       string   `json:"source"`               // "topic" or "search"
	TopicDepth   *int     `json:"topicDepth,omitempty"` // 0=root, 1=child, etc.
	TopicID      string   `json:"topicId,omitempty"`    // which topic sourced this skill
	TopicName    string   `json:"topicName,omitempty"`  // resolved topic name for display
	ContentChars int      `json:"contentChars"`
	Status       string   `json:"status,omitempty"`
	Owner        string   `json:"owner,omitempty"`
	ShowCommand  string   `json:"showCommand,omitempty"`
	RunCommand   string   `json:"runCommand,omitempty"`
}

// DiscoverResponse wraps discovery results with budget metadata.
type DiscoverResponse struct {
	Results                []DiscoverResult `json:"results"`
	Total                  int              `json:"total"`
	Query                  string           `json:"query"`
	Method                 string           `json:"method"`
	TotalContentChars      int              `json:"totalContentChars"`
	ReadCommand            string           `json:"readCommand"`
	ShowCommand            string           `json:"showCommand,omitempty"`
	RunCommand             string           `json:"runCommand,omitempty"`
	BudgetChars            int              `json:"budgetChars,omitempty"`
	BudgetStatus           string           `json:"budgetStatus,omitempty"`           // under|over|at
	RecommendedReadCommand string           `json:"recommendedReadCommand,omitempty"` // trimmed command if over budget
	Complexity             string           `json:"complexity,omitempty"`
}
