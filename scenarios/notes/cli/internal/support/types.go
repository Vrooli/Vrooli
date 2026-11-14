package support

import "time"

// Note mirrors the API shape returned by /api/notes endpoints.
type Note struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id,omitempty"`
	FolderID     *string   `json:"folder_id,omitempty"`
	Title        string    `json:"title"`
	Content      string    `json:"content,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	Summary      *string   `json:"summary,omitempty"`
	IsPinned     bool      `json:"is_pinned,omitempty"`
	IsArchived   bool      `json:"is_archived,omitempty"`
	IsFavorite   bool      `json:"is_favorite,omitempty"`
	WordCount    int       `json:"word_count,omitempty"`
	ReadingTime  int       `json:"reading_time_minutes,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	LastAccessed time.Time `json:"last_accessed,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
}

// Folder mirrors /api/folders entries.
type Folder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Color     string    `json:"color,omitempty"`
	Position  int       `json:"position,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Tag mirrors /api/tags entries.
type Tag struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id,omitempty"`
	Name       string `json:"name"`
	Color      string `json:"color,omitempty"`
	UsageCount int    `json:"usage_count,omitempty"`
}

// Template mirrors /api/templates entries.
type Template struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Content     string    `json:"content,omitempty"`
	Category    string    `json:"category,omitempty"`
	UsageCount  int       `json:"usage_count,omitempty"`
	IsPublic    bool      `json:"is_public,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// SearchResponse wraps /api/search results.
type SearchResponse struct {
	Results []Note `json:"results"`
	Query   string `json:"query"`
	Count   int    `json:"count"`
}

// SemanticSearchResult is one hit from /api/search/semantic.
type SemanticSearchResult struct {
	ID       string  `json:"id"`
	Score    float64 `json:"score,omitempty"`
	Title    string  `json:"title,omitempty"`
	Content  string  `json:"content,omitempty"`
	Summary  string  `json:"summary,omitempty"`
	FolderID string  `json:"folder_id,omitempty"`
}

// SemanticSearchResponse wraps /api/search/semantic results.
type SemanticSearchResponse struct {
	Results []SemanticSearchResult `json:"results"`
	Query   string                 `json:"query"`
	Count   int                    `json:"count"`
}
