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
	"strconv"
	"strings"
	"time"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/attribution"
	"prompt-manager/internal/teamconfig"
	"prompt-manager/internal/teamcontract"

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

// AttributionInfo mirrors the API-side attribution object for read output.
type AttributionInfo struct {
	Kind          string  `json:"kind"`
	MemberID      *string `json:"member_id"`
	TeamID        *string `json:"team_id"`
	RunID         *string `json:"run_id"`
	SpawnOrigin   string  `json:"spawn_origin"`
	SourceSkillID *string `json:"source_skill_id"`
}

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

type KnowledgeListResponse struct {
	TeamID  string           `json:"teamId"`
	Entries []KnowledgeEntry `json:"entries"`
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
				Description: "Manage teams (list|show|create|update|delete|knowledge-*|bug-*|friction-*|add-member|heartbeat-*|retention|prune|import-cc|export-cc|trigger)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
			{
				Name:        "heartbeat-control",
				Aliases:     []string{"heartbeats-control"},
				NeedsAPI:    true,
				Description: "Manage global heartbeat auto-pause status, policy, pause, and resume",
				Run: func(args []string) error {
					return cmdHeartbeatControl(ctx, args)
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
	case "heartbeat-fleet-health":
		return cmdHeartbeatFleetHealth(ctx, subArgs)
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
	case "heartbeat-control":
		return cmdTeamHeartbeatControl(ctx, subArgs)
	case "queue-clear":
		return cmdQueueClear(ctx, subArgs)
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
	case "knowledge-add":
		return cmdKnowledgeAdd(ctx, subArgs)
	case "knowledge-list":
		return cmdKnowledgeList(ctx, subArgs)
	case "knowledge-update":
		return cmdKnowledgeUpdate(ctx, subArgs)
	case "knowledge-delete":
		return cmdKnowledgeDelete(ctx, subArgs)
	case "bug-capture":
		return cmdBugCapture(ctx, subArgs)
	case "bug-repair":
		return cmdBugRepair(ctx, subArgs)
	case "friction-capture":
		return cmdFrictionCapture(ctx, subArgs)
	case "friction-repair":
		return cmdFrictionRepair(ctx, subArgs)
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
  heartbeat-fleet-health                      Count durable work products, blocked runs, and failures in the last 24 hours
  heartbeat <team-id> <agent-id>              Show heartbeat config
  heartbeat-enable <team-id> <agent-id>       Enable heartbeat with schedule
  heartbeat-disable <team-id> <agent-id>      Disable heartbeat
  heartbeat-trigger <team-id> <agent-id>      Manually trigger heartbeat
  heartbeat-logs <team-id> <agent-id>         List execution logs
  heartbeat-control <team-id> <action>        Status, pause, resume, or set team auto-pause policy
  queue-clear <team-id> <agent-id>            Clear a stuck running entry from the team queue

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

Knowledge Log Commands:
  knowledge-add <team-id>               Add a knowledge entry
  knowledge-list <team-id>              List knowledge entries
  knowledge-update <team-id> <id>       Update a knowledge entry
  knowledge-delete <team-id> <id>       Delete a knowledge entry
  bug-capture <team-id>                 Capture a Scenario QA bug (publishes or saves a private draft)
  bug-repair <team-id> <draft-id>       Repair a private bug draft; publishes when complete

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
	configFlags := registerTeamConfigFlags(fs, true)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team create <name> [--mission=...] [--runtime-mode=...] [--coordination-pattern=...]")
	}
	name := fs.Arg(0)

	runtime, coordination, execution, err := resolveCreateTeamConfig(configFlags)
	if err != nil {
		return err
	}

	req := CreateTeamRequest{
		DisplayName:       name,
		Mission:           *mission,
		Runtime:           runtime,
		Coordination:      coordination,
		Execution:         execution,
		OperatingContract: teamcontract.Minimal(""),
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
	configFlags := registerTeamConfigFlags(fs, false)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team update <id> [--name=...] [--mission=...] [--enabled=true|false] [--runtime-mode=...] [--coordination-pattern=...]")
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
	jsonOut := fs.Bool("json", false, "Output as JSON")
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
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"deleted": true, "id": teamID, "displayName": team.DisplayName})
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
	jsonOut := fs.Bool("json", false, "Output as JSON")
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
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"removed": true, "teamId": teamID, "agentId": agentID})
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
	TeamID                  string               `json:"teamId"`
	AgentID                 string               `json:"agentId"`
	Enabled                 bool                 `json:"enabled"`
	Schedule                string               `json:"schedule"`
	ProfileKey              string               `json:"profileKey,omitempty"`
	TimeoutSeconds          int                  `json:"timeoutSeconds,omitempty"`
	ConsecutiveFailures     int                  `json:"consecutiveFailures"`
	LifecycleState          string               `json:"lifecycleState"`
	LastExecution           *HeartbeatExecResult `json:"lastExecution,omitempty"`
	LastSuccessfulExecution *HeartbeatExecResult `json:"lastSuccessfulExecution,omitempty"`
	NextExecution           string               `json:"nextExecution,omitempty"`
	CreatedAt               string               `json:"createdAt"`
	UpdatedAt               string               `json:"updatedAt"`
}

