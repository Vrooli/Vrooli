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

// --- Agent search types ---

// AgentSearchQuery represents agent search parameters.
type AgentSearchQuery struct {
	Query  string `json:"q"`
	Tag    string `json:"tag"`
	Status string `json:"status"`
}

// AgentSearchResult represents an agent search result.
type AgentSearchResult struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags,omitempty"`
	Score       float64  `json:"score,omitempty"`
	Highlight   string   `json:"highlight,omitempty"`
}

// AgentSearchResponse wraps agent search results.
type AgentSearchResponse struct {
	Results []AgentSearchResult `json:"results"`
	Total   int                 `json:"total"`
	Query   string              `json:"query"`
}

// AgentContentSearchQuery represents agent content search parameters.
type AgentContentSearchQuery struct {
	Query         string   `json:"q"`
	Tags          []string `json:"tags,omitempty"`
	CaseSensitive bool     `json:"caseSensitive"`
	WholeWord     bool     `json:"wholeWord"`
	Regex         bool     `json:"regex"`
	Limit         int      `json:"limit"`
}

// AgentContentSearchMatch represents a line-level match in agent files.
type AgentContentSearchMatch struct {
	AgentID     string       `json:"agentId"`
	AgentName   string       `json:"agentName"`
	File        string       `json:"file"`
	LineNumber  int          `json:"lineNumber"`
	Line        string       `json:"line"`
	MatchRanges []MatchRange `json:"matchRanges"`
}

// AgentContentSearchResponse wraps agent content search results.
type AgentContentSearchResponse struct {
	Matches []AgentContentSearchMatch `json:"matches"`
	Total   int                       `json:"total"`
	Query   string                    `json:"query"`
}

// --- Team search types ---

// TeamSearchQuery represents team search parameters.
type TeamSearchQuery struct {
	Query   string `json:"q"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// TeamSearchResult represents a team search result.
type TeamSearchResult struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Mission     string  `json:"mission,omitempty"`
	Enabled     bool    `json:"enabled"`
	MemberCount int     `json:"memberCount"`
	Score       float64 `json:"score,omitempty"`
	Highlight   string  `json:"highlight,omitempty"`
}

// TeamSearchResponse wraps team search results.
type TeamSearchResponse struct {
	Results []TeamSearchResult `json:"results"`
	Total   int                `json:"total"`
	Query   string             `json:"query"`
}

// TeamContentSearchQuery represents team content search parameters.
type TeamContentSearchQuery struct {
	Query         string `json:"q"`
	CaseSensitive bool   `json:"caseSensitive"`
	WholeWord     bool   `json:"wholeWord"`
	Regex         bool   `json:"regex"`
	Limit         int    `json:"limit"`
}

// TeamContentSearchMatch represents a line-level match in team shared files.
type TeamContentSearchMatch struct {
	TeamID      string       `json:"teamId"`
	TeamName    string       `json:"teamName"`
	File        string       `json:"file"`
	LineNumber  int          `json:"lineNumber"`
	Line        string       `json:"line"`
	MatchRanges []MatchRange `json:"matchRanges"`
}

// TeamContentSearchResponse wraps team content search results.
type TeamContentSearchResponse struct {
	Matches []TeamContentSearchMatch `json:"matches"`
	Total   int                      `json:"total"`
	Query   string                   `json:"query"`
}
