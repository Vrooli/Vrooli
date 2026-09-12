// Package interop provides types and converters for mapping prompt-manager
// team definitions to and from external tool-specific team configurations.
package interop

import "prompt-manager/internal/store"

// Converter translates between prompt-manager's internal team model and
// an external tool's team configuration format.
type Converter interface {
	// ToolID returns the unique identifier for the target tool (e.g. "claude-code").
	ToolID() string

	// FromPMTeam converts a prompt-manager team snapshot into the tool's team config.
	FromPMTeam(snapshot *PMTeamSnapshot) (*ToolTeamConfig, error)

	// ToPMTeam converts a tool team config into prompt-manager import structs.
	ToPMTeam(config *ToolTeamConfig) (*PMTeamImport, error)

	// FormatSpawnPrompt generates a structured prompt that instructs the tool's
	// lead agent to bootstrap a team session.
	FormatSpawnPrompt(config *ToolTeamConfig, ctx SpawnContext) (string, error)
}

// PMTeamSnapshot is a read-only view of a prompt-manager team and its members,
// roles, and org structure. It is the input for Converter.FromPMTeam.
type PMTeamSnapshot struct {
	Team     store.Team
	Members  []PMTeamMember
	Roles    []store.Role
	OrgEdges []store.OrgEdge
}

// PMTeamMember pairs a team-member relation with its resolved agent and any
// supplemental metadata carried during export.
type PMTeamMember struct {
	Relation         store.TeamMemberRelation
	Agent            store.Agent
	Responsibilities string
	HeartbeatInstr   string
}

// ToolTeamConfig is the tool-agnostic representation of a team configuration
// that external tools produce and consume.
type ToolTeamConfig struct {
	TeamName    string         `json:"teamName"`
	Description string         `json:"description,omitempty"`
	Members     []ToolMember   `json:"members"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ToolMember describes a single team member in tool-agnostic terms.
type ToolMember struct {
	Name      string         `json:"name"`
	AgentType string         `json:"agentType"`
	Model     string         `json:"model,omitempty"`
	Mode      string         `json:"mode,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// PMTeamImport holds the set of entities that should be created or updated
// when importing a team from an external tool.
type PMTeamImport struct {
	Team     store.Team
	Agents   []store.Agent
	Members  []store.TeamMemberRelation
	OrgEdges []store.OrgEdge
}

// SpawnContext provides runtime context for prompt generation.
type SpawnContext struct {
	WorkingDir    string
	VrooliRoot    string
	TeamID        string
	AdditionalCtx string
}
