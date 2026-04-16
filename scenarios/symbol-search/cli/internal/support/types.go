package support

// Character mirrors the API's character shape returned by /api/search,
// /api/character/:codepoint, and /api/bulk/range.
type Character struct {
	Codepoint      string                 `json:"codepoint"`
	Decimal        int                    `json:"decimal"`
	Name           string                 `json:"name"`
	Category       string                 `json:"category"`
	Block          string                 `json:"block"`
	UnicodeVersion string                 `json:"unicode_version"`
	Description    *string                `json:"description,omitempty"`
	HTMLEntity     *string                `json:"html_entity,omitempty"`
	CSSContent     *string                `json:"css_content,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
}

// SearchResponse is the envelope returned by GET /api/search.
type SearchResponse struct {
	Characters     []Character            `json:"characters"`
	Total          int                    `json:"total"`
	QueryTimeMs    float64                `json:"query_time_ms"`
	FiltersApplied map[string]interface{} `json:"filters_applied"`
}

// CharacterDetailResponse is returned by GET /api/character/:codepoint.
type CharacterDetailResponse struct {
	Character         Character   `json:"character"`
	RelatedCharacters []Character `json:"related_characters,omitempty"`
	UsageExamples     []string    `json:"usage_examples,omitempty"`
}

// Category describes a Unicode category entry from GET /api/categories.
type Category struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CharacterCount int    `json:"character_count,omitempty"`
}

// CategoriesResponse is the envelope returned by GET /api/categories.
type CategoriesResponse struct {
	Categories []Category `json:"categories"`
}

// CharacterBlock describes a Unicode block entry from GET /api/blocks.
type CharacterBlock struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	StartCodepoint int    `json:"start_codepoint"`
	EndCodepoint   int    `json:"end_codepoint"`
	Description    string `json:"description"`
	CharacterCount int    `json:"character_count,omitempty"`
}

// BlocksResponse is the envelope returned by GET /api/blocks.
type BlocksResponse struct {
	Blocks []CharacterBlock `json:"blocks"`
}

// BulkRangeResponse is returned by POST /api/bulk/range.
type BulkRangeResponse struct {
	Characters      []Character `json:"characters"`
	TotalCharacters int         `json:"total_characters"`
	RangesProcessed int         `json:"ranges_processed"`
}
