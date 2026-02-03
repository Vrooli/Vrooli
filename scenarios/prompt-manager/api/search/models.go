// Package search provides skill search functionality.
package search

// SearchQuery represents search parameters.
type SearchQuery struct {
	Query  string `json:"q"`      // Text query
	Tag    string `json:"tag"`    // Filter by tag
	Folder string `json:"folder"` // Filter by folder
}

// ContentSearchQuery represents search parameters for content search.
type ContentSearchQuery struct {
	Query         string   `json:"q"`                 // Text query
	Tags          []string `json:"tags,omitempty"`    // Filter by tags (any match)
	Folders       []string `json:"folders,omitempty"` // Filter by folders (any match)
	CaseSensitive bool     `json:"caseSensitive"`     // Case-sensitive match
	WholeWord     bool     `json:"wholeWord"`         // Whole word match
	Regex         bool     `json:"regex"`             // Treat query as regex
	Limit         int      `json:"limit"`             // Max number of matches
}

// SearchResult represents a search result item.
type SearchResult struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Folder      string   `json:"folder"`
	Tags        []string `json:"tags"`
	Modes       []string `json:"modes"`
	Score       float64  `json:"score,omitempty"` // Relevance score
	Highlight   string   `json:"highlight,omitempty"`
}

// MatchRange represents a match range in a line.
type MatchRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ContentSearchMatch represents a line-level content search match.
type ContentSearchMatch struct {
	SkillID     string       `json:"skillId"`
	SkillName   string       `json:"skillName"`
	File        string       `json:"file"`
	Folder      string       `json:"folder"`
	LineNumber  int          `json:"lineNumber"`
	Line        string       `json:"line"`
	MatchRanges []MatchRange `json:"matchRanges"`
}

// SearchResponse wraps search results with metadata.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
}

// ContentSearchResponse wraps content search results with metadata.
type ContentSearchResponse struct {
	Matches []ContentSearchMatch `json:"matches"`
	Total   int                  `json:"total"`
	Query   string               `json:"query"`
}
