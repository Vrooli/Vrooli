package tasks

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"task-planner/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `task` subcommand group. Each subcommand maps 1:1 to an
// endpoint exposed by the task-planner API; no client-side filtering or
// orchestration is done here.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "task",
		Description: "Parse, list, research, and update planner tasks",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List tasks with optional filters", Run: func(args []string) error { return runList(core, args) }},
			{Name: "parse", Description: "Parse raw text into tasks via POST /api/parse-text", Run: func(args []string) error { return runParse(core, args) }},
			{Name: "research", Description: "Trigger research for a backlog task", Run: func(args []string) error { return runResearch(core, args) }},
			{Name: "status-set", Description: "Update task status via PUT /api/tasks/status", Run: func(args []string) error { return runStatusSet(core, args) }},
			{Name: "status-history", Description: "Fetch status history for a task", Run: func(args []string) error { return runStatusHistory(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("task list")
	appID := fs.String("app-id", "", "Filter by application ID")
	status := fs.String("status", "", "Filter by status")
	priority := fs.String("priority", "", "Filter by priority")
	limit := fs.Int("limit", 50, "Maximum tasks to retrieve")
	offset := fs.Int("offset", 0, "Pagination offset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"app_id":   *appID,
		"status":   *status,
		"priority": *priority,
		"limit":    strconv.Itoa(*limit),
		"offset":   strconv.Itoa(*offset),
	})
	body, err := core.Get("/tasks", query)
	if err != nil {
		return err
	}
	var resp support.TasksResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tasks: %d", resp.Count)},
		ResultsHeading: "Tasks",
		Results:        taskRows(resp.Tasks),
		RetrievalHints: []string{
			fmt.Sprintf("%s task status-history <task-id>", support.CLIName),
			fmt.Sprintf("%s task research <task-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runParse(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("task parse")
	appID := fs.String("app-id", "", "Application ID to attach tasks to (required)")
	text := fs.String("text", "", "Raw text to parse")
	textFile := fs.String("text-file", "", "Read raw text from a file")
	apiToken := fs.String("api-token", "cli_token", "App API token for the parser")
	inputType := fs.String("input-type", "", "Optional input type hint")
	submittedBy := fs.String("submitted-by", "cli", "Identifier recorded as submitter")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		resolvedText := strings.TrimSpace(*text)
		if resolvedText == "" && strings.TrimSpace(*textFile) != "" {
			data, err := os.ReadFile(strings.TrimSpace(*textFile))
			if err != nil {
				return fmt.Errorf("read %s: %w", *textFile, err)
			}
			resolvedText = strings.TrimSpace(string(data))
		}
		if strings.TrimSpace(*appID) == "" {
			return fmt.Errorf("--app-id is required (or pass --body-file)")
		}
		if resolvedText == "" {
			return fmt.Errorf("--text or --text-file is required (or pass --body-file)")
		}
		payload = map[string]interface{}{
			"app_id":       strings.TrimSpace(*appID),
			"raw_text":     resolvedText,
			"api_token":    strings.TrimSpace(*apiToken),
			"input_type":   strings.TrimSpace(*inputType),
			"submitted_by": strings.TrimSpace(*submittedBy),
		}
	}

	body, err := core.Request("POST", "/parse-text", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ParseTextResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := fmt.Sprintf("Parsed text: %d task(s) created", resp.TasksCreated)
	if resp.SessionID != "" {
		result += fmt.Sprintf(" (session %s)", support.ShortID(resp.SessionID))
	}
	changes := taskRows(resp.Tasks)
	report := cliapp.MutationReport{
		Result:  []string{result},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s task list --app-id %s", support.CLIName, strings.TrimSpace(*appID)),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runResearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("task research")
	force := fs.Bool("force", false, "Force refresh even if prior research exists")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full research payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task research <task-id> [--force] [--body-file PATH]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		payload = map[string]interface{}{
			"force_refresh": *force,
		}
	}

	body, err := core.Request("POST", "/tasks/"+id+"/research", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ResearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Task: %s", resp.TaskID),
		fmt.Sprintf("Complexity: %s", resp.Complexity),
		fmt.Sprintf("Estimated hours: %g", resp.EstimatedHours),
	}
	if resp.ResearchSummary != "" {
		results = append(results, "Summary: "+resp.ResearchSummary)
	}
	if len(resp.Requirements) > 0 {
		results = append(results, "Requirements: "+strings.Join(resp.Requirements, "; "))
	}
	if len(resp.Dependencies) > 0 {
		results = append(results, "Dependencies: "+strings.Join(resp.Dependencies, "; "))
	}
	if len(resp.Recommendations) > 0 {
		results = append(results, "Recommendations: "+strings.Join(resp.Recommendations, "; "))
	}
	if resp.Error != "" {
		results = append(results, "Error: "+resp.Error)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Research for %s", id)},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s task status-set %s --to-status in_progress", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStatusSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("task status-set")
	toStatus := fs.String("to-status", "", "Target status (required)")
	reason := fs.String("reason", "", "Optional reason")
	notes := fs.String("notes", "", "Optional notes")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full status-update payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	var id string
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
		// best-effort: let the server validate; we keep id for hints
		if fs.NArg() >= 1 {
			id = fs.Arg(0)
		}
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: task status-set <task-id> --to-status STATUS [--reason ...] [--notes ...]")
		}
		id = fs.Arg(0)
		if strings.TrimSpace(*toStatus) == "" {
			return fmt.Errorf("--to-status is required")
		}
		payload = map[string]interface{}{
			"task_id":   id,
			"to_status": strings.TrimSpace(*toStatus),
			"reason":    strings.TrimSpace(*reason),
			"notes":     strings.TrimSpace(*notes),
		}
	}

	body, err := core.Request("PUT", "/tasks/status", nil, payload)
	if err != nil {
		return err
	}
	var resp support.StatusUpdateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := "Status update dispatched"
	if resp.StatusChanged {
		result = fmt.Sprintf("Status %s -> %s", resp.PreviousStatus, resp.NewStatus)
	}
	if resp.Error != "" {
		result = "Error: " + resp.Error
	}

	changes := make([]string, 0, len(resp.NextActions))
	for _, a := range resp.NextActions {
		changes = append(changes, "next: "+a)
	}

	report := cliapp.MutationReport{
		Result:  []string{result},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s task status-history %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runStatusHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("task status-history")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task status-history <task-id>")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{"task_id": id})
	body, err := core.Get("/tasks/status-history", query)
	if err != nil {
		return err
	}
	var resp support.StatusHistoryResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Status history for %s: %d entries", id, len(resp.History))},
		ResultsHeading: "Transitions",
		Results:        historyRows(resp.History),
		RetrievalHints: []string{
			fmt.Sprintf("%s task list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func taskRows(tasks []support.Task) []string {
	if len(tasks) == 0 {
		return []string{"(no tasks)"}
	}
	rows := make([]string, 0, len(tasks))
	for _, t := range tasks {
		title := t.Title
		if len(title) > 60 {
			title = title[:60] + "..."
		}
		rows = append(rows, fmt.Sprintf("%s | %s | status=%s priority=%s app=%s",
			support.ShortID(t.ID), title, t.Status, t.Priority, t.AppName))
	}
	return rows
}

func historyRows(entries []support.StatusChange) []string {
	if len(entries) == 0 {
		return []string{"(no history)"}
	}
	rows := make([]string, 0, len(entries))
	for _, e := range entries {
		ts := "unknown"
		if e.ChangedAt != nil {
			ts = support.FormatTimeValue(*e.ChangedAt)
		}
		line := fmt.Sprintf("%s | %s -> %s", ts, e.FromStatus, e.ToStatus)
		if e.ChangedBy != "" {
			line += fmt.Sprintf(" by=%s", e.ChangedBy)
		}
		if e.Reason != "" {
			line += fmt.Sprintf(" reason=%s", e.Reason)
		}
		rows = append(rows, line)
	}
	return rows
}
