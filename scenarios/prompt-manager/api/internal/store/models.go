package store

import (
	"encoding/json"
	"time"

	"prompt-manager/internal/teamconfig"
	"prompt-manager/internal/teamcontract"
)

// Skill represents a skill entity from skill.json
type Skill struct {
	BaseEntity
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	Status       string   `json:"status"`
	Entry        string   `json:"entry"`
	TargetToolID *string  `json:"targetToolId,omitempty"`
	DefaultScope string   `json:"defaultScope,omitempty"` // Default scope skill to include
	// TargetDimensions are optional maturity dimensions this skill helps address.
	// Finding-to-skill recommendations remain descriptor-owned by Test Genie.
	TargetDimensions []string       `json:"targetDimensions,omitempty"`
	Requires         *SkillRequires `json:"requires,omitempty"`
	Origin           *SkillOrigin   `json:"origin,omitempty"`
	// ProgrammaticHome is a record-of-fact pointer (format "engine:identifier",
	// e.g. "test-genie:architecture") set when this skill's detection has
	// graduated into a programmatic engine. Nil while detection is still agentic.
	// Meaningful for steer skills; the quality-auditor derives its audit rotation
	// from steer skills where this is nil. NOT part of the generic health score.
	ProgrammaticHome *string `json:"programmaticHome,omitempty"`
	Timestamps

	// Runtime fields (not persisted in skill.json)
	Pack      string `json:"-"` // Which pack this skill belongs to
	SourceDir string `json:"-"` // Direct scenario-owned source directory, when set
}

// SkillOrigin records immutable provenance and independent review state for a
// vendored skill. Imported skills remain inactive until explicitly passed.
type SkillOrigin struct {
	Kind            string      `json:"kind"`
	SourceURL       string      `json:"sourceUrl"`
	Commit          string      `json:"commit"`
	License         string      `json:"license"`
	Checksum        string      `json:"checksum"`
	ImportedBy      string      `json:"importedBy"`
	ImportedAt      string      `json:"importedAt"`
	UpstreamVersion string      `json:"upstreamVersion,omitempty"`
	Review          SkillReview `json:"review"`
}

type SkillReview struct {
	Verdict    string `json:"verdict"`
	Reviewer   string `json:"reviewer,omitempty"`
	ReviewedAt string `json:"reviewedAt,omitempty"`
}

const (
	OriginAuthored        = "authored"
	OriginImported        = "imported"
	ReviewVerdictPending  = "pending"
	ReviewVerdictPassed   = "passed"
	ReviewVerdictRejected = "rejected"
)

// Action represents a typed executable wrapper over one controlled CLI command.
//
// DOC: docs/concepts/ACTIONS.md
type Action struct {
	BaseEntity
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Status      string                  `json:"status"`
	Owner       ActionOwner             `json:"owner"`
	Command     ActionCommand           `json:"command"`
	Inputs      map[string]ActionInput  `json:"inputs,omitempty"`
	Outputs     map[string]ActionOutput `json:"outputs,omitempty"`
	Permissions ActionPermissions       `json:"permissions,omitempty"`
	Examples    []ActionExample         `json:"examples,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Execution   *ActionExecution        `json:"execution,omitempty"`
	Validation  *ActionValidation       `json:"validation,omitempty"`
	Timestamps

	// Runtime fields (not persisted in action.json)
	Pack string `json:"-"`
}

// ActionOwner identifies the Vrooli-owned surface responsible for the command.
type ActionOwner struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ActionCommand declares the exact argv template this Action wraps.
type ActionCommand struct {
	Argv []string `json:"argv"`
}

// ActionInput declares one typed input accepted by an Action.
type ActionInput struct {
	Type           string   `json:"type"`
	Description    string   `json:"description,omitempty"`
	Required       bool     `json:"required,omitempty"`
	Enum           []string `json:"enum,omitempty"`
	Default        any      `json:"default,omitempty"`
	Pattern        string   `json:"pattern,omitempty"`
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	MaxLength      *int     `json:"maxLength,omitempty"`
	AllowMultiline bool     `json:"allowMultiline,omitempty"`
}

// ActionOutput declares one structured output emitted by an Action.
type ActionOutput struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// ActionPermissions declares side effects the Action may require.
type ActionPermissions struct {
	FilesystemRead   bool `json:"filesystemRead,omitempty"`
	FilesystemWrite  bool `json:"filesystemWrite,omitempty"`
	LocalhostNetwork bool `json:"localhostNetwork,omitempty"`
	ExternalNetwork  bool `json:"externalNetwork,omitempty"`
	APIRead          bool `json:"apiRead,omitempty"`
	APIWrite         bool `json:"apiWrite,omitempty"`
	ProcessStart     bool `json:"processStart,omitempty"`
	ProcessStop      bool `json:"processStop,omitempty"`
	HostConfigure    bool `json:"hostConfigure,omitempty"`
	SecretRead       bool `json:"secretRead,omitempty"`
	SecretWrite      bool `json:"secretWrite,omitempty"`
	Destructive      bool `json:"destructive,omitempty"`
}

// ActionExample shows a concrete input payload for an Action.
type ActionExample struct {
	Description string         `json:"description,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
}

