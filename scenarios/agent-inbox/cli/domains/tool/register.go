package tool

import (
	"fmt"
	"net/url"
	"strings"

	"agent-inbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type toolSetResponse struct {
	Scenarios   []scenarioInfo  `json:"scenarios"`
	Tools       []effectiveTool `json:"tools"`
	Categories  []toolCategory  `json:"categories"`
	GeneratedAt string          `json:"generated_at"`
}

type scenarioInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	BaseURL     string `json:"base_url"`
}

type toolCategory struct {
	Name string `json:"name"`
}

type effectiveTool struct {
	Scenario         string `json:"scenario"`
	Enabled          bool   `json:"enabled"`
	Source           string `json:"source"`
	RequiresApproval bool   `json:"requires_approval"`
	ApprovalSource   string `json:"approval_source"`
	Tool             struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tool"`
}

type scenarioStatus struct {
	Scenario  string `json:"scenario"`
	Available bool   `json:"available"`
	LastCheck string `json:"last_checked"`
	ToolCount int    `json:"tool_count"`
	Error     string `json:"error"`
}

type basicTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"function"`
}

type chatToolCall struct {
	ID           string `json:"id"`
	ToolName     string `json:"tool_name"`
	Status       string `json:"status"`
	ScenarioName string `json:"scenario_name"`
	StartedAt    string `json:"started_at"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tool",
		Description: "Tool discovery and configuration",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List effective tools", Run: func(args []string) error { return runList(core, args) }},
			{Name: "raw", NeedsAPI: true, Description: "List OpenAI-compatible tool definitions", Run: func(args []string) error { return runRaw(core, args) }},
			{Name: "scenarios", NeedsAPI: true, Description: "List configured scenario tool providers", Run: func(args []string) error { return runScenarios(core, args) }},
			{Name: "enable", NeedsAPI: true, Description: "Enable a tool globally or for one chat", Run: func(args []string) error { return runSetEnabled(core, true, args) }},
			{Name: "disable", NeedsAPI: true, Description: "Disable a tool globally or for one chat", Run: func(args []string) error { return runSetEnabled(core, false, args) }},
			{Name: "reset", NeedsAPI: true, Description: "Reset tool configuration to default", Run: func(args []string) error { return runReset(core, args) }},
			{Name: "sync", NeedsAPI: true, Description: "Re-discover available tools", Run: func(args []string) error { return runSync(core, args) }},
			{Name: "calls", NeedsAPI: true, Description: "List tool calls for one chat", Run: func(args []string) error { return runCalls(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool list")
	chatID := fs.String("chat", "", "Chat-specific effective tool set")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*chatID) != "" {
		query.Set("chat_id", strings.TrimSpace(*chatID))
	}
	body, err := core.Get("/tools/set", query)
	if err != nil {
		return err
	}

	var resp toolSetResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Tools))
	for _, item := range resp.Tools {
		line := fmt.Sprintf("%s/%s | enabled=%t | source=%s", item.Scenario, item.Tool.Name, item.Enabled, item.Source)
		if item.RequiresApproval {
			line += " | approval"
		}
		if item.Tool.Description != "" {
			line += "\n  " + support.Truncate(item.Tool.Description, 100)
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tools: %d", len(resp.Tools)), fmt.Sprintf("Scenarios: %d", len(resp.Scenarios))},
		ResultsHeading: "Effective Tools",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " tool enable --scenario agent-manager --tool spawn_coding_agent", support.CLIName + " tool sync"},
	}
	return support.PrintList(*jsonOutput, report)
}

func runRaw(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool raw")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var tools []basicTool
	if err := support.GetJSON(core, "/tools", &tools); err != nil {
		return err
	}

	results := make([]string, 0, len(tools))
	for _, item := range tools {
		results = append(results, fmt.Sprintf("%s\n  %s", item.Function.Name, support.Truncate(item.Function.Description, 100)))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("OpenAI tool definitions: %d", len(tools))},
		ResultsHeading: "Tools",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " tool list", support.CLIName + " tool scenarios"},
	}
	return support.PrintList(*jsonOutput, report)
}

