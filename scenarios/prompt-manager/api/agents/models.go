// Package agents provides types and operations for agent management.
//
// DOC: docs/reference/api-endpoints.md#agents
package agents

// Response is the API response for an agent.
type Response struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"displayName"`
	Status      string         `json:"status"`
	Appearance  *AppearanceDTO `json:"appearance,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

// AppearanceDTO represents visual appearance for 3D UI
type AppearanceDTO struct {
	Body   string `json:"body"`
	Head   string `json:"head"`
	Accent string `json:"accent"`
}

// CreateRequest is the request body for creating an agent.
type CreateRequest struct {
	ID          string         `json:"id,omitempty"`
	DisplayName string         `json:"displayName"`
	Appearance  *AppearanceDTO `json:"appearance,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
}

// UpdateRequest is the request body for updating an agent.
type UpdateRequest struct {
	DisplayName *string        `json:"displayName,omitempty"`
	Status      *string        `json:"status,omitempty"`
	Appearance  *AppearanceDTO `json:"appearance,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
}

// SkillAssignmentRequest is the request body for assigning skills to an agent.
type SkillAssignmentRequest struct {
	SkillIDs []string `json:"skillIds"`
}

// EffectiveSkillsResponse is the response for the effective skills endpoint.
type EffectiveSkillsResponse struct {
	AgentID string   `json:"agentId"`
	TeamID  *string  `json:"teamId,omitempty"`
	Skills  []string `json:"skills"`
}

