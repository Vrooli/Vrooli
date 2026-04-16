package issue

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"app-issue-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `issue` subcommand group. This is the primary surface of
// the CLI — create/list/show/search/investigate/fix/export all delegate directly
// to the corresponding API endpoints. `fix` is an alias for `investigate` per
// the API contract: the unified-resolver agent both investigates and resolves.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "issue",
		Description: "Create, inspect, and act on tracked issues",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a new issue", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "list", Aliases: []string{"ls"}, Description: "List issues with filters", Run: func(args []string) error { return runList(core, args) }},
			{Name: "show", Aliases: []string{"get"}, Description: "Show one issue", Run: func(args []string) error { return runShow(core, args) }},
			{Name: "search", Description: "Search issues by text", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "investigate", Description: "Trigger the unified AI agent run", Run: func(args []string) error { return runInvestigate(core, args, false) }},
			{Name: "fix", Description: "Alias for investigate (investigation + resolution)", Run: func(args []string) error { return runInvestigate(core, args, true) }},
			{Name: "export", Description: "Export issues (json, csv, markdown)", Run: func(args []string) error { return runExport(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("issue create")
	title := fs.String("title", "", "Issue title (required)")
	description := fs.String("description", "", "Detailed description")
	issueType := fs.String("type", "bug", "Issue type (bug/feature/performance/security)")
	priority := fs.String("priority", "medium", "Priority (critical/high/medium/low)")
	appID := fs.String("app-id", "", "Application identifier")
	errorMessage := fs.String("error", "", "Error message that occurred")
	stackTrace := fs.String("stack", "", "Stack trace information")
	tags := fs.String("tags", "", "Comma-separated tags")
	reporterName := fs.String("reporter-name", "", "Reporter's name")
	reporterEmail := fs.String("reporter-email", "", "Reporter's email")
	environment := fs.String("environment", "", "Environment details (JSON or free text)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	trimmedTitle := strings.TrimSpace(*title)
	if trimmedTitle == "" {
		return fmt.Errorf("usage: issue create --title <title> [--description ...] [--type bug] [--priority medium]")
	}
	descriptionValue := strings.TrimSpace(*description)
	if descriptionValue == "" {
		descriptionValue = trimmedTitle
	}

	body := map[string]interface{}{
		"title":          trimmedTitle,
		"description":    descriptionValue,
		"type":           strings.TrimSpace(*issueType),
		"priority":       strings.TrimSpace(*priority),
		"app_id":         strings.TrimSpace(*appID),
		"error_message":  strings.TrimSpace(*errorMessage),
		"stack_trace":    strings.TrimSpace(*stackTrace),
		"reporter_name":  strings.TrimSpace(*reporterName),
		"reporter_email": strings.TrimSpace(*reporterEmail),
		"tags":           parseTags(*tags),
		"environment":    parseEnvironment(*environment),
	}

	respBody, err := core.Request("POST", "/issues", nil, body)
	if err != nil {
		return err
	}
	var data support.IssueCreateData
	if err := support.Decode(respBody, &data); err != nil {
		return err
	}

	message := support.EnvelopeMessage(respBody)
	if message == "" {
		message = "Issue created"
	}
	issueID := strings.TrimSpace(data.IssueID)
	if issueID == "" && data.Issue != nil {
		issueID = data.Issue.ID
	}

	result := []string{message}
	if issueID != "" {
		result = append(result, fmt.Sprintf("Issue ID: %s", issueID))
	}
	if data.Issue != nil {
		if t := strings.TrimSpace(data.Issue.Title); t != "" {
			result = append(result, fmt.Sprintf("Title: %s", t))
		}
		if p := strings.TrimSpace(data.Issue.Priority); p != "" {
			result = append(result, fmt.Sprintf("Priority: %s", p))
		}
	}
	if data.StoragePath != "" {
		result = append(result, fmt.Sprintf("Storage path: %s", data.StoragePath))
	}

	report := cliapp.MutationReport{
		Result: result,
		Changes: []string{
			fmt.Sprintf("Created issue %q (priority=%s)", trimmedTitle, strings.TrimSpace(*priority)),
		},
		NextCommand: nextCommandsForIssue(issueID),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("issue list")
	status := fs.String("status", "", "Filter by status (open/active/completed/failed)")
	priority := fs.String("priority", "", "Filter by priority (critical/high/medium/low)")
	issueType := fs.String("type", "", "Filter by type (bug/feature/performance/security)")
	targetID := fs.String("target-id", "", "Filter by target id")
	targetType := fs.String("target-type", "", "Filter by target type (scenario|resource)")
	limit := fs.Int("limit", 20, "Maximum number of results")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"status":      *status,
		"priority":    *priority,
		"type":        *issueType,
		"target_id":   *targetID,
		"target_type": *targetType,
		"limit":       strconv.Itoa(*limit),
	})
	body, err := core.Get("/issues", query)
	if err != nil {
		return err
	}
	var data support.IssueListData
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Found %d issues", data.Count)}
	if *status != "" || *priority != "" || *issueType != "" || *targetID != "" {
		summary = append(summary, describeFilters(*status, *priority, *issueType, *targetID, *targetType))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Issues",
		Results:        issueRows(data.Issues),
		RetrievalHints: []string{
			fmt.Sprintf("%s issue show <issue-id>", support.CLIName),
			fmt.Sprintf("%s issue search \"<query>\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("issue show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: issue show <issue-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/issues/"+id, nil)
	if err != nil {
		return err
	}
	var data support.IssueDetailData
	if err := support.Decode(body, &data); err != nil {
		return err
	}
	if data.Issue == nil {
		return fmt.Errorf("issue %s not found", id)
	}

	issue := data.Issue
	results := []string{
		fmt.Sprintf("ID: %s", issue.ID),
		fmt.Sprintf("Title: %s", issue.Title),
		fmt.Sprintf("Status: %s", defaultIfEmpty(issue.Status, "unknown")),
		fmt.Sprintf("Priority: %s", defaultIfEmpty(issue.Priority, "unset")),
		fmt.Sprintf("Type: %s", defaultIfEmpty(issue.Type, "unset")),
	}
	if targets := support.FormatTargets(issue.Targets); targets != "" {
		results = append(results, fmt.Sprintf("Targets: %s", targets))
	}
	if issue.Metadata.CreatedAt != "" {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTime(issue.Metadata.CreatedAt)))
	}
	if issue.Metadata.UpdatedAt != "" {
		results = append(results, fmt.Sprintf("Updated: %s", support.FormatTime(issue.Metadata.UpdatedAt)))
	}
	if len(issue.Metadata.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %s", strings.Join(issue.Metadata.Tags, ", ")))
	}
	if desc := strings.TrimSpace(issue.Description); desc != "" {
		results = append(results, "", "Description:", desc)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Issue: %s", issue.Title)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: nextCommandsForIssue(issue.ID),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("issue search")
	limit := fs.Int("limit", 10, "Maximum number of results")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: issue search <query> [--limit N]")
	}
	query := strings.Join(fs.Args(), " ")

	values := support.BuildQuery(map[string]string{
		"q":     query,
		"limit": strconv.Itoa(*limit),
	})
	body, err := core.Get("/issues/search", values)
	if err != nil {
		return err
	}
	var data support.IssueSearchData
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Query: %s", query),
		fmt.Sprintf("Found %d results", data.Count),
	}
	if data.Method != "" {
		summary = append(summary, fmt.Sprintf("Method: %s", data.Method))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Matches",
		Results:        issueRows(data.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s issue show <issue-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runInvestigate(core *cliapp.ScenarioApp, args []string, isFix bool) error {
	label := "issue investigate"
	if isFix {
		label = "issue fix"
	}
	fs := support.NewFlagSet(label)
	agentID := fs.String("agent", "", "Agent to use (default: unified-resolver)")
	priority := fs.String("priority", "normal", "Run priority (normal/high/urgent)")
	analysisOnly := fs.Bool("analysis-only", false, "Skip fix generation; investigation only")
	force := fs.Bool("force", false, "Bypass running/slot checks")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: %s <issue-id>", label)
	}
	id := fs.Arg(0)

	autoResolve := !*analysisOnly
	if isFix {
		// `fix` is the resolution-required alias of investigate. Don't let
		// --analysis-only silently subvert the verb's contract.
		autoResolve = true
	}

	body := map[string]interface{}{
		"issue_id":     id,
		"agent_id":     strings.TrimSpace(*agentID),
		"priority":     strings.TrimSpace(*priority),
		"auto_resolve": autoResolve,
		"force":        *force,
	}

	respBody, err := core.Request("POST", "/investigate", nil, body)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(respBody, &payload); err != nil {
		return err
	}

	runID := asString(payload["run_id"])
	resolutionID := asString(payload["resolution_id"])
	status := asString(payload["status"])
	workflow := asString(payload["workflow"])

	message := support.EnvelopeMessage(respBody)
	if message == "" {
		if autoResolve {
			message = "Agent run started (investigation + resolution)"
		} else {
			message = "Agent run started (investigation only)"
		}
	}

	result := []string{message}
	if runID != "" {
		result = append(result, fmt.Sprintf("Run ID: %s", runID))
	}
	if resolutionID != "" {
		result = append(result, fmt.Sprintf("Resolution ID: %s", resolutionID))
	}
	if status != "" {
		result = append(result, fmt.Sprintf("Status: %s", status))
	}
	if workflow != "" {
		result = append(result, fmt.Sprintf("Workflow: %s", workflow))
	}

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{fmt.Sprintf("Issue %s: agent run triggered (auto_resolve=%t)", id, autoResolve)},
		NextCommand: nextCommandsForIssue(id),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("issue export")
	format := fs.String("format", "markdown", "Export format (json|csv|markdown)")
	status := fs.String("status", "", "Filter by status")
	priority := fs.String("priority", "", "Filter by priority")
	issueType := fs.String("type", "", "Filter by type")
	targetID := fs.String("target-id", "", "Filter by target id")
	outputFile := fs.String("output", "", "Write to file instead of stdout")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	format2 := strings.ToLower(strings.TrimSpace(*format))
	if format2 == "md" {
		format2 = "markdown"
	}
	switch format2 {
	case "json", "csv", "markdown":
	default:
		return fmt.Errorf("unsupported format %q (use json, csv, or markdown)", *format)
	}

	query := support.BuildQuery(map[string]string{
		"format":    format2,
		"status":    *status,
		"priority":  *priority,
		"type":      *issueType,
		"target_id": *targetID,
	})
	body, err := core.Get("/export", query)
	if err != nil {
		return err
	}

	// The export endpoint returns raw JSON/CSV/Markdown, not the envelope.
	// --json preserves the raw bytes so downstream tooling gets whatever the
	// server produced. Otherwise we write to file (if --output) or stdout.
	if *jsonOutput && format2 == "json" {
		return support.WriteOutput(strings.TrimSpace(*outputFile), body)
	}
	if strings.TrimSpace(*outputFile) != "" {
		if err := support.WriteOutput(*outputFile, body); err != nil {
			return err
		}
		report := cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Exported %d bytes to %s", len(body), *outputFile)},
			Changes: []string{fmt.Sprintf("Wrote %s export", format2)},
			NextCommand: []string{
				fmt.Sprintf("%s issue list --limit 50", support.CLIName),
			},
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderMutationReport(os.Stdout, report)
	}

	// Stream directly to stdout for shell pipelines.
	_, err = os.Stdout.Write(body)
	return err
}

func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// parseEnvironment accepts either a JSON object or a free-text description.
// Free text is wrapped as {"description": <text>} so the API always receives
// a structured object.
func parseEnvironment(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed
	}
	return map[string]interface{}{"description": raw}
}

func issueRows(issues []support.Issue) []string {
	if len(issues) == 0 {
		return []string{"(no matching issues)"}
	}
	rows := make([]string, 0, len(issues)*2)
	for _, issue := range issues {
		title := strings.TrimSpace(issue.Title)
		if title == "" {
			title = "(untitled)"
		}
		priority := defaultIfEmpty(issue.Priority, "medium")
		status := defaultIfEmpty(issue.Status, "unknown")
		issueType := defaultIfEmpty(issue.Type, "issue")
		rows = append(rows, fmt.Sprintf("%s | %s | priority=%s | type=%s | status=%s",
			title, support.ShortID(issue.ID), priority, issueType, status))
		if targets := support.FormatTargets(issue.Targets); targets != "" {
			rows = append(rows, fmt.Sprintf("  targets: %s", targets))
		}
	}
	return rows
}

func describeFilters(status, priority, issueType, targetID, targetType string) string {
	parts := []string{}
	if status != "" {
		parts = append(parts, "status="+status)
	}
	if priority != "" {
		parts = append(parts, "priority="+priority)
	}
	if issueType != "" {
		parts = append(parts, "type="+issueType)
	}
	if targetID != "" {
		parts = append(parts, "target_id="+targetID)
	}
	if targetType != "" {
		parts = append(parts, "target_type="+targetType)
	}
	if len(parts) == 0 {
		return ""
	}
	return "Filters: " + strings.Join(parts, ", ")
}

func nextCommandsForIssue(id string) []string {
	if strings.TrimSpace(id) == "" {
		return []string{
			fmt.Sprintf("%s issue list", support.CLIName),
		}
	}
	return []string{
		fmt.Sprintf("%s issue show %s", support.CLIName, id),
		fmt.Sprintf("%s issue investigate %s", support.CLIName, id),
	}
}

func defaultIfEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func asString(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
