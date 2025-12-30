// Package types defines the data structures used by the prompt-manager CLI.
package types

import "time"

// Campaign represents a collection of related prompts.
type Campaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Color       string     `json:"color"`
	Icon        string     `json:"icon"`
	ParentID    *string    `json:"parent_id"`
	SortOrder   int        `json:"sort_order"`
	IsFavorite  bool       `json:"is_favorite"`
	PromptCount int        `json:"prompt_count"`
	LastUsed    *time.Time `json:"last_used"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Prompt represents a saved prompt with metadata.
type Prompt struct {
	ID                  string     `json:"id"`
	CampaignID          string     `json:"campaign_id"`
	Title               string     `json:"title"`
	Content             string     `json:"content"`
	Description         *string    `json:"description"`
	Variables           []string   `json:"variables"`
	UsageCount          int        `json:"usage_count"`
	LastUsed            *time.Time `json:"last_used"`
	IsFavorite          bool       `json:"is_favorite"`
	IsArchived          bool       `json:"is_archived"`
	QuickAccessKey      *string    `json:"quick_access_key"`
	Version             int        `json:"version"`
	ParentVersionID     *string    `json:"parent_version_id"`
	WordCount           *int       `json:"word_count"`
	EstimatedTokens     *int       `json:"estimated_tokens"`
	EffectivenessRating *int       `json:"effectiveness_rating"`
	Notes               *string    `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	// Joined fields from API
	CampaignName *string  `json:"campaign_name,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// PromptVersion represents a historical version of a prompt.
type PromptVersion struct {
	ID            string    `json:"id"`
	PromptID      string    `json:"prompt_id"`
	VersionNumber int       `json:"version_number"`
	FilePath      string    `json:"file_path"`
	ContentCache  *string   `json:"content_cache"`
	Variables     []string  `json:"variables"`
	ChangeSummary *string   `json:"change_summary"`
	CreatedBy     *string   `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateCampaignRequest is the payload for creating a new campaign.
type CreateCampaignRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
	IsFavorite  *bool   `json:"is_favorite,omitempty"`
}

// CreatePromptRequest is the payload for creating a new prompt.
type CreatePromptRequest struct {
	CampaignID          string   `json:"campaign_id"`
	Title               string   `json:"title"`
	Content             string   `json:"content"`
	Description         *string  `json:"description,omitempty"`
	Variables           []string `json:"variables,omitempty"`
	QuickAccessKey      *string  `json:"quick_access_key,omitempty"`
	EffectivenessRating *int     `json:"effectiveness_rating,omitempty"`
	Notes               *string  `json:"notes,omitempty"`
	Tags                []string `json:"tags,omitempty"`
}

// UpdatePromptRequest is the payload for updating an existing prompt.
type UpdatePromptRequest struct {
	Title               *string  `json:"title,omitempty"`
	Content             *string  `json:"content,omitempty"`
	Description         *string  `json:"description,omitempty"`
	Variables           []string `json:"variables,omitempty"`
	IsFavorite          *bool    `json:"is_favorite,omitempty"`
	IsArchived          *bool    `json:"is_archived,omitempty"`
	QuickAccessKey      *string  `json:"quick_access_key,omitempty"`
	EffectivenessRating *int     `json:"effectiveness_rating,omitempty"`
	Notes               *string  `json:"notes,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	ChangeSummary       *string  `json:"change_summary,omitempty"`
}

// Helper functions for optional string values
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// BoolPtr returns a pointer to a bool value.
func BoolPtr(b bool) *bool {
	return &b
}