type HeartbeatFleetHealth struct {
	GeneratedAt            string  `json:"generatedAt"`
	WindowHours            int     `json:"windowHours"`
	EnabledMembers         int     `json:"enabledMembers"`
	SucceededMembers       int     `json:"succeededMembers"`
	ProducedMembers        int     `json:"producedMembers"`
	BlockedMembers         int     `json:"blockedMembers"`
	FailedMembers          int     `json:"failedMembers"`
	MembersWithTwoFailures int     `json:"membersWithTwoFailures"`
	SuccessPercent         float64 `json:"successPercent"`
	ThresholdPercent       int     `json:"thresholdPercent"`
	MeetsThreshold         bool    `json:"meetsThreshold"`
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

type HeartbeatControlPolicy struct {
	Enabled                                bool   `json:"enabled"`
	PauseAfterDaysWithoutHumanEngagement   int    `json:"pauseAfterDaysWithoutHumanEngagement"`
	WarningAfterDaysWithoutHumanEngagement int    `json:"warningAfterDaysWithoutHumanEngagement"`
	ResumeMode                             string `json:"resumeMode"`
}

type HeartbeatControlTeamOverride struct {
	Mode                                   string `json:"mode"`
	PauseAfterDaysWithoutHumanEngagement   *int   `json:"pauseAfterDaysWithoutHumanEngagement,omitempty"`
	WarningAfterDaysWithoutHumanEngagement *int   `json:"warningAfterDaysWithoutHumanEngagement,omitempty"`
	ResumeMode                             string `json:"resumeMode,omitempty"`
}

type HeartbeatControlStatus struct {
	Scope                     string                        `json:"scope"`
	TeamID                    string                        `json:"teamId,omitempty"`
	Status                    string                        `json:"status"`
	EffectivePolicy           HeartbeatControlPolicy        `json:"effectivePolicy"`
	GlobalPolicy              HeartbeatControlPolicy        `json:"globalPolicy,omitempty"`
	TeamOverride              *HeartbeatControlTeamOverride `json:"teamOverride,omitempty"`
	LastHumanEngagementAt     *string                       `json:"lastHumanEngagementAt,omitempty"`
	LastHumanEngagementReason string                        `json:"lastHumanEngagementReason,omitempty"`
	LastHumanEngagementTeamID string                        `json:"lastHumanEngagementTeamId,omitempty"`
	PausedAt                  *string                       `json:"pausedAt,omitempty"`
	PausedReason              string                        `json:"pausedReason,omitempty"`
	WarningAt                 *string                       `json:"warningAt,omitempty"`
	AutoPauseAt               *string                       `json:"autoPauseAt,omitempty"`
	ResumeHint                string                        `json:"resumeHint,omitempty"`
	Teams                     []HeartbeatControlStatus      `json:"teams,omitempty"`
}

type HeartbeatControlPolicyRequest struct {
	Enabled                                *bool   `json:"enabled,omitempty"`
	PauseAfterDaysWithoutHumanEngagement   *int    `json:"pauseAfterDaysWithoutHumanEngagement,omitempty"`
	WarningAfterDaysWithoutHumanEngagement *int    `json:"warningAfterDaysWithoutHumanEngagement,omitempty"`
	ResumeMode                             *string `json:"resumeMode,omitempty"`
}

type HeartbeatControlTeamPolicyRequest struct {
	Mode                                   *string `json:"mode,omitempty"`
	PauseAfterDaysWithoutHumanEngagement   *int    `json:"pauseAfterDaysWithoutHumanEngagement,omitempty"`
	WarningAfterDaysWithoutHumanEngagement *int    `json:"warningAfterDaysWithoutHumanEngagement,omitempty"`
	ResumeMode                             *string `json:"resumeMode,omitempty"`
}

type HeartbeatControlPauseRequest struct {
	Reason string `json:"reason,omitempty"`
}

func cmdHeartbeatControl(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: heartbeat-control <status|pause|resume|policy> [args]")
	}
	switch args[0] {
	case "status":
		return cmdHeartbeatControlStatus(ctx, args[1:], "")
	case "pause":
		return cmdHeartbeatControlPause(ctx, args[1:], "")
	case "resume":
		return cmdHeartbeatControlResume(ctx, args[1:], "")
	case "policy":
		return cmdHeartbeatControlPolicy(ctx, args[1:], "")
	default:
		return fmt.Errorf("unknown heartbeat-control action %q", args[0])
	}
}

