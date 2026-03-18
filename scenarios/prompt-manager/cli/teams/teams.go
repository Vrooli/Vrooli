// Package teams provides CLI commands for team management.
//
// DOC: docs/reference/cli-commands.md#teams
package teams

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"prompt-manager/cli/internal/appctx"
)

// Team represents a team from the API (brief response)
type Team struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Mission     string `json:"mission,omitempty"`
	SpawnMode   string `json:"spawnMode,omitempty"`
	MemberCount int    `json:"memberCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
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

// DecisionEntry represents a decision log entry.
type DecisionEntry struct {
	ID         string `json:"id"`
	At         string `json:"at"`
	By         string `json:"by"`
	Decision   string `json:"decision"`
	Rationale  string `json:"rationale"`
	Context    string `json:"context,omitempty"`
	Supersedes string `json:"supersedes,omitempty"`
}

// DecisionListResponse represents the decision list API response.
type DecisionListResponse struct {
	TeamID  string          `json:"teamId"`
	Entries []DecisionEntry `json:"entries"`
}

// AddDecisionRequest is the request body for adding a decision.
type AddDecisionRequest struct {
	By         string `json:"by"`
	Decision   string `json:"decision"`
	Rationale  string `json:"rationale"`
	Context    string `json:"context,omitempty"`
	Supersedes string `json:"supersedes,omitempty"`
}

// CreateTeamRequest is the request body for creating a team
type CreateTeamRequest struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"displayName"`
	Mission     string `json:"mission,omitempty"`
	SpawnMode   string `json:"spawnMode,omitempty"`
}

