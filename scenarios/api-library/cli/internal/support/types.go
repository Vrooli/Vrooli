package support

import (
	"encoding/json"
	"time"
)

// API mirrors the API struct exposed by the api-library service.
type API struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Provider         string     `json:"provider"`
	Description      string     `json:"description,omitempty"`
	BaseURL          string     `json:"base_url,omitempty"`
	DocumentationURL string     `json:"documentation_url,omitempty"`
	PricingURL       string     `json:"pricing_url,omitempty"`
	Category         string     `json:"category,omitempty"`
	Status           string     `json:"status,omitempty"`
	AuthType         string     `json:"auth_type,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	Capabilities     []string   `json:"capabilities,omitempty"`
	Version          string     `json:"version,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	LastRefreshed    *time.Time `json:"last_refreshed,omitempty"`
	SourceURL        string     `json:"source_url,omitempty"`
}

// Note mirrors a note entry attached to an API.
type Note struct {
	ID        string     `json:"id"`
	APIID     string     `json:"api_id,omitempty"`
	Content   string     `json:"content"`
	Type      string     `json:"type,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	CreatedBy string     `json:"created_by,omitempty"`
}

// SearchResult mirrors one hit from /search.
type SearchResult struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	Description    string   `json:"description,omitempty"`
	RelevanceScore float64  `json:"relevance_score,omitempty"`
	Configured     bool     `json:"configured,omitempty"`
	PricingSummary string   `json:"pricing_summary,omitempty"`
	Category       string   `json:"category,omitempty"`
	MinPrice       *float64 `json:"min_price,omitempty"`
}

// SearchResponse is the envelope returned by /search.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
	Query   string         `json:"query"`
	Method  string         `json:"method"`
}

// APIDetailResponse mirrors the /apis/{id} response shape.
type APIDetailResponse struct {
	API   API    `json:"api"`
	Notes []Note `json:"notes"`
}

// ResearchResponse mirrors the /request-research 201 body.
type ResearchResponse struct {
	ResearchID    string `json:"research_id"`
	Status        string `json:"status"`
	EstimatedTime int    `json:"estimated_time"`
}

// ConfiguredAPI is one entry returned by /configured.
type ConfiguredAPI struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Provider          string     `json:"provider"`
	Description       string     `json:"description,omitempty"`
	Category          string     `json:"category,omitempty"`
	Environment       string     `json:"environment,omitempty"`
	ConfigurationDate *time.Time `json:"configuration_date,omitempty"`
}

// CategoryCount is a category summary returned by /categories.
type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// TagCount is a tag summary returned by /tags.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// WebhookSubscription mirrors a webhook row.
type WebhookSubscription struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Events        []string   `json:"events,omitempty"`
	Active        bool       `json:"active"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
	FailureCount  int        `json:"failure_count,omitempty"`
}

// Snippet mirrors an integration snippet.
type Snippet struct {
	ID                string                 `json:"id"`
	APIID             string                 `json:"api_id,omitempty"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description,omitempty"`
	Language          string                 `json:"language,omitempty"`
	Framework         string                 `json:"framework,omitempty"`
	Code              string                 `json:"code,omitempty"`
	SnippetType       string                 `json:"snippet_type,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Tested            bool                   `json:"tested,omitempty"`
	Official          bool                   `json:"official,omitempty"`
	CommunityVerified bool                   `json:"community_verified,omitempty"`
	UsageCount        int                    `json:"usage_count,omitempty"`
	HelpfulCount      int                    `json:"helpful_count,omitempty"`
	NotHelpfulCount   int                    `json:"not_helpful_count,omitempty"`
	APIName           string                 `json:"api_name,omitempty"`
	APIProvider       string                 `json:"api_provider,omitempty"`
	CreatedAt         *time.Time             `json:"created_at,omitempty"`
	Dependencies      map[string]interface{} `json:"dependencies,omitempty"`
}

// Recipe mirrors an integration recipe.
type Recipe struct {
	ID                   string                 `json:"id"`
	APIID                string                 `json:"api_id,omitempty"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description,omitempty"`
	UseCase              string                 `json:"use_case,omitempty"`
	ExpectedOutcome      string                 `json:"expected_outcome,omitempty"`
	EstimatedTimeMinutes int                    `json:"estimated_time_minutes,omitempty"`
	DifficultyLevel      string                 `json:"difficulty_level,omitempty"`
	TimesUsed            int                    `json:"times_used,omitempty"`
	SuccessRate          float64                `json:"success_rate,omitempty"`
	Rating               float64                `json:"rating,omitempty"`
	RatingCount          int                    `json:"rating_count,omitempty"`
	Tags                 []string               `json:"tags,omitempty"`
	CreatedBy            string                 `json:"created_by,omitempty"`
	CreatedAt            *time.Time             `json:"created_at,omitempty"`
	UpdatedAt            *time.Time             `json:"updated_at,omitempty"`
	Steps                json.RawMessage        `json:"steps,omitempty"`
	Prerequisites        map[string]interface{} `json:"prerequisites,omitempty"`
	RelatedAPIIDs        []string               `json:"related_api_ids,omitempty"`
}

// RecipesEnvelope mirrors {recipes, count}.
type RecipesEnvelope struct {
	Recipes []Recipe `json:"recipes"`
	Count   int      `json:"count"`
}

// SnippetsEnvelope mirrors {snippets, count}.
type SnippetsEnvelope struct {
	Snippets []Snippet `json:"snippets"`
	Count    int       `json:"count"`
}

// APIVersion mirrors an /apis/{id}/versions row.
type APIVersion struct {
	ID              string     `json:"id"`
	APIID           string     `json:"api_id,omitempty"`
	Version         string     `json:"version"`
	ChangeSummary   string     `json:"change_summary,omitempty"`
	BreakingChanges bool       `json:"breaking_changes,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}