func runScenarios(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool scenarios")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var statuses []scenarioStatus
	if err := support.GetJSON(core, "/tools/scenarios", &statuses); err != nil {
		return err
	}

	results := make([]string, 0, len(statuses))
	for _, item := range statuses {
		line := fmt.Sprintf("%s | available=%t | tools=%d", item.Scenario, item.Available, item.ToolCount)
		if item.Error != "" {
			line += "\n  " + item.Error
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario providers: %d", len(statuses))},
		ResultsHeading: "Providers",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " tool sync"},
	}
	return support.PrintList(*jsonOutput, report)
}

func runSetEnabled(core *cliapp.ScenarioApp, enabled bool, args []string) error {
	commandName := "tool disable"
	if enabled {
		commandName = "tool enable"
	}
	fs := support.NewFlagSet(commandName)
	chatID := fs.String("chat", "", "Optional chat ID for chat-specific override")
	scenario := fs.String("scenario", "", "Scenario provider name")
	toolName := fs.String("tool", "", "Tool name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" || strings.TrimSpace(*toolName) == "" {
		return fmt.Errorf("--scenario and --tool are required")
	}

	input := map[string]interface{}{
		"scenario":  strings.TrimSpace(*scenario),
		"tool_name": strings.TrimSpace(*toolName),
		"enabled":   enabled,
	}
	if strings.TrimSpace(*chatID) != "" {
		input["chat_id"] = strings.TrimSpace(*chatID)
	}

	if _, err := core.Request("POST", "/tools/config", nil, input); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Tool configuration updated"},
		Changes:     []string{fmt.Sprintf("Scenario: %s", input["scenario"]), fmt.Sprintf("Tool: %s", input["tool_name"]), fmt.Sprintf("Enabled: %t", enabled)},
		NextCommand: []string{support.CLIName + " tool list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runReset(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool reset")
	chatID := fs.String("chat", "", "Optional chat ID for chat-specific override")
	scenario := fs.String("scenario", "", "Scenario provider name")
	toolName := fs.String("tool", "", "Tool name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" || strings.TrimSpace(*toolName) == "" {
		return fmt.Errorf("--scenario and --tool are required")
	}

	query := url.Values{}
	query.Set("scenario", strings.TrimSpace(*scenario))
	query.Set("tool_name", strings.TrimSpace(*toolName))
	if strings.TrimSpace(*chatID) != "" {
		query.Set("chat_id", strings.TrimSpace(*chatID))
	}
	if _, err := core.Request("DELETE", "/tools/config", query, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Tool configuration reset"},
		Changes:     []string{fmt.Sprintf("Scenario: %s", query.Get("scenario")), fmt.Sprintf("Tool: %s", query.Get("tool_name"))},
		NextCommand: []string{support.CLIName + " tool list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runSync(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool sync")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp struct {
		ScenariosWithTools int      `json:"scenarios_with_tools"`
		NewScenarios       []string `json:"new_scenarios"`
		RemovedScenarios   []string `json:"removed_scenarios"`
		TotalTools         int      `json:"total_tools"`
	}
	body, err := core.Request("POST", "/tools/sync", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Tool sync completed", fmt.Sprintf("Providers with tools: %d", resp.ScenariosWithTools)},
		Changes: []string{
			fmt.Sprintf("Total tools: %d", resp.TotalTools),
			"New scenarios: " + strings.Join(resp.NewScenarios, ", "),
			"Removed scenarios: " + strings.Join(resp.RemovedScenarios, ", "),
		},
		NextCommand: []string{support.CLIName + " tool list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runCalls(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool calls")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tool calls <chat-id> [--json]")
	}
	id := fs.Arg(0)

	var calls []chatToolCall
	if err := support.GetJSON(core, "/chats/"+id+"/tool-calls", &calls); err != nil {
		return err
	}

	results := make([]string, 0, len(calls))
	for _, call := range calls {
		results = append(results, fmt.Sprintf("%s | %s | %s | %s", call.Status, call.ToolName, call.ScenarioName, support.FormatTime(call.StartedAt)))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tool calls: %d", len(calls)), "Chat ID: " + id},
		ResultsHeading: "Tool Calls",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " chat get " + id, support.CLIName + " agent status " + id},
	}
	return support.PrintList(*jsonOutput, report)
}
