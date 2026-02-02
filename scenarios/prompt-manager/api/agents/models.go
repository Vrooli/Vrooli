// Package agents provides types and operations for agent management.
//
// DOC: docs/reference/api-endpoints.md#agents
package agents

// Response is the API response for an agent.
type Response struct {
	ID                string           `json:"id"`
	DisplayName       string           `json:"displayName"`
	Description       string           `json:"description,omitempty"`
	Status            string           `json:"status"`
	Appearance        *AppearanceDTO   `json:"appearance,omitempty"`
	Capabilities      *CapabilitiesDTO `json:"capabilities,omitempty"`
	Connectors        []ConnectorDTO   `json:"connectors,omitempty"`
	DefaultProfileRef string           `json:"defaultProfileRef,omitempty"`
	Heartbeat         *HeartbeatDTO    `json:"heartbeat,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
	Skills            []string         `json:"skills,omitempty"`
	CreatedAt         string           `json:"createdAt"`
	UpdatedAt         string           `json:"updatedAt"`
}

// AppearanceDTO represents visual appearance for 3D UI
type AppearanceDTO struct {
	Body   string `json:"body"`
	Head   string `json:"head"`
	Accent string `json:"accent"`
}

// CapabilityDTO represents a single capability
type CapabilityDTO struct {
	CapabilityID string   `json:"capabilityId"`
	Verbs        []string `json:"verbs,omitempty"`
}

// CapabilitiesDTO represents capability requirements and provisions
type CapabilitiesDTO struct {
	Provides []CapabilityDTO `json:"provides,omitempty"`
	Requires []CapabilityDTO `json:"requires,omitempty"`
}

// ConnectorDTO represents an external system connector
type ConnectorDTO struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// HeartbeatDTO represents health monitoring configuration
type HeartbeatDTO struct {
	IntervalSeconds int `json:"intervalSeconds,omitempty"`
	TimeoutSeconds  int `json:"timeoutSeconds,omitempty"`
	MaxMissedBeats  int `json:"maxMissedBeats,omitempty"`
}

// CreateRequest is the request body for creating an agent.
type CreateRequest struct {
	ID                string           `json:"id,omitempty"`
	DisplayName       string           `json:"displayName"`
	Description       string           `json:"description,omitempty"`
	Appearance        *AppearanceDTO   `json:"appearance,omitempty"`
	Capabilities      *CapabilitiesDTO `json:"capabilities,omitempty"`
	Connectors        []ConnectorDTO   `json:"connectors,omitempty"`
	DefaultProfileRef string           `json:"defaultProfileRef,omitempty"`
	Heartbeat         *HeartbeatDTO    `json:"heartbeat,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
	Skills            []string         `json:"skills,omitempty"`
}

// UpdateRequest is the request body for updating an agent.
type UpdateRequest struct {
	DisplayName       *string          `json:"displayName,omitempty"`
	Description       *string          `json:"description,omitempty"`
	Status            *string          `json:"status,omitempty"`
	Appearance        *AppearanceDTO   `json:"appearance,omitempty"`
	Capabilities      *CapabilitiesDTO `json:"capabilities,omitempty"`
	Connectors        []ConnectorDTO   `json:"connectors,omitempty"`
	DefaultProfileRef *string          `json:"defaultProfileRef,omitempty"`
	Heartbeat         *HeartbeatDTO    `json:"heartbeat,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
	Skills            []string         `json:"skills,omitempty"`
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

// SoulRequest is the request body for setting SOUL.md content.
type SoulRequest struct {
	Content string `json:"content"`
}

// SoulResponse is the response for SOUL.md operations.
type SoulResponse struct {
	AgentID string `json:"agentId"`
	Content string `json:"content"`
}