// ActionExecution contains optional runtime policy for future execution support.
type ActionExecution struct {
	TimeoutSeconds *int   `json:"timeoutSeconds,omitempty"`
	OutputMode     string `json:"outputMode,omitempty"`
	RunEligible    *bool  `json:"runEligible,omitempty"`
}

// ActionValidation contains optional owner-specific validation metadata.
type ActionValidation struct {
	Mode string   `json:"mode,omitempty"`
	Argv []string `json:"argv,omitempty"`
}

// ActionRunHistoryEntry records a bounded audit entry for Action execution.
type ActionRunHistoryEntry struct {
	ActionID        string    `json:"actionId"`
	StartedAt       time.Time `json:"startedAt"`
	FinishedAt      time.Time `json:"finishedAt"`
	DurationMs      int64     `json:"durationMs"`
	Status          string    `json:"status"`
	Argv            []string  `json:"argv,omitempty"`
	ExitCode        *int      `json:"exitCode,omitempty"`
	Stdout          string    `json:"stdout,omitempty"`
	Stderr          string    `json:"stderr,omitempty"`
	StdoutTruncated bool      `json:"stdoutTruncated,omitempty"`
	StderrTruncated bool      `json:"stderrTruncated,omitempty"`
	Error           string    `json:"error,omitempty"`
	ValidationValid bool      `json:"validationValid"`
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
	ID                string                          `json:"id"`
	DisplayName       string                          `json:"displayName"`
	Mission           string                          `json:"mission,omitempty"`
	Enabled           bool                            `json:"enabled"`
	EnabledSet        bool                            `json:"-"`
	Runtime           teamconfig.Runtime              `json:"runtime"`
	Coordination      teamconfig.Coordination         `json:"coordination"`
	Execution         teamconfig.Execution            `json:"execution"`
	OperatingContract *teamcontract.OperatingContract `json:"operatingContract"`
	// ValidationFindings are read-time diagnostics for an otherwise readable
	// team.json. They are never persisted and do not weaken write validation.
	ValidationFindings []TeamValidationFinding `json:"validationFindings,omitempty"`
	Shared             *TeamShared             `json:"shared,omitempty"`
	Retention          *RetentionConfig        `json:"retention,omitempty"`
	// AttributionValidFrom is the per-team Pillar 3 cutoff. Team-corpus
	// entries written on or after this ISO-8601 date (YYYY-MM-DD) MUST
	// carry runtime attribution conforming to docs/agent-system/RUNTIME_ATTRIBUTION.md;
	// pre-cutoff entries are treated as kind="legacy" and skipped by
	// ruleActualWriterUndeclared. The value is set at migration time
	// by the cutover migration and may be widened retroactively
	// by the operator if a migration introduces unexpected drift.
	// Empty string means the team has not yet adopted the runtime contract;
	// the validator skips such teams entirely.
	AttributionValidFrom string `json:"attributionValidFrom,omitempty"`
	// Policy carries opt-in per-team thresholds the runtime-attribution
	// validators consult. Nil / omitted means "every default applies"; in
	// particular, nil disables the per-week thresholds (external writes
	// are tracked but never flagged). See docs/agent-system/RUNTIME_ATTRIBUTION.md
	// § kind enum for the field-by-field semantics, and the TeamPolicy
	// struct below for the active set.
	Policy *TeamPolicy `json:"policy,omitempty"`
	Timestamps
}

