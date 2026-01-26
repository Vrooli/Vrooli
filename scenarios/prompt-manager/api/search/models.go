// Package search provides skill search functionality.
package search

// SearchQuery represents search parameters.
type SearchQuery struct {
	Query  string `json:"q"`      // Text query
	Tag    string `json:"tag"`    // Filter by tag
	Folder string `json:"folder"` // Filter by folder
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

// SearchResponse wraps search results with metadata.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
}
