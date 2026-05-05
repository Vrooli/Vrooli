// Package teams provides CLI commands for team management.
//
// DOC: docs/reference/cli-commands.md#teams
package teams

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"prompt-manager/cli/internal/appctx"
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Team represents a team from the API (brief response)
type Team struct {
	ID                string                          `json:"id"`
	DisplayName       string                          `json:"displayName"`
	Mission           string                          `json:"mission,omitempty"`
	Enabled           bool                            `json:"enabled"`
	Runtime           Runtime                         `json:"runtime"`
	Coordination      Coordination                    `json:"coordination"`
	Execution         Execution                       `json:"execution"`
	DecisionMode      string                          `json:"decisionMode,omitempty"`
	OperatingContract *teamcontract.OperatingContract `json:"operatingContract"`
	MemberCount       int                             `json:"memberCount"`
	CreatedAt         string                          `json:"createdAt"`
	UpdatedAt         string                          `json:"updatedAt"`
}

// TeamDetails represents full team details
type TeamDetails struct {
	Team
	Roles   []Role   `json:"roles"`
	Members []Member `json:"members"`
}

// Role represents a team role
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Member represents a team member
type Member struct {
	AgentID     string   `json:"agentId"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	Status      string   `json:"status"`
}

// OrgEdge represents a manager-report relationship in the org chart.
type OrgEdge struct {
	ManagerAgentID string `json:"managerAgentId"`
	ReportAgentID  string `json:"reportAgentId"`
}

// OrgChartResponse represents org chart data from the API.
type OrgChartResponse struct {
	TeamID string    `json:"teamId"`
	Edges  []OrgEdge `json:"edges"`
}

// UpdateOrgEdgeRequest sets a member's manager.
type UpdateOrgEdgeRequest struct {
	ManagerAgentID string `json:"managerAgentId"`
}

// TeamDetailsWithOrg includes org chart data for JSON output.
type TeamDetailsWithOrg struct {
	TeamDetails
	OrgChart *OrgChartResponse `json:"orgChart,omitempty"`
}

// TeamMessage represents a message between team members.
type TeamMessage struct {
	ID          string `json:"id"`
	TeamID      string `json:"teamId"`
	FromAgentID string `json:"fromAgentId"`
	ToAgentID   string `json:"toAgentId"`
	Content     string `json:"content"`
	CreatedAt   string `json:"createdAt"`
}

// TeamInboxResponse represents an inbox listing.
type TeamInboxResponse struct {
	TeamID   string        `json:"teamId"`
	AgentID  string        `json:"agentId"`
	Messages []TeamMessage `json:"messages"`
}

// SendTeamMessageRequest is the request body for sending a message.
type SendTeamMessageRequest struct {
	FromAgentID string `json:"fromAgentId"`
	Content     string `json:"content"`
}

type (
	Runtime                  = teamconfig.Runtime
	CoordinationCapabilities = teamconfig.Capabilities
	Coordination             = teamconfig.Coordination
	Execution                = teamconfig.Execution
)

// HandoffResponse represents a handoff API response.
type HandoffResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	Content string `json:"content"`
}

// HandoffHistoryEntry represents a handoff history entry.
type HandoffHistoryEntry struct {
	AgentID   string `json:"agentId"`
	RunID     string `json:"runId"`
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
}

// HandoffHistoryResponse represents the handoff history API response.
type HandoffHistoryResponse struct {
	TeamID  string                `json:"teamId"`
	Entries []HandoffHistoryEntry `json:"entries"`
}

// TaskNote represents a note on a task.
type TaskNote struct {
	At   string `json:"at"`
	By   string `json:"by"`
	Text string `json:"text"`
}

// TeamTask represents a task on the task board.
type TeamTask struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`
	Assignee  string     `json:"assignee"`
	Priority  string     `json:"priority"`
	CreatedBy string     `json:"createdBy"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
	Notes     []TaskNote `json:"notes,omitempty"`
}

// TaskBoardResponse represents the task board API response.
type TaskBoardResponse struct {
	TeamID string     `json:"teamId"`
	Tasks  []TeamTask `json:"tasks"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

// AddTaskRequest is the request body for adding a task.
type AddTaskRequest struct {
	Title    string `json:"title"`
	Assignee string `json:"assignee,omitempty"`
	Priority string `json:"priority,omitempty"`
	From     string `json:"from"`
}

// UpdateTaskRequest is the request body for updating a task.
type UpdateTaskRequest struct {
	Status   *string `json:"status,omitempty"`
	Assignee *string `json:"assignee,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Note     *string `json:"note,omitempty"`
}

// DecisionOption represents a lettered choice in a multi-option decision.
type DecisionOption struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Rationale   string `json:"rationale"`
	Recommended bool   `json:"recommended,omitempty"`
}

// DecisionEntry represents a decision log entry.
type DecisionEntry struct {
	ID                      string                      `json:"id"`
	At                      string                      `json:"at"`
	By                      string                      `json:"by"`
	Decision                string                      `json:"decision"`
	Rationale               string                      `json:"rationale"`
	Context                 string                      `json:"context,omitempty"`
	Supersedes              string                      `json:"supersedes,omitempty"`
	Status                  string                      `json:"status,omitempty"`
	Topic                   string                      `json:"topic,omitempty"`
	Description             string                      `json:"description,omitempty"`
	Options                 []DecisionOption            `json:"options,omitempty"`
	Selected                string                      `json:"selected,omitempty"`
	Freeform                string                      `json:"freeform,omitempty"`
	Notes                   string                      `json:"notes,omitempty"`
	Modifications           *DecisionModifications      `json:"modifications,omitempty"`
	RevisitAfter            *string                     `json:"revisit_after,omitempty"`
	AcceptedAsProposed      bool                        `json:"accepted_as_proposed,omitempty"`
	InitiativeMetadata      *DecisionInitiativeMetadata `json:"initiative_metadata,omitempty"`
	AutoCreateStatus        string                      `json:"auto_create_status,omitempty"`
	AutoCreateError         string                      `json:"auto_create_error,omitempty"`
	AutoCreateInitiativeRef string                      `json:"auto_create_initiative_ref,omitempty"`
}

// DecisionInitiativeMetadata mirrors the API store struct of the same name.
// Carried on decisions whose context is "initiative-proposal"; consumed by
// the API at decision-accept time to auto-create a swarm-manager initiative.
// Contract: docs/reference/decision-initiative-proposal-contract.md.
type DecisionInitiativeMetadata struct {
	Name           string   `json:"name"`
	Priority       int      `json:"priority,omitempty"`
	DependsOn      []string `json:"depends_on,omitempty"`
	TargetScenario string   `json:"target_scenario,omitempty"`
	Title          string   `json:"title,omitempty"`
}

// DecisionModifications is a structured, scoped exception an operator attaches
// to an accepted option. See docs/reference/decision-modifications-contract.md.
type DecisionModifications struct {
	ExcludedClauses []string `json:"excluded_clauses,omitempty"`
	Additions       []string `json:"additions,omitempty"`
	Rationale       string   `json:"rationale,omitempty"`
}

// DecisionListResponse represents the decision list API response.
type DecisionListResponse struct {
	TeamID  string          `json:"teamId"`
	Entries []DecisionEntry `json:"entries"`
	Total   int             `json:"total"`
	Last    int             `json:"last"`
}

// AddDecisionRequest is the request body for adding a decision.
type AddDecisionRequest struct {
	By                 string                      `json:"by"`
	Decision           string                      `json:"decision"`
	Rationale          string                      `json:"rationale"`
	Context            string                      `json:"context,omitempty"`
	Supersedes         string                      `json:"supersedes,omitempty"`
	Topic              string                      `json:"topic,omitempty"`
	Description        string                      `json:"description,omitempty"`
	Options            []DecisionOption            `json:"options,omitempty"`
	InitiativeMetadata *DecisionInitiativeMetadata `json:"initiative_metadata,omitempty"`
}

// AttributionInfo mirrors the API-side store.AttributionInfo so the CLI can
// deserialize knowledge-list responses cleanly. Field semantics, JSON tags,
// and per-kind required-field rules are owned by
// docs/agent-system/RUNTIME_ATTRIBUTION.md and the canonical Go struct in
// scenarios/prompt-manager/api/store/models.go::AttributionInfo. Keep this
// shape in lockstep with that struct — drift surfaces as silent JSON-decode
// gaps. Construction of outgoing attribution payloads lives in
// cli/internal/attribution; this struct is read-only — it deserializes
// attribution objects on knowledge-list responses for display.
type AttributionInfo struct {
	Kind          string  `json:"kind"`
	MemberID      *string `json:"member_id"`
	TeamID        *string `json:"team_id"`
	RunID         *string `json:"run_id"`
	SpawnOrigin   string  `json:"spawn_origin"`
	SourceSkillID *string `json:"source_skill_id"`
}

// KnowledgeEntry mirrors the API-side store.KnowledgeEntry. Caller is the
// derived display string the API computes at write time; CallerNote is
// optional freeform context; Attribution is the structured truth. See
// docs/agent-system/RUNTIME_ATTRIBUTION.md for the contract.
type KnowledgeEntry struct {
	ID          string          `json:"id"`
	At          string          `json:"at"`
	Topic       string          `json:"topic"`
	Content     string          `json:"content"`
	Source      string          `json:"source,omitempty"`
	Supersedes  string          `json:"supersedes,omitempty"`
	Caller      string          `json:"caller"`
	CallerNote  string          `json:"caller_note,omitempty"`
	Attribution AttributionInfo `json:"attribution"`
}

// KnowledgeListResponse represents the knowledge list API response.
type KnowledgeListResponse struct {
	TeamID  string           `json:"teamId"`
	Entries []KnowledgeEntry `json:"entries"`
}

type ChallengeListEntry struct {
	Decision    DecisionEntry    `json:"decision"`
	Reports     []KnowledgeEntry `json:"reports"`
	Resolutions []KnowledgeEntry `json:"resolutions"`
	Status      string           `json:"status"`
}

// AddKnowledgeRequest is the request body for adding a knowledge entry.
//
// Identity is carried out-of-band on the X-Vrooli-Attribution header
// (set by cli-core's HTTPClient extra-header source — see app.go and
// docs/agent-system/RUNTIME_ATTRIBUTION.md). The body never carries
// caller identity. CallerNote is freeform context only — it cannot
// override or contradict the header's attribution.
type AddKnowledgeRequest struct {
	Topic      string `json:"topic"`
	Content    string `json:"content"`
	CallerNote string `json:"caller_note,omitempty"`
	Source     string `json:"source,omitempty"`
	Supersedes string `json:"supersedes,omitempty"`
}

// CreateTeamRequest is the request body for creating a team
type CreateTeamRequest struct {
	ID                string                          `json:"id,omitempty"`
	DisplayName       string                          `json:"displayName"`
	Mission           string                          `json:"mission,omitempty"`
	Runtime           Runtime                         `json:"runtime"`
	Coordination      Coordination                    `json:"coordination"`
	Execution         Execution                       `json:"execution"`
	DecisionMode      string                          `json:"decisionMode,omitempty"`
	OperatingContract *teamcontract.OperatingContract `json:"operatingContract"`
}

// UpdateTeamRequest is the request body for updating a team
type UpdateTeamRequest struct {
	DisplayName       *string                         `json:"displayName,omitempty"`
	Mission           *string                         `json:"mission,omitempty"`
	Enabled           *bool                           `json:"enabled,omitempty"`
	Runtime           *Runtime                        `json:"runtime,omitempty"`
	Coordination      *Coordination                   `json:"coordination,omitempty"`
	Execution         *Execution                      `json:"execution,omitempty"`
	DecisionMode      *string                         `json:"decisionMode,omitempty"`
	OperatingContract *teamcontract.OperatingContract `json:"operatingContract,omitempty"`
}

type teamConfigFlagSet struct {
	runtimeMode              *string
	coordinationPattern      *string
	leadAgentID              *string
	reportingMode            *string
	messagingMode            *string
	queuePolicy              *string
	maxConcurrentRuns        *int
	showOrgContext           *string
	injectInbox              *string
	allowPeerTriggers        *string
	showTaskBoardGuidance    *string
	showDecisionLogGuidance  *string
	showKnowledgeLogGuidance *string
	requireHandoff           *string
}

func registerTeamConfigFlags(fs *flag.FlagSet, includeDefaults bool) teamConfigFlagSet {
	runtimeDefault := ""
	patternDefault := ""
	queueDefault := ""
	maxConcurrentDefault := 0
	if includeDefaults {
		runtimeDefault = teamconfig.RuntimeModeMultiProcess
		patternDefault = teamconfig.CoordinationPatternIndependent
		queueDefault = teamconfig.QueuePolicyBoundedParallel
		maxConcurrentDefault = 2
	}

	return teamConfigFlagSet{
		runtimeMode:              fs.String("runtime-mode", runtimeDefault, "Runtime mode (multi-process|single-process)"),
		coordinationPattern:      fs.String("coordination-pattern", patternDefault, "Coordination pattern (independent|peer|leader-led)"),
		leadAgentID:              fs.String("lead-agent-id", "", "Explicit lead agent ID for leader-led teams"),
		reportingMode:            fs.String("reporting-mode", "", "Reporting mode (none|org-chart|leader)"),
		messagingMode:            fs.String("messaging-mode", "", "Messaging mode (disabled|async-inbox|in-session)"),
		queuePolicy:              fs.String("queue-policy", queueDefault, "Execution queue policy (serialized|bounded-parallel)"),
		maxConcurrentRuns:        fs.Int("max-concurrent-runs", maxConcurrentDefault, "Maximum concurrent runs for bounded-parallel execution"),
		showOrgContext:           fs.String("show-org-context", "", "Show org context in prompts (true|false)"),
		injectInbox:              fs.String("inject-inbox", "", "Inject inbox contents into prompts (true|false)"),
		allowPeerTriggers:        fs.String("allow-peer-triggers", "", "Allow peer-triggered heartbeats (true|false)"),
		showTaskBoardGuidance:    fs.String("show-task-board-guidance", "", "Show task board guidance in prompts (true|false)"),
		showDecisionLogGuidance:  fs.String("show-decision-log-guidance", "", "Show decision log guidance in prompts (true|false)"),
		showKnowledgeLogGuidance: fs.String("show-knowledge-log-guidance", "", "Show knowledge log guidance in prompts (true|false)"),
		requireHandoff:           fs.String("require-handoff", "", "Require a final handoff section (true|false)"),
	}
}

func parseOptionalBool(raw string) (*bool, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func buildCoordinationPreset(pattern, runtimeMode, leadAgentID string) (Coordination, error) {
	coordination, err := teamconfig.BuildCoordinationPreset(pattern, runtimeMode, leadAgentID)
	if err != nil {
		if strings.TrimSpace(pattern) == teamconfig.CoordinationPatternLeaderLed &&
			strings.TrimSpace(leadAgentID) == "" {
			return Coordination{}, fmt.Errorf("leader-led teams require --lead-agent-id")
		}
		return Coordination{}, err
	}
	return coordination, nil
}

func buildExecutionConfig(runtimeMode, queuePolicy string, maxConcurrentRuns int) (Execution, error) {
	return teamconfig.BuildExecutionConfig(runtimeMode, queuePolicy, maxConcurrentRuns)
}

func applyCapabilityOverrides(coordination *Coordination, flags teamConfigFlagSet) error {
	if coordination == nil {
		return nil
	}

	overrides := []struct {
		raw string
		set func(bool)
	}{
		{raw: *flags.showOrgContext, set: func(v bool) { coordination.Capabilities.ShowOrgContext = v }},
		{raw: *flags.injectInbox, set: func(v bool) { coordination.Capabilities.InjectInbox = v }},
		{raw: *flags.allowPeerTriggers, set: func(v bool) { coordination.Capabilities.AllowPeerTriggers = v }},
		{raw: *flags.showTaskBoardGuidance, set: func(v bool) { coordination.Capabilities.ShowTaskBoardGuidance = v }},
		{raw: *flags.showDecisionLogGuidance, set: func(v bool) { coordination.Capabilities.ShowDecisionLogGuidance = v }},
		{raw: *flags.showKnowledgeLogGuidance, set: func(v bool) { coordination.Capabilities.ShowKnowledgeLogGuidance = v }},
		{raw: *flags.requireHandoff, set: func(v bool) { coordination.Capabilities.RequireHandoff = v }},
	}

	for _, override := range overrides {
		parsed, err := parseOptionalBool(override.raw)
		if err != nil {
			return err
		}
		if parsed != nil {
			override.set(*parsed)
		}
	}

	return nil
}

func resolveCreateTeamConfig(flags teamConfigFlagSet) (Runtime, Coordination, Execution, error) {
	runtimeMode := strings.TrimSpace(*flags.runtimeMode)
	pattern := strings.TrimSpace(*flags.coordinationPattern)
	if runtimeMode == teamconfig.RuntimeModeSingleProcess && pattern == teamconfig.CoordinationPatternIndependent {
		pattern = teamconfig.CoordinationPatternLeaderLed
	}

	runtime := Runtime{Mode: runtimeMode}
	coordination, err := buildCoordinationPreset(pattern, runtimeMode, strings.TrimSpace(*flags.leadAgentID))
	if err != nil {
		return Runtime{}, Coordination{}, Execution{}, err
	}
	if strings.TrimSpace(*flags.reportingMode) != "" {
		coordination.ReportingMode = strings.TrimSpace(*flags.reportingMode)
	}
	if strings.TrimSpace(*flags.messagingMode) != "" {
		coordination.MessagingMode = strings.TrimSpace(*flags.messagingMode)
	}
	if err := applyCapabilityOverrides(&coordination, flags); err != nil {
		return Runtime{}, Coordination{}, Execution{}, err
	}
	execution, err := buildExecutionConfig(runtimeMode, strings.TrimSpace(*flags.queuePolicy), *flags.maxConcurrentRuns)
	if err != nil {
		return Runtime{}, Coordination{}, Execution{}, err
	}
	return runtime, coordination, execution, nil
}

func resolveUpdatedTeamConfig(current TeamDetails, flags teamConfigFlagSet) (*Runtime, *Coordination, *Execution, error) {
	changed := false

	runtime := current.Runtime
	if strings.TrimSpace(*flags.runtimeMode) != "" {
		runtime.Mode = strings.TrimSpace(*flags.runtimeMode)
		changed = true
	}

	coordination := current.Coordination
	if strings.TrimSpace(*flags.runtimeMode) != "" {
		leadAgentID := firstNonEmpty(strings.TrimSpace(*flags.leadAgentID), coordination.LeadAgentID)
		if runtime.Mode == teamconfig.RuntimeModeSingleProcess {
			nextCoordination, err := buildCoordinationPreset(teamconfig.CoordinationPatternLeaderLed, runtime.Mode, leadAgentID)
			if err != nil {
				return nil, nil, nil, err
			}
			coordination = nextCoordination
			changed = true
		} else if coordination.Pattern == teamconfig.CoordinationPatternLeaderLed {
			nextCoordination, err := buildCoordinationPreset(teamconfig.CoordinationPatternLeaderLed, runtime.Mode, leadAgentID)
			if err != nil {
				return nil, nil, nil, err
			}
			coordination = nextCoordination
			changed = true
		}
	}

	if strings.TrimSpace(*flags.coordinationPattern) != "" {
		pattern := strings.TrimSpace(*flags.coordinationPattern)
		leadAgentID := coordination.LeadAgentID
		if strings.TrimSpace(*flags.leadAgentID) != "" {
			leadAgentID = strings.TrimSpace(*flags.leadAgentID)
		}
		nextCoordination, err := buildCoordinationPreset(pattern, runtime.Mode, leadAgentID)
		if err != nil {
			return nil, nil, nil, err
		}
		coordination = nextCoordination
		changed = true
	}

	if strings.TrimSpace(*flags.leadAgentID) != "" {
		coordination.LeadAgentID = strings.TrimSpace(*flags.leadAgentID)
		changed = true
	}
	if strings.TrimSpace(*flags.reportingMode) != "" {
		coordination.ReportingMode = strings.TrimSpace(*flags.reportingMode)
		changed = true
	}
	if strings.TrimSpace(*flags.messagingMode) != "" {
		coordination.MessagingMode = strings.TrimSpace(*flags.messagingMode)
		changed = true
	}
	if err := applyCapabilityOverrides(&coordination, flags); err != nil {
		return nil, nil, nil, err
	}
	if strings.TrimSpace(*flags.showOrgContext) != "" ||
		strings.TrimSpace(*flags.injectInbox) != "" ||
		strings.TrimSpace(*flags.allowPeerTriggers) != "" ||
		strings.TrimSpace(*flags.showTaskBoardGuidance) != "" ||
		strings.TrimSpace(*flags.showDecisionLogGuidance) != "" ||
		strings.TrimSpace(*flags.showKnowledgeLogGuidance) != "" ||
		strings.TrimSpace(*flags.requireHandoff) != "" {
		changed = true
	}

	execution := current.Execution
	if strings.TrimSpace(*flags.queuePolicy) != "" || *flags.maxConcurrentRuns > 0 || strings.TrimSpace(*flags.runtimeMode) != "" {
		nextExecution, err := buildExecutionConfig(runtime.Mode, firstNonEmpty(strings.TrimSpace(*flags.queuePolicy), execution.QueuePolicy), firstPositive(*flags.maxConcurrentRuns, execution.MaxConcurrentRuns))
		if err != nil {
			return nil, nil, nil, err
		}
		execution = nextExecution
		changed = true
	}

	if !changed {
		return nil, nil, nil, nil
	}

	return &runtime, &coordination, &execution, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

// AddMemberRequest is the request body for adding a member
type AddMemberRequest struct {
	AgentID string   `json:"agentId"`
	Roles   []string `json:"roles,omitempty"`
}

// UpdateMemberRequest is the request body for updating a member
type UpdateMemberRequest struct {
	Roles  []string `json:"roles,omitempty"`
	Status *string  `json:"status,omitempty"`
}

// Commands returns the team command groups using noun-verb pattern.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Teams",
		Commands: []cliapp.Command{
			{
				Name:        "team",
				Aliases:     []string{"teams", "t"},
				NeedsAPI:    true,
				Description: "Manage teams (list|show|create|update|delete|add-member|update-member|remove-member|roles|org-*|message-*|heartbeat-*|retention|prune|import-cc|export-cc|trigger)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
			{
				Name:        "decisions-pending",
				Aliases:     []string{"pending-decisions"},
				NeedsAPI:    true,
				Description: "List all pending decisions across all teams",
				Run: func(args []string) error {
					return cmdDecisionsPending(ctx, args)
				},
			},
		},
	}
}

// route dispatches to the appropriate subcommand.
func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list", "ls":
		return cmdList(ctx, subArgs)
	case "show", "get":
		return cmdShow(ctx, subArgs)
	case "create", "add":
		return cmdCreate(ctx, subArgs)
	case "update", "edit":
		return cmdUpdate(ctx, subArgs)
	case "delete", "rm":
		return cmdDelete(ctx, subArgs)
	case "add-member":
		return cmdAddMember(ctx, subArgs)
	case "update-member":
		return cmdUpdateMember(ctx, subArgs)
	case "remove-member":
		return cmdRemoveMember(ctx, subArgs)
	case "roles":
		return cmdRoles(ctx, subArgs)
	case "org-list":
		return cmdOrgList(ctx, subArgs)
	case "org-set":
		return cmdOrgSet(ctx, subArgs)
	case "org-remove":
		return cmdOrgRemove(ctx, subArgs)
	case "message-list":
		return cmdMessageList(ctx, subArgs)
	case "message-send":
		return cmdMessageSend(ctx, subArgs)
	case "message-delete":
		return cmdMessageDelete(ctx, subArgs)
	case "message-clear":
		return cmdMessageClear(ctx, subArgs)
	case "heartbeat-list":
		return cmdHeartbeatList(ctx, subArgs)
	case "heartbeat":
		return cmdHeartbeat(ctx, subArgs)
	case "heartbeat-enable":
		return cmdHeartbeatEnable(ctx, subArgs)
	case "heartbeat-disable":
		return cmdHeartbeatDisable(ctx, subArgs)
	case "heartbeat-trigger":
		return cmdHeartbeatTrigger(ctx, subArgs)
	case "heartbeat-logs":
		return cmdHeartbeatLogs(ctx, subArgs)
	case "responsibilities":
		return cmdResponsibilities(ctx, subArgs)
	case "heartbeat-instructions":
		return cmdHeartbeatInstructions(ctx, subArgs)
	case "import-cc":
		return cmdImportCC(ctx, subArgs)
	case "export-cc":
		return cmdExportCC(ctx, subArgs)
	case "trigger":
		return cmdTriggerTeam(ctx, subArgs)
	case "prompt-preview":
		return cmdPromptPreview(ctx, subArgs)
	case "prompt-preview-structured":
		return cmdPromptPreviewStructured(ctx, subArgs)
	case "prompt-matrix":
		return cmdPromptMatrix(ctx, subArgs)
	case "member-context":
		return cmdMemberContext(ctx, subArgs)
	case "operating-contract":
		return cmdOperatingContract(ctx, subArgs)
	case "validate-contract":
		return cmdValidateContract(ctx, subArgs)
	case "search", "find":
		return cmdSearch(ctx, subArgs)
	case "handoff-latest":
		return cmdHandoffLatest(ctx, subArgs)
	case "handoff-history":
		return cmdHandoffHistory(ctx, subArgs)
	case "task-list":
		return cmdTaskList(ctx, subArgs)
	case "task-add":
		return cmdTaskAdd(ctx, subArgs)
	case "task-update":
		return cmdTaskUpdate(ctx, subArgs)
	case "task-delete":
		return cmdTaskDelete(ctx, subArgs)
	case "decision-add":
		return cmdDecisionAdd(ctx, subArgs)
	case "decision-list":
		return cmdDecisionList(ctx, subArgs)
	case "decision-show":
		return cmdDecisionShow(ctx, subArgs)
	case "decision-update":
		return cmdDecisionUpdate(ctx, subArgs)
	case "decision-accept":
		return cmdDecisionAccept(ctx, subArgs)
	case "decision-reject":
		return cmdDecisionReject(ctx, subArgs)
	case "decision-defer":
		return cmdDecisionDefer(ctx, subArgs)
	case "decision-delete":
		return cmdDecisionDelete(ctx, subArgs)
	case "challenge-list":
		return cmdChallengeList(ctx, subArgs)
	case "challenge-resolve":
		return cmdChallengeResolve(ctx, subArgs)
	case "knowledge-add":
		return cmdKnowledgeAdd(ctx, subArgs)
	case "knowledge-list":
		return cmdKnowledgeList(ctx, subArgs)
	case "knowledge-update":
		return cmdKnowledgeUpdate(ctx, subArgs)
	case "knowledge-delete":
		return cmdKnowledgeDelete(ctx, subArgs)
	case "retention":
		return cmdRetention(ctx, subArgs)
	case "prune":
		return cmdPrune(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager team <subcommand> [args]

Subcommands:
  list, ls                          List all teams
  show, get <id>                    Show team details
  create, add <name>                Create a new team
  update, edit <id>                 Update a team
  delete, rm <id>                   Delete a team
  add-member <team-id> <agent-id>   Add a member to a team
  update-member <team-id> <agent-id> Update a team member
  remove-member <team-id> <agent-id> Remove a member from a team
  roles <team-id>                   List team roles
  org-list <team-id>                List org chart edges
  org-set <team-id> <report-id> <manager-id> Set reporting line
  org-remove <team-id> <report-id>  Remove reporting line
  message-list <team-id> <agent-id> List inbox messages for a member
  message-send <team-id> <agent-id> Send a message to a member
  message-delete <team-id> <agent-id> <message-id> Delete a message
  message-clear <team-id> <agent-id> Clear all messages for a member

Heartbeat Commands:
  heartbeat-list <team-id>                    List all heartbeat configs
  heartbeat <team-id> <agent-id>              Show heartbeat config
  heartbeat-enable <team-id> <agent-id>       Enable heartbeat with schedule
  heartbeat-disable <team-id> <agent-id>      Disable heartbeat
  heartbeat-trigger <team-id> <agent-id>      Manually trigger heartbeat
  heartbeat-logs <team-id> <agent-id>         List execution logs

Member Document Commands:
  responsibilities <team-id> <agent-id>       Get/set RESPONSIBILITIES.md
  heartbeat-instructions <team-id> <agent-id> Get/set HEARTBEAT.md

Context Commands:
  prompt-preview <team-id> <agent-id>         Preview the full runtime heartbeat prompt
  prompt-preview-structured <team-id> <agent-id> Preview ordered prompt sections
  prompt-matrix <team-id>                     Show prompt sections for all members
  member-context <team-id> <agent-id>         Get standing member context without HEARTBEAT.md
  operating-contract <team-id>                Print the team's stored operating contract
  validate-contract <team-id>                 Validate the team's operating contract

Handoff Commands:
  handoff-latest <team-id> <agent-id>   Show latest handoff for a member
  handoff-history <team-id>             Show handoff history

Task Board Commands:
  task-list <team-id>                   List tasks on the task board
  task-add <team-id>                    Add a task to the board
  task-update <team-id> <task-id>       Update a task
  task-delete <team-id> <task-id>       Delete a task

Decision Log Commands:
  decision-add <team-id>                Log a decision (supports --options for multi-option)
  decision-list <team-id>               List decisions (--context, --status, --last)
  decision-show <team-id> <id>          Show a single decision by id
  decision-update <team-id> <id>        Update any field of a decision (PATCH semantics)
  decision-accept <team-id> <id>        Accept a decision (--selected required for multi-option; omit for single-proposal; --notes recommended)
  decision-reject <team-id> <id>        Reject a decision (--notes required)
  decision-defer <team-id> <id>         Defer a pending decision (--revisit-after=YYYY-MM-DD required)
  decision-delete <team-id> <id>        Delete a decision (use --yes to skip confirm)
  challenge-list <team-id>              List challenge reports joined to pending decisions
  challenge-resolve <team-id> <id>      Add a challenge-resolution-record entry

Knowledge Log Commands:
  knowledge-add <team-id>               Add a knowledge entry
  knowledge-list <team-id>              List knowledge entries
  knowledge-update <team-id> <id>       Update a knowledge entry
  knowledge-delete <team-id> <id>       Delete a knowledge entry

Retention Commands:
  retention <team-id>                         Show effective retention config
  prune <team-id>                             Prune stale shared state

Claude Code Interop Commands:
  import-cc <team-name>                       Import a Claude Code team
  export-cc <team-id>                         Export team as Claude Code config
  trigger <team-id>                           Trigger team using the configured team policy`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var teams []Team
	if err := ctx.Get("/teams", &teams); err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(teams)
	}

	if len(teams) == 0 {
		fmt.Println("No teams found")
		return nil
	}

	fmt.Println("Teams:")
	for _, t := range teams {
		fmt.Printf("  %s - %d members [%s]\n", t.DisplayName, t.MemberCount, t.ID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team show <id>")
	}
	teamID := fs.Arg(0)

	var team TeamDetails
	if err := ctx.Get(fmt.Sprintf("/teams/%s", teamID), &team); err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}

	var orgChart *OrgChartResponse
	var orgErr error
	var orgResp OrgChartResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/org", teamID), &orgResp); err == nil {
		orgChart = &orgResp
	} else {
		orgErr = err
	}

	if *jsonOut {
		resp := TeamDetailsWithOrg{
			TeamDetails: team,
			OrgChart:    orgChart,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Name: %s\n", team.DisplayName)
	fmt.Printf("ID: %s\n", team.ID)
	if team.Mission != "" {
		fmt.Printf("Mission: %s\n", team.Mission)
	}
	fmt.Printf("Enabled: %v\n", team.Enabled)
	fmt.Printf("Runtime: %s\n", team.Runtime.Mode)
	fmt.Printf("Coordination: %s\n", team.Coordination.Pattern)
	if team.Coordination.LeadAgentID != "" {
		fmt.Printf("Lead Agent: %s\n", team.Coordination.LeadAgentID)
	}
	fmt.Printf("Reporting Mode: %s\n", team.Coordination.ReportingMode)
	fmt.Printf("Messaging Mode: %s\n", team.Coordination.MessagingMode)
	fmt.Printf("Queue Policy: %s (max concurrent: %d)\n", team.Execution.QueuePolicy, team.Execution.MaxConcurrentRuns)
	if team.DecisionMode != "" {
		fmt.Printf("Decision Mode: %s\n", team.DecisionMode)
	}
	fmt.Printf("Members: %d\n", team.MemberCount)
	if len(team.Members) > 0 {
		fmt.Println("  Members:")
		for _, m := range team.Members {
			rolesStr := ""
			if len(m.Roles) > 0 {
				rolesStr = fmt.Sprintf(" [%s]", strings.Join(m.Roles, ", "))
			}
			fmt.Printf("    - %s (%s)%s\n", m.DisplayName, m.Status, rolesStr)
		}
	}
	fmt.Printf("Roles: %d defined\n", len(team.Roles))
	if len(team.Roles) > 0 {
		for _, r := range team.Roles {
			desc := ""
			if r.Description != "" {
				desc = fmt.Sprintf(" - %s", r.Description)
			}
			fmt.Printf("    - %s%s\n", r.Name, desc)
		}
	}

	if orgChart != nil && len(orgChart.Edges) > 0 {
		memberNames := make(map[string]string, len(team.Members))
		for _, m := range team.Members {
			memberNames[m.AgentID] = m.DisplayName
		}
		fmt.Println("Org Chart:")
		for _, edge := range orgChart.Edges {
			managerName := memberNames[edge.ManagerAgentID]
			if managerName == "" {
				managerName = edge.ManagerAgentID
			}
			reportName := memberNames[edge.ReportAgentID]
			if reportName == "" {
				reportName = edge.ReportAgentID
			}
			fmt.Printf("  - %s -> %s\n", managerName, reportName)
		}
	} else if orgErr != nil {
		fmt.Printf("Org Chart: unavailable (%v)\n", orgErr)
	} else {
		fmt.Println("Org Chart: none")
	}
	fmt.Printf("Created: %s\n", team.CreatedAt)
	fmt.Printf("Updated: %s\n", team.UpdatedAt)
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	mission := fs.String("mission", "", "Team mission statement")
	decisionMode := fs.String("decision-mode", "", "Decision mode (yolo|approval)")
	configFlags := registerTeamConfigFlags(fs, true)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team create <name> [--mission=...] [--runtime-mode=...] [--coordination-pattern=...] [--decision-mode=...]")
	}
	name := fs.Arg(0)

	runtime, coordination, execution, err := resolveCreateTeamConfig(configFlags)
	if err != nil {
		return err
	}

	resolvedDecisionMode := strings.TrimSpace(*decisionMode)
	if resolvedDecisionMode == "" {
		resolvedDecisionMode = "yolo"
	}

	req := CreateTeamRequest{
		DisplayName:  name,
		Mission:      *mission,
		Runtime:      runtime,
		Coordination: coordination,
		Execution:    execution,
		DecisionMode: resolvedDecisionMode,
	}

	var team TeamDetails
	if err := ctx.Post("/teams", req, &team); err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(team)
	}

	fmt.Printf("Created team: %s [%s]\n", team.DisplayName, team.ID)
	return nil
}

func cmdUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "New display name")
	mission := fs.String("mission", "", "New mission statement")
	enabled := fs.String("enabled", "", "Set team enabled state (true|false)")
	decisionMode := fs.String("decision-mode", "", "Decision mode (yolo|approval)")
	configFlags := registerTeamConfigFlags(fs, false)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team update <id> [--name=...] [--mission=...] [--enabled=true|false] [--runtime-mode=...] [--coordination-pattern=...] [--decision-mode=...]")
	}
	teamID := fs.Arg(0)

	req := UpdateTeamRequest{}
	if *name != "" {
		req.DisplayName = name
	}
	if *mission != "" {
		req.Mission = mission
	}
	if *enabled != "" {
		parsed, err := strconv.ParseBool(*enabled)
		if err != nil {
			return fmt.Errorf("invalid --enabled value %q: %w", *enabled, err)
		}
		req.Enabled = &parsed
	}
	if *decisionMode != "" {
		req.DecisionMode = decisionMode
	}

	if strings.TrimSpace(*configFlags.runtimeMode) != "" ||
		strings.TrimSpace(*configFlags.coordinationPattern) != "" ||
		strings.TrimSpace(*configFlags.leadAgentID) != "" ||
		strings.TrimSpace(*configFlags.reportingMode) != "" ||
		strings.TrimSpace(*configFlags.messagingMode) != "" ||
		strings.TrimSpace(*configFlags.queuePolicy) != "" ||
		*configFlags.maxConcurrentRuns > 0 ||
		strings.TrimSpace(*configFlags.showOrgContext) != "" ||
		strings.TrimSpace(*configFlags.injectInbox) != "" ||
		strings.TrimSpace(*configFlags.allowPeerTriggers) != "" ||
		strings.TrimSpace(*configFlags.showTaskBoardGuidance) != "" ||
		strings.TrimSpace(*configFlags.showDecisionLogGuidance) != "" ||
		strings.TrimSpace(*configFlags.showKnowledgeLogGuidance) != "" ||
		strings.TrimSpace(*configFlags.requireHandoff) != "" {
		var current TeamDetails
		if err := ctx.Get(fmt.Sprintf("/teams/%s", teamID), &current); err != nil {
			return fmt.Errorf("failed to load current team config: %w", err)
		}
		runtime, coordination, execution, err := resolveUpdatedTeamConfig(current, configFlags)
		if err != nil {
			return err
		}
		req.Runtime = runtime
		req.Coordination = coordination
		req.Execution = execution
	}

	var team TeamDetails
	if err := ctx.Put(fmt.Sprintf("/teams/%s", teamID), req, &team); err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(team)
	}

	fmt.Printf("Updated team: %s [%s]\n", team.DisplayName, team.ID)
	return nil
}