func cmdTeamHeartbeatControl(ctx appctx.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: team heartbeat-control <team-id> <status|pause|resume|policy> [args]")
	}
	teamID := args[0]
	switch args[1] {
	case "status":
		return cmdHeartbeatControlStatus(ctx, args[2:], teamID)
	case "pause":
		return cmdHeartbeatControlPause(ctx, args[2:], teamID)
	case "resume":
		return cmdHeartbeatControlResume(ctx, args[2:], teamID)
	case "policy":
		return cmdHeartbeatControlPolicy(ctx, args[2:], teamID)
	default:
		return fmt.Errorf("unknown team heartbeat-control action %q", args[1])
	}
}

func cmdHeartbeatControlStatus(ctx appctx.Context, args []string, teamID string) error {
	fs := flag.NewFlagSet("heartbeat-control status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp HeartbeatControlStatus
	path := "/heartbeats/control"
	if teamID != "" {
		path = fmt.Sprintf("/teams/%s/heartbeats/control", teamID)
	}
	if err := ctx.Get(path, &resp); err != nil {
		return fmt.Errorf("failed to get heartbeat control status: %w", err)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	printHeartbeatControlStatus(resp)
	return nil
}

func cmdHeartbeatControlPause(ctx appctx.Context, args []string, teamID string) error {
	fs := flag.NewFlagSet("heartbeat-control pause", flag.ContinueOnError)
	reason := fs.String("reason", "", "Pause reason")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp HeartbeatControlStatus
	path := "/heartbeats/control/pause"
	if teamID != "" {
		path = fmt.Sprintf("/teams/%s/heartbeats/control/pause", teamID)
	}
	req := HeartbeatControlPauseRequest{Reason: *reason}
	if err := ctx.Post(path, req, &resp); err != nil {
		return fmt.Errorf("failed to pause heartbeats: %w", err)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	fmt.Printf("Heartbeats paused (%s)\n", resp.Status)
	if resp.ResumeHint != "" {
		fmt.Println(resp.ResumeHint)
	}
	return nil
}

func cmdHeartbeatControlResume(ctx appctx.Context, args []string, teamID string) error {
	fs := flag.NewFlagSet("heartbeat-control resume", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp HeartbeatControlStatus
	path := "/heartbeats/control/resume"
	if teamID != "" {
		path = fmt.Sprintf("/teams/%s/heartbeats/control/resume", teamID)
	}
	if err := ctx.Post(path, nil, &resp); err != nil {
		return fmt.Errorf("failed to resume heartbeats: %w", err)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	fmt.Printf("Heartbeats resumed (%s)\n", resp.Status)
	if resp.AutoPauseAt != nil {
		fmt.Printf("Next auto-pause check: %s\n", *resp.AutoPauseAt)
	}
	return nil
}

func cmdHeartbeatControlPolicy(ctx appctx.Context, args []string, teamID string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: heartbeat-control policy <show|set> [args]")
	}
	switch args[0] {
	case "show":
		return cmdHeartbeatControlStatus(ctx, args[1:], teamID)
	case "set":
		return cmdHeartbeatControlPolicySet(ctx, args[1:], teamID)
	default:
		return fmt.Errorf("unknown heartbeat-control policy action %q", args[0])
	}
}

func cmdHeartbeatControlPolicySet(ctx appctx.Context, args []string, teamID string) error {
	fs := flag.NewFlagSet("heartbeat-control policy set", flag.ContinueOnError)
	enabledRaw := fs.String("enabled", "", "Enable global auto-pause (true|false)")
	modeRaw := fs.String("mode", "", "Team override mode (inherit|disabled|custom)")
	pauseAfterRaw := fs.String("pause-after", "", "Days before auto-pause, e.g. 14d")
	warningAfterRaw := fs.String("warning-after", "", "Days before warning, e.g. 10d")
	resumeModeRaw := fs.String("resume-mode", "", "Resume mode (manual)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp HeartbeatControlStatus
	if teamID == "" {
		req := HeartbeatControlPolicyRequest{}
		if strings.TrimSpace(*enabledRaw) != "" {
			v, err := strconv.ParseBool(strings.TrimSpace(*enabledRaw))
			if err != nil {
				return fmt.Errorf("invalid --enabled: %w", err)
			}
			req.Enabled = &v
		}
		if strings.TrimSpace(*pauseAfterRaw) != "" {
			v, err := parseDaysFlag(*pauseAfterRaw)
			if err != nil {
				return err
			}
			req.PauseAfterDaysWithoutHumanEngagement = &v
		}
		if strings.TrimSpace(*warningAfterRaw) != "" {
			v, err := parseDaysFlag(*warningAfterRaw)
			if err != nil {
				return err
			}
			req.WarningAfterDaysWithoutHumanEngagement = &v
		}
		if strings.TrimSpace(*resumeModeRaw) != "" {
			v := strings.TrimSpace(*resumeModeRaw)
			req.ResumeMode = &v
		}
		if err := ctx.Put("/heartbeats/control/policy", req, &resp); err != nil {
			return fmt.Errorf("failed to update heartbeat control policy: %w", err)
		}
	} else {
		req := HeartbeatControlTeamPolicyRequest{}
		if strings.TrimSpace(*modeRaw) != "" {
			v := strings.TrimSpace(*modeRaw)
			req.Mode = &v
		}
		if strings.TrimSpace(*pauseAfterRaw) != "" {
			v, err := parseDaysFlag(*pauseAfterRaw)
			if err != nil {
				return err
			}
			req.PauseAfterDaysWithoutHumanEngagement = &v
		}
		if strings.TrimSpace(*warningAfterRaw) != "" {
			v, err := parseDaysFlag(*warningAfterRaw)
			if err != nil {
				return err
			}
			req.WarningAfterDaysWithoutHumanEngagement = &v
		}
		if strings.TrimSpace(*resumeModeRaw) != "" {
			v := strings.TrimSpace(*resumeModeRaw)
			req.ResumeMode = &v
		}
		if err := ctx.Put(fmt.Sprintf("/teams/%s/heartbeats/control/policy", teamID), req, &resp); err != nil {
			return fmt.Errorf("failed to update team heartbeat control policy: %w", err)
		}
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	fmt.Println("Heartbeat auto-pause policy updated")
	printHeartbeatControlStatus(resp)
	return nil
}

func parseDaysFlag(raw string) (int, error) {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "d"))
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("invalid day duration %q", raw)
	}
	return v, nil
}

func printHeartbeatControlStatus(resp HeartbeatControlStatus) {
	label := "Global"
	if resp.TeamID != "" {
		label = "Team " + resp.TeamID
	}
	fmt.Printf("%s heartbeat control: %s\n", label, resp.Status)
	if resp.LastHumanEngagementAt != nil {
		fmt.Printf("Last human engagement: %s", *resp.LastHumanEngagementAt)
		if resp.LastHumanEngagementReason != "" {
			fmt.Printf(" (%s)", resp.LastHumanEngagementReason)
		}
		fmt.Println()
	}
	fmt.Printf("Auto-pause: enabled=%v warning=%dd pause=%dd resume=%s\n",
		resp.EffectivePolicy.Enabled,
		resp.EffectivePolicy.WarningAfterDaysWithoutHumanEngagement,
		resp.EffectivePolicy.PauseAfterDaysWithoutHumanEngagement,
		resp.EffectivePolicy.ResumeMode,
	)
	if resp.PausedAt != nil {
		fmt.Printf("Paused at: %s\n", *resp.PausedAt)
	}
	if resp.PausedReason != "" {
		fmt.Printf("Paused reason: %s\n", resp.PausedReason)
	}
	if resp.WarningAt != nil {
		fmt.Printf("Warning at: %s\n", *resp.WarningAt)
	}
	if resp.AutoPauseAt != nil {
		fmt.Printf("Auto-pause at: %s\n", *resp.AutoPauseAt)
	}
	if resp.ResumeHint != "" && strings.HasPrefix(resp.Status, "paused") {
		fmt.Println(resp.ResumeHint)
	}
	if len(resp.Teams) > 0 {
		fmt.Println("Teams:")
		for _, team := range resp.Teams {
			fmt.Printf("  %s: %s", team.TeamID, team.Status)
			if team.Scope == "global" && team.Status != "active" {
				fmt.Print(" (global)")
			}
			fmt.Println()
		}
	}
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
		fmt.Printf("  %s: %s [%s] - lifecycle: %s, failures: %d, last: %s\n", c.AgentID, c.Schedule, status, c.LifecycleState, c.ConsecutiveFailures, lastRun)
	}
	return nil
}

func cmdHeartbeatFleetHealth(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-fleet-health", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: team heartbeat-fleet-health [--json]")
	}

	var teams []Team
	if err := ctx.Get("/teams", &teams); err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}
	now := time.Now().UTC()
	summary := HeartbeatFleetHealth{GeneratedAt: now.Format(time.RFC3339), WindowHours: 24, ThresholdPercent: 90}
	cutoff := now.Add(-24 * time.Hour)
	for _, team := range teams {
		if !team.Enabled {
			continue
		}
		var configs []HeartbeatConfig
		if err := ctx.Get(fmt.Sprintf("/teams/%s/heartbeats", team.ID), &configs); err != nil {
			return fmt.Errorf("failed to list heartbeats for %s: %w", team.ID, err)
		}
		for _, config := range configs {
			if !config.Enabled {
				continue
			}
			summary.EnabledMembers++
			if config.ConsecutiveFailures >= 2 {
				summary.MembersWithTwoFailures++
			}
			if heartbeatCompletedSince(config, cutoff) {
				summary.SucceededMembers++
			}
			switch heartbeatProductStatus(ctx, team.ID, config, cutoff) {
			case "produced":
				summary.ProducedMembers++
			case "failed":
				summary.FailedMembers++
			default:
				summary.BlockedMembers++
			}
		}
	}
	if summary.EnabledMembers > 0 {
		summary.SuccessPercent = 100 * float64(summary.ProducedMembers) / float64(summary.EnabledMembers)
		summary.MeetsThreshold = summary.ProducedMembers*100 >= summary.EnabledMembers*summary.ThresholdPercent
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}
	fmt.Printf("Fleet health: %d of %d enabled members produced a durable work product in the last %d hours\n", summary.ProducedMembers, summary.EnabledMembers, summary.WindowHours)
	fmt.Printf("Success: %.2f%% (threshold: %d%%, meets threshold: %t)\n", summary.SuccessPercent, summary.ThresholdPercent, summary.MeetsThreshold)
	fmt.Printf("Members at failure streak 2 or higher: %d\n", summary.MembersWithTwoFailures)
	return nil
}

