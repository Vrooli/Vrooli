package store

// Skill represents a skill entity from skill.json
type Skill struct {
	BaseEntity
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Modes        []string       `json:"modes,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Icon         string         `json:"icon,omitempty"`
	Status       string         `json:"status"`
	Entry        string         `json:"entry"`
	TargetToolID *string        `json:"targetToolId,omitempty"`
	Requires     *SkillRequires `json:"requires,omitempty"`
	Timestamps

	// Runtime fields (not persisted in skill.json)
	Pack string `json:"-"` // Which pack this skill belongs to
}

// Agent represents an agent entity from agent.json
type Agent struct {
	BaseEntity
	ID                string             `json:"id"`
	DisplayName       string             `json:"displayName"`
	Description       string             `json:"description,omitempty"`
	Status            string             `json:"status"`
	Appearance        *AgentAppearance   `json:"appearance,omitempty"`
	Persona           *AgentPersona      `json:"persona,omitempty"`
	Capabilities      *AgentCapabilities `json:"capabilities,omitempty"`
	Connectors        []AgentConnector   `json:"connectors,omitempty"`
	DefaultProfileRef string             `json:"defaultProfileRef,omitempty"`
	Heartbeat         *AgentHeartbeat    `json:"heartbeat,omitempty"`
	Tags              []string           `json:"tags,omitempty"`
	SkillPins         []SkillPin         `json:"skillPins,omitempty"`
	Runtime           *AgentRuntime      `json:"runtime,omitempty"`
	Timestamps
}

// AgentAppearance represents visual appearance for 3D UI
type AgentAppearance struct {
	Body   string `json:"body"`
	Head   string `json:"head"`
	Accent string `json:"accent"`
}

// AgentPersona represents agent personality and behavior configuration
type AgentPersona struct {
	Entry              string   `json:"entry,omitempty"`
	Voice              string   `json:"voice,omitempty"`
	Traits             []string `json:"traits,omitempty"`
	SystemPromptPrefix string   `json:"systemPromptPrefix,omitempty"`
}

// AgentCapability represents a single capability with verbs
type AgentCapability struct {
	CapabilityID string   `json:"capabilityId"`
	Verbs        []string `json:"verbs,omitempty"`
}

// AgentCapabilities represents capability requirements and provisions
type AgentCapabilities struct {
	Provides []AgentCapability `json:"provides,omitempty"`
	Requires []AgentCapability `json:"requires,omitempty"`
}

// AgentConnector represents an external system connector
type AgentConnector struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// AgentHeartbeat represents health monitoring configuration
type AgentHeartbeat struct {
	IntervalSeconds int `json:"intervalSeconds,omitempty"`
	TimeoutSeconds  int `json:"timeoutSeconds,omitempty"`
	MaxMissedBeats  int `json:"maxMissedBeats,omitempty"`
}

// SkillPin represents an inline skill pin on an agent
type SkillPin struct {
	SkillID string `json:"skillId"`
	Version string `json:"version,omitempty"`
}

// AgentRuntime contains runtime state pointers
type AgentRuntime struct {
	WorkspaceRef string `json:"workspaceRef,omitempty"`
}

// Team represents a team entity from team.json
type Team struct {
	BaseEntity
	ID          string        `json:"id"`
	DisplayName string        `json:"displayName"`
	Mission     string        `json:"mission,omitempty"`
	Shared      *TeamShared   `json:"shared,omitempty"`
	Defaults    *TeamDefaults `json:"defaults,omitempty"`
	Timestamps
}

// TeamShared configures shared document access
type TeamShared struct {
	Path      string `json:"path"`
	MountHint string `json:"mountHint,omitempty"` // readOnly or readWrite
}

// TeamDefaults contains default policies for team members
type TeamDefaults struct {
	SkillGrantsByRole map[string][]string `json:"skillGrantsByRole,omitempty"`
}

// TeamRoles represents role definitions for a team
type TeamRoles struct {
	BaseEntity
	TeamID string `json:"teamId"`
	Roles  []Role `json:"roles"`
}

// Role represents a single role definition
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// OrgChart represents organizational hierarchy for a team
type OrgChart struct {
	BaseEntity
	TeamID string    `json:"teamId"`
	Edges  []OrgEdge `json:"edges"`
}

// OrgEdge represents a manager-report relationship
type OrgEdge struct {
	ManagerAgentID string `json:"managerAgentId"`
	ReportAgentID  string `json:"reportAgentId"`
}

// AgentSkillRelation represents an agent-skill relationship
type AgentSkillRelation struct {
	BaseEntity
	AgentID string `json:"agentId"`
	SkillID string `json:"skillId"`
	Pin     string `json:"pin"`
	Enabled bool   `json:"enabled"`
}

// TeamMemberRelation represents a team-member relationship
type TeamMemberRelation struct {
	BaseEntity
	TeamID  string   `json:"teamId"`
	AgentID string   `json:"agentId"`
	Roles   []string `json:"roles,omitempty"`
	Status  string   `json:"status"`
}

// HistoryEntry represents a version in the skill history
type HistoryEntry struct {
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Summary   string `json:"summary"`
}

// PackOrder represents the pack precedence configuration
type PackOrder struct {
	ActivePacks   []string `json:"activePacks"`
	InactivePacks []string `json:"inactivePacks"`
}

// Index types

// SkillsIndex is the generated index of all skills
type SkillsIndex struct {
	GeneratedAt string             `json:"generatedAt"`
	Items       []SkillsIndexEntry `json:"items"`
}

// SkillsIndexEntry is a single entry in the skills index
type SkillsIndexEntry struct {
	ID     string   `json:"id"`
	Pack   string   `json:"pack"`
	Name   string   `json:"name"`
	Modes  []string `json:"modes,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Status string   `json:"status"`
}

// AgentsIndex is the generated index of all agents
type AgentsIndex struct {
	GeneratedAt string             `json:"generatedAt"`
	Items       []AgentsIndexEntry `json:"items"`
}

// AgentsIndexEntry is a single entry in the agents index
type AgentsIndexEntry struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

// TeamsIndex is the generated index of all teams
type TeamsIndex struct {
	GeneratedAt string            `json:"generatedAt"`
	Items       []TeamsIndexEntry `json:"items"`
}

// TeamsIndexEntry is a single entry in the teams index
type TeamsIndexEntry struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	MemberCount int    `json:"memberCount"`
}
