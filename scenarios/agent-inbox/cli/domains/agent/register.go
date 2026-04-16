package agent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"agent-inbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type runSummary struct {
	RunID           string `json:"run_id"`
	TaskID          string `json:"task_id"`
	Tag             string `json:"tag"`
	Status          string `json:"status"`
	Phase           string `json:"phase"`
	ProgressPercent int    `json:"progress_percent"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type listRunsResponse struct {
	Runs    []runSummary `json:"runs"`
	Total   int          `json:"total"`
	HasMore bool         `json:"has_more"`
}

type agentStatusResponse struct {
	ChatMode        string `json:"chat_mode"`
	IsAgent         bool   `json:"is_agent"`
	TaskID          string `json:"task_id"`
	RunID           string `json:"run_id"`
	Status          string `json:"status"`
	Phase           string `json:"phase"`
	ProgressPercent int    `json:"progress_percent"`
	SessionID       string `json:"session_id"`
	ErrorMsg        string `json:"error_msg"`
	Error           string `json:"error"`
}

type agentMutationResponse struct {
	ChatID    string `json:"chat_id"`
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
}

type agentEvent struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"`
	Sequence   int    `json:"sequence"`
	ToolName   string `json:"tool_name"`
	RunStatus  string `json:"run_status"`
	Phase      string `json:"phase"`
	Progress   int    `json:"progress"`
	RawData    string `json:"raw_data"`
	ToolOutput string `json:"tool_output"`
}

