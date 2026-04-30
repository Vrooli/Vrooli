package store

import (
	"encoding/json"
	"prompt-manager/teamconfig"
	"time"
)

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
	DefaultScope string         `json:"defaultScope,omitempty"` // Default scope skill to include
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
	Capabilities      *AgentCapabilities `json:"capabilities,omitempty"`
	Connectors        []AgentConnector   `json:"connectors,omitempty"`
	DefaultProfileRef string             `json:"defaultProfileRef,omitempty"`
	Heartbeat         *AgentHeartbeat    `json:"heartbeat,omitempty"`
	Tags              []string           `json:"tags,omitempty"`
	FileOrder         []string           `json:"fileOrder,omitempty"`
	Runtime           *AgentRuntime      `json:"runtime,omitempty"`
	Timestamps
}

// AgentAppearance represents visual appearance for 3D UI
type AgentAppearance struct {
	Body   string `json:"body"`
	Head   string `json:"head"`
	Accent string `json:"accent"`
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

// AgentRuntime contains runtime state pointers
type AgentRuntime struct {
	WorkspaceRef string `json:"workspaceRef,omitempty"`
}

// Team represents a team entity from team.json
type Team struct {
	BaseEntity
	ID           string                  `json:"id"`
	DisplayName  string                  `json:"displayName"`
	Mission      string                  `json:"mission,omitempty"`
	Enabled      bool                    `json:"enabled"`
	EnabledSet   bool                    `json:"-"`
	Runtime      teamconfig.Runtime      `json:"runtime"`
	Coordination teamconfig.Coordination `json:"coordination"`
	Execution    teamconfig.Execution    `json:"execution"`
	DecisionMode string                  `json:"decisionMode,omitempty"` // "yolo" or "approval"
	Shared       *TeamShared             `json:"shared,omitempty"`
	Retention    *RetentionConfig        `json:"retention,omitempty"`
	Timestamps
}

// TeamShared configures shared document access
type TeamShared struct {
	Path      string `json:"path"`
	MountHint string `json:"mountHint,omitempty"` // readOnly or readWrite
}

// RetentionConfig controls automatic cleanup of completed tasks, decisions, and knowledge.
type RetentionConfig struct {
	Tasks     *TaskRetention  `json:"tasks,omitempty"`
	Decisions *EntryRetention `json:"decisions,omitempty"`
	Knowledge *EntryRetention `json:"knowledge,omitempty"`
}

// TaskRetention controls how completed tasks are pruned.
type TaskRetention struct {
	MaxCompleted int `json:"maxCompleted"` // keep N newest completed tasks (0 = unlimited)
	MaxAgeDays   int `json:"maxAgeDays"`   // remove completed tasks older than N days (0 = unlimited)
}

// EntryRetention controls how decision/knowledge entries are pruned.
type EntryRetention struct {
	MaxEntries int `json:"maxEntries"` // keep N newest entries (0 = unlimited)
	MaxAgeDays int `json:"maxAgeDays"` // remove entries older than N days (0 = unlimited)
}

// DefaultRetentionConfig returns the default retention policy.
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		Tasks:     &TaskRetention{MaxCompleted: 20, MaxAgeDays: 30},
		Decisions: &EntryRetention{MaxEntries: 50, MaxAgeDays: 90},
		Knowledge: &EntryRetention{MaxEntries: 100, MaxAgeDays: 180},
	}
}

// EffectiveRetention returns the team's retention config with defaults for nil sub-fields.
func (t *Team) EffectiveRetention() RetentionConfig {
	d := DefaultRetentionConfig()
	if t.Retention == nil {
		return d
	}
	r := *t.Retention
	if r.Tasks == nil {
		r.Tasks = d.Tasks
	}
	if r.Decisions == nil {
		r.Decisions = d.Decisions
	}
	if r.Knowledge == nil {
		r.Knowledge = d.Knowledge
	}
	return r
}

