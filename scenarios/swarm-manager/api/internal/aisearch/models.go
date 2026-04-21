// Package aisearch provides embedding-based semantic search over swarm-manager
// backlog items and initiatives, with graceful fallback to text search when
// Ollama or Qdrant is unavailable.
package aisearch

// EntityType identifies which collection a search result belongs to.
type EntityType string

const (
	EntityBacklog    EntityType = "backlog"
	EntityInitiative EntityType = "initiative"
	EntityBoth       EntityType = "both"
)

// Valid reports whether e is one of the known entity values accepted by the
// API surface.
func (e EntityType) Valid() bool {
	switch e {
	case EntityBacklog, EntityInitiative, EntityBoth:
		return true
	default:
		return false
	}
}

// SearchFilters constrains results to a subset of the indexed entities. All
// fields are optional; zero-valued filters match everything.
type SearchFilters struct {
	Status          []string `json:"status,omitempty"`
	Kind            []string `json:"kind,omitempty"`
	Initiative      string   `json:"initiative,omitempty"`
	IncludeArchived bool     `json:"include_archived,omitempty"`
}

// AISearchRequest is the input shape for a semantic search.
type AISearchRequest struct {
	Query     string        `json:"query"`
	Entity    EntityType    `json:"entity,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Threshold float64       `json:"threshold,omitempty"`
	Filters   SearchFilters `json:"filters,omitempty"`
}

// AISearchResult is a single ranked result. Payload preserves the raw Qdrant
// payload for the client; typed helpers (BacklogPayloadFrom, InitiativePayloadFrom)
// can unwrap it when the entity is known.
type AISearchResult struct {
	Entity       EntityType             `json:"entity"`
	ID           string                 `json:"id"`
	Score        float64                `json:"score"`
	ScorePercent int                    `json:"scorePercent"`
	Payload      map[string]interface{} `json:"payload"`
}

// FallbackMethod describes whether and how a request fell back when embeddings
// were unavailable.
type FallbackMethod string

const (
	FallbackNone        FallbackMethod = "none"
	FallbackTextSearch  FallbackMethod = "text-search"
	FallbackUnavailable FallbackMethod = "unavailable"
)

// AISearchResponse wraps ranked results with metadata describing how the
// search was fulfilled.
type AISearchResponse struct {
	Results   []AISearchResult `json:"results"`
	Total     int              `json:"total"`
	Query     string           `json:"query"`
	Entity    EntityType       `json:"entity"`
	Fallback  FallbackMethod   `json:"fallback"`
	LatencyMs int64            `json:"latencyMs"`
}

// AvailabilityStatus reports whether AI search is currently usable. Exposed by
// GET /api/v1/search/ai/status.
type AvailabilityStatus struct {
	Available          bool   `json:"available"`
	Ollama             bool   `json:"ollama"`
	Qdrant             bool   `json:"qdrant"`
	IndexedBacklog     int    `json:"indexedBacklog"`
	IndexedInitiatives int    `json:"indexedInitiatives"`
	OnDiskBacklog      int    `json:"onDiskBacklog"`
	OnDiskInitiatives  int    `json:"onDiskInitiatives"`
	Message            string `json:"message,omitempty"`
}

// ReindexResponse is the terminal result returned by ReindexAll.
type ReindexResponse struct {
	Indexed int    `json:"indexed"`
	Skipped int    `json:"skipped"`
	Errors  int    `json:"errors"`
	Message string `json:"message"`
}

// ReindexStatus is the mutable, in-flight progress of a reindex job.
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

// BacklogPayload is the typed shape of what aisearch stores in a backlog
// vector's payload. Keys here must match the map keys used in
// index.go:buildBacklogPayload.
type BacklogPayload struct {
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Priority   int      `json:"priority"`
	Tags       []string `json:"tags"`
	Initiative string   `json:"initiative"`
	Effort     string   `json:"effort"`
	Archived   bool     `json:"archived"`
}

// InitiativePayload is the typed shape of what aisearch stores in an
// initiative vector's payload.
type InitiativePayload struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Archived bool   `json:"archived"`
}
