// Package avatars provides types and operations for avatar management.
// Avatars represent visual characters in the 3D skill tree that can have prompts (skills) assigned.
package avatars

// Avatar represents a visual character in the skill tree.
type Avatar struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`   // hex color
	HeadColor   string   `json:"headColor"`   // hex color
	AccentColor string   `json:"accentColor"` // hex color
	Skills      []string `json:"skills"`      // Prompt IDs assigned to this avatar
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// AvatarsFile represents the structure of avatars.json.
type AvatarsFile struct {
	Avatars []Avatar `json:"avatars"`
}

// CreateRequest is the request body for creating an avatar.
type CreateRequest struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`
	HeadColor   string   `json:"headColor"`
	AccentColor string   `json:"accentColor"`
	Skills      []string `json:"skills,omitempty"`
}

// UpdateRequest is the request body for updating an avatar.
type UpdateRequest struct {
	Name        *string  `json:"name,omitempty"`
	BodyColor   *string  `json:"bodyColor,omitempty"`
	HeadColor   *string  `json:"headColor,omitempty"`
	AccentColor *string  `json:"accentColor,omitempty"`
	Skills      []string `json:"skills,omitempty"`
}

// Response is the API response for an avatar.
type Response struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`
	HeadColor   string   `json:"headColor"`
	AccentColor string   `json:"accentColor"`
	Skills      []string `json:"skills"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}
