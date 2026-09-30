package assistant

import (
	"fmt"
	"os"
	"strconv"

	"vrooli-assistant/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `assistant` subcommand group covering the scenario's
// issue-capture and agent-coordination surface:
//
//   - stats         GET  /api/v1/assistant/status            (scenario-specific counts)
//   - capture       POST /api/v1/assistant/capture
//   - history       GET  /api/v1/assistant/history
//   - get           GET  /api/v1/assistant/issues/{id}
//   - update-status PUT  /api/v1/assistant/issues/{id}/status
//   - spawn-agent   POST /api/v1/assistant/spawn-agent
//
// The root /health probe is owned by cli-core's built-in `status` command and
// is intentionally not duplicated here.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "assistant",
		Description: "Capture issues, inspect history, and coordinate spawn-agent requests",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "stats", Description: "Show scenario-specific counters (issues captured, agents spawned, uptime)", Run: func(args []string) error { return runStats(core, args) }},
			{Name: "capture", Description: "Capture a new issue", Run: func(args []string) error { return runCapture(core, args) }},
			{Name: "history", Aliases: []string{"list", "ls"}, Description: "List recent issues", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show a single issue by id", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update-status", Description: "Update an issue's status and resolution notes", Run: func(args []string) error { return runUpdateStatus(core, args) }},
			{Name: "spawn-agent", Description: "Spawn an agent session for an issue", Run: func(args []string) error { return runSpawnAgent(core, args) }},
		},
	}
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("assistant stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/assistant/status", nil)
	if err != nil {
		return err
	}
	var stats support.StatsResponse
	if err := support.Decode(body, &stats); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Status: %s", valueOr(stats.Status, "unknown")),
		fmt.Sprintf("Issues captured: %d", stats.IssuesCaptured),
		fmt.Sprintf("Agents spawned: %d", stats.AgentsSpawned),
		fmt.Sprintf("Uptime: %s", valueOr(stats.Uptime, "unknown")),
	}
	report := cliapp.ListReport{
		Summary:        []string{"Vrooli Assistant counters"},
		ResultsHeading: "Stats",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s assistant history", support.CLIName),
			fmt.Sprintf("%s status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCapture(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("assistant capture")
	description := fs.String("description", "", "Description of the issue (required)")
	scenarioName := fs.String("scenario-name", "", "Scenario the issue was observed in")
	issueURL := fs.String("url", "", "URL or location where the issue was observed")
	screenshotPath := fs.String("screenshot-path", "", "Path to a pre-captured screenshot on the server-accessible filesystem")
	contextFile := fs.String("context-file", "", "Path to a JSON file providing the `context` payload")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to send as the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	// Support --body-file for advanced / scripted callers per the thin-client rule.
	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *description == "" {
			// Fallback: allow positional `capture "text"` for the common case.
			if fs.NArg() >= 1 {
				s := fs.Arg(0)
				description = &s
			}
		}
		if *description == "" {
			return fmt.Errorf("usage: assistant capture --description <text> [--scenario-name NAME] [--url URL] [--screenshot-path PATH] [--context-file PATH]")
		}
		contextData, err := support.ReadJSONFile(*contextFile, false)
		if err != nil {
			return err
		}
		p := map[string]interface{}{
			"description": *description,
		}
		if *scenarioName != "" {
			p["scenario_name"] = *scenarioName
		}
		if *issueURL != "" {
			p["url"] = *issueURL
		}
		if *screenshotPath != "" {
			p["screenshot_path"] = *screenshotPath
		}
		if contextData != nil {
			p["context_data"] = contextData
		}
		payload = p
	}

	body, err := core.Request("POST", "/assistant/capture", nil, payload)
	if err != nil {
		return err
	}

	var resp support.CaptureResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := resp.Message
	if message == "" {
		message = "Issue captured"
	}
	changes := []string{fmt.Sprintf("Issue %s created", valueOr(resp.IssueID, "(unknown id)"))}
	next := []string{
		fmt.Sprintf("%s assistant get %s", support.CLIName, resp.IssueID),
		fmt.Sprintf("%s assistant spawn-agent --issue-id %s --agent-type claude-code", support.CLIName, resp.IssueID),
	}
	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: next,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("assistant history")
	limit := fs.Int("limit", 20, "Maximum issues to retrieve (informational; the API returns up to its own cap)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"limit": strconv.Itoa(*limit),
	})
	body, err := core.Get("/assistant/history", query)
	if err != nil {
		return err
	}
	var history support.HistoryResponse
	if err := support.Decode(body, &history); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recent issues: %d (count reported by API: %d)", len(history.Issues), history.Count)},
		ResultsHeading: "Issues",
		Results:        issueRows(history.Issues),
		RetrievalHints: []string{
			fmt.Sprintf("%s assistant get <issue-id>", support.CLIName),
			fmt.Sprintf("%s assistant update-status <issue-id> --status resolved", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("assistant get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: assistant get <issue-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/assistant/issues/"+id, nil)
	if err != nil {
		return err
	}
	var issue support.Issue
	if err := support.Decode(body, &issue); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", issue.ID),
		fmt.Sprintf("Status: %s", valueOr(issue.Status, "unknown")),
		fmt.Sprintf("Captured: %s", support.FormatTimeValue(issue.Timestamp)),
		fmt.Sprintf("Description: %s", valueOr(issue.Description, "(none)")),
	}
	if issue.ScenarioName != "" {
		results = append(results, fmt.Sprintf("Scenario: %s", issue.ScenarioName))
	}
	if issue.URL != "" {
		results = append(results, fmt.Sprintf("URL: %s", issue.URL))
	}
	if issue.ScreenshotPath != "" {
		results = append(results, fmt.Sprintf("Screenshot: %s", issue.ScreenshotPath))
	}
	if issue.AgentSessionID != "" {
		results = append(results, fmt.Sprintf("Agent session: %s", issue.AgentSessionID))
	}
	if issue.ResolutionNotes != "" {
		results = append(results, fmt.Sprintf("Resolution notes: %s", issue.ResolutionNotes))
	}
	if len(issue.ContextData) > 0 {
		results = append(results, "Context:")
		for _, row := range support.MapRows(issue.ContextData) {
			results = append(results, "  "+row)
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Issue: %s (%s)", support.ShortID(issue.ID), valueOr(issue.Status, "unknown"))},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s assistant update-status %s --status resolved", support.CLIName, issue.ID),
			fmt.Sprintf("%s assistant spawn-agent --issue-id %s --agent-type claude-code", support.CLIName, issue.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdateStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("assistant update-status")
	status := fs.String("status", "", "New status value (required unless --body-file is set)")
	notes := fs.String("notes", "", "Resolution notes")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to send as the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: assistant update-status <issue-id> --status <value> [--notes <text>]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *status == "" {
			return fmt.Errorf("--status is required (or provide --body-file)")
		}
		p := map[string]interface{}{"status": *status}
		if *notes != "" {
			p["notes"] = *notes
		}
		payload = p
	}

	body, err := core.Request("PUT", "/assistant/issues/"+id+"/status", nil, payload)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		if s, ok := resp["status"].(string); ok && s != "" {
			message = fmt.Sprintf("Issue status: %s", s)
		} else {
			message = "Issue updated"
		}
	}

	changes := []string{fmt.Sprintf("Issue %s updated", id)}
	if *status != "" {
		changes = append(changes, fmt.Sprintf("status=%s", *status))
	}
	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s assistant get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSpawnAgent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("assistant spawn-agent")
	issueID := fs.String("issue-id", "", "Issue ID to spawn an agent for (required unless --body-file is set)")
	agentType := fs.String("agent-type", "", "Agent type to spawn, e.g. claude-code (required unless --body-file is set)")
	description := fs.String("description", "", "Optional description forwarded to the agent")
	screenshot := fs.String("screenshot", "", "Optional screenshot reference forwarded to the agent")
	contextFile := fs.String("context-file", "", "Path to a JSON file providing the `context` payload")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to send as the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *issueID == "" || *agentType == "" {
			return fmt.Errorf("--issue-id and --agent-type are required (or provide --body-file)")
		}
		contextData, err := support.ReadJSONFile(*contextFile, false)
		if err != nil {
			return err
		}
		p := map[string]interface{}{
			"issue_id":   *issueID,
			"agent_type": *agentType,
		}
		if *description != "" {
			p["description"] = *description
		}
		if *screenshot != "" {
			p["screenshot"] = *screenshot
		}
		if contextData != nil {
			p["context"] = contextData
		}
		payload = p
	}

	body, err := core.Request("POST", "/assistant/spawn-agent", nil, payload)
	if err != nil {
		return err
	}
	var resp support.SpawnResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := resp.Message
	if message == "" {
		message = "Agent spawn requested"
	}
	changes := []string{fmt.Sprintf("Agent session %s created", valueOr(resp.SessionID, "(unknown id)"))}
	next := []string{}
	if *issueID != "" {
		next = append(next, fmt.Sprintf("%s assistant get %s", support.CLIName, *issueID))
	}
	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: next,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func issueRows(issues []support.Issue) []string {
	if len(issues) == 0 {
		return []string{"No issues captured"}
	}
	rows := make([]string, 0, len(issues))
	for _, iss := range issues {
		scenario := iss.ScenarioName
		if scenario == "" {
			scenario = "-"
		}
		desc := iss.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | scenario=%s | %s",
			support.ShortID(iss.ID),
			support.FormatTimeValue(iss.Timestamp),
			valueOr(iss.Status, "unknown"),
			scenario,
			valueOr(desc, "(no description)")))
	}
	return rows
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