func cmdOperatingContract(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("operating-contract", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team operating-contract <team-id>")
	}

	var team TeamDetails
	if err := ctx.Get(fmt.Sprintf("/teams/%s", fs.Arg(0)), &team); err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}
	if team.OperatingContract == nil {
		return fmt.Errorf("team %q does not have an operating contract", team.ID)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(team.OperatingContract)
}

func cmdValidateContract(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("validate-contract", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team validate-contract <team-id> [--json]")
	}

	var team TeamDetails
	if err := ctx.Get(fmt.Sprintf("/teams/%s", fs.Arg(0)), &team); err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}
	if team.OperatingContract == nil {
		return fmt.Errorf("contract validation failed: team %q does not have an operating contract", team.ID)
	}

	if *jsonOut {
		resp := map[string]any{"teamId": team.ID, "valid": true}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	fmt.Printf("Operating contract valid: %s\n", team.ID)
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team delete <id> [--force]")
	}
	teamID := fs.Arg(0)

	// Get team info first for confirmation
	var team Team
	if err := ctx.Get(fmt.Sprintf("/teams/%s", teamID), &team); err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}

	if !*force {
		fmt.Printf("Delete team %q (%s) with %d members? [y/N]: ", team.DisplayName, teamID, team.MemberCount)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/teams/%s", teamID)); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	fmt.Printf("Deleted team: %s\n", team.DisplayName)
	return nil
}

func cmdAddMember(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("add-member", flag.ContinueOnError)
	roles := fs.String("roles", "", "Comma-separated role IDs")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team add-member <team-id> <agent-id> [--roles=role1,role2]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var roleList []string
	if *roles != "" {
		roleList = strings.Split(*roles, ",")
		for i, r := range roleList {
			roleList[i] = strings.TrimSpace(r)
		}
	}

	req := AddMemberRequest{
		AgentID: agentID,
		Roles:   roleList,
	}

	var member Member
	if err := ctx.Post(fmt.Sprintf("/teams/%s/members", teamID), req, &member); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(member)
	}

	fmt.Printf("Added member: %s to team %s\n", member.DisplayName, teamID)
	return nil
}

func cmdUpdateMember(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("update-member", flag.ContinueOnError)
	roles := fs.String("roles", "", "Comma-separated role IDs (replaces existing)")
	status := fs.String("status", "", "New status (active|inactive|pending)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team update-member <team-id> <agent-id> [--roles=...] [--status=...]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	req := UpdateMemberRequest{}
	if *roles != "" {
		roleList := strings.Split(*roles, ",")
		for i, r := range roleList {
			roleList[i] = strings.TrimSpace(r)
		}
		req.Roles = roleList
	}
	if *status != "" {
		req.Status = status
	}

	var member Member
	if err := ctx.Put(fmt.Sprintf("/teams/%s/members/%s", teamID, agentID), req, &member); err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(member)
	}

	fmt.Printf("Updated member: %s in team %s\n", member.DisplayName, teamID)
	return nil
}

func cmdRemoveMember(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("remove-member", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team remove-member <team-id> <agent-id> [--force]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	if !*force {
		fmt.Printf("Remove member %s from team %s? [y/N]: ", agentID, teamID)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/members/%s", teamID, agentID)); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	fmt.Printf("Removed member %s from team %s\n", agentID, teamID)
	return nil
}

func cmdRoles(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("roles", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team roles <team-id>")
	}
	teamID := fs.Arg(0)

	var roles []Role
	if err := ctx.Get(fmt.Sprintf("/teams/%s/roles", teamID), &roles); err != nil {
		return fmt.Errorf("failed to get roles: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(roles)
	}

	if len(roles) == 0 {
		fmt.Println("No roles defined for this team")
		return nil
	}

	fmt.Println("Roles:")
	for _, r := range roles {
		desc := ""
		if r.Description != "" {
			desc = fmt.Sprintf(" - %s", r.Description)
		}
		fmt.Printf("  %s [%s]%s\n", r.Name, r.ID, desc)
	}
	return nil
}

func cmdOrgList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("org-list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team org-list <team-id>")
	}
	teamID := fs.Arg(0)

	var org OrgChartResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/org", teamID), &org); err != nil {
		return fmt.Errorf("failed to get org chart: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(org)
	}

	if len(org.Edges) == 0 {
		fmt.Println("No reporting relationships defined")
		return nil
	}

	var team TeamDetails
	_ = ctx.Get(fmt.Sprintf("/teams/%s", teamID), &team)
	memberNames := make(map[string]string, len(team.Members))
	for _, m := range team.Members {
		memberNames[m.AgentID] = m.DisplayName
	}

	fmt.Println("Org Chart:")
	for _, edge := range org.Edges {
		managerName := memberNames[edge.ManagerAgentID]
		if managerName == "" {
			managerName = edge.ManagerAgentID
		}
		reportName := memberNames[edge.ReportAgentID]
		if reportName == "" {
			reportName = edge.ReportAgentID
		}
		fmt.Printf("  - %s -> %s\n", managerName, reportName)
	}
	return nil
}

func cmdOrgSet(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("org-set", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 3 {
		return fmt.Errorf("usage: team org-set <team-id> <report-id> <manager-id>")
	}
	teamID := fs.Arg(0)
	reportID := fs.Arg(1)
	managerID := fs.Arg(2)

	req := UpdateOrgEdgeRequest{ManagerAgentID: managerID}
	var resp OrgEdge
	if err := ctx.Put(fmt.Sprintf("/teams/%s/org/edges/%s", teamID, reportID), req, &resp); err != nil {
		return fmt.Errorf("failed to update reporting line: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Updated reporting line: %s reports to %s\n", reportID, managerID)
	return nil
}

func cmdOrgRemove(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("org-remove", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team org-remove <team-id> <report-id>")
	}
	teamID := fs.Arg(0)
	reportID := fs.Arg(1)

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/org/edges/%s", teamID, reportID)); err != nil {
		return fmt.Errorf("failed to remove reporting line: %w", err)
	}

	fmt.Printf("Removed reporting line for %s\n", reportID)
	return nil
}

func cmdMessageList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("message-list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team message-list <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var inbox TeamInboxResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/members/%s/messages", teamID, agentID), &inbox); err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(inbox)
	}

	if len(inbox.Messages) == 0 {
		fmt.Println("No messages found")
		return nil
	}

	fmt.Printf("Messages for %s:\n", agentID)
	for _, msg := range inbox.Messages {
		fmt.Printf("  - [%s] from %s: %s\n", msg.ID, msg.FromAgentID, msg.Content)
	}
	return nil
}

func cmdMessageSend(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("message-send", flag.ContinueOnError)
	fromAgent := fs.String("from", "", "Sender agent ID")
	content := fs.String("content", "", "Message content")
	file := fs.String("file", "", "Read message content from file")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team message-send <team-id> <agent-id> --from=<agent-id> --content=\"...\"")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	if *file != "" {
		data, err := cliutil.ReadFileString(*file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		*content = data
	}

	if strings.TrimSpace(*fromAgent) == "" {
		return fmt.Errorf("from agent is required")
	}
	if strings.TrimSpace(*content) == "" {
		return fmt.Errorf("content is required")
	}

	req := SendTeamMessageRequest{
		FromAgentID: *fromAgent,
		Content:     *content,
	}

	var resp TeamMessage
	if err := ctx.Post(fmt.Sprintf("/teams/%s/members/%s/messages", teamID, agentID), req, &resp); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Sent message to %s (%s)\n", agentID, resp.ID)
	return nil
}

func cmdMessageDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("message-delete", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 3 {
		return fmt.Errorf("usage: team message-delete <team-id> <agent-id> <message-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)
	messageID := fs.Arg(2)

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/members/%s/messages/%s", teamID, agentID, messageID)); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	fmt.Printf("Deleted message %s\n", messageID)
	return nil
}

func cmdMessageClear(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("message-clear", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team message-clear <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/members/%s/messages", teamID, agentID)); err != nil {
		return fmt.Errorf("failed to clear messages: %w", err)
	}

	fmt.Printf("Cleared messages for %s\n", agentID)
	return nil
}

// HeartbeatConfig represents a heartbeat configuration from the API
type HeartbeatConfig struct {
	TeamID         string               `json:"teamId"`
	AgentID        string               `json:"agentId"`
	Enabled        bool                 `json:"enabled"`
	Schedule       string               `json:"schedule"`
	ProfileKey     string               `json:"profileKey,omitempty"`
	TimeoutSeconds int                  `json:"timeoutSeconds,omitempty"`
	LastExecution  *HeartbeatExecResult `json:"lastExecution,omitempty"`
	NextExecution  string               `json:"nextExecution,omitempty"`
	CreatedAt      string               `json:"createdAt"`
	UpdatedAt      string               `json:"updatedAt"`
}

// HeartbeatExecResult represents execution result
type HeartbeatExecResult struct {
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt,omitempty"`
	Status    string `json:"status"`
	RunID     string `json:"runId,omitempty"`
	LogPath   string `json:"logPath,omitempty"`
	Error     string `json:"error,omitempty"`
}

// CreateHeartbeatRequest is the request for creating a heartbeat
type CreateHeartbeatRequest struct {
	Schedule       string `json:"schedule"`
	ProfileKey     string `json:"profileKey,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

// UpdateHeartbeatRequest is the request for updating a heartbeat
type UpdateHeartbeatRequest struct {
	Schedule       *string `json:"schedule,omitempty"`
	ProfileKey     *string `json:"profileKey,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	TimeoutSeconds *int    `json:"timeoutSeconds,omitempty"`
}

// TriggerResponse is the response from triggering a heartbeat
type TriggerResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	RunID   string `json:"runId"`
	Status  string `json:"status"`
	LogPath string `json:"logPath,omitempty"`
}

// LogEntry represents a log file entry
type LogEntry struct {
	Filename  string `json:"filename"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status,omitempty"`
}

// LogListResponse is the response for listing logs
type LogListResponse struct {
	TeamID  string     `json:"teamId"`
	AgentID string     `json:"agentId"`
	Logs    []LogEntry `json:"logs"`
}

// MemberDocResponse is the response for member document operations
type MemberDocResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	Content string `json:"content"`
}

// MemberDocRequest is the request for setting member documents
type MemberDocRequest struct {
	Content string `json:"content"`
}

func cmdHeartbeatList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team heartbeat-list <team-id>")
	}
	teamID := fs.Arg(0)

	var configs []HeartbeatConfig
	if err := ctx.Get(fmt.Sprintf("/teams/%s/heartbeats", teamID), &configs); err != nil {
		return fmt.Errorf("failed to list heartbeats: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(configs)
	}

	if len(configs) == 0 {
		fmt.Println("No heartbeat configurations found")
		return nil
	}

	fmt.Println("Heartbeat Configurations:")
	for _, c := range configs {
		status := "disabled"
		if c.Enabled {
			status = "enabled"
		}
		lastRun := "never"
		if c.LastExecution != nil {
			lastRun = c.LastExecution.Status + " at " + c.LastExecution.StartedAt
		}
		fmt.Printf("  %s: %s [%s] - last: %s\n", c.AgentID, c.Schedule, status, lastRun)
	}
	return nil
}