type eventsResponse struct {
	Events []agentEvent `json:"events"`
	RunID  string       `json:"run_id"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "agent",
		Description: "Agent-mode and agent-run operations",
		Subcommands: []cliapp.Command{
			{Name: "runs", NeedsAPI: true, Description: "List agent-manager runs", Run: func(args []string) error { return RunRuns(core, args) }},
			{Name: "status", NeedsAPI: true, Description: "Show agent status for a chat", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "start", NeedsAPI: true, Description: "Start agent mode for a chat", Run: func(args []string) error { return runStart(core, args) }},
			{Name: "send", NeedsAPI: true, Description: "Send a follow-up message to an agent run", Run: func(args []string) error { return runSend(core, args) }},
			{Name: "stop", NeedsAPI: true, Description: "Stop the active agent run", Run: func(args []string) error { return runStop(core, args) }},
			{Name: "clear", NeedsAPI: true, Description: "Clear agent mode and return to LLM mode", Run: func(args []string) error { return runClear(core, args) }},
			{Name: "attach", NeedsAPI: true, Description: "Attach an existing run to a chat", Run: func(args []string) error { return runAttach(core, args) }},
			{Name: "events", NeedsAPI: true, Description: "Show translated events for a chat's agent run", Run: func(args []string) error { return runEvents(core, args) }},
			{Name: "run-events", NeedsAPI: true, Description: "Show translated events for a run ID", Run: func(args []string) error { return runRunEvents(core, args) }},
		},
	}
}

func RunRuns(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent runs")
	status := fs.String("status", "", "Filter by run status")
	tagPrefix := fs.String("tag-prefix", "", "Filter by tag prefix")
	limit := fs.Int("limit", 20, "Maximum runs to return")
	offset := fs.Int("offset", 0, "Runs to skip")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*tagPrefix) != "" {
		query.Set("tag_prefix", strings.TrimSpace(*tagPrefix))
	}
	query.Set("limit", strconv.Itoa(*limit))
	query.Set("offset", strconv.Itoa(*offset))

	body, err := core.Get("/agent-runs", query)
	if err != nil {
		return err
	}

	var resp listRunsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Runs))
	for _, run := range resp.Runs {
		line := fmt.Sprintf("%s | %s | %s | %d%%", run.RunID, run.Status, run.Phase, run.ProgressPercent)
		if run.Tag != "" {
			line += " | " + run.Tag
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Runs returned: %d", len(resp.Runs)), fmt.Sprintf("Total runs: %d", resp.Total)},
		ResultsHeading: "Agent Runs",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " agent run-events <run-id>", support.CLIName + " agent status <chat-id>"},
	}
	return support.PrintList(*jsonOutput, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent status <chat-id> [--json]")
	}
	id := fs.Arg(0)

	var resp agentStatusResponse
	if err := support.GetJSON(core, "/chats/"+id+"/agent-mode/status", &resp); err != nil {
		return err
	}

	statusLines := []string{
		"Chat mode: " + resp.ChatMode,
		fmt.Sprintf("Agent mode active: %t", resp.IsAgent),
	}
	if resp.RunID != "" {
		statusLines = append(statusLines, "Run ID: "+resp.RunID)
	}
	if resp.TaskID != "" {
		statusLines = append(statusLines, "Task ID: "+resp.TaskID)
	}
	if resp.Status != "" {
		statusLines = append(statusLines, "Status: "+resp.Status)
	}
	if resp.Phase != "" {
		statusLines = append(statusLines, "Phase: "+resp.Phase)
	}
	if resp.ProgressPercent > 0 {
		statusLines = append(statusLines, fmt.Sprintf("Progress: %d%%", resp.ProgressPercent))
	}

	triage := []cliapp.TriageGroup{}
	if resp.Error != "" || resp.ErrorMsg != "" {
		items := []string{}
		if resp.Error != "" {
			items = append(items, resp.Error)
		}
		if resp.ErrorMsg != "" {
			items = append(items, resp.ErrorMsg)
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "Errors", Items: items})
	}

	report := cliapp.OperationalReport{
		Status:    statusLines,
		Triage:    triage,
		NextSteps: []string{support.CLIName + " agent events " + id, support.CLIName + " agent send " + id + " --message \"...\""},
	}
	return support.PrintOperational(*jsonOutput, report)
}

func runStart(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent start")
	message := fs.String("message", "", "Initial agent message")
	runner := fs.String("runner", "claude-code", "Runner type: claude-code, codex, opencode")
	projectPath := fs.String("project", ".", "Project path for the agent workspace")
	model := fs.String("model", "", "Optional model override")
	maxTurns := fs.Int("max-turns", 0, "Optional max turns")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent start <chat-id> --message TEXT [--runner claude-code|codex|opencode] [--project PATH] [--json]")
	}
	if strings.TrimSpace(*message) == "" {
		return fmt.Errorf("--message is required")
	}
	id := fs.Arg(0)

	input := map[string]interface{}{
		"message":      *message,
		"runner_type":  strings.TrimSpace(*runner),
		"project_path": support.AbsPath(*projectPath),
	}
	if strings.TrimSpace(*model) != "" {
		input["model"] = strings.TrimSpace(*model)
	}
	if *maxTurns > 0 {
		input["max_turns"] = *maxTurns
	}

	body, err := core.Request("POST", "/chats/"+id+"/agent-mode/start", nil, input)
	if err != nil {
		return err
	}

	var resp agentMutationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Agent mode started", "Run ID: " + resp.RunID},
		Changes: []string{
			"Chat ID: " + resp.ChatID,
			"Task ID: " + resp.TaskID,
			"Runner: " + strings.TrimSpace(*runner),
			"Project: " + support.AbsPath(*projectPath),
		},
		NextCommand: []string{support.CLIName + " agent status " + id, support.CLIName + " agent events " + id},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runSend(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent send")
	message := fs.String("message", "", "Follow-up message")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent send <chat-id> --message TEXT [--json]")
	}
	if strings.TrimSpace(*message) == "" {
		return fmt.Errorf("--message is required")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/chats/"+id+"/agent-mode/message", nil, map[string]interface{}{"message": *message})
	if err != nil {
		return err
	}
	var resp struct {
		Success bool   `json:"success"`
		RunID   string `json:"run_id"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Agent message sent", "Run ID: " + resp.RunID},
		Changes:     []string{"Chat ID: " + id, "Message: " + support.Truncate(*message, 96)},
		NextCommand: []string{support.CLIName + " agent events " + id, support.CLIName + " agent status " + id},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runStop(core *cliapp.ScenarioApp, args []string) error {
	return runSimpleMutation(core, "agent stop", "/chats/%s/agent-mode/stop", "Agent run stop requested", args)
}

func runClear(core *cliapp.ScenarioApp, args []string) error {
	return runSimpleMutation(core, "agent clear", "/chats/%s/agent-mode/clear", "Agent mode cleared", args)
}

func runSimpleMutation(core *cliapp.ScenarioApp, commandName, pathTemplate, resultLine string, args []string) error {
	fs := support.NewFlagSet(commandName)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: %s <chat-id> [--json]", commandName)
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", fmt.Sprintf(pathTemplate, id), nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	changes := []string{"Chat ID: " + id}
	if runID, ok := raw["run_id"].(string); ok && strings.TrimSpace(runID) != "" {
		changes = append(changes, "Run ID: "+runID)
	}
	if mode, ok := raw["chat_mode"].(string); ok && strings.TrimSpace(mode) != "" {
		changes = append(changes, "Chat mode: "+mode)
	}

	report := cliapp.MutationReport{
		Result:      []string{resultLine},
		Changes:     changes,
		NextCommand: []string{support.CLIName + " agent status " + id},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runAttach(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent attach")
	runID := fs.String("run", "", "Existing run ID")
	taskID := fs.String("task", "", "Existing task ID")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent attach <chat-id> --run <run-id> --task <task-id> [--json]")
	}
	if strings.TrimSpace(*runID) == "" || strings.TrimSpace(*taskID) == "" {
		return fmt.Errorf("--run and --task are required")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/chats/"+id+"/agent-mode/attach", nil, map[string]interface{}{
		"run_id":  strings.TrimSpace(*runID),
		"task_id": strings.TrimSpace(*taskID),
	})
	if err != nil {
		return err
	}
	var resp agentMutationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Agent run attached", "Run ID: " + resp.RunID},
		Changes: []string{
			"Chat ID: " + resp.ChatID,
			"Task ID: " + resp.TaskID,
		},
		NextCommand: []string{support.CLIName + " agent status " + id, support.CLIName + " agent events " + id},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runEvents(core *cliapp.ScenarioApp, args []string) error {
	return runEventsForPath(core, "agent events", "/chats/%s/agent-mode/events", args)
}

func runRunEvents(core *cliapp.ScenarioApp, args []string) error {
	return runEventsForPath(core, "agent run-events", "/agent-runs/%s/events", args)
}

func runEventsForPath(core *cliapp.ScenarioApp, commandName, pathTemplate string, args []string) error {
	fs := support.NewFlagSet(commandName)
	after := fs.Int("after", 0, "Only show events after this sequence number")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: %s <id> [--after N] [--json]", commandName)
	}
	id := fs.Arg(0)

	query := url.Values{}
	if *after > 0 {
		query.Set("after_sequence", strconv.Itoa(*after))
	}
	body, err := core.Get(fmt.Sprintf(pathTemplate, id), query)
	if err != nil {
		return err
	}

	var resp eventsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Events))
	for _, event := range resp.Events {
		line := fmt.Sprintf("#%d | %s | %s", event.Sequence, event.Type, support.FormatTime(event.Timestamp))
		if event.ToolName != "" {
			line += " | tool=" + event.ToolName
		}
		if event.RunStatus != "" {
			line += " | status=" + event.RunStatus
		}
		content := support.Truncate(event.Content, 120)
		if content == "" {
			content = support.Truncate(event.RawData, 120)
		}
		if content != "" {
			line += "\n  " + content
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Events: %d", len(resp.Events)), "Run ID: " + resp.RunID},
		ResultsHeading: "Events",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " agent status <chat-id>", support.CLIName + " agent send <chat-id> --message \"...\""},
	}
	return support.PrintList(*jsonOutput, report)
}