func heartbeatProductStatus(ctx appctx.Context, teamID string, config HeartbeatConfig, cutoff time.Time) string {
	last := config.LastExecution
	if last == nil {
		return "blocked"
	}
	if last.Status == "running" && config.LastSuccessfulExecution != nil {
		last = config.LastSuccessfulExecution
	}
	ended, ok := heartbeatCompletionTime(last.StartedAt, last.EndedAt)
	if !ok || ended.Before(cutoff) {
		return "blocked"
	}
	if last.Status != "completed" {
		return "failed"
	}
	var response KnowledgeListResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/knowledge?last=100", teamID), &response); err != nil {
		return "blocked"
	}
	for _, entry := range response.Entries {
		if entry.At < cutoff.Format(time.RFC3339) {
			continue
		}
		if entry.Attribution.RunID != nil && strings.TrimSpace(*entry.Attribution.RunID) == strings.TrimSpace(last.RunID) {
			return "produced"
		}
	}
	if reader, ok := ctx.(interface {
		HasDurableProduct(string, string, time.Time) (bool, error)
	}); ok {
		produced, err := reader.HasDurableProduct(teamID, last.RunID, cutoff)
		if err == nil && produced {
			return "produced"
		}
	}
	return "blocked"
}

func heartbeatCompletedSince(config HeartbeatConfig, cutoff time.Time) bool {
	for _, execution := range []*HeartbeatExecResult{config.LastSuccessfulExecution, config.LastExecution} {
		if execution == nil || execution.Status != "completed" {
			continue
		}
		completedAt, ok := heartbeatCompletionTime(execution.StartedAt, execution.EndedAt)
		if ok && !completedAt.Before(cutoff) {
			return true
		}
	}
	return false
}

