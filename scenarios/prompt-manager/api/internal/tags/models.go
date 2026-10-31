// Package tags provides tag management for prompt categorization.
// Tags are stored in embedded SQLite and can be used to filter/organize prompts.
package tags

// Tag represents a tag in the database.
type Tag struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}
