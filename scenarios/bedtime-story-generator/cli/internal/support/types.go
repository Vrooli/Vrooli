package support

import "time"

// Story mirrors the Story shape returned by /api/v1/stories and /api/v1/stories/{id}.
type Story struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	AgeGroup       string            `json:"age_group"`
	Theme          string            `json:"theme"`
	StoryLength    string            `json:"story_length"`
	ReadingTime    int               `json:"reading_time_minutes"`
	CharacterNames []string          `json:"character_names"`
	PageCount      int               `json:"page_count"`
	CreatedAt      time.Time         `json:"created_at"`
	TimesRead      int               `json:"times_read"`
	LastRead       *time.Time        `json:"last_read"`
	IsFavorite     bool              `json:"is_favorite"`
	Illustrations  map[string]string `json:"illustrations,omitempty"`
}

// GenerateStoryRequest is the JSON body sent to POST /api/v1/stories/generate.
type GenerateStoryRequest struct {
	AgeGroup       string   `json:"age_group"`
	Theme          string   `json:"theme"`
	Length         string   `json:"length"`
	CharacterNames []string `json:"character_names"`
}

// FavoriteResponse is the shape returned by POST /api/v1/stories/{id}/favorite.
type FavoriteResponse struct {
	Success    bool `json:"success"`
	IsFavorite bool `json:"is_favorite"`
}

// ReadingSessionResponse is the shape returned by POST /api/v1/stories/{id}/read.
type ReadingSessionResponse struct {
	SessionID string `json:"session_id"`
	StoryID   string `json:"story_id"`
}

// Theme is one entry returned by GET /api/v1/themes.
type Theme struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Emoji       string `json:"emoji,omitempty"`
	Color       string `json:"color,omitempty"`
}