func cmdHeartbeat(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var config HeartbeatConfig
	if err := ctx.Get(fmt.Sprintf("/teams/%s/heartbeats/%s", teamID, agentID), &config); err != nil {
		return fmt.Errorf("failed to get heartbeat config: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(config)
	}

	fmt.Printf("Team: %s\n", config.TeamID)
	fmt.Printf("Agent: %s\n", config.AgentID)
	fmt.Printf("Schedule: %s\n", config.Schedule)
	fmt.Printf("Enabled: %v\n", config.Enabled)
	if config.ProfileKey != "" {
		fmt.Printf("Profile Key: %s\n", config.ProfileKey)
	}
	if config.TimeoutSeconds > 0 {
		fmt.Printf("Timeout: %s\n", (time.Duration(config.TimeoutSeconds) * time.Second).String())
	} else {
		fmt.Printf("Timeout: 45m0s (default)\n")
	}
	if config.NextExecution != "" {
		fmt.Printf("Next Execution: %s\n", config.NextExecution)
	}
	if config.LastExecution != nil {
		fmt.Printf("Last Execution:\n")
		fmt.Printf("  Started: %s\n", config.LastExecution.StartedAt)
		if config.LastExecution.EndedAt != "" {
			fmt.Printf("  Ended: %s\n", config.LastExecution.EndedAt)
		}
		fmt.Printf("  Status: %s\n", config.LastExecution.Status)
		if config.LastExecution.RunID != "" {
			fmt.Printf("  Run ID: %s\n", config.LastExecution.RunID)
		}
		if config.LastExecution.Error != "" {
			fmt.Printf("  Error: %s\n", config.LastExecution.Error)
		}
	}
	return nil
}

func cmdHeartbeatEnable(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-enable", flag.ContinueOnError)
	schedule := fs.String("schedule", "0 */6 * * *", "Cron schedule expression")
	profileKey := fs.String("profile", "", "Agent-manager profile key")
	timeout := fs.String("timeout", "", "Execution timeout (e.g., 45m, 1h)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat-enable <team-id> <agent-id> [--schedule='0 */6 * * *'] [--profile=key] [--timeout=45m]")
	}

	var timeoutSeconds int
	if *timeout != "" {
		d, err := time.ParseDuration(*timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout duration %q: %w", *timeout, err)
		}
		timeoutSeconds = int(d.Seconds())
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	// Check if config exists
	var existing HeartbeatConfig
	existsErr := ctx.Get(fmt.Sprintf("/teams/%s/heartbeats/%s", teamID, agentID), &existing)

	enabled := true
	if existsErr != nil {
		// Create new config
		req := CreateHeartbeatRequest{
			Schedule:       *schedule,
			ProfileKey:     *profileKey,
			Enabled:        &enabled,
			TimeoutSeconds: timeoutSeconds,
		}
		var config HeartbeatConfig
		if err := ctx.Post(fmt.Sprintf("/teams/%s/heartbeats/%s", teamID, agentID), req, &config); err != nil {
			return fmt.Errorf("failed to create heartbeat config: %w", err)
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(config)
		}
		fmt.Printf("Created and enabled heartbeat for %s/%s with schedule: %s\n", teamID, agentID, config.Schedule)
	} else {
		// Update existing config
		req := UpdateHeartbeatRequest{
			Enabled: &enabled,
		}
		if *schedule != "0 */6 * * *" {
			req.Schedule = schedule
		}
		if *profileKey != "" {
			req.ProfileKey = profileKey
		}
		if timeoutSeconds > 0 {
			req.TimeoutSeconds = &timeoutSeconds
		}
		var config HeartbeatConfig
		if err := ctx.Put(fmt.Sprintf("/teams/%s/heartbeats/%s", teamID, agentID), req, &config); err != nil {
			return fmt.Errorf("failed to update heartbeat config: %w", err)
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(config)
		}
		fmt.Printf("Enabled heartbeat for %s/%s with schedule: %s\n", teamID, agentID, config.Schedule)
	}
	return nil
}

func cmdHeartbeatDisable(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-disable", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat-disable <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	enabled := false
	req := UpdateHeartbeatRequest{
		Enabled: &enabled,
	}

	var config HeartbeatConfig
	if err := ctx.Put(fmt.Sprintf("/teams/%s/heartbeats/%s", teamID, agentID), req, &config); err != nil {
		return fmt.Errorf("failed to disable heartbeat: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(config)
	}

	fmt.Printf("Disabled heartbeat for %s/%s\n", teamID, agentID)
	return nil
}

func cmdHeartbeatTrigger(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-trigger", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat-trigger <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var resp TriggerResponse
	if err := ctx.Post(fmt.Sprintf("/teams/%s/heartbeats/%s/trigger", teamID, agentID), nil, &resp); err != nil {
		return fmt.Errorf("failed to trigger heartbeat: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Triggered heartbeat for %s/%s\n", teamID, agentID)
	fmt.Printf("Run ID: %s\n", resp.RunID)
	fmt.Printf("Status: %s\n", resp.Status)
	return nil
}

func cmdHeartbeatLogs(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-logs", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat-logs <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var resp LogListResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/heartbeats/%s/logs", teamID, agentID), &resp); err != nil {
		return fmt.Errorf("failed to list logs: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Logs) == 0 {
		fmt.Println("No logs found")
		return nil
	}

	fmt.Println("Execution Logs:")
	for _, log := range resp.Logs {
		fmt.Printf("  %s\n", log.Filename)
	}
	return nil
}

func cmdResponsibilities(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("responsibilities", flag.ContinueOnError)
	setContent := fs.String("set", "", "Set content from string")
	setFile := fs.String("file", "", "Set content from file")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team responsibilities <team-id> <agent-id> [--set='content'] [--file=path]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	// If setting content
	if *setContent != "" || *setFile != "" {
		var content string
		if *setFile != "" {
			data, err := os.ReadFile(*setFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		} else {
			content = *setContent
		}

		req := MemberDocRequest{Content: content}
		var resp MemberDocResponse
		if err := ctx.Put(fmt.Sprintf("/teams/%s/members/%s/responsibilities", teamID, agentID), req, &resp); err != nil {
			return fmt.Errorf("failed to set responsibilities: %w", err)
		}

		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		}

		fmt.Printf("Updated RESPONSIBILITIES.md for %s/%s (%d bytes)\n", teamID, agentID, len(content))
		return nil
	}

	// Get content
	var resp MemberDocResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/members/%s/responsibilities", teamID, agentID), &resp); err != nil {
		return fmt.Errorf("failed to get responsibilities: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Content == "" {
		fmt.Println("No RESPONSIBILITIES.md content defined")
		return nil
	}

	fmt.Println(resp.Content)
	return nil
}

func cmdHeartbeatInstructions(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-instructions", flag.ContinueOnError)
	setContent := fs.String("set", "", "Set content from string")
	setFile := fs.String("file", "", "Set content from file")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat-instructions <team-id> <agent-id> [--set='content'] [--file=path]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	// If setting content
	if *setContent != "" || *setFile != "" {
		var content string
		if *setFile != "" {
			data, err := os.ReadFile(*setFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		} else {
			content = *setContent
		}

		req := MemberDocRequest{Content: content}
		var resp MemberDocResponse
		if err := ctx.Put(fmt.Sprintf("/teams/%s/members/%s/heartbeat-instructions", teamID, agentID), req, &resp); err != nil {
			return fmt.Errorf("failed to set heartbeat instructions: %w", err)
		}

		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		}

		fmt.Printf("Updated HEARTBEAT.md for %s/%s (%d bytes)\n", teamID, agentID, len(content))
		return nil
	}

	// Get content
	var resp MemberDocResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/members/%s/heartbeat-instructions", teamID, agentID), &resp); err != nil {
		return fmt.Errorf("failed to get heartbeat instructions: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Content == "" {
		fmt.Println("No HEARTBEAT.md content defined")
		return nil
	}

	fmt.Println(resp.Content)
	return nil
}

// ImportCCRequest is the request body for importing a Claude Code team.
type ImportCCRequest struct {
	TeamName string `json:"teamName"`
}

func cmdImportCC(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("import-cc", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team import-cc <team-name> [--json]")
	}
	teamName := fs.Arg(0)

	req := ImportCCRequest{TeamName: teamName}
	var team TeamDetails
	if err := ctx.Post("/teams/import/claude-code", req, &team); err != nil {
		return fmt.Errorf("failed to import CC team: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(team)
	}

	fmt.Printf("Imported team: %s [%s]\n", team.DisplayName, team.ID)
	fmt.Printf("Members: %d\n", team.MemberCount)
	for _, m := range team.Members {
		fmt.Printf("  - %s (%s)\n", m.DisplayName, m.AgentID)
	}
	return nil
}

// CCExportResponse matches the ToolTeamConfig from the interop package.
type CCExportResponse struct {
	TeamName    string           `json:"teamName"`
	Description string           `json:"description,omitempty"`
	Members     []CCExportMember `json:"members"`
}

// CCExportMember represents a member in the CC export.
type CCExportMember struct {
	Name      string `json:"name"`
	AgentType string `json:"agentType"`
	Model     string `json:"model,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

func cmdExportCC(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("export-cc", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	output := fs.String("output", "", "Write output to file path")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team export-cc <team-id> [--json] [--output=path]")
	}
	teamID := fs.Arg(0)

	var resp CCExportResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/export/claude-code", teamID), &resp); err != nil {
		return fmt.Errorf("failed to export CC team: %w", err)
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0o644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Exported team %s to %s\n", teamID, *output)
		return nil
	}

	if *jsonOut {
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Team: %s\n", resp.TeamName)
	if resp.Description != "" {
		fmt.Printf("Description: %s\n", resp.Description)
	}
	fmt.Printf("Members: %d\n", len(resp.Members))
	for _, m := range resp.Members {
		fmt.Printf("  - %s (type: %s)\n", m.Name, m.AgentType)
	}
	return nil
}

// TriggerTeamResponse matches the heartbeat TriggerTeamResponse.
type TriggerTeamResponse struct {
	TeamID              string            `json:"teamId"`
	RuntimeMode         string            `json:"runtimeMode"`
	CoordinationPattern string            `json:"coordinationPattern"`
	QueuePolicy         string            `json:"queuePolicy"`
	Triggers            []TriggerResponse `json:"triggers"`
}

func cmdTriggerTeam(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("trigger", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team trigger <team-id> [--json]")
	}
	teamID := fs.Arg(0)

	var resp TriggerTeamResponse
	if err := ctx.Post(fmt.Sprintf("/teams/%s/trigger", teamID), nil, &resp); err != nil {
		return fmt.Errorf("failed to trigger team: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf(
		"Triggered team %s (runtime: %s, coordination: %s, queue: %s)\n",
		teamID,
		resp.RuntimeMode,
		resp.CoordinationPattern,
		resp.QueuePolicy,
	)
	for _, t := range resp.Triggers {
		fmt.Printf("  - %s: run=%s status=%s\n", t.AgentID, t.RunID, t.Status)
	}
	return nil
}

// MemberContextResponse is the response from the member context endpoint.
type MemberContextResponse struct {
	TeamID  string `json:"teamId"`
	AgentID string `json:"agentId"`
	Prompt  string `json:"prompt"`
}

// PromptPreviewRequest is the request body for previewing a built prompt.
type PromptPreviewRequest struct {
	AgentID string `json:"agentId"`
	TeamID  string `json:"teamId,omitempty"`
}

// PromptPreviewResponse is the flat full prompt preview response.
type PromptPreviewResponse struct {
	TeamID  string `json:"teamId,omitempty"`
	AgentID string `json:"agentId"`
	Prompt  string `json:"prompt"`
}

// PromptSection is one backend-ordered section in a structured prompt preview.
type PromptSection struct {
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	SourcePath string `json:"sourcePath,omitempty"`
	Content    string `json:"content"`
}

// StructuredPromptPreviewResponse is the structured prompt preview response.
type StructuredPromptPreviewResponse struct {
	TeamID   string          `json:"teamId,omitempty"`
	AgentID  string          `json:"agentId"`
	Sections []PromptSection `json:"sections"`
}

// TeamPromptMatrixEntry is one member row in the team prompt matrix.
type TeamPromptMatrixEntry struct {
	AgentID     string          `json:"agentId"`
	DisplayName string          `json:"displayName"`
	Sections    []PromptSection `json:"sections"`
	Error       string          `json:"error,omitempty"`
}

// TeamPromptMatrixResponse is the prompt matrix response.
type TeamPromptMatrixResponse struct {
	TeamID  string                  `json:"teamId"`
	Entries []TeamPromptMatrixEntry `json:"entries"`
}

func cmdPromptPreview(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("prompt-preview", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team prompt-preview <team-id> <agent-id> [--json]")
	}

	req := PromptPreviewRequest{TeamID: fs.Arg(0), AgentID: fs.Arg(1)}
	var resp PromptPreviewResponse
	if err := ctx.Post("/prompt-preview", req, &resp); err != nil {
		return fmt.Errorf("failed to preview prompt: %w", err)
	}

	if *jsonOut {
		return writeJSON(os.Stdout, resp)
	}
	fmt.Print(resp.Prompt)
	return nil
}

func cmdPromptPreviewStructured(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("prompt-preview-structured", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team prompt-preview-structured <team-id> <agent-id> [--json]")
	}

	req := PromptPreviewRequest{TeamID: fs.Arg(0), AgentID: fs.Arg(1)}
	var resp StructuredPromptPreviewResponse
	if err := ctx.Post("/prompt-preview-structured", req, &resp); err != nil {
		return fmt.Errorf("failed to preview structured prompt: %w", err)
	}

	if *jsonOut {
		return writeJSON(os.Stdout, resp)
	}
	formatStructuredPromptPreview(os.Stdout, resp)
	return nil
}

func cmdPromptMatrix(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("prompt-matrix", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team prompt-matrix <team-id> [--json]")
	}

	teamID := fs.Arg(0)
	var resp TeamPromptMatrixResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/prompt-matrix", teamID), &resp); err != nil {
		return fmt.Errorf("failed to get prompt matrix: %w", err)
	}

	if *jsonOut {
		return writeJSON(os.Stdout, resp)
	}
	formatPromptMatrix(os.Stdout, resp)
	return nil
}

func formatStructuredPromptPreview(w io.Writer, resp StructuredPromptPreviewResponse) {
	fmt.Fprintf(w, "Prompt preview: team=%s agent=%s sections=%d\n\n", resp.TeamID, resp.AgentID, len(resp.Sections))
	for i, section := range resp.Sections {
		fmt.Fprintf(w, "## %d. %s\n", i+1, section.Label)
		fmt.Fprintf(w, "Kind: %s\n", section.Kind)
		if section.SourcePath != "" {
			fmt.Fprintf(w, "Source: %s\n", section.SourcePath)
		}
		fmt.Fprintf(w, "Chars: %d\n\n", len(section.Content))
		fmt.Fprintln(w, section.Content)
		if i < len(resp.Sections)-1 {
			fmt.Fprintln(w, "\n---")
		}
	}
}

func formatPromptMatrix(w io.Writer, resp TeamPromptMatrixResponse) {
	fmt.Fprintf(w, "Prompt matrix: team=%s members=%d\n", resp.TeamID, len(resp.Entries))
	if len(resp.Entries) == 0 {
		return
	}

	kinds := promptMatrixKinds(resp.Entries)
	fmt.Fprint(w, "Member")
	for _, kind := range kinds {
		fmt.Fprintf(w, "\t%s", kind)
	}
	fmt.Fprintln(w)

	for _, entry := range resp.Entries {
		name := entry.DisplayName
		if name == "" {
			name = entry.AgentID
		}
		if entry.Error != "" {
			fmt.Fprintf(w, "%s\tERROR: %s\n", name, entry.Error)
			continue
		}
		sectionsByKind := map[string]int{}
		for _, section := range entry.Sections {
			sectionsByKind[section.Kind] += len(section.Content)
		}
		fmt.Fprint(w, name)
		for _, kind := range kinds {
			if count, ok := sectionsByKind[kind]; ok {
				fmt.Fprintf(w, "\t%d", count)
			} else {
				fmt.Fprint(w, "\t-")
			}
		}
		fmt.Fprintln(w)
	}
}

func promptMatrixKinds(entries []TeamPromptMatrixEntry) []string {
	seen := map[string]struct{}{}
	var kinds []string
	for _, entry := range entries {
		for _, section := range entry.Sections {
			if _, ok := seen[section.Kind]; ok {
				continue
			}
			seen[section.Kind] = struct{}{}
			kinds = append(kinds, section.Kind)
		}
	}
	return kinds
}

func writeJSON(w io.Writer, value interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func cmdMemberContext(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("member-context", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team member-context <team-id> <agent-id> [--json]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var resp MemberContextResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/members/%s/context", teamID, agentID), &resp); err != nil {
		return fmt.Errorf("failed to get member context: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// Default: print prompt text directly (most useful for piping)
	fmt.Print(resp.Prompt)
	return nil
}

// --- Team search types ---

// TeamSearchResult represents a text search result for teams.
type TeamSearchResult struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Mission     string  `json:"mission,omitempty"`
	Enabled     bool    `json:"enabled"`
	MemberCount int     `json:"memberCount"`
	Score       float64 `json:"score,omitempty"`
	Highlight   string  `json:"highlight,omitempty"`
}

// TeamSearchResponse wraps team text search results.
type TeamSearchResponse struct {
	Results []TeamSearchResult `json:"results"`
	Total   int                `json:"total"`
	Query   string             `json:"query"`
}

// AITeamSearchResult represents an AI search result for teams.
type AITeamSearchResult struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"displayName"`
	Mission      string  `json:"mission,omitempty"`
	Enabled      bool    `json:"enabled"`
	MemberCount  int     `json:"memberCount"`
	Score        float64 `json:"score"`
	ScorePercent int     `json:"scorePercent"`
}

// AITeamSearchResponse wraps team AI search results.
type AITeamSearchResponse struct {
	Results []AITeamSearchResult `json:"results,omitempty"`
	Total   int                  `json:"total"`
	Query   string               `json:"query"`
	Method  string               `json:"method"`
}

// AITeamSearchRequest is the request body for team AI search.
type AITeamSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// TeamContentSearchMatch represents a content search match in team files.
type TeamContentSearchMatch struct {
	TeamID     string `json:"teamId"`
	TeamName   string `json:"teamName"`
	File       string `json:"file"`
	LineNumber int    `json:"lineNumber"`
	Line       string `json:"line"`
}

// TeamContentSearchResponse wraps team content search results.
type TeamContentSearchResponse struct {
	Matches []TeamContentSearchMatch `json:"matches"`
	Total   int                      `json:"total"`
	Query   string                   `json:"query"`
}

func cmdSearch(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("team search", flag.ContinueOnError)
	textOnly := fs.Bool("text", false, "Force text-only search (skip AI)")
	contentOnly := fs.Bool("content", false, "Search within team shared file contents")
	caseSensitive := fs.Bool("case-sensitive", false, "Case-sensitive content search")
	wholeWord := fs.Bool("whole-word", false, "Whole word matching for content search")
	regex := fs.Bool("regex", false, "Treat query as regex for content search")
	limit := fs.Int("limit", 5, "Maximum number of results")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team search <query> [--text] [--content] [--case-sensitive] [--whole-word] [--regex] [--limit=N] [--json]")
	}

	query := strings.Join(fs.Args(), " ")

	if *contentOnly {
		return teamContentSearch(ctx, query, *limit, *caseSensitive, *wholeWord, *regex, *jsonOut)
	}

	if *textOnly {
		return teamTextSearch(ctx, query, *jsonOut)
	}

	return teamAISearch(ctx, query, *limit, *jsonOut)
}

func teamAISearch(ctx appctx.Context, query string, limit int, jsonOut bool) error {
	req := AITeamSearchRequest{
		Query: query,
		Limit: limit,
	}

	var resp AITeamSearchResponse
	if err := ctx.Post("/search/teams/ai", req, &resp); err != nil {
		fmt.Fprintln(os.Stderr, "(AI search unavailable, using text search)")
		return teamTextSearch(ctx, query, jsonOut)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	methodLabel := "AI"
	if resp.Method == "text" {
		methodLabel = "text (AI unavailable)"
	}

	if resp.Total == 0 {
		fmt.Printf("No teams found matching: %s (%s search)\n", query, methodLabel)
		return nil
	}

	fmt.Printf("Team Search Results (%d found, %s search):\n", resp.Total, methodLabel)
	for _, r := range resp.Results {
		enabled := "enabled"
		if !r.Enabled {
			enabled = "disabled"
		}
		score := fmt.Sprintf(" (%d%%)", r.ScorePercent)
		fmt.Printf("  %s%s (%s, %d members) [%s]\n", r.DisplayName, score, enabled, r.MemberCount, r.ID)
		if r.Mission != "" {
			mission := r.Mission
			if len(mission) > 80 {
				mission = mission[:77] + "..."
			}
			fmt.Printf("    → %s\n", mission)
		}
	}
	return nil
}

func teamTextSearch(ctx appctx.Context, query string, jsonOut bool) error {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}

	var resp TeamSearchResponse
	if err := ctx.GetWithQuery("/search/teams", params, &resp); err != nil {
		return fmt.Errorf("team search failed: %w", err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No teams found matching: %s\n", query)
		return nil
	}

	fmt.Printf("Team Search Results (%d found, text search):\n", resp.Total)
	for _, r := range resp.Results {
		enabled := "enabled"
		if !r.Enabled {
			enabled = "disabled"
		}
		score := ""
		if r.Score > 0 {
			score = fmt.Sprintf(" (%.1f)", r.Score)
		}
		fmt.Printf("  %s%s (%s, %d members) [%s]\n", r.DisplayName, score, enabled, r.MemberCount, r.ID)
		if r.Highlight != "" {
			highlight := r.Highlight
			if len(highlight) > 80 {
				highlight = highlight[:77] + "..."
			}
			fmt.Printf("    → %s\n", highlight)
		}
	}
	return nil
}

func teamContentSearch(ctx appctx.Context, query string, limit int, caseSensitive, wholeWord, regex, jsonOut bool) error {
	params := url.Values{}
	params.Set("q", query)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if caseSensitive {
		params.Set("caseSensitive", "true")
	}
	if wholeWord {
		params.Set("wholeWord", "true")
	}
	if regex {
		params.Set("regex", "true")
	}

	var resp TeamContentSearchResponse
	if err := ctx.GetWithQuery("/search/teams/content", params, &resp); err != nil {
		return fmt.Errorf("team content search failed: %w", err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No content matches found for: %s\n", query)
		return nil
	}

	fmt.Printf("Team Content Matches (%d found):\n", resp.Total)
	for _, m := range resp.Matches {
		line := m.Line
		if len(line) > 120 {
			line = line[:117] + "..."
		}
		fmt.Printf("  %s/%s:%d: %s\n", m.TeamName, m.File, m.LineNumber, line)
	}
	return nil
}

// --- Handoff commands ---

func cmdHandoffLatest(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("handoff-latest", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team handoff-latest <team-id> <agent-id>")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	var resp HandoffResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/members/%s/handoff", teamID, agentID), &resp); err != nil {
		return fmt.Errorf("failed to get handoff: %w", err)
	}

	fmt.Println(resp.Content)
	return nil
}

func cmdHandoffHistory(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("handoff-history", flag.ContinueOnError)
	agent := fs.String("agent", "", "Filter by agent ID")
	last := fs.Int("last", 20, "Number of entries to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team handoff-history <team-id> [--agent=<id>] [--last=N] [--json]")
	}
	teamID := fs.Arg(0)

	query := fmt.Sprintf("/teams/%s/handoff-history?last=%d", teamID, *last)
	if *agent != "" {
		query += "&agent=" + url.QueryEscape(*agent)
	}

	var resp HandoffHistoryResponse
	if err := ctx.Get(query, &resp); err != nil {
		return fmt.Errorf("failed to get handoff history: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Entries) == 0 {
		fmt.Println("No handoff history found")
		return nil
	}

	for _, entry := range resp.Entries {
		fmt.Printf("--- %s [%s] run=%s ---\n", entry.AgentID, entry.Timestamp, entry.RunID)
		fmt.Println(entry.Content)
		fmt.Println()
	}
	return nil
}

// --- Task Board commands ---

func cmdTaskList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("task-list", flag.ContinueOnError)
	limit := fs.Int("limit", 25, "Maximum number of tasks to return")
	offset := fs.Int("offset", 0, "Number of tasks to skip")
	status := fs.String("status", "", "Filter by status (todo|in-progress|blocked|done)")
	assignee := fs.String("assignee", "", "Filter by assignee agent ID")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team task-list <team-id> [--limit=N] [--offset=N] [--status=X] [--assignee=X] [--json]")
	}
	teamID := fs.Arg(0)

	query := fmt.Sprintf("/teams/%s/tasks?limit=%d&offset=%d", teamID, *limit, *offset)
	if *status != "" {
		query += "&status=" + url.QueryEscape(*status)
	}
	if *assignee != "" {
		query += "&assignee=" + url.QueryEscape(*assignee)
	}

	var resp TaskBoardResponse
	if err := ctx.Get(query, &resp); err != nil {
		return fmt.Errorf("failed to get task board: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	fmt.Printf("%-8s %-30s %-12s %-20s %-5s %-5s\n", "PRIO", "TITLE", "STATUS", "ASSIGNEE", "NOTES", "UPDATED")
	for _, task := range resp.Tasks {
		title := task.Title
		if len(title) > 28 {
			title = title[:28] + ".."
		}
		assigneeStr := task.Assignee
		if len(assigneeStr) > 18 {
			assigneeStr = assigneeStr[:18] + ".."
		}
		fmt.Printf("%-8s %-30s %-12s %-20s %-5d %s\n",
			task.Priority, title, task.Status, assigneeStr, len(task.Notes), task.UpdatedAt)
	}

	// Pagination hint
	remaining := resp.Total - resp.Offset - len(resp.Tasks)
	if remaining > 0 {
		nextOffset := resp.Offset + resp.Limit
		fmt.Printf("\n+%d more items. Run `prompt-manager team task-list %s --offset=%d` to see next batch\n", remaining, teamID, nextOffset)
	}
	return nil
}

func cmdTaskAdd(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("task-add", flag.ContinueOnError)
	title := fs.String("title", "", "Task title (required)")
	assignee := fs.String("assignee", "", "Assignee agent ID")
	priority := fs.String("priority", "P3", "Priority (P1-P5)")
	from := fs.String("from", "", "Creator agent ID (required)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team task-add <team-id> --title=\"...\" --from=<id> [--assignee=<id>] [--priority=P2]")
	}
	teamID := fs.Arg(0)

	if strings.TrimSpace(*title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(*from) == "" {
		return fmt.Errorf("from is required")
	}

	req := AddTaskRequest{
		Title:    *title,
		Assignee: *assignee,
		Priority: *priority,
		From:     *from,
	}

	var resp TeamTask
	if err := ctx.Post(fmt.Sprintf("/teams/%s/tasks", teamID), req, &resp); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Created task %s: %s\n", resp.ID, resp.Title)
	return nil
}

func cmdTaskUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("task-update", flag.ContinueOnError)
	status := fs.String("status", "", "New status")
	assignee := fs.String("assignee", "", "New assignee")
	priority := fs.String("priority", "", "New priority")
	note := fs.String("note", "", "Add a note")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team task-update <team-id> <task-id> [--status=X] [--assignee=X] [--priority=X] [--note=\"...\"]")
	}
	teamID := fs.Arg(0)
	taskID := fs.Arg(1)

	req := UpdateTaskRequest{}
	if *status != "" {
		req.Status = status
	}
	if *assignee != "" {
		req.Assignee = assignee
	}
	if *priority != "" {
		req.Priority = priority
	}
	if *note != "" {
		req.Note = note
	}

	var resp TeamTask
	if err := ctx.Put(fmt.Sprintf("/teams/%s/tasks/%s", teamID, taskID), req, &resp); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Updated task %s: %s [%s]\n", resp.ID, resp.Title, resp.Status)
	return nil
}

func cmdTaskDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("task-delete", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team task-delete <team-id> <task-id>")
	}
	teamID := fs.Arg(0)
	taskID := fs.Arg(1)

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/tasks/%s", teamID, taskID)); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	fmt.Printf("Deleted task %s\n", taskID)
	return nil
}

