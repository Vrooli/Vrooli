// Package aisearch provides AI-powered semantic search functionality.
//
// DOC: docs/reference/api-endpoints.md#ai-search
package aisearch

import "time"

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

// VectorPayload represents the metadata stored with each vector point.
type VectorPayload struct {
	SkillID     string   `json:"skill_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Folder      string   `json:"folder"`
	Tags        []string `json:"tags"`
	Modes       []string `json:"modes"`
}

// EntityKind names the collection a reconciler item belongs to.
type EntityKind string

const (
	KindSkill  EntityKind = "skill"
	KindAgent  EntityKind = "agent"
	KindTeam   EntityKind = "team"
	KindTopic  EntityKind = "topic"
	KindAction EntityKind = "action"
)

// ItemRef is the reconciler's handle on a single planned item — enough to
// embed-and-upsert during Apply without re-reading disk.
type ItemRef struct {
	Kind        EntityKind  `json:"kind"`
	PointID     string      `json:"pointId"`
	Name        string      `json:"name"`
	PayloadHash string      `json:"payloadHash"`
	Snapshot    interface{} `json:"-"`
}

// CollectionDriftReport captures Plan output for one collection.
type CollectionDriftReport struct {
	Kind           EntityKind `json:"kind"`
	ToUpsert       []ItemRef  `json:"toUpsert,omitempty"`
	ToDelete       []string   `json:"toDelete,omitempty"`
	UnchangedCount int        `json:"unchangedCount"`
	LegacyCount    int        `json:"legacyCount"`
}

// DriftReport is the full Plan output across all configured collections.
type DriftReport struct {
	PlannedAt   time.Time               `json:"plannedAt"`
	Collections []CollectionDriftReport `json:"collections"`
}

// HasWork returns true when any collection has upserts or deletes pending.
func (d *DriftReport) HasWork() bool {
	if d == nil {
		return false
	}
	for _, c := range d.Collections {
		if len(c.ToUpsert) > 0 || len(c.ToDelete) > 0 {
			return true
		}
	}
	return false
}

// CollectionApplyResult captures Apply output for one collection.
type CollectionApplyResult struct {
	Kind     EntityKind `json:"kind"`
	Upserted int        `json:"upserted"`
	Deleted  int        `json:"deleted"`
}

// ReconcileError is one entry in ApplyResult.Errors. The Op field names which
// step in the Plan/Apply pipeline failed for at-a-glance triage.
type ReconcileError struct {
	Kind    EntityKind `json:"kind"`
	PointID string     `json:"pointId,omitempty"`
	Name    string     `json:"name,omitempty"`
	Op      string     `json:"op"`
	Err     string     `json:"err"`
}

// ApplyResult captures the actions Apply took.
type ApplyResult struct {
	StartedAt   time.Time               `json:"startedAt"`
	FinishedAt  time.Time               `json:"finishedAt"`
	Collections []CollectionApplyResult `json:"collections"`
	Errors      []ReconcileError        `json:"errors,omitempty"`
}

// ReconcileStatus is the read-only Reconciler state surfaced to handlers/CLI.
type ReconcileStatus struct {
	Running    bool         `json:"running"`
	StartedAt  string       `json:"startedAt,omitempty"`
	FinishedAt string       `json:"finishedAt,omitempty"`
	LastPlan   *DriftReport `json:"lastPlan,omitempty"`
	LastResult *ApplyResult `json:"lastResult,omitempty"`
	LastError  string       `json:"lastError,omitempty"`
	Canceled   bool         `json:"canceled,omitempty"`
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