func (t *Team) Contract() teamconfig.Contract {
	if t == nil {
		return teamconfig.Contract{}
	}
	return teamconfig.Contract{
		Runtime:      t.Runtime,
		Coordination: t.Coordination,
		Execution:    t.Execution,
		DecisionMode: t.DecisionMode,
	}
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

// TeamInbox stores inbound messages for a team member.
type TeamInbox struct {
	BaseEntity
	TeamID   string        `json:"teamId"`
	AgentID  string        `json:"agentId"`
	Messages []TeamMessage `json:"messages"`
}

// TeamMessage represents a message sent between team members.
type TeamMessage struct {
	ID          string `json:"id"`
	TeamID      string `json:"teamId"`
	FromAgentID string `json:"fromAgentId"`
	ToAgentID   string `json:"toAgentId"`
	Content     string `json:"content"`
	CreatedAt   string `json:"createdAt"`
}

// TeamMemberRelation represents a team-member relationship
type TeamMemberRelation struct {
	BaseEntity
	TeamID  string   `json:"teamId"`
	AgentID string   `json:"agentId"`
	Roles   []string `json:"roles,omitempty"`
	Status  string   `json:"status"`
}

// Topic represents a topic entity from topic.json
type Topic struct {
	BaseEntity
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ParentTopicID *string  `json:"parentTopicId,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	Status        string   `json:"status"`
	Timestamps
}

// TopicsIndex is the generated index of all topics
type TopicsIndex struct {
	GeneratedAt string             `json:"generatedAt"`
	Items       []TopicsIndexEntry `json:"items"`
}

// TopicsIndexEntry is a single entry in the topics index
type TopicsIndexEntry struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ParentTopicID *string `json:"parentTopicId,omitempty"`
	SkillCount    int     `json:"skillCount"`
	Status        string  `json:"status"`
}

// HistoryEntry represents a version in the skill history
type HistoryEntry struct {
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Summary   string `json:"summary"`
}

// Variant represents an alternative version of a skill's content for A/B testing.
type Variant struct {
	BaseEntity
	ID          string `json:"id"`
	SkillID     string `json:"skillId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Entry       string `json:"entry"` // filename, e.g., "VARIANT.md"
	Timestamps
}

// Experiment ties skill variants together for A/B testing with weighted traffic splitting.
type Experiment struct {
	BaseEntity
	ID              string          `json:"id"`
	SkillID         string          `json:"skillId"`
	Name            string          `json:"name"`
	Hypothesis      string          `json:"hypothesis,omitempty"`
	Status          string          `json:"status"` // draft, running, concluded
	Arms            []ExperimentArm `json:"arms"`
	StartedAt       *string         `json:"startedAt,omitempty"`
	ConcludedAt     *string         `json:"concludedAt,omitempty"`
	WinnerVariantID *string         `json:"winnerVariantId,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	Timestamps
}

// ExperimentArm is one variant in an experiment with its traffic weight.
type ExperimentArm struct {
	VariantID string  `json:"variantId"` // "control" = original SKILL.md
	Weight    float64 `json:"weight"`    // 0.0-1.0, arms must sum to 1.0
}

// ExperimentOutcome is the opaque outcome envelope stored per experiment.
// Prompt-manager never parses Data — it belongs to the source system (e.g. swarm-manager).
type ExperimentOutcome struct {
	VariantID     string          `json:"variantId"`
	Source        string          `json:"source"`        // e.g. "swarm-manager"
	SchemaVersion int             `json:"schemaVersion"` // consumer-defined
	RecordedAt    string          `json:"recordedAt"`
	Data          json.RawMessage `json:"data"` // opaque to PM
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

// HeartbeatConfig represents the heartbeat configuration for a team member
//
// DOC: docs/concepts/HEARTBEATS.md
type HeartbeatConfig struct {
	BaseEntity
	TeamID         string               `json:"teamId"`
	AgentID        string               `json:"agentId"`
	Enabled        bool                 `json:"enabled"`
	Schedule       string               `json:"schedule"`                 // Cron expression
	ProfileKey     string               `json:"profileKey,omitempty"`     // agent-manager profile key
	TimeoutSeconds int                  `json:"timeoutSeconds,omitempty"` // 0 = use default
	LastExecution  *HeartbeatExecResult `json:"lastExecution,omitempty"`
	Timestamps
}

// HeartbeatExecResult represents the result of a heartbeat execution
type HeartbeatExecResult struct {
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt,omitempty"`
	Status    string `json:"status"` // running, completed, failed
	RunID     string `json:"runId,omitempty"`
	LogPath   string `json:"logPath,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Heartbeat config constants
const (
	KindHeartbeatConfig = "heartbeat-config"

	HeartbeatStatusRunning   = "running"
	HeartbeatStatusCompleted = "completed"
	HeartbeatStatusFailed    = "failed"
	HeartbeatStatusCancelled = "cancelled"

	DefaultHeartbeatTimeoutSeconds = 2700 // 45 minutes
	MinHeartbeatTimeoutSeconds     = 60   // 1 minute
	MaxHeartbeatTimeoutSeconds     = 7200 // 120 minutes
)

// EffectiveTimeout returns the timeout duration for this config,
// clamped to [Min, Max] and defaulting to 45 minutes when unset.
func (c *HeartbeatConfig) EffectiveTimeout() time.Duration {
	s := c.TimeoutSeconds
	if s <= 0 {
		s = DefaultHeartbeatTimeoutSeconds
	}
	if s < MinHeartbeatTimeoutSeconds {
		s = MinHeartbeatTimeoutSeconds
	}
	if s > MaxHeartbeatTimeoutSeconds {
		s = MaxHeartbeatTimeoutSeconds
	}
	return time.Duration(s) * time.Second
}

// --- Handoff ---

// HandoffEntry represents a single handoff record in the team's handoff history.
type HandoffEntry struct {
	AgentID   string `json:"agentId"`
	RunID     string `json:"runId"`
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
}

// --- Task Board ---

// TaskNote represents a note appended to a task.
type TaskNote struct {
	At   string `json:"at"`
	By   string `json:"by"`
	Text string `json:"text"`
}

// TeamTask represents a task on the team's shared task board.
type TeamTask struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`    // "todo", "in-progress", "blocked", "done"
	Assignee  string     `json:"assignee"`  // agent ID
	Priority  string     `json:"priority"`  // "P1"-"P5"
	CreatedBy string     `json:"createdBy"` // agent ID
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
	Notes     []TaskNote `json:"notes,omitempty"`
}

// TeamTaskBoard holds the full task board for a team.
type TeamTaskBoard struct {
	Tasks []TeamTask `json:"tasks"`
}

// --- Decision Log ---

// Decision status constants.
const (
	DecisionStatusPending   = "pending"
	DecisionStatusAccepted  = "accepted"
	DecisionStatusRejected  = "rejected"
	DecisionStatusRunning   = "running"
	DecisionStatusCompleted = "completed"
	DecisionStatusDeferred  = "deferred"
)

// MaxRevisitAfterDays caps how far in the future a deferred decision's
// revisit_after may be set. Catches typo'd far-future dates cheaply.
const MaxRevisitAfterDays = 365

// DecisionOption represents a lettered choice in a multi-option decision.
type DecisionOption struct {
	Key         string `json:"key"`                   // "A", "B", "C", etc.
	Label       string `json:"label"`                 // short description
	Rationale   string `json:"rationale"`             // why this option
	Recommended bool   `json:"recommended,omitempty"` // agent-suggested pick
}

// DecisionModifications captures structured, scoped exceptions an operator
// attaches to an accepted option. `excluded_clauses` lists parts of the
// option's rationale the operator does NOT accept; `additions` lists extra
// constraints or scope added on top; `rationale` explains the modification.
// Contract: see docs/reference/decision-modifications-contract.md.
type DecisionModifications struct {
	ExcludedClauses []string `json:"excluded_clauses,omitempty"`
	Additions       []string `json:"additions,omitempty"`
	Rationale       string   `json:"rationale,omitempty"`
}

// Auto-create status constants for `initiative-proposal` decisions.
const (
	AutoCreateStatusUnset   = ""
	AutoCreateStatusPending = "pending"
	AutoCreateStatusCreated = "created"
	AutoCreateStatusFailed  = "failed"
)

// DecisionContextInitiativeProposal is the well-known context tag that
// triggers initiative auto-creation on decision-accept.
const DecisionContextInitiativeProposal = "initiative-proposal"

// DecisionInitiativeMetadata is the structured block carried on a decision
// whose context is `initiative-proposal`. It is the authoritative source of
// the initiative's identity (name, priority, depends_on, target scenario)
// and is consumed at decision-accept time to construct the swarm-manager
// initiative. Immutable once the decision has been accepted.
// Contract: see docs/reference/decision-initiative-proposal-contract.md.
type DecisionInitiativeMetadata struct {
	Name           string   `json:"name"`                      // initiative name (kebab-case)
	Priority       int      `json:"priority,omitempty"`        // 0 (unset) or 1-10
	DependsOn      []string `json:"depends_on,omitempty"`      // initiative name refs
	TargetScenario string   `json:"target_scenario,omitempty"` // e.g. "swarm-manager"
	Title          string   `json:"title,omitempty"`           // optional override; default = decision topic
}

// DecisionEntry represents a recorded decision in the team's decision log.
type DecisionEntry struct {
	ID            string                 `json:"id"`
	At            string                 `json:"at"`
	By            string                 `json:"by"` // agent ID
	Decision      string                 `json:"decision"`
	Rationale     string                 `json:"rationale"`
	Context       string                 `json:"context,omitempty"`     // tag/topic grouping
	Supersedes    string                 `json:"supersedes,omitempty"`  // ID of decision this replaces
	Status        string                 `json:"status,omitempty"`      // "pending", "accepted", "rejected"
	Topic         string                 `json:"topic,omitempty"`       // what is being decided (multi-option)
	Description   string                 `json:"description,omitempty"` // background/context for multi-option decisions
	Options       []DecisionOption       `json:"options,omitempty"`     // lettered choices
	Selected      string                 `json:"selected,omitempty"`    // chosen option key or "__other__"
	Freeform      string                 `json:"freeform,omitempty"`    // custom response text
	Notes         string                 `json:"notes,omitempty"`       // additional human context
	Modifications *DecisionModifications `json:"modifications,omitempty"`
	RevisitAfter  *string                `json:"revisit_after,omitempty"` // ISO-8601 date (YYYY-MM-DD); set when status=deferred
	// AcceptedAsProposed is true when a single-proposal decision (no Options)
	// was accepted without picking from a list. Selected/Freeform stay empty
	// in this case; this flag is the explicit, queryable marker.
	AcceptedAsProposed bool `json:"accepted_as_proposed,omitempty"`
	// InitiativeMetadata is set when Context == "initiative-proposal" and
	// drives initiative auto-creation on accept. Immutable post-accept.
	InitiativeMetadata *DecisionInitiativeMetadata `json:"initiative_metadata,omitempty"`
	// AutoCreateStatus tracks whether the auto-create attempt succeeded.
	// One of: "" (unset), "pending", "created", "failed".
	AutoCreateStatus string `json:"auto_create_status,omitempty"`
	// AutoCreateError carries the failure reason when AutoCreateStatus="failed".
	AutoCreateError string `json:"auto_create_error,omitempty"`
	// AutoCreateInitiativeRef is the "<scenario>/<name>" reference of the
	// successfully created initiative.
	AutoCreateInitiativeRef string `json:"auto_create_initiative_ref,omitempty"`
}

// --- Knowledge Log ---

// KnowledgeEntry represents a piece of team knowledge persisted across heartbeats.
type KnowledgeEntry struct {
	ID         string `json:"id"`
	At         string `json:"at"`
	By         string `json:"by"`
	Topic      string `json:"topic"`
	Content    string `json:"content"`
	Source     string `json:"source,omitempty"`
	Supersedes string `json:"supersedes,omitempty"`
}