// --- Decision Log commands ---

func cmdDecisionAdd(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-add", flag.ContinueOnError)
	by := fs.String("by", "", "Agent ID who made the decision (required)")
	decision := fs.String("decision", "", "The decision (simple mode, required without --options)")
	rationale := fs.String("rationale", "", "Why this decision was made")
	contextTag := fs.String("context", "", "Context tag for grouping")
	supersedes := fs.String("supersedes", "", "ID of decision this replaces")
	topic := fs.String("topic", "", "What is being decided (multi-option mode, required with --options)")
	description := fs.String("description", "", "Background/context for the decision (multi-option mode)")
	options := fs.String("options", "", `JSON array of options, e.g. '[{"key":"A","label":"Option A","rationale":"Why A","recommended":true}]'`)
	initiativeMetadataInline := fs.String("initiative-metadata", "", `Structured initiative metadata for context=initiative-proposal decisions, as JSON: '{"name":"my-init","priority":5,"depends_on":["..."],"target_scenario":"swarm-manager","title":"..."}'`)
	initiativeMetadataFile := fs.String("initiative-metadata-file", "", "Path to a JSON file containing the same payload as --initiative-metadata (mutually exclusive)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team decision-add <team-id> --by=<id> [--decision=\"...\" --rationale=\"...\"] [--topic=\"...\" --description=\"...\" --options='[...]'] [--initiative-metadata='{...}']")
	}
	teamID := fs.Arg(0)

	if strings.TrimSpace(*by) == "" {
		return fmt.Errorf("by is required")
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	meta, err := loadInitiativeMetadataFromFlags(provided, *initiativeMetadataInline, *initiativeMetadataFile)
	if err != nil {
		return err
	}
	if meta != nil && *contextTag != "initiative-proposal" {
		return fmt.Errorf("--initiative-metadata is only valid when --context=initiative-proposal")
	}

	req := AddDecisionRequest{
		By:                 *by,
		Context:            *contextTag,
		Supersedes:         *supersedes,
		Rationale:          *rationale,
		InitiativeMetadata: meta,
	}

	if strings.TrimSpace(*options) != "" {
		// Multi-option mode
		if strings.TrimSpace(*topic) == "" {
			return fmt.Errorf("topic is required when options are provided")
		}
		var opts []DecisionOption
		if err := json.Unmarshal([]byte(*options), &opts); err != nil {
			return fmt.Errorf("invalid options JSON: %w", err)
		}
		req.Topic = *topic
		req.Description = *description
		req.Options = opts
	} else {
		// Simple mode
		if strings.TrimSpace(*decision) == "" {
			return fmt.Errorf("decision is required (or use --topic + --options for multi-option)")
		}
		if strings.TrimSpace(*rationale) == "" {
			return fmt.Errorf("rationale is required")
		}
		req.Decision = *decision
	}

	var resp DecisionEntry
	if err := ctx.Post(fmt.Sprintf("/teams/%s/decisions", teamID), req, &resp); err != nil {
		return fmt.Errorf("failed to add decision: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Topic != "" {
		fmt.Printf("Logged decision %s: %s (%d options)\n", resp.ID, resp.Topic, len(resp.Options))
	} else {
		fmt.Printf("Logged decision %s: %s\n", resp.ID, resp.Decision)
	}
	return nil
}

func cmdDecisionList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-list", flag.ContinueOnError)
	contextTag := fs.String("context", "", "Filter by context tag")
	status := fs.String("status", "", "Filter by status (pending|accepted|rejected|running|completed|deferred)")
	last := fs.Int("last", 10, "Number of entries to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team decision-list <team-id> [--context=<tag>] [--status=<status>] [--last=N] [--json]")
	}
	teamID := fs.Arg(0)

	query := fmt.Sprintf("/teams/%s/decisions?last=%d", teamID, *last)
	if *contextTag != "" {
		query += "&context=" + url.QueryEscape(*contextTag)
	}
	if *status != "" {
		query += "&status=" + url.QueryEscape(*status)
	}

	var resp DecisionListResponse
	if err := ctx.Get(query, &resp); err != nil {
		return fmt.Errorf("failed to get decisions: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Entries) == 0 {
		fmt.Println("No decisions found")
		return nil
	}

	for _, entry := range resp.Entries {
		contextStr := ""
		if entry.Context != "" {
			contextStr = fmt.Sprintf(" [%s]", entry.Context)
		}
		supersededStr := ""
		if entry.Supersedes != "" {
			supersededStr = fmt.Sprintf(" (supersedes %s)", entry.Supersedes)
		}
		statusStr := ""
		if entry.Status != "" {
			statusStr = fmt.Sprintf(" (%s)", entry.Status)
		}
		fmt.Printf("--- %s by %s%s%s%s ---\n", entry.ID, entry.By, contextStr, supersededStr, statusStr)

		if len(entry.Options) > 0 {
			fmt.Printf("Topic: %s\n", entry.Topic)
			if entry.Description != "" {
				fmt.Printf("Description: %s\n", entry.Description)
			}
			if entry.Rationale != "" {
				fmt.Printf("Rationale: %s\n", entry.Rationale)
			}
			for _, opt := range entry.Options {
				marker := "  "
				if entry.Selected == opt.Key {
					marker = "→ "
				}
				rec := ""
				if opt.Recommended {
					rec = " [RECOMMENDED]"
				}
				fmt.Printf("  %s%s) %s%s — %s\n", marker, opt.Key, opt.Label, rec, opt.Rationale)
			}
			if entry.Selected != "" {
				if entry.Selected == "__other__" {
					fmt.Printf("Selected: Other — %s\n", entry.Freeform)
				} else {
					fmt.Printf("Selected: %s\n", entry.Selected)
				}
			}
			if entry.Notes != "" {
				fmt.Printf("Notes: %s\n", entry.Notes)
			}
		} else {
			fmt.Printf("Decision: %s\n", entry.Decision)
			fmt.Printf("Rationale: %s\n", entry.Rationale)
		}
		fmt.Println()
	}

	// Pagination hint
	remaining := resp.Total - len(resp.Entries)
	if remaining > 0 {
		fmt.Printf("+%d more entries. Run `prompt-manager team decision-list %s --last=%d` to see more\n", remaining, teamID, resp.Total)
	}
	return nil
}

