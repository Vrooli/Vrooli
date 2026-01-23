// Package members provides types and operations for member management.
// Members represent virtual employees in the 3D skill tree that can have skills assigned.
package members

// Member represents a virtual employee in the skill tree.
type Member struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`   // hex color
	HeadColor   string   `json:"headColor"`   // hex color
	AccentColor string   `json:"accentColor"` // hex color
	Skills      []string `json:"skills"`      // Skill IDs assigned to this member
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// MembersFile represents the structure of members.json.
type MembersFile struct {
	Members []Member `json:"members"`
}

// CreateRequest is the request body for creating a member.
type CreateRequest struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`
	HeadColor   string   `json:"headColor"`
	AccentColor string   `json:"accentColor"`
	Skills      []string `json:"skills,omitempty"`
}

// UpdateRequest is the request body for updating a member.
type UpdateRequest struct {
	Name        *string  `json:"name,omitempty"`
	BodyColor   *string  `json:"bodyColor,omitempty"`
	HeadColor   *string  `json:"headColor,omitempty"`
	AccentColor *string  `json:"accentColor,omitempty"`
	Skills      []string `json:"skills,omitempty"`
}

// Response is the API response for a member.
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
