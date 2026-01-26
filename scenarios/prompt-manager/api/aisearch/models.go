// Package aisearch provides AI-powered semantic search functionality.
//
// DOC: docs/reference/api-endpoints.md#ai-search
package aisearch

// AISearchRequest represents a search request.
type AISearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// AISearchResult represents a single search result.
type AISearchResult struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Folder      string   `json:"folder"`
	Tags        []string `json:"tags,omitempty"`
	Modes       []string `json:"modes,omitempty"`
	Score       float64  `json:"score"`
	ScorePercent int     `json:"scorePercent"` // 0-100 for display
}

// AISearchResponse wraps search results with metadata.
type AISearchResponse struct {
	Results []AISearchResult `json:"results"`
	Total   int              `json:"total"`
	Query   string           `json:"query"`
	Method  string           `json:"method"` // "ai" or "text"
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