// cmdDecisionUpdate is the generic PATCH wrapper exposing every field of the
// API's UpdateDecisionRequest as an optional flag.
func cmdDecisionUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-update", flag.ContinueOnError)
	decision := fs.String("decision", "", "Update decision text")
	rationale := fs.String("rationale", "", "Update rationale")
	contextTag := fs.String("context", "", "Update context tag")
	status := fs.String("status", "", "Update status (pending|accepted|rejected|running|completed)")
	supersedes := fs.String("supersedes", "", "ID of decision this supersedes")
	topic := fs.String("topic", "", "Update topic (multi-option mode)")
	description := fs.String("description", "", "Update description (multi-option mode)")
	options := fs.String("options", "", `JSON array of options, e.g. '[{"key":"A","label":"...","rationale":"..."}]'`)
	selected := fs.String("selected", "", "Selected option key (use __other__ with --freeform for write-in)")
	freeform := fs.String("freeform", "", "Freeform answer when --selected=__other__")
	notes := fs.String("notes", "", "Operator notes attached to the decision")
	initiativeMetadataInline := fs.String("initiative-metadata", "", `Update structured initiative metadata, as JSON. Pre-accept only.`)
	initiativeMetadataFile := fs.String("initiative-metadata-file", "", "Path to a JSON file containing the same payload as --initiative-metadata (mutually exclusive)")
	autoCreateStatus := fs.String("auto-create-status", "", "Set auto_create_status (manual-recovery: failed→created or failed→failed)")
	autoCreateRef := fs.String("auto-create-initiative-ref", "", "Initiative reference (\"<scenario>/<name>\") set alongside --auto-create-status=created")
	autoCreateError := fs.String("auto-create-error", "", "Updated error message for failed→failed re-record (optional)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team decision-update <team-id> <decision-id> [--decision=...] [--rationale=...] [--context=...] [--status=...] [--supersedes=...] [--topic=...] [--description=...] [--options='[...]'] [--selected=...] [--freeform=...] [--notes=...] [--initiative-metadata='{...}'] [--auto-create-status=created --auto-create-initiative-ref=<ref>]")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)

	type updateReq struct {
		Decision                *string                     `json:"decision,omitempty"`
		Rationale               *string                     `json:"rationale,omitempty"`
		Context                 *string                     `json:"context,omitempty"`
		Status                  *string                     `json:"status,omitempty"`
		Supersedes              *string                     `json:"supersedes,omitempty"`
		Topic                   *string                     `json:"topic,omitempty"`
		Description             *string                     `json:"description,omitempty"`
		Options                 *[]DecisionOption           `json:"options,omitempty"`
		Selected                *string                     `json:"selected,omitempty"`
		Freeform                *string                     `json:"freeform,omitempty"`
		Notes                   *string                     `json:"notes,omitempty"`
		InitiativeMetadata      *DecisionInitiativeMetadata `json:"initiative_metadata,omitempty"`
		AutoCreateStatus        *string                     `json:"auto_create_status,omitempty"`
		AutoCreateInitiativeRef *string                     `json:"auto_create_initiative_ref,omitempty"`
		AutoCreateError         *string                     `json:"auto_create_error,omitempty"`
	}

	// Track which flags were explicitly set so that empty-string values can
	// still be sent (e.g., --notes "" to clear notes). flag.Visit only
	// iterates over flags the user actually passed.
	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })

	req := updateReq{}
	if provided["decision"] {
		req.Decision = decision
	}
	if provided["rationale"] {
		req.Rationale = rationale
	}
	if provided["context"] {
		req.Context = contextTag
	}
	if provided["status"] {
		req.Status = status
	}
	if provided["supersedes"] {
		req.Supersedes = supersedes
	}
	if provided["topic"] {
		req.Topic = topic
	}
	if provided["description"] {
		req.Description = description
	}
	if provided["options"] {
		var opts []DecisionOption
		if strings.TrimSpace(*options) != "" {
			if err := json.Unmarshal([]byte(*options), &opts); err != nil {
				return fmt.Errorf("invalid options JSON: %w", err)
			}
		}
		req.Options = &opts
	}
	if provided["selected"] {
		req.Selected = selected
	}
	if provided["freeform"] {
		req.Freeform = freeform
	}
	if provided["notes"] {
		req.Notes = notes
	}

	// initiative-metadata: mutually-exclusive flag pair, parsed via shared helper.
	meta, err := loadInitiativeMetadataFromFlags(provided, *initiativeMetadataInline, *initiativeMetadataFile)
	if err != nil {
		return err
	}
	if meta != nil {
		req.InitiativeMetadata = meta
	}

	// auto-create-status manual-recovery flags (per d8=C). Status is the
	// gate; ref/error only meaningful when status is set.
	if provided["auto-create-status"] {
		req.AutoCreateStatus = autoCreateStatus
	}
	if provided["auto-create-initiative-ref"] {
		req.AutoCreateInitiativeRef = autoCreateRef
	}
	if provided["auto-create-error"] {
		req.AutoCreateError = autoCreateError
	}

	// Reject a no-op call early — a PATCH with no fields is almost certainly a typo.
	if req.Decision == nil && req.Rationale == nil && req.Context == nil && req.Status == nil &&
		req.Supersedes == nil && req.Topic == nil && req.Description == nil && req.Options == nil &&
		req.Selected == nil && req.Freeform == nil && req.Notes == nil &&
		req.InitiativeMetadata == nil && req.AutoCreateStatus == nil &&
		req.AutoCreateInitiativeRef == nil && req.AutoCreateError == nil {
		return fmt.Errorf("decision-update requires at least one field flag")
	}

	var resp UpdateDecisionResponse
	if err := ctx.Put(fmt.Sprintf("/teams/%s/decisions/%s", teamID, decisionID), req, &resp); err != nil {
		return fmt.Errorf("failed to update decision: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	statusStr := ""
	if resp.Status != "" {
		statusStr = fmt.Sprintf(" [%s]", resp.Status)
	}
	selectedStr := ""
	if resp.Selected != "" {
		selectedStr = fmt.Sprintf(" selected=%s", resp.Selected)
	}
	fmt.Printf("Updated decision %s%s%s\n", resp.ID, statusStr, selectedStr)
	fprintAutoCreateOutcome(os.Stdout, resp.AutoCreateOutcome)
	return nil
}

// parseInitiativeMetadataJSON parses a JSON payload into a
// DecisionInitiativeMetadata. CLI-boundary validation mirrors the API's
// policy (kebab-case name, priority bounds) so operators get fast errors.
// Server-side validation remains authoritative.
func parseInitiativeMetadataJSON(data []byte) (*DecisionInitiativeMetadata, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("payload is empty")
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.DisallowUnknownFields()
	var m DecisionInitiativeMetadata
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if strings.TrimSpace(m.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if m.Priority != 0 && (m.Priority < 1 || m.Priority > 10) {
		return nil, fmt.Errorf("priority %d must be 0 (unset) or in 1-10", m.Priority)
	}
	for i, d := range m.DependsOn {
		if strings.TrimSpace(d) == "" {
			return nil, fmt.Errorf("depends_on[%d] must be a non-empty string", i)
		}
	}
	return &m, nil
}

// loadInitiativeMetadataFromFlags reads --initiative-metadata or
// --initiative-metadata-file (mutually exclusive) and returns the parsed
// metadata or nil when neither was supplied.
func loadInitiativeMetadataFromFlags(provided map[string]bool, inline, filePath string) (*DecisionInitiativeMetadata, error) {
	if provided["initiative-metadata"] && provided["initiative-metadata-file"] {
		return nil, fmt.Errorf("--initiative-metadata and --initiative-metadata-file are mutually exclusive")
	}
	switch {
	case provided["initiative-metadata"]:
		m, err := parseInitiativeMetadataJSON([]byte(inline))
		if err != nil {
			return nil, fmt.Errorf("--initiative-metadata: %w", err)
		}
		return m, nil
	case provided["initiative-metadata-file"]:
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("--initiative-metadata-file: %w", err)
		}
		m, err := parseInitiativeMetadataJSON(data)
		if err != nil {
			return nil, fmt.Errorf("--initiative-metadata-file: %w", err)
		}
		return m, nil
	}
	return nil, nil
}

// AutoCreateOutcome mirrors the API response field returned alongside
// decision-update / decision-accept when an `initiative-proposal` decision
// is accepted. The CLI renders this as the structured "Auto-Created
// Initiative" block (per d9=A).
type AutoCreateOutcome struct {
	Status             string `json:"status"`
	InitiativeRef      string `json:"initiative_ref,omitempty"`
	Error              string `json:"error,omitempty"`
	WorkaroundCommand  string `json:"workaround_command,omitempty"`
	ResolveCommand     string `json:"resolve_command,omitempty"`
	DescriptionTmpFile string `json:"description_tmp_file,omitempty"`
	TargetScenario     string `json:"target_scenario,omitempty"`
	InitiativeName     string `json:"initiative_name,omitempty"`
	Priority           int    `json:"priority,omitempty"`
}

// UpdateDecisionResponse extends DecisionEntry with an optional
// AutoCreateOutcome payload (set when an initiative-proposal decision is
// accepted). Mirrors the API response shape.
type UpdateDecisionResponse struct {
	DecisionEntry
	AutoCreateOutcome *AutoCreateOutcome `json:"auto_create_outcome,omitempty"`
}

// fprintAutoCreateOutcome renders the structured auto-create block per d9=A.
// Success: shows ref + get/edit hints + a "Reminder:" to enrich the initiative.
// Failure: shows the error and the pre-filled workaround + resolve commands.
func fprintAutoCreateOutcome(w io.Writer, o *AutoCreateOutcome) {
	if o == nil {
		return
	}
	fmt.Fprintln(w, "Auto-Created Initiative:")
	if o.Status == "created" {
		if o.InitiativeName != "" {
			fmt.Fprintf(w, "  Name:        %s\n", o.InitiativeName)
		}
		if o.Priority != 0 {
			fmt.Fprintf(w, "  Priority:    %d\n", o.Priority)
		}
		target := o.TargetScenario
		if target == "" {
			target = "swarm-manager"
		}
		fmt.Fprintf(w, "  Inspect:     %s initiatives get --name %s\n", target, o.InitiativeName)
		fmt.Fprintf(w, "  Edit:        %s initiatives update --name %s ...\n", target, o.InitiativeName)
		fmt.Fprintln(w, "Reminder: enrich the initiative (description, members, additional depends_on) as needed.")
		return
	}
	// Failure path: print exact workaround + resolve commands.
	fmt.Fprintf(w, "  Status: failed (%s)\n", o.Error)
	fmt.Fprintln(w)
	if o.WorkaroundCommand != "" {
		fmt.Fprintln(w, "To create the initiative manually, run:")
		fmt.Fprintf(w, "  %s\n", o.WorkaroundCommand)
		if o.DescriptionTmpFile != "" {
			fmt.Fprintf(w, "  (description body materialised to %s)\n", o.DescriptionTmpFile)
		}
		fmt.Fprintln(w)
	}
	if o.ResolveCommand != "" {
		fmt.Fprintln(w, "Once created, mark the decision as resolved:")
		fmt.Fprintf(w, "  %s\n", o.ResolveCommand)
	}
}

// parseDecisionModificationsJSON parses a JSON payload into a
// DecisionModifications and performs CLI-boundary validation that mirrors the
// API's policy so operators get fast, clear errors. Server-side validation
// remains authoritative (see
// docs/reference/decision-modifications-contract.md).
func parseDecisionModificationsJSON(data []byte) (*DecisionModifications, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("payload is empty")
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.DisallowUnknownFields()
	var m DecisionModifications
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	for i, s := range m.ExcludedClauses {
		if strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("excluded_clauses[%d] must be a non-empty string", i)
		}
	}
	for i, s := range m.Additions {
		if strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("additions[%d] must be a non-empty string", i)
		}
	}
	if len(m.ExcludedClauses) == 0 && len(m.Additions) == 0 && strings.TrimSpace(m.Rationale) == "" {
		return nil, fmt.Errorf("must contain at least one of excluded_clauses, additions, or rationale")
	}
	return &m, nil
}

// cmdDecisionAccept is the convenience wrapper for the most common operator
// action: set status=accepted, record the chosen option key, and (optionally)
// attach notes. --selected is required; an accept-without-choice is almost
// always a mistake.
func cmdDecisionAccept(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-accept", flag.ContinueOnError)
	selected := fs.String("selected", "", "Selected option key (required for multi-option decisions; omit for single-proposal decisions; use __other__ with --freeform for write-in)")
	freeform := fs.String("freeform", "", "Freeform answer when --selected=__other__")
	notes := fs.String("notes", "", "Operator notes (free-form commentary; for structured exceptions use --modifications)")
	modsJSON := fs.String("modifications", "", `Structured exception against the selected option's rationale, as JSON: '{"excluded_clauses":[...],"additions":[...],"rationale":"..."}'. Distinct from --notes. Immutable once set.`)
	modsFile := fs.String("modifications-file", "", "Path to a JSON file containing the same payload as --modifications (mutually exclusive)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team decision-accept <team-id> <decision-id> [--selected=<option-key>] [--freeform=\"...\"] [--notes=\"...\"] [--modifications='{...}' | --modifications-file=<path>]")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)

	// Pre-fetch the decision to detect single-proposal (no Options) shape.
	// Single-proposal decisions are accepted with no --selected; multi-option
	// decisions still require it.
	var existing DecisionEntry
	{
		var listResp DecisionListResponse
		if err := ctx.Get(fmt.Sprintf("/teams/%s/decisions?last=0", teamID), &listResp); err != nil {
			return fmt.Errorf("failed to fetch decision: %w", err)
		}
		found := false
		for _, e := range listResp.Entries {
			if e.ID == decisionID {
				existing = e
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("decision %s not found in team %s", decisionID, teamID)
		}
	}
	singleProposal := len(existing.Options) == 0

	if singleProposal {
		if strings.TrimSpace(*selected) != "" {
			return fmt.Errorf("single-proposal decisions are accepted with no --selected; rerun without it")
		}
	} else {
		if strings.TrimSpace(*selected) == "" {
			return fmt.Errorf("--selected is required for decision-accept on multi-option decisions (use decision-update for status-only changes)")
		}
		if *selected == "__other__" && strings.TrimSpace(*freeform) == "" {
			return fmt.Errorf("--freeform is required when --selected=__other__")
		}
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })

	var mods *DecisionModifications
	if provided["modifications"] && provided["modifications-file"] {
		return fmt.Errorf("--modifications and --modifications-file are mutually exclusive")
	}
	switch {
	case provided["modifications"]:
		parsed, err := parseDecisionModificationsJSON([]byte(*modsJSON))
		if err != nil {
			return fmt.Errorf("--modifications: %w", err)
		}
		mods = parsed
	case provided["modifications-file"]:
		data, err := os.ReadFile(*modsFile)
		if err != nil {
			return fmt.Errorf("--modifications-file: %w", err)
		}
		parsed, err := parseDecisionModificationsJSON(data)
		if err != nil {
			return fmt.Errorf("--modifications-file: %w", err)
		}
		mods = parsed
	}

	type acceptReq struct {
		Status        string                 `json:"status"`
		Selected      *string                `json:"selected,omitempty"`
		Freeform      *string                `json:"freeform,omitempty"`
		Notes         *string                `json:"notes,omitempty"`
		Modifications *DecisionModifications `json:"modifications,omitempty"`
	}
	req := acceptReq{Status: "accepted"}
	if !singleProposal {
		s := *selected
		req.Selected = &s
	}
	if provided["freeform"] {
		req.Freeform = freeform
	}
	if provided["notes"] {
		req.Notes = notes
	}
	if mods != nil {
		req.Modifications = mods
	}

	var resp UpdateDecisionResponse
	if err := ctx.Put(fmt.Sprintf("/teams/%s/decisions/%s", teamID, decisionID), req, &resp); err != nil {
		return fmt.Errorf("failed to accept decision: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.AcceptedAsProposed || singleProposal {
		fmt.Printf("Decision %s accepted (as proposed).\n", resp.ID)
	} else {
		fmt.Printf("Decision %s accepted (selected: %s).\n", resp.ID, resp.Selected)
	}
	fprintAutoCreateOutcome(os.Stdout, resp.AutoCreateOutcome)
	return nil
}

