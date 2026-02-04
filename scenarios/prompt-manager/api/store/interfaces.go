package store

import (
	"context"
)

// SkillStore defines operations for skill storage
type SkillStore interface {
	// List returns all skills from active packs
	List(ctx context.Context) ([]Skill, error)

	// Get retrieves a skill by ID, searching through active packs
	Get(ctx context.Context, id string) (*Skill, error)

	// GetWithContent retrieves a skill with its content
	GetWithContent(ctx context.Context, id string) (*Skill, string, error)

	// Create creates a new skill in the specified pack
	Create(ctx context.Context, pack string, skill *Skill, content string) error

	// Update updates an existing skill
	Update(ctx context.Context, id string, skill *Skill, content *string) error

	// Delete removes a skill
	Delete(ctx context.Context, id string) error

	// GetVersionHistory returns version history for a skill
	GetVersionHistory(ctx context.Context, id string) ([]HistoryEntry, error)

	// ContentPath returns the absolute path where a skill's content is stored.
	// Used by adapters for direct content operations during move operations.
	ContentPath(pack, skillID string) string

	// Rename changes a skill's ID by renaming its directory and updating skill.json.
	// Returns the updated skill with the new ID.
	Rename(ctx context.Context, oldID, newID string) (*Skill, error)
}

// AgentStore defines operations for agent storage
type AgentStore interface {
	// List returns all agents
	List(ctx context.Context) ([]Agent, error)

	// Get retrieves an agent by ID
	Get(ctx context.Context, id string) (*Agent, error)

	// Create creates a new agent
	Create(ctx context.Context, agent *Agent) error

	// Update updates an existing agent
	Update(ctx context.Context, id string, agent *Agent) error

	// Delete removes an agent
	Delete(ctx context.Context, id string) error
}

// TeamStore defines operations for team storage
type TeamStore interface {
	// List returns all teams
	List(ctx context.Context) ([]Team, error)

	// Get retrieves a team by ID
	Get(ctx context.Context, id string) (*Team, error)

	// Create creates a new team
	Create(ctx context.Context, team *Team) error

	// Update updates an existing team
	Update(ctx context.Context, id string, team *Team) error

	// Delete removes a team
	Delete(ctx context.Context, id string) error

	// GetRoles returns role definitions for a team
	GetRoles(ctx context.Context, teamID string) (*TeamRoles, error)

	// SetRoles sets role definitions for a team
	SetRoles(ctx context.Context, teamID string, roles *TeamRoles) error

	// GetOrgChart returns the org chart for a team
	GetOrgChart(ctx context.Context, teamID string) (*OrgChart, error)

	// SetOrgChart sets the org chart for a team
	SetOrgChart(ctx context.Context, teamID string, org *OrgChart) error

	// GetMembers returns all members of a team
	GetMembers(ctx context.Context, teamID string) ([]TeamMemberRelation, error)

	// GetInbox returns the inbox for a team member
	GetInbox(ctx context.Context, teamID, agentID string) (*TeamInbox, error)

	// SetInbox stores the inbox for a team member
	SetInbox(ctx context.Context, teamID, agentID string, inbox *TeamInbox) error
}

// RelationStore defines operations for relationship storage
type RelationStore interface {
	// TeamMember operations
	GetTeamMember(ctx context.Context, teamID, agentID string) (*TeamMemberRelation, error)
	SetTeamMember(ctx context.Context, rel *TeamMemberRelation) error
	DeleteTeamMember(ctx context.Context, teamID, agentID string) error
	ListTeamMembers(ctx context.Context, teamID string) ([]TeamMemberRelation, error)
}

// IndexStore defines operations for index management
type IndexStore interface {
	// RegenerateAll regenerates all indexes from entity files
	RegenerateAll(ctx context.Context) error

	// RegenerateSkills regenerates the skills index
	RegenerateSkills(ctx context.Context) error

	// RegenerateAgents regenerates the agents index
	RegenerateAgents(ctx context.Context) error

	// RegenerateTeams regenerates the teams index
	RegenerateTeams(ctx context.Context) error

	// GetSkillsIndex returns the current skills index
	GetSkillsIndex(ctx context.Context) (*SkillsIndex, error)

	// GetAgentsIndex returns the current agents index
	GetAgentsIndex(ctx context.Context) (*AgentsIndex, error)

	// GetTeamsIndex returns the current teams index
	GetTeamsIndex(ctx context.Context) (*TeamsIndex, error)
}

// Store combines all store interfaces
type Store interface {
	Skills() SkillStore
	Agents() AgentStore
	Teams() TeamStore
	Relations() RelationStore
	Indexes() IndexStore
}

// ContentIO abstracts filesystem operations for skill content.
// This is the testing seam for filesystem interactions in the adapter layer.
// Production: Uses real filesystem via FileContentIO
// Testing: Uses MockContentIO for side-effect-free tests
type ContentIO interface {
	// ReadContent reads content from a file path
	ReadContent(path string) (string, error)

	// WriteContent writes content to a file path, creating directories as needed
	WriteContent(path, content string) error
}
