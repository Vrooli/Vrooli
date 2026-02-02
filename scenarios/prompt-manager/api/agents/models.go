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

// AgentFileEntry represents a file or directory in an agent folder.
type AgentFileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

// AgentFileListResponse is the response for listing agent files.
type AgentFileListResponse struct {
	AgentID string           `json:"agentId"`
	Files   []AgentFileEntry `json:"files"`
}

// AgentFileContentResponse is the response for file content operations.
type AgentFileContentResponse struct {
	AgentID string `json:"agentId"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// AgentFileWriteRequest is the request body for writing file content.
type AgentFileWriteRequest struct {
	Content string `json:"content"`
}

// AgentFileCreateRequest is the request body for creating a file or directory.
type AgentFileCreateRequest struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	IsDir   bool   `json:"isDir,omitempty"`
}

// AgentFileRenameRequest is the request body for renaming a file.
type AgentFileRenameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}