// cmdDecisionReject is the convenience wrapper for rejecting a decision.
// --notes is required at the CLI layer; the API does not enforce it but a
// reject-without-reason is a nudge against future-self confusion.
func cmdDecisionReject(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-reject", flag.ContinueOnError)
	notes := fs.String("notes", "", "Reason for rejection (required)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team decision-reject <team-id> <decision-id> --notes=\"reason\"")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)

	if strings.TrimSpace(*notes) == "" {
		return fmt.Errorf("--notes is required for decision-reject (capture the reason for future reference)")
	}

	type rejectReq struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	req := rejectReq{Status: "rejected", Notes: *notes}

	var resp DecisionEntry
	if err := ctx.Put(fmt.Sprintf("/teams/%s/decisions/%s", teamID, decisionID), req, &resp); err != nil {
		return fmt.Errorf("failed to reject decision: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Rejected decision %s\n", resp.ID)
	return nil
}

// cmdDecisionDefer parks a pending or already-deferred decision until a
// specified revisit date. The decision disappears from the pending queue
// until that date passes, at which point the next pending-queue read
// auto-promotes it back to pending.
func cmdDecisionDefer(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-defer", flag.ContinueOnError)
	revisitAfter := fs.String("revisit-after", "", "Date to revisit the decision (YYYY-MM-DD, required)")
	notes := fs.String("notes", "", "Optional reason for deferring")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team decision-defer <team-id> <decision-id> --revisit-after=YYYY-MM-DD [--notes=\"...\"]")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)

	if strings.TrimSpace(*revisitAfter) == "" {
		return fmt.Errorf("--revisit-after is required (format: YYYY-MM-DD)")
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*revisitAfter))
	if err != nil {
		return fmt.Errorf("--revisit-after must be in YYYY-MM-DD format: %w", err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if parsed.UTC().Truncate(24 * time.Hour).Before(today) {
		return fmt.Errorf("--revisit-after must be today or in the future (got %s)", *revisitAfter)
	}

	type deferReq struct {
		Status       string  `json:"status"`
		RevisitAfter string  `json:"revisit_after"`
		Notes        *string `json:"notes,omitempty"`
	}
	req := deferReq{Status: "deferred", RevisitAfter: parsed.Format("2006-01-02")}
	if strings.TrimSpace(*notes) != "" {
		n := *notes
		req.Notes = &n
	}

	var resp DecisionEntry
	if err := ctx.Put(fmt.Sprintf("/teams/%s/decisions/%s", teamID, decisionID), req, &resp); err != nil {
		return fmt.Errorf("failed to defer decision: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	revisit := req.RevisitAfter
	if resp.RevisitAfter != nil {
		revisit = *resp.RevisitAfter
	}
	fmt.Printf("Deferred decision %s in team %s — revisit on %s\n", resp.ID, teamID, revisit)
	if resp.Notes != "" {
		fmt.Printf("Notes: %s\n", resp.Notes)
	}
	return nil
}

// cmdDecisionDelete removes a decision entry. Pass --yes to skip confirmation.
func cmdDecisionDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-delete", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team decision-delete <team-id> <decision-id> [--yes]")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)

	if !*yes {
		fmt.Fprintf(os.Stderr, "Delete decision %s from team %s? [y/N]: ", decisionID, teamID)
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			return fmt.Errorf("aborted")
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/decisions/%s", teamID, decisionID)); err != nil {
		return fmt.Errorf("failed to delete decision: %w", err)
	}

	fmt.Printf("Deleted decision %s\n", decisionID)
	return nil
}

// cmdDecisionShow returns a single decision by id. The API exposes no
// single-show endpoint; this command paginates through the list endpoint and
// filters client-side. Cheap for typical decision-log sizes.
func cmdDecisionShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team decision-show <team-id> <decision-id> [--json]")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)

	// last=0 fetches all entries (the API treats non-positive as no limit).
	var resp DecisionListResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/decisions?last=0", teamID), &resp); err != nil {
		return fmt.Errorf("failed to fetch decisions: %w", err)
	}

	for _, entry := range resp.Entries {
		if entry.ID == decisionID {
			if *jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(entry)
			}
			formatDecisionShow(os.Stdout, entry)
			return nil
		}
	}

	return fmt.Errorf("decision %s not found in team %s", decisionID, teamID)
}

// formatDecisionShow renders a single decision for `decision-show`'s default
// human output. Distinct from printDecisionEntry (used by decision-list /
// decisions-pending) because the show view is the operator's primary surface
// for inspecting a decision's options before accepting; it always renders the
// options block — with key, label, rationale, and a `(recommended)` marker —
// when options are present. Decisions without options omit the section
// entirely (no empty header). See workshop d2 = A in plan.md.
func formatDecisionShow(w io.Writer, entry DecisionEntry) {
	contextStr := ""
	if entry.Context != "" {
		contextStr = fmt.Sprintf(" [%s]", entry.Context)
	}
	supersededStr := ""
	if entry.Supersedes != "" {
		supersededStr = fmt.Sprintf(" (supersedes %s)", entry.Supersedes)
	}
	statusStr := ""
	if entry.Status != "" {
		statusStr = fmt.Sprintf(" (%s)", entry.Status)
	}
	fmt.Fprintf(w, "--- %s by %s%s%s%s ---\n", entry.ID, entry.By, contextStr, supersededStr, statusStr)
	if entry.RevisitAfter != nil && *entry.RevisitAfter != "" {
		fmt.Fprintf(w, "Revisit after: %s\n", *entry.RevisitAfter)
	}

	if len(entry.Options) > 0 {
		if entry.Topic != "" {
			fmt.Fprintf(w, "Topic: %s\n", entry.Topic)
		}
		if entry.Description != "" {
			fmt.Fprintf(w, "Description: %s\n", entry.Description)
		}
		if entry.Rationale != "" {
			fmt.Fprintf(w, "Rationale: %s\n", entry.Rationale)
		}
		fmt.Fprintln(w, "Options:")
		for _, opt := range entry.Options {
			marker := "  "
			if entry.Selected == opt.Key {
				marker = "→ "
			}
			rec := ""
			if opt.Recommended {
				rec = " (recommended)"
			}
			fmt.Fprintf(w, "%s%s  %s%s\n", marker, opt.Key, opt.Label, rec)
			if opt.Rationale != "" {
				fmt.Fprintf(w, "     %s\n", opt.Rationale)
			}
		}
		if entry.Selected != "" {
			if entry.Selected == "__other__" {
				fmt.Fprintf(w, "Selected: Other — %s\n", entry.Freeform)
			} else {
				fmt.Fprintf(w, "Selected: %s\n", entry.Selected)
			}
		}
		if entry.Notes != "" {
			fmt.Fprintf(w, "Notes: %s\n", entry.Notes)
		}
	} else {
		fmt.Fprintf(w, "Decision: %s\n", entry.Decision)
		fmt.Fprintf(w, "Rationale: %s\n", entry.Rationale)
		if isAcceptedAsProposed(entry) {
			fmt.Fprintln(w, "Selected: accepted as proposed")
		}
		if entry.Notes != "" {
			fmt.Fprintf(w, "Notes: %s\n", entry.Notes)
		}
	}
	fprintDecisionModifications(w, entry.Modifications)
	fprintInitiativeMetadata(w, entry.InitiativeMetadata)
	fprintAutoCreateStatus(w, entry)
}

// fprintInitiativeMetadata renders the structured initiative_metadata block
// on `initiative-proposal` decisions. Empty metadata renders nothing.
func fprintInitiativeMetadata(w io.Writer, m *DecisionInitiativeMetadata) {
	if m == nil {
		return
	}
	fmt.Fprintln(w, "Initiative Metadata:")
	fmt.Fprintf(w, "  Name:            %s\n", m.Name)
	if m.Title != "" {
		fmt.Fprintf(w, "  Title:           %s\n", m.Title)
	}
	if m.Priority != 0 {
		fmt.Fprintf(w, "  Priority:        %d\n", m.Priority)
	}
	if len(m.DependsOn) > 0 {
		fmt.Fprintf(w, "  Depends on:      %s\n", strings.Join(m.DependsOn, ", "))
	}
	target := m.TargetScenario
	if target == "" {
		target = "swarm-manager (default)"
	}
	fmt.Fprintf(w, "  Target scenario: %s\n", target)
}

// fprintAutoCreateStatus renders the auto_create_status block when a status
// has been recorded (created / failed / pending).
func fprintAutoCreateStatus(w io.Writer, entry DecisionEntry) {
	if entry.AutoCreateStatus == "" {
		return
	}
	fmt.Fprintln(w, "Auto-Create Status:")
	fmt.Fprintf(w, "  Status:    %s\n", entry.AutoCreateStatus)
	if entry.AutoCreateInitiativeRef != "" {
		fmt.Fprintf(w, "  Reference: %s\n", entry.AutoCreateInitiativeRef)
	}
	if entry.AutoCreateError != "" {
		fmt.Fprintf(w, "  Error:     %s\n", entry.AutoCreateError)
	}
}

func fprintDecisionModifications(w io.Writer, m *DecisionModifications) {
	if m == nil {
		return
	}
	fmt.Fprintln(w, "Modifications:")
	if len(m.ExcludedClauses) > 0 {
		fmt.Fprintln(w, "  Excluded clauses:")
		for _, c := range m.ExcludedClauses {
			fmt.Fprintf(w, "    - %s\n", c)
		}
	}
	if len(m.Additions) > 0 {
		fmt.Fprintln(w, "  Additions:")
		for _, c := range m.Additions {
			fmt.Fprintf(w, "    + %s\n", c)
		}
	}
	if m.Rationale != "" {
		fmt.Fprintf(w, "  Rationale: %s\n", m.Rationale)
	}
}

// PendingDecisionTeamGroup mirrors the API response for grouped pending decisions.
type PendingDecisionTeamGroup struct {
	TeamID   string          `json:"teamId"`
	TeamName string          `json:"teamName"`
	Entries  []DecisionEntry `json:"entries"`
}

// AllPendingDecisionsResponse mirrors the API response for cross-team pending decisions.
type AllPendingDecisionsResponse struct {
	Teams      []PendingDecisionTeamGroup `json:"teams"`
	TotalCount int                        `json:"totalCount"`
}

// cmdDecisionsPending lists every pending decision grouped by team — the
// surface the morning-vision-walk prep agent and operator-side dashboards
// rely on for triage.
func cmdDecisionsPending(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decisions-pending", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp AllPendingDecisionsResponse
	if err := ctx.Get("/decisions/pending", &resp); err != nil {
		return fmt.Errorf("failed to fetch pending decisions: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.TotalCount == 0 {
		fmt.Println("No pending decisions across any team")
		return nil
	}

	fmt.Printf("%d pending decision(s) across %d team(s):\n\n", resp.TotalCount, len(resp.Teams))
	for _, group := range resp.Teams {
		fmt.Printf("=== %s (%s) — %d pending ===\n", group.TeamName, group.TeamID, len(group.Entries))
		for _, entry := range group.Entries {
			printDecisionEntry(entry)
			fmt.Println()
		}
	}
	return nil
}

// printDecisionEntry renders a single decision in the same format used by decision-list.
func printDecisionEntry(entry DecisionEntry) {
	contextStr := ""
	if entry.Context != "" {
		contextStr = fmt.Sprintf(" [%s]", entry.Context)
	}
	supersededStr := ""
	if entry.Supersedes != "" {
		supersededStr = fmt.Sprintf(" (supersedes %s)", entry.Supersedes)
	}
	statusStr := ""
	if entry.Status != "" {
		statusStr = fmt.Sprintf(" (%s)", entry.Status)
	}
	fmt.Printf("--- %s by %s%s%s%s ---\n", entry.ID, entry.By, contextStr, supersededStr, statusStr)
	if entry.RevisitAfter != nil && *entry.RevisitAfter != "" {
		fmt.Printf("Revisit after: %s\n", *entry.RevisitAfter)
	}

	if len(entry.Options) > 0 {
		fmt.Printf("Topic: %s\n", entry.Topic)
		if entry.Description != "" {
			fmt.Printf("Description: %s\n", entry.Description)
		}
		if entry.Rationale != "" {
			fmt.Printf("Rationale: %s\n", entry.Rationale)
		}
		for _, opt := range entry.Options {
			marker := "  "
			if entry.Selected == opt.Key {
				marker = "→ "
			}
			rec := ""
			if opt.Recommended {
				rec = " [RECOMMENDED]"
			}
			fmt.Printf("  %s%s) %s%s — %s\n", marker, opt.Key, opt.Label, rec, opt.Rationale)
		}
		if entry.Selected != "" {
			if entry.Selected == "__other__" {
				fmt.Printf("Selected: Other — %s\n", entry.Freeform)
			} else {
				fmt.Printf("Selected: %s\n", entry.Selected)
			}
		}
		if entry.Notes != "" {
			fmt.Printf("Notes: %s\n", entry.Notes)
		}
		printDecisionModifications(entry.Modifications)
	} else {
		fmt.Printf("Decision: %s\n", entry.Decision)
		fmt.Printf("Rationale: %s\n", entry.Rationale)
		if isAcceptedAsProposed(entry) {
			fmt.Println("Selected: accepted as proposed")
		}
		if entry.Notes != "" {
			fmt.Printf("Notes: %s\n", entry.Notes)
		}
		printDecisionModifications(entry.Modifications)
	}
}

// isAcceptedAsProposed returns true for both the new `accepted_as_proposed`
// flag and historical entries that used the `__other__ + freeform="accept as
// proposed"` workaround on a single-proposal decision (read-side normalization).
func isAcceptedAsProposed(entry DecisionEntry) bool {
	if entry.AcceptedAsProposed {
		return true
	}
	if len(entry.Options) == 0 && entry.Selected == "__other__" {
		return strings.Contains(strings.ToLower(strings.TrimSpace(entry.Freeform)), "accept as proposed")
	}
	return false
}

// printDecisionModifications renders the structured-exceptions block beneath
// notes. Keeps modifications visually distinct from free-form notes.
func printDecisionModifications(m *DecisionModifications) {
	if m == nil {
		return
	}
	fmt.Println("Modifications:")
	if len(m.ExcludedClauses) > 0 {
		fmt.Println("  Excluded clauses:")
		for _, c := range m.ExcludedClauses {
			fmt.Printf("    - %s\n", c)
		}
	}
	if len(m.Additions) > 0 {
		fmt.Println("  Additions:")
		for _, c := range m.Additions {
			fmt.Printf("    + %s\n", c)
		}
	}
	if m.Rationale != "" {
		fmt.Printf("  Rationale: %s\n", m.Rationale)
	}
}

// --- Challenge Review commands ---

func cmdChallengeList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("challenge-list", flag.ContinueOnError)
	memberID := fs.String("member", "", "Filter to pending decisions authored by this member")
	decisionID := fs.String("decision", "", "Filter to one decision id")
	includeClosed := fs.Bool("include-closed", false, "Include resolved, stale, and overridden challenges")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team challenge-list <team-id> [--member=<member-id> | --decision=<decision-id>] [--include-closed] [--json]")
	}
	if *memberID != "" && *decisionID != "" {
		return fmt.Errorf("--member and --decision are mutually exclusive")
	}
	teamID := fs.Arg(0)

	var decisionsResp DecisionListResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/decisions?status=pending&last=0", teamID), &decisionsResp); err != nil {
		return fmt.Errorf("failed to get pending decisions: %w", err)
	}

	ownedContexts := map[string]struct{}{}
	if *memberID != "" {
		var team Team
		if err := ctx.Get(fmt.Sprintf("/teams/%s", teamID), &team); err == nil && team.OperatingContract != nil {
			if member, ok := team.OperatingContract.Members[*memberID]; ok {
				for _, contextID := range member.OwnedDecisionContexts {
					if strings.TrimSpace(contextID) != "" {
						ownedContexts[contextID] = struct{}{}
					}
				}
			}
		}
	}

	entries := make([]ChallengeListEntry, 0)
	for _, decision := range decisionsResp.Entries {
		if *decisionID != "" && decision.ID != *decisionID {
			continue
		}
		if *memberID != "" && decision.By != *memberID && !mapContains(ownedContexts, decision.Context) {
			continue
		}

		var reports KnowledgeListResponse
		reportQuery := fmt.Sprintf("/teams/%s/knowledge?topic=%s&last=0", teamID, url.QueryEscape("challenge-report/"+decision.ID))
		if err := ctx.Get(reportQuery, &reports); err != nil {
			return fmt.Errorf("failed to get challenge reports for %s: %w", decision.ID, err)
		}
		if len(reports.Entries) == 0 {
			continue
		}

		var resolutions KnowledgeListResponse
		resolutionQuery := fmt.Sprintf("/teams/%s/knowledge?topic=%s&last=0", teamID, url.QueryEscape("challenge-resolution-record/"+decision.ID))
		if err := ctx.Get(resolutionQuery, &resolutions); err != nil {
			return fmt.Errorf("failed to get challenge resolution records for %s: %w", decision.ID, err)
		}

		status := "open"
		if len(resolutions.Entries) > 0 {
			status = challengeStatusFromContent(resolutions.Entries[len(resolutions.Entries)-1].Content)
			if status == "" {
				status = "open"
			}
		}
		if !*includeClosed && challengeStatusIsClosed(status) {
			continue
		}

		entries = append(entries, ChallengeListEntry{
			Decision:    decision,
			Reports:     reports.Entries,
			Resolutions: resolutions.Entries,
			Status:      status,
		})
	}

	if *jsonOut {
		resp := struct {
			TeamID  string               `json:"teamId"`
			Entries []ChallengeListEntry `json:"entries"`
		}{TeamID: teamID, Entries: entries}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(entries) == 0 {
		fmt.Println("No matching challenge reports found")
		return nil
	}

	for _, entry := range entries {
		fmt.Printf("--- %s by %s [%s] challenge status: %s ---\n", entry.Decision.ID, entry.Decision.By, entry.Decision.Context, entry.Status)
		fmt.Printf("Decision: %s\n", truncate(firstNonEmpty(entry.Decision.Decision, entry.Decision.Topic), 180))
		latestReport := entry.Reports[len(entry.Reports)-1]
		fmt.Printf("Latest report: %s [%s]\n", latestReport.ID, latestReport.Topic)
		fmt.Printf("%s\n", truncate(firstCLIContentLine(latestReport.Content), 220))
		if len(entry.Resolutions) > 0 {
			latestResolution := entry.Resolutions[len(entry.Resolutions)-1]
			fmt.Printf("Latest resolution: %s [%s]\n", latestResolution.ID, latestResolution.Topic)
			fmt.Printf("%s\n", truncate(firstCLIContentLine(latestResolution.Content), 220))
		} else {
			fmt.Println("Latest resolution: none")
		}
		fmt.Printf("Inspect: prompt-manager team knowledge-list %s --topic challenge-report/%s\n", teamID, entry.Decision.ID)
		fmt.Printf("Resolve: prompt-manager team knowledge-add %s --topic challenge-resolution-record/%s --content '<yaml>'\n\n", teamID, entry.Decision.ID)
	}

	return nil
}