func heartbeatCompletionTime(startedAt, endedAt string) (time.Time, bool) {
	value := endedAt
	if value == "" {
		value = startedAt
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return parsed, err == nil
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

// teamQueueStatus mirrors the heartbeat.TeamExecutionStatus shape returned
// by GET /teams/{id}/execution-status and DELETE /teams/{id}/queue/running/{agentId}.
type teamQueueStatus struct {
	TeamID            string   `json:"teamId"`
	State             string   `json:"state"`
	RunningAgentIDs   []string `json:"runningAgentIds"`
	Queue             []string `json:"queue"`
	QueuePolicy       string   `json:"queuePolicy"`
	MaxConcurrentRuns int      `json:"maxConcurrentRuns"`
}

func cmdQueueClear(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("queue-clear", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	force := fs.Bool("force", false, "Clear even if the backing run is still active in agent-manager")
	yes := fs.Bool("yes", false, "Skip the confirmation prompt that --force triggers")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team queue-clear <team-id> <agent-id> [--force [--yes]]")
	}
	teamID := fs.Arg(0)
	agentID := fs.Arg(1)

	if *force && !*yes {
		fmt.Fprintf(os.Stderr, "--force will clear the running entry even if the agent-manager run is still active.\nPass --yes to confirm. Refusing for safety.\n")
		return fmt.Errorf("force without confirmation")
	}

	query := url.Values{}
	if *force {
		query.Set("force", "true")
	}

	var resp teamQueueStatus
	if err := ctx.DeleteWithQuery(fmt.Sprintf("/teams/%s/queue/running/%s", teamID, agentID), query, &resp); err != nil {
		return fmt.Errorf("failed to clear queue entry: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Cleared running entry %s/%s\n", teamID, agentID)
	fmt.Printf("Team state: %s\n", resp.State)
	if len(resp.RunningAgentIDs) > 0 {
		fmt.Printf("Still running: %v\n", resp.RunningAgentIDs)
	}
	if len(resp.Queue) > 0 {
		fmt.Printf("Queue: %v\n", resp.Queue)
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

type frictionCaptureRequest struct {
	Scope        string            `json:"scope"`
	Severity     string            `json:"severity"`
	Expected     string            `json:"expected"`
	Actual       string            `json:"actual"`
	Description  string            `json:"description"`
	Context      map[string]string `json:"context,omitempty"`
	HonestyFlags []string          `json:"honesty_flags,omitempty"`
}

func cmdFrictionCapture(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("friction-capture", flag.ContinueOnError)
	scope := fs.String("scope", "", "toolchain|run-execution|prompt-team-agent-storage|recurring-workaround|unknown")
	severity := fs.String("severity", "", "blocking|recurring|one-off")
	expected := fs.String("expected", "", "Expected behavior")
	actual := fs.String("actual", "", "Observed behavior")
	description := fs.String("description", "", "What happened and why this is structural friction")
	slug := fs.String("slug", "", "Short kebab-case report slug")
	honesty := fs.String("honesty-flags", "", "Comma-separated honesty flags")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 || fs.Arg(0) != "meta-optimization" {
		return fmt.Errorf("usage: team friction-capture meta-optimization --scope=... --severity=... --expected=... --actual=... --description=... --slug=...")
	}
	validScopes := map[string]bool{"toolchain": true, "run-execution": true, "prompt-team-agent-storage": true, "recurring-workaround": true, "unknown": true}
	validSeverities := map[string]bool{"blocking": true, "recurring": true, "one-off": true}
	if !validScopes[*scope] || !validSeverities[*severity] || strings.TrimSpace(*expected) == "" || strings.TrimSpace(*actual) == "" || strings.TrimSpace(*description) == "" || strings.TrimSpace(*slug) == "" {
		return fmt.Errorf("scope, severity, expected, actual, description, and slug are required; scope and severity must match the taxonomy")
	}
	content := fmt.Sprintf("---\nseverity: %s\nscope: %s\nreporter: operator\nreporter_team: operator\nobserved_at: %s\ncontext:\n  scenario: null\n  skill: report-friction\n  member: null\n  command: prompt-manager team friction-capture\n  doc: null\n  task: null\nexpected: %s\nactual: %s\ndescription: |\n  %s\nhonesty_flags: [%s]\n---\n", *severity, *scope, time.Now().UTC().Format("2006-01-02"), frictionYAMLScalar(*expected), frictionYAMLScalar(*actual), strings.ReplaceAll(strings.TrimSpace(*description), "\n", "\n  "), strings.Join(csvValues(*honesty), ", "))
	var entry KnowledgeEntry
	post := func() error {
		return ctx.Post("/teams/meta-optimization/knowledge", AddKnowledgeRequest{Topic: "friction-inbox/" + *scope + "/" + *slug, Content: content, Source: "prompt-manager friction capture", CallerNote: "filed via report-friction skill"}, &entry)
	}
	if err := attribution.WithWriterSkill("report-friction", "meta-optimization", post); err != nil {
		return fmt.Errorf("friction capture failed: %w", err)
	}
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(entry)
	}
	fmt.Printf("Filed friction %s\n", entry.ID)
	return nil
}

func frictionYAMLScalar(value string) string {
	encoded, _ := json.Marshal(strings.TrimSpace(value))
	return string(encoded)
}

func cmdFrictionRepair(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("friction-repair", flag.ContinueOnError)
	content := fs.String("content", "", "Replacement complete friction report body")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 || strings.TrimSpace(*content) == "" {
		return fmt.Errorf("usage: team friction-repair meta-optimization <knowledge-id> --content=...")
	}
	var entry KnowledgeEntry
	if err := ctx.Put(fmt.Sprintf("/teams/%s/knowledge/%s", fs.Arg(0), fs.Arg(1)), map[string]string{"content": *content}, &entry); err != nil {
		return fmt.Errorf("friction repair failed: %w", err)
	}
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(entry)
	}
	fmt.Printf("Repaired friction %s\n", entry.ID)
	return nil
}

// bugCaptureRequest mirrors heartbeat.BugCaptureRequest. Keeping the CLI's
// shape local avoids importing API packages while preserving the typed wire
// contract at the HTTP boundary.
type bugCaptureRequest struct {
	Title          string            `json:"title"`
	SignalType     string            `json:"signal_type"`
	Severity       string            `json:"severity"`
	Repro          []string          `json:"repro"`
	Expected       string            `json:"expected"`
	Actual         string            `json:"actual"`
	Description    string            `json:"description"`
	Context        map[string]string `json:"context,omitempty"`
	HonestyFlags   []string          `json:"honesty_flags"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type bugCaptureResponse struct {
	Disposition string               `json:"disposition"`
	DraftID     string               `json:"draft_id,omitempty"`
	Knowledge   *KnowledgeEntry      `json:"knowledge,omitempty"`
	Accepted    map[string]string    `json:"accepted"`
	Needs       []string             `json:"needs"`
	Invalid     []bugFieldDiagnostic `json:"invalid"`
	Warnings    []string             `json:"warnings"`
	NextAction  []string             `json:"next_action"`
}

type bugFieldDiagnostic struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

func cmdBugCapture(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("bug-capture", flag.ContinueOnError)
	flags := defineBugCaptureFlags(fs)
	jsonOut := fs.Bool("json", false, "Output the structured capture result as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("team bug-capture usage: scenario-qa --title=<title> --signal-type=<type> --severity=<severity> --repro=<step> --expected=<expected> --actual=<actual> --description=<description>")
	}
	teamID := fs.Arg(0)
	return attribution.WithWriterSkill("report-bug", teamID, func() error {
		return runBugCapture(ctx, false, fmt.Sprintf("/teams/%s/bugs/capture", teamID), flags.request(), *jsonOut)
	})
}

func cmdBugRepair(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("bug-repair", flag.ContinueOnError)
	flags := defineBugCaptureFlags(fs)
	jsonOut := fs.Bool("json", false, "Output the structured capture result as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team bug-repair scenario-qa <draft-id> [--signal-type=... --severity=... --expected=... --actual=...]")
	}
	teamID := fs.Arg(0)
	return attribution.WithWriterSkill("report-bug", teamID, func() error {
		return runBugCapture(ctx, true, fmt.Sprintf("/teams/%s/bugs/%s/capture", teamID, fs.Arg(1)), flags.request(), *jsonOut)
	})
}

type bugCaptureFlags struct{ title, signalType, severity, repro, expected, actual, description, scenario, skill, member, command, honestyFlags, idempotencyKey *string }

func defineBugCaptureFlags(fs *flag.FlagSet) bugCaptureFlags {
	return bugCaptureFlags{title: fs.String("title", "", "Short report title"), signalType: fs.String("signal-type", "", "code-defect|regression|prompt-confusion|data-shape-mismatch|unexpected-error|unknown"), severity: fs.String("severity", "", "blocker|major|minor"), repro: fs.String("repro", "", "Comma-separated reproduction steps"), expected: fs.String("expected", "", "Expected behavior"), actual: fs.String("actual", "", "Observed behavior"), description: fs.String("description", "", "Free-form report details"), scenario: fs.String("scenario", "", "Affected scenario (optional)"), skill: fs.String("skill", "", "Affected skill (optional)"), member: fs.String("member", "", "Affected member (optional)"), command: fs.String("command", "", "Affected command (optional)"), honestyFlags: fs.String("honesty-flags", "", "Comma-separated taxonomy honesty flags"), idempotencyKey: fs.String("idempotency-key", "", "Optional retry-safe capture key")}
}

func (f bugCaptureFlags) request() bugCaptureRequest {
	return bugCaptureRequest{Title: *f.title, SignalType: *f.signalType, Severity: *f.severity, Repro: csvValues(*f.repro), Expected: *f.expected, Actual: *f.actual, Description: *f.description, Context: compactContext(*f.scenario, *f.skill, *f.member, *f.command), HonestyFlags: csvValues(*f.honestyFlags), IdempotencyKey: *f.idempotencyKey}
}

func runBugCapture(ctx appctx.Context, repair bool, path string, req bugCaptureRequest, jsonOut bool) error {
	var resp bugCaptureResponse
	var err error
	if repair {
		err = ctx.Put(path, req, &resp)
	} else {
		err = ctx.Post(path, req, &resp)
	}
	if err != nil {
		return fmt.Errorf("bug capture failed: %w", err)
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	if resp.Disposition == "published" && resp.Knowledge != nil {
		fmt.Printf("Published Scenario QA bug %s [%s].\n", resp.Knowledge.ID, resp.Knowledge.Topic)
		return nil
	}
	fmt.Printf("Draft saved privately: %s (not published to bug-inbox).\n", resp.DraftID)
	if len(resp.Needs) > 0 {
		fmt.Printf("Needs: %s\n", strings.Join(resp.Needs, ", "))
	}
	for _, invalid := range resp.Invalid {
		fmt.Printf("Invalid %s: %s\n", invalid.Field, invalid.Message)
	}
	if len(resp.NextAction) > 0 {
		fmt.Printf("Repair: %s\n", strings.Join(resp.NextAction, " "))
	}
	return nil
}

func csvValues(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func compactContext(scenario, skill, member, command string) map[string]string {
	context := map[string]string{}
	for key, value := range map[string]string{"scenario": scenario, "skill": skill, "member": member, "command": command} {
		if strings.TrimSpace(value) != "" {
			context[key] = value
		}
	}
	return context
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
	Tasks *TaskRetention `json:"tasks,omitempty"`
}

// TaskRetention mirrors the API task retention settings.
type TaskRetention struct {
	MaxCompleted int `json:"maxCompleted"`
	MaxAgeDays   int `json:"maxAgeDays"`
}

// EntryRetention mirrors the API entry retention settings.
// PruneResult mirrors the API prune result.
type PruneResult struct {
	TasksRemoved int `json:"tasksRemoved"`
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

	total := result.TasksRemoved
	if total == 0 {
		fmt.Println("Nothing to prune.")
		return nil
	}

	fmt.Printf("Pruned %d items:\n", total)
	if result.TasksRemoved > 0 {
		fmt.Printf("  Tasks removed:     %d\n", result.TasksRemoved)
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