// TeamValidationFinding identifies one contract defect without hiding the
// team that needs repair from operators.
type TeamValidationFinding struct {
	Source string `json:"source"`
	// Rule is the catalogued rule id when the finding came from a rule family
	// that has one. teamconfig checks are config-load errors and leave it empty.
	Rule    string `json:"rule,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// TeamPolicy holds opt-in thresholds and toggles consulted by the Pillar 3
// runtime-attribution validator (and any future runtime-only validators
// that a team can dial up or down without changing topics.json shape).
//
// Every field is "off" at zero — adding a new field never silently changes
// existing teams' validator behavior. The whole struct is optional on
// team.json; the validator treats a missing Policy as if every threshold
// were zero.
type TeamPolicy struct {
	// FlagExternalWritesPerWeek is the per-week count above which
	// kind="external" knowledge writes fire `actual_writer_undeclared`
	// findings. Zero (or omitted) means "track but never flag" — the
	// expected default for teams that are not yet routing external
	// writers through prompt-manager. Set to a positive integer to opt
	// in; the validator emits one warning per entry past the threshold
	// in any ISO week from `attributionValidFrom` forward.
	//
	// The "per week" semantics are deterministic and side-effect-free:
	// each entry's ISO week (year-week, derived from the entry's `at`
	// timestamp) is the bucket; the threshold is compared against the
	// bucket's count. No rolling clock, no `time.Now()` dependency —
	// `prompt-manager graph topics` produces the same findings on a
	// frozen team-corpus ledger regardless of when it runs.
	FlagExternalWritesPerWeek int `json:"flagExternalWritesPerWeek,omitempty"`
}

// TeamShared configures shared document access
type TeamShared struct {
	Path      string `json:"path"`
	MountHint string `json:"mountHint,omitempty"` // readOnly or readWrite
}

// RetentionConfig controls automatic cleanup of completed tasks. Team-corpus
// entries live in Source Ledger and are append-only, so they are never pruned
// by prompt-manager.
type RetentionConfig struct {
	Tasks *TaskRetention `json:"tasks,omitempty"`
}

// TaskRetention controls how completed tasks are pruned.
type TaskRetention struct {
	MaxCompleted int `json:"maxCompleted"` // keep N newest completed tasks (0 = unlimited)
	MaxAgeDays   int `json:"maxAgeDays"`   // remove completed tasks older than N days (0 = unlimited)
}

// DefaultRetentionConfig returns the default retention policy.
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		Tasks: &TaskRetention{MaxCompleted: 20, MaxAgeDays: 30},
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
	ID                       string             `json:"id"`
	SkillID                  string             `json:"skillId"`
	Name                     string             `json:"name"`
	Hypothesis               string             `json:"hypothesis,omitempty"`
	Protocol                 ExperimentProtocol `json:"protocol"`
	Status                   string             `json:"status"` // draft, running, concluded
	Arms                     []ExperimentArm    `json:"arms"`
	StartedAt                *string            `json:"startedAt,omitempty"`
	ConcludedAt              *string            `json:"concludedAt,omitempty"`
	WinnerVariantID          *string            `json:"winnerVariantId,omitempty"`
	PromotionWorkItemRef     string             `json:"promotionWorkItemRef,omitempty"`
	HoldoutFindingsHash      string             `json:"holdoutFindingsHash,omitempty"`
	HoldoutCompletedAt       string             `json:"holdoutCompletedAt,omitempty"`
	HoldoutIdempotencyKey    string             `json:"holdoutIdempotencyKey,omitempty"`
	HoldoutSignature         string             `json:"holdoutSignature,omitempty"` // historical development artifact only
	HoldoutSignatureEnvelope json.RawMessage    `json:"holdoutSignatureEnvelope,omitempty"`
	PromotedAt               string             `json:"promotedAt,omitempty"`
	Notes                    string             `json:"notes,omitempty"`
	Timestamps
}

// ExperimentProtocol is the immutable, pre-registered evidence contract for a
// controlled experiment. Its explicit fields prevent promotion criteria from
// being changed after results are known.
type ExperimentProtocol struct {
	Population                   string   `json:"population"`
	RandomizationUnit            string   `json:"randomizationUnit"`
	PrimaryMetric                string   `json:"primaryMetric"`
	EffectThreshold              float64  `json:"effectThreshold"`
	ProbabilityThreshold         float64  `json:"probabilityThreshold"`
	CostNonInferiorityRatio      float64  `json:"costNonInferiorityRatio"`
	Strata                       []string `json:"strata"`
	ExposurePolicy               string   `json:"exposurePolicy"`
	OutcomeCompletenessThreshold float64  `json:"outcomeCompletenessThreshold"`
	Budget                       string   `json:"budget"`
	StoppingRule                 string   `json:"stoppingRule"`
	HoldoutRequired              bool     `json:"holdoutRequired"`
	HoldoutPopulationHash        string   `json:"holdoutPopulationHash"`
	PromotionAuthority           string   `json:"promotionAuthority"`
	EvaluatorRubricHash          string   `json:"evaluatorRubricHash"`
	EvaluatorAuthor              string   `json:"evaluatorAuthor"`
	ProtocolHash                 string   `json:"protocolHash,omitempty"`
}

// ExperimentArm is one variant in an experiment with its traffic weight.
type ExperimentArm struct {
	VariantID string  `json:"variantId"` // "control" = original SKILL.md
	Weight    float64 `json:"weight"`    // 0.0-1.0, arms must sum to 1.0
}

// ExperimentOutcome is the opaque outcome envelope stored per experiment.
// Prompt-manager never parses Data — it belongs to the source system (e.g. swarm-manager).
type ExperimentOutcome struct {
	IdempotencyKey string                       `json:"idempotencyKey,omitempty"`
	VariantID      string                       `json:"variantId"`
	Source         string                       `json:"source"`        // e.g. "swarm-manager"
	SchemaVersion  int                          `json:"schemaVersion"` // consumer-defined
	RecordedAt     string                       `json:"recordedAt"`
	Data           json.RawMessage              `json:"data"` // opaque to PM
	Controlled     *ControlledExperimentOutcome `json:"controlled,omitempty"`
}

// ControlledExperimentOutcome is the typed, evaluator-bound primary evidence
// projection. Data remains available for bounded auxiliary evidence, but a
// controlled record is never interpreted from terminal run status alone.
type ControlledExperimentOutcome struct {
	AssignmentID         string `json:"assignmentId"`
	ExecutionID          string `json:"executionId"`
	EvaluatorAttemptID   string `json:"evaluatorAttemptId"`
	EvaluatorRunID       string `json:"evaluatorRunId"`
	Verdict              string `json:"verdict,omitempty"`
	Success              *bool  `json:"success,omitempty"`
	OutcomeStatus        string `json:"outcomeStatus"` // complete or incomplete
	RubricHash           string `json:"rubricHash"`
	EvaluatorPromptHash  string `json:"evaluatorPromptHash"`
	StructuredSchemaHash string `json:"structuredSchemaHash"`
	EvidenceHash         string `json:"evidenceHash,omitempty"`
}

// ExperimentAuditReceipt binds a qualitative audit to immutable quantitative
// evidence. Signature is generated server-side over canonical fields.
type ExperimentAuditReceipt struct {
	ExperimentID         string          `json:"experimentId"`
	ProtocolHash         string          `json:"protocolHash"`
	SampledAssignmentIDs []string        `json:"sampledAssignmentIds"`
	FindingsHash         string          `json:"findingsHash"`
	ChallengeState       string          `json:"challengeState"`
	AnomalyCount         int             `json:"anomalyCount"`
	GamingCount          int             `json:"gamingCount"`
	CompletedAt          string          `json:"completedAt"`
	Signature            string          `json:"signature,omitempty"` // historical development artifact only
	SignatureEnvelope    json.RawMessage `json:"signatureEnvelope,omitempty"`
	IdempotencyKey       string          `json:"idempotencyKey"`
}

// ExperimentServe records that a variant was served for a skill read. It is the
// serve-side counterpart to ExperimentOutcome: appended when blind serving (or an
// explicit experiment) selects a variant, so a served variant can later be joined
// to its outcome. Written to serve.jsonl beside outcomes.jsonl.
type ExperimentServe struct {
	ExperimentID string `json:"experimentId"`
	SkillID      string `json:"skillId"`
	VariantID    string `json:"variantId"`
	Source       string `json:"source,omitempty"` // caller identity, e.g. "agent-manager"
	ServedAt     string `json:"servedAt"`
}

// ExperimentAssignment is the immutable treatment receipt created before a
// workflow child is dispatched. Content is stored with the receipt so retries
// cannot observe a later variant edit or a different random draw.
type ExperimentAssignment struct {
	ExperimentID   string `json:"experimentId"`
	SkillID        string `json:"skillId"`
	VariantID      string `json:"variantId"`
	ExecutionID    string `json:"executionId"`
	NodeID         string `json:"nodeId"`
	AttemptKey     string `json:"attemptKey"`
	IdempotencyKey string `json:"idempotencyKey"`
	Content        string `json:"content"`
	ContentHash    string `json:"contentHash"`
	AssignedAt     string `json:"assignedAt"`
}

// ExperimentExposure is a verified skill-read receipt issued from an active
// workflow run. Identity fields are copied only after agent-manager validates
// the signed token; callers cannot supply them in the read request body.
type ExperimentExposure struct {
	ExperimentID   string `json:"experimentId"`
	VariantID      string `json:"variantId"`
	RunID          string `json:"runId"`
	ExecutionID    string `json:"executionId"`
	NodeID         string `json:"nodeId"`
	AttemptID      string `json:"attemptId"`
	ReadSkillID    string `json:"readSkillId"`
	Provenance     string `json:"provenance"`
	ObservedAt     string `json:"observedAt"`
	IdempotencyKey string `json:"idempotencyKey"`
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

// HeartbeatAttempt records an attempted heartbeat dispatch, including failures
// that happen before agent-manager can create a run.
type HeartbeatAttempt struct {
	ID            string `json:"id"`
	TeamID        string `json:"teamId"`
	AgentID       string `json:"agentId"`
	ProfileKey    string `json:"profileKey,omitempty"`
	TaskID        string `json:"taskId,omitempty"`
	RunID         string `json:"runId,omitempty"`
	Tag           string `json:"tag,omitempty"`
	Status        string `json:"status"`
	Phase         string `json:"phase"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt,omitempty"`
	ErrorCategory string `json:"errorCategory,omitempty"`
	Error         string `json:"error,omitempty"`
	Recovery      string `json:"recovery,omitempty"`
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

// --- Knowledge Log ---

// Knowledge writer-kind enum. Closed vocabulary; new kinds require a
// meta-optimization gated work item and a migration plan. The canon contract is
// docs/agent-system/RUNTIME_ATTRIBUTION.md § kind enum.
const (
	// KnowledgeKindAgentMember is a team member's agent process running
	// under agent-manager. Requires MemberID, TeamID, RunID on attribution.
	KnowledgeKindAgentMember = "agent-member"
	// KnowledgeKindWriterSkill is a registered writer skill (e.g. report-bug,
	// report-friction, morning-vision-walk). Requires SourceSkillID and
	// TeamID; MemberID/RunID are populated when invoked from a run.
	KnowledgeKindWriterSkill = "writer-skill"
	// KnowledgeKindOperatorDirect is a human at the CLI not running under
	// any agent context. No required fields beyond Kind.
	KnowledgeKindOperatorDirect = "operator-direct"
	// KnowledgeKindExternal is a non-Vrooli system (webhook, future
	// integration). No required fields beyond Kind.
	KnowledgeKindExternal = "external"
	// KnowledgeKindLegacy is set during the cutover on every pre-cutoff entry.
	// The original freeform `by` value is preserved on
	// the entry's CallerNote field. Skipped by ruleActualWriterUndeclared.
	KnowledgeKindLegacy = "legacy"
	// KnowledgeKindInvestigation is a bug-investigator or root-cause-analysis
	// run reproducing or annotating an existing entry. Requires RunID;
	// treated as read-only relative to topic-flow declarations.
	KnowledgeKindInvestigation = "investigation"
)

// KnowledgeKinds enumerates every valid AttributionInfo.Kind value. Used by
// the API handler to reject unknown kinds at write time and by tests
// to assert constant coverage.
var KnowledgeKinds = []string{
	KnowledgeKindAgentMember,
	KnowledgeKindWriterSkill,
	KnowledgeKindOperatorDirect,
	KnowledgeKindExternal,
	KnowledgeKindLegacy,
	KnowledgeKindInvestigation,
}

// Spawn-origin enum. Closed vocabulary; informational signal for triage and
// curation (debt-curator joins on this to spot patterns), not a gating field.
// The canon contract is docs/agent-system/RUNTIME_ATTRIBUTION.md
// § spawn_origin enum.
const (
	// SpawnOriginHeartbeat — run was spawned by prompt-manager's heartbeat
	// scheduler.
	SpawnOriginHeartbeat = "heartbeat"
	// SpawnOriginOperatorCLI — run was spawned directly from an operator's
	// terminal.
	SpawnOriginOperatorCLI = "operator-cli"
	// SpawnOriginSwarmTask — run was spawned as part of a swarm-manager
	// initiative or sub-task.
	SpawnOriginSwarmTask = "swarm-task"
	// SpawnOriginVisionWalk — run was spawned from morning-vision-walk's
	// seeded inbox entries.
	SpawnOriginVisionWalk = "vision-walk"
	// SpawnOriginInvestigation — run was spawned by another run's
	// investigate action.
	SpawnOriginInvestigation = "investigation"
	// SpawnOriginLegacy — pre-cutoff entry; origin not recorded.
	SpawnOriginLegacy = "legacy"
	// SpawnOriginUnknown — origin could not be determined (e.g., a writer
	// skill invoked outside any run context).
	SpawnOriginUnknown = "unknown"
)

// SpawnOrigins enumerates every valid AttributionInfo.SpawnOrigin value.
// Used by the API handler to reject unknown origins at write time
// and by tests to assert constant coverage.
var SpawnOrigins = []string{
	SpawnOriginHeartbeat,
	SpawnOriginOperatorCLI,
	SpawnOriginSwarmTask,
	SpawnOriginVisionWalk,
	SpawnOriginInvestigation,
	SpawnOriginLegacy,
	SpawnOriginUnknown,
}

// AttributionInfo is the structured-attribution payload carried on every
// post-cutoff knowledge entry. The API handler populates it at write time
// from the X-Vrooli-Attribution HTTP header.
//
// The canon contract is docs/agent-system/RUNTIME_ATTRIBUTION.md
// § The structured-attribution payload — that doc owns field semantics,
// per-kind required-field rules, and the cryptographic-strengthening path.
//
// Optional pointer fields marshal as JSON null when nil; the canon
// preserves null over omission to keep the on-disk shape uniform across
// kinds (so every legacy entry has the same field set as every live
// entry, with kind-driven nulls rather than missing keys).
type AttributionInfo struct {
	// Kind is the category of writer; required, closed vocabulary
	// (KnowledgeKind*). Unknown values are rejected by the API at write
	// time and surfaced as attribution_malformed by the validator.
	Kind string `json:"kind"`
	// MemberID is the team-scoped member id (matches
	// store/teams/<team>/members/<id>/) for agent-member and writer-skill
	// writes. Nil for operator-direct, external, legacy. Marshals as null.
	MemberID *string `json:"member_id"`
	// TeamID is the team id this write is attributed to; the API handler
	// rejects mismatches against the URL path's team id (HTTP 400
	// team_mismatch). Nil for operator-direct when the operator did not
	// specify a team scope. Marshals as null.
	TeamID *string `json:"team_id"`
	// RunID is the agent-manager run UUID this write is attributed to —
	// allows joining a write to its run lineage. Required for
	// agent-member and investigation; optional for writer-skill (when
	// invoked from a run); nil for operator-direct, external, legacy.
	// Marshals as null.
	RunID *string `json:"run_id"`
	// SpawnOrigin is informational — how the writer process was started;
	// required, closed vocabulary (SpawnOrigin*). Used by debt-curator
	// to spot patterns ("most undeclared writes come from vision-walk
	// spawns"); never gates writes.
	SpawnOrigin string `json:"spawn_origin"`
	// SourceSkillID names the skill that performed or mediated the
	// write. Required for writer-skill; optional for others (when a
	// skill mediated the write). Marshals as null.
	SourceSkillID *string `json:"source_skill_id"`
}

// KnowledgeEntry represents a piece of team knowledge persisted across heartbeats.
//
// Identity: every post-cutoff entry carries structured Attribution that the
// API populates from the X-Vrooli-Attribution HTTP header. Caller is a
// derived display string the API computes at write time — never accepted
// as input. CallerNote is optional freeform context. The full contract is
// docs/agent-system/RUNTIME_ATTRIBUTION.md.
//
// Pre-cutover entries use kind=legacy with the original `by` field value
// preserved on CallerNote. Post-cutover entries with kind=legacy are not
// produced.
type KnowledgeEntry struct {
	ID         string `json:"id"`
	At         string `json:"at"`
	Topic      string `json:"topic"`
	Content    string `json:"content"`
	Source     string `json:"source,omitempty"`
	Supersedes string `json:"supersedes,omitempty"`
	// Caller is the human-readable display label derived at write time
	// from Attribution. Format depends on Attribution.Kind (see
	// docs/agent-system/RUNTIME_ATTRIBUTION.md § Derived display: caller).
	// Persisted verbatim so list/search reads don't need to parse
	// Attribution. NEVER accepted as input — clients send Attribution and
	// the API derives Caller.
	Caller string `json:"caller"`
	// CallerNote is optional freeform context the writer may attach (debug
	// breadcrumb, retry note, cutover trail). Capped at 256 chars by the
	// API handler. No identity meaning; never read by validators.
	// The cutover preserves the original `by` field's value here on every
	// pre-cutover entry.
	CallerNote string `json:"caller_note,omitempty"`
	// Attribution is the structured truth about who wrote this entry.
	// Always present (post-cutoff and migrated entries alike). The API
	// rejects mutating writes that lack the X-Vrooli-Attribution header
	// or carry a malformed payload.
	Attribution AttributionInfo `json:"attribution"`
}

// BugDraft is a private, repairable Scenario QA intake submission. It is kept
// outside the team corpus deliberately: only published reports are bug-inbox
// entries and therefore visible to the investigator or knowledge search.
type BugDraft struct {
	ID          string            `json:"id"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Raw         map[string]any    `json:"raw"`
	Accepted    map[string]string `json:"accepted"`
	Needs       []string          `json:"needs"`
	Invalid     []FieldDiagnostic `json:"invalid"`
	Warnings    []string          `json:"warnings"`
	Attribution AttributionInfo   `json:"attribution"`
}

// FieldDiagnostic explains one rejected capture field without discarding its
// supplied value. It is shared by API and CLI recovery output.
type FieldDiagnostic struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}