// UpdateTeamRequest is the request body for updating a team
type UpdateTeamRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Mission     *string `json:"mission,omitempty"`
	SpawnMode   *string `json:"spawnMode,omitempty"`
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
				Description: "Manage teams (list|show|create|update|delete|add-member|update-member|remove-member|roles|org-*|message-*|heartbeat-*|import-cc|export-cc|trigger)",
				Run: func(args []string) error {
					return route(ctx, args)
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
	case "member-context":
		return cmdMemberContext(ctx, subArgs)
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
  member-context <team-id> <agent-id>         Get full member context prompt

Handoff Commands:
  handoff-latest <team-id> <agent-id>   Show latest handoff for a member
  handoff-history <team-id>             Show handoff history

Task Board Commands:
  task-list <team-id>                   List tasks on the task board
  task-add <team-id>                    Add a task to the board
  task-update <team-id> <task-id>       Update a task
  task-delete <team-id> <task-id>       Delete a task

Decision Log Commands:
  decision-add <team-id>                Log a decision
  decision-list <team-id>               List decisions

Claude Code Interop Commands:
  import-cc <team-name>                       Import a Claude Code team
  export-cc <team-id>                         Export team as Claude Code config
  trigger <team-id>                           Trigger team (spawn mode aware)`
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
	if team.SpawnMode != "" {
		fmt.Printf("Spawn Mode: %s\n", team.SpawnMode)
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
	spawnMode := fs.String("spawn-mode", "", "Spawn mode (multi-process|single-process)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team create <name> [--mission=...] [--spawn-mode=...]")
	}
	name := fs.Arg(0)

	req := CreateTeamRequest{
		DisplayName: name,
		Mission:     *mission,
		SpawnMode:   *spawnMode,
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
	spawnMode := fs.String("spawn-mode", "", "Spawn mode (multi-process|single-process)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team update <id> [--name=...] [--mission=...] [--spawn-mode=...]")
	}
	teamID := fs.Arg(0)

	req := UpdateTeamRequest{}
	if *name != "" {
		req.DisplayName = name
	}
	if *mission != "" {
		req.Mission = mission
	}
	if *spawnMode != "" {
		req.SpawnMode = spawnMode
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
	TeamID        string               `json:"teamId"`
	AgentID       string               `json:"agentId"`
	Enabled       bool                 `json:"enabled"`
	Schedule      string               `json:"schedule"`
	ProfileKey    string               `json:"profileKey,omitempty"`
	LastExecution *HeartbeatExecResult `json:"lastExecution,omitempty"`
	NextExecution string               `json:"nextExecution,omitempty"`
	CreatedAt     string               `json:"createdAt"`
	UpdatedAt     string               `json:"updatedAt"`
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
	Schedule   string `json:"schedule"`
	ProfileKey string `json:"profileKey,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

// UpdateHeartbeatRequest is the request for updating a heartbeat
type UpdateHeartbeatRequest struct {
	Schedule   *string `json:"schedule,omitempty"`
	ProfileKey *string `json:"profileKey,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
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
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: team heartbeat-enable <team-id> <agent-id> [--schedule='0 */6 * * *'] [--profile=key]")
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
			Schedule:   *schedule,
			ProfileKey: *profileKey,
			Enabled:    &enabled,
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
	TeamID    string            `json:"teamId"`
	SpawnMode string            `json:"spawnMode"`
	Triggers  []TriggerResponse `json:"triggers"`
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

	fmt.Printf("Triggered team %s (mode: %s)\n", teamID, resp.SpawnMode)
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
	status := fs.String("status", "", "Filter by status (todo|in-progress|blocked|done)")
	assignee := fs.String("assignee", "", "Filter by assignee agent ID")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team task-list <team-id> [--status=X] [--assignee=X] [--json]")
	}
	teamID := fs.Arg(0)

	var resp TaskBoardResponse
	if err := ctx.Get(fmt.Sprintf("/teams/%s/tasks", teamID), &resp); err != nil {
		return fmt.Errorf("failed to get task board: %w", err)
	}

	// Client-side filtering
	var filtered []TeamTask
	for _, task := range resp.Tasks {
		if *status != "" && task.Status != *status {
			continue
		}
		if *assignee != "" && task.Assignee != *assignee {
			continue
		}
		filtered = append(filtered, task)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(filtered)
	}

	if len(filtered) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	fmt.Printf("%-8s %-30s %-12s %-20s %-5s %-5s\n", "PRIO", "TITLE", "STATUS", "ASSIGNEE", "NOTES", "UPDATED")
	for _, task := range filtered {
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
	decision := fs.String("decision", "", "The decision (required)")
	rationale := fs.String("rationale", "", "Why this decision was made (required)")
	contextTag := fs.String("context", "", "Context tag for grouping")
	supersedes := fs.String("supersedes", "", "ID of decision this replaces")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team decision-add <team-id> --by=<id> --decision=\"...\" --rationale=\"...\" [--context=<tag>]")
	}
	teamID := fs.Arg(0)

	if strings.TrimSpace(*by) == "" {
		return fmt.Errorf("by is required")
	}
	if strings.TrimSpace(*decision) == "" {
		return fmt.Errorf("decision is required")
	}
	if strings.TrimSpace(*rationale) == "" {
		return fmt.Errorf("rationale is required")
	}

	req := AddDecisionRequest{
		By:         *by,
		Decision:   *decision,
		Rationale:  *rationale,
		Context:    *contextTag,
		Supersedes: *supersedes,
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

	fmt.Printf("Logged decision %s: %s\n", resp.ID, resp.Decision)
	return nil
}

func cmdDecisionList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("decision-list", flag.ContinueOnError)
	contextTag := fs.String("context", "", "Filter by context tag")
	last := fs.Int("last", 20, "Number of entries to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: team decision-list <team-id> [--context=<tag>] [--last=N] [--json]")
	}
	teamID := fs.Arg(0)

	query := fmt.Sprintf("/teams/%s/decisions?last=%d", teamID, *last)
	if *contextTag != "" {
		query += "&context=" + url.QueryEscape(*contextTag)
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
		fmt.Printf("--- %s by %s%s%s ---\n", entry.ID, entry.By, contextStr, supersededStr)
		fmt.Printf("Decision: %s\n", entry.Decision)
		fmt.Printf("Rationale: %s\n\n", entry.Rationale)
	}
	return nil
}