func cmdChallengeResolve(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("challenge-resolve", flag.ContinueOnError)
	status := fs.String("status", "", "Resolution status (open|author-responded|resolved|escalated|overridden|stale)")
	challengeReport := fs.String("challenge-report", "", "Knowledge id of the challenge report being addressed")
	responseKind := fs.String("response-kind", "none", "Author response kind (none|revised|superseded|accepted|disagreed|no-response)")
	responseRef := fs.String("response-ref", "", "Decision, knowledge, or notes reference for the author response")
	dispositionKind := fs.String("disposition", "pending", "Contrarian disposition (pending|resolved|escalate-rejection|escalate-framework|operator-call|watch)")
	notes := fs.String("notes", "", "Short rationale")
	supersedes := fs.String("supersedes", "", "Prior challenge-resolution-record knowledge id superseded by this entry")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 || strings.TrimSpace(*status) == "" {
		return fmt.Errorf("usage: team challenge-resolve <team-id> <decision-id> --status=<status> [--challenge-report=<knowledge-id>] [--response-kind=<kind>] [--response-ref=<ref>] [--disposition=<kind>] [--notes=<text>] [--supersedes=<knowledge-id>] [--json]")
	}
	teamID := fs.Arg(0)
	decisionID := fs.Arg(1)
	normalizedStatus := strings.TrimSpace(*status)
	if !validChallengeStatus(normalizedStatus) {
		return fmt.Errorf("invalid --status %q", normalizedStatus)
	}

	content := buildChallengeResolutionContent(decisionID, *challengeReport, normalizedStatus, *responseKind, *responseRef, *dispositionKind, *notes)
	req := AddKnowledgeRequest{
		Topic:      "challenge-resolution-record/" + decisionID,
		Content:    content,
		CallerNote: "challenge-resolution",
		Supersedes: strings.TrimSpace(*supersedes),
	}

	var resp KnowledgeEntry
	if err := ctx.Post(fmt.Sprintf("/teams/%s/knowledge", teamID), req, &resp); err != nil {
		return fmt.Errorf("failed to add challenge resolution: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Added challenge resolution %s [%s] status %s\n", resp.ID, resp.Topic, normalizedStatus)
	return nil
}

func buildChallengeResolutionContent(decisionID, challengeReport, status, responseKind, responseRef, dispositionKind, notes string) string {
	var b strings.Builder
	b.WriteString("target_decision: " + strings.TrimSpace(decisionID) + "\n")
	b.WriteString("challenge_report: " + strings.TrimSpace(challengeReport) + "\n")
	b.WriteString("status: " + strings.TrimSpace(status) + "\n")
	b.WriteString("author_response:\n")
	b.WriteString("  kind: " + strings.TrimSpace(firstNonEmpty(responseKind, "none")) + "\n")
	b.WriteString("  reference: " + strings.TrimSpace(responseRef) + "\n")
	b.WriteString("contrarian_disposition:\n")
	b.WriteString("  kind: " + strings.TrimSpace(firstNonEmpty(dispositionKind, "pending")) + "\n")
	b.WriteString("updated_at: " + time.Now().UTC().Format(time.RFC3339) + "\n")
	if strings.TrimSpace(notes) != "" {
		b.WriteString("notes: " + strings.TrimSpace(notes) + "\n")
	}
	return b.String()
}

func validChallengeStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open", "author-responded", "resolved", "escalated", "overridden", "stale":
		return true
	default:
		return false
	}
}

func mapContains(values map[string]struct{}, key string) bool {
	_, ok := values[strings.TrimSpace(key)]
	return ok
}

// --- Knowledge Log commands ---

func cmdKnowledgeAdd(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("knowledge-add", flag.ContinueOnError)
	topic := fs.String("topic", "", "Topic/category tag (required)")
	content := fs.String("content", "", "The knowledge content (required)")
	source := fs.String("source", "", "Where this was learned from")
	supersedes := fs.String("supersedes", "", "ID of knowledge entry this replaces")
	callerNote := fs.String("caller-note", "", "Optional freeform context (debug breadcrumb, retry note). Does not carry identity — attribution is auto-derived from the runtime context. Capped at 256 chars by the API.")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	// --by is removed; identity now flows over the
	// X-Vrooli-Attribution header (canon:
	// docs/agent-system/RUNTIME_ATTRIBUTION.md). Defining the flag
	// lets us emit a clean migration message instead of "flag
	// provided but not defined".
	by := fs.String("by", "", "[removed] use --caller-note for freeform context; identity is auto-attributed")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*by) != "" {
		return fmt.Errorf("--by is removed; identity is auto-attributed from the runtime context. Use --caller-note for freeform notes. See docs/agent-system/RUNTIME_ATTRIBUTION.md")
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team knowledge-add <team-id> --topic=\"...\" --content=\"...\" [--caller-note=\"...\"]")
	}
	teamID := fs.Arg(0)

	if strings.TrimSpace(*topic) == "" {
		return fmt.Errorf("topic is required")
	}
	if strings.TrimSpace(*content) == "" {
		return fmt.Errorf("content is required")
	}

	req := AddKnowledgeRequest{
		Topic:      *topic,
		Content:    *content,
		CallerNote: *callerNote,
		Source:     *source,
		Supersedes: *supersedes,
	}

	var resp KnowledgeEntry
	if err := ctx.Post(fmt.Sprintf("/teams/%s/knowledge", teamID), req, &resp); err != nil {
		return fmt.Errorf("failed to add knowledge: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Added knowledge %s [%s]: %s\n", resp.ID, resp.Topic, truncate(resp.Content, 80))
	return nil
}

func cmdKnowledgeList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("knowledge-list", flag.ContinueOnError)
	topic := fs.String("topic", "", "Filter by exact topic match")
	topicPrefix := fs.String("topic-prefix", "", "Filter by topic prefix (e.g. 'research-inbox/')")
	last := fs.Int("last", 20, "Number of entries to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team knowledge-list <team-id> [--topic=<tag> | --topic-prefix=<prefix>] [--last=N] [--json]")
	}
	if *topic != "" && *topicPrefix != "" {
		return fmt.Errorf("--topic and --topic-prefix are mutually exclusive")
	}
	teamID := fs.Arg(0)

	query := fmt.Sprintf("/teams/%s/knowledge?last=%d", teamID, *last)
	if *topic != "" {
		query += "&topic=" + url.QueryEscape(*topic)
	}
	if *topicPrefix != "" {
		query += "&topic_prefix=" + url.QueryEscape(*topicPrefix)
	}

	var resp KnowledgeListResponse
	if err := ctx.Get(query, &resp); err != nil {
		return fmt.Errorf("failed to get knowledge: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Entries) == 0 {
		fmt.Println("No knowledge entries found")
		return nil
	}

	for _, entry := range resp.Entries {
		supersededStr := ""
		if entry.Supersedes != "" {
			supersededStr = fmt.Sprintf(" (supersedes %s)", entry.Supersedes)
		}
		sourceStr := ""
		if entry.Source != "" {
			sourceStr = fmt.Sprintf(" (source: %s)", entry.Source)
		}
		fmt.Printf("--- %s [%s] by %s%s%s ---\n", entry.ID, entry.Topic, entry.Caller, supersededStr, sourceStr)
		fmt.Printf("%s\n\n", entry.Content)
	}
	return nil
}

func cmdKnowledgeUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("knowledge-update", flag.ContinueOnError)
	topic := fs.String("topic", "", "Update topic")
	content := fs.String("content", "", "Update content")
	source := fs.String("source", "", "Update source")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team knowledge-update <team-id> <knowledge-id> [--topic=...] [--content=...] [--source=...]")
	}
	teamID := fs.Arg(0)
	knowledgeID := fs.Arg(1)

	type updateReq struct {
		Topic   *string `json:"topic,omitempty"`
		Content *string `json:"content,omitempty"`
		Source  *string `json:"source,omitempty"`
	}
	req := updateReq{}
	if *topic != "" {
		req.Topic = topic
	}
	if *content != "" {
		req.Content = content
	}
	if *source != "" {
		req.Source = source
	}

	var resp KnowledgeEntry
	if err := ctx.Put(fmt.Sprintf("/teams/%s/knowledge/%s", teamID, knowledgeID), req, &resp); err != nil {
		return fmt.Errorf("failed to update knowledge: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Updated knowledge %s [%s]\n", resp.ID, resp.Topic)
	return nil
}

func cmdKnowledgeDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("knowledge-delete", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team knowledge-delete <team-id> <knowledge-id>")
	}
	teamID := fs.Arg(0)
	knowledgeID := fs.Arg(1)

	if err := ctx.Delete(fmt.Sprintf("/teams/%s/knowledge/%s", teamID, knowledgeID)); err != nil {
		return fmt.Errorf("failed to delete knowledge: %w", err)
	}

	fmt.Printf("Deleted knowledge %s\n", knowledgeID)
	return nil
}

// --- Retention / Prune CLI ---

// RetentionConfig mirrors the API response for retention settings.
type RetentionConfig struct {
	Tasks     *TaskRetention  `json:"tasks,omitempty"`
	Decisions *EntryRetention `json:"decisions,omitempty"`
	Knowledge *EntryRetention `json:"knowledge,omitempty"`
}

// TaskRetention mirrors the API task retention settings.
type TaskRetention struct {
	MaxCompleted int `json:"maxCompleted"`
	MaxAgeDays   int `json:"maxAgeDays"`
}

// EntryRetention mirrors the API entry retention settings.
type EntryRetention struct {
	MaxEntries int `json:"maxEntries"`
	MaxAgeDays int `json:"maxAgeDays"`
}

// PruneResult mirrors the API prune result.
type PruneResult struct {
	TasksRemoved     int `json:"tasksRemoved"`
	DecisionsRemoved int `json:"decisionsRemoved"`
	KnowledgeRemoved int `json:"knowledgeRemoved"`
}

func cmdRetention(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("retention", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team retention <team-id> [--json]")
	}
	teamID := fs.Arg(0)

	var cfg RetentionConfig
	if err := ctx.Get(fmt.Sprintf("/teams/%s/retention", teamID), &cfg); err != nil {
		return fmt.Errorf("failed to get retention config: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	}

	fmt.Printf("Retention Config for %s\n\n", teamID)
	if cfg.Tasks != nil {
		fmt.Printf("  Tasks:\n")
		fmt.Printf("    Max completed:  %s\n", fmtLimit(cfg.Tasks.MaxCompleted))
		fmt.Printf("    Max age (days): %s\n", fmtLimit(cfg.Tasks.MaxAgeDays))
	}
	if cfg.Decisions != nil {
		fmt.Printf("  Decisions:\n")
		fmt.Printf("    Max entries:    %s\n", fmtLimit(cfg.Decisions.MaxEntries))
		fmt.Printf("    Max age (days): %s\n", fmtLimit(cfg.Decisions.MaxAgeDays))
	}
	if cfg.Knowledge != nil {
		fmt.Printf("  Knowledge:\n")
		fmt.Printf("    Max entries:    %s\n", fmtLimit(cfg.Knowledge.MaxEntries))
		fmt.Printf("    Max age (days): %s\n", fmtLimit(cfg.Knowledge.MaxAgeDays))
	}
	return nil
}

func cmdPrune(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team prune <team-id> [--json]")
	}
	teamID := fs.Arg(0)

	var result PruneResult
	if err := ctx.Post(fmt.Sprintf("/teams/%s/prune", teamID), nil, &result); err != nil {
		return fmt.Errorf("failed to prune: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	total := result.TasksRemoved + result.DecisionsRemoved + result.KnowledgeRemoved
	if total == 0 {
		fmt.Println("Nothing to prune.")
		return nil
	}

	fmt.Printf("Pruned %d items:\n", total)
	if result.TasksRemoved > 0 {
		fmt.Printf("  Tasks removed:     %d\n", result.TasksRemoved)
	}
	if result.DecisionsRemoved > 0 {
		fmt.Printf("  Decisions removed: %d\n", result.DecisionsRemoved)
	}
	if result.KnowledgeRemoved > 0 {
		fmt.Printf("  Knowledge removed: %d\n", result.KnowledgeRemoved)
	}
	return nil
}

func fmtLimit(v int) string {
	if v == 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", v)
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func firstCLIContentLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

func challengeStatusFromContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "status:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "status:")), `"'`)
		}
		if strings.Contains(line, `"status"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(strings.TrimRight(parts[1], ",}")), `"'`)
			}
		}
	}
	return ""
}

func challengeStatusIsClosed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "overridden", "stale":
		return true
	default:
		return false
	}
}
