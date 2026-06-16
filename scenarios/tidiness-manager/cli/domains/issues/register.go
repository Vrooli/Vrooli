package issues

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "issues",
		Description: "List and manage tidiness issues",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List issues for a scenario", Run: func(args []string) error { return runList(core, args) }},
			{Name: "resolve", Description: "Mark an issue as resolved", Run: func(args []string) error { return runUpdate(core, args, "resolved") }},
			{Name: "ignore", Description: "Ignore an issue", Run: func(args []string) error { return runUpdate(core, args, "ignored") }},
			{Name: "reopen", Description: "Reopen an issue", Run: func(args []string) error { return runUpdate(core, args, "open") }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("issues list")
	scenario := fs.String("scenario", "", "Scenario name")
	file := fs.String("file", "", "Exact file path filter")
	folder := fs.String("folder", "", "Folder prefix filter")
	category := fs.String("category", "", "Category filter")
	severity := fs.String("severity", "", "Severity filter")
	limit := fs.Int("limit", 50, "Maximum number of issues to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *scenario == "" && fs.NArg() > 0 {
		*scenario = fs.Arg(0)
	}
	if *scenario == "" {
		return fmt.Errorf("scenario is required")
	}

	query := url.Values{}
	query.Set("scenario", *scenario)
	query.Set("limit", strconv.Itoa(*limit))
	if strings.TrimSpace(*file) != "" {
		query.Set("file", *file)
	}
	if strings.TrimSpace(*folder) != "" {
		query.Set("folder", *folder)
	}
	if strings.TrimSpace(*category) != "" && *category != "all" {
		query.Set("category", *category)
	}
	if strings.TrimSpace(*severity) != "" && *severity != "all" {
		query.Set("severity", *severity)
	}

	body, err := core.Get("/agent/issues", query)
	if err != nil {
		return err
	}

	var issues []support.Issue
	if err := support.Decode(body, &issues); err != nil {
		return err
	}
	sort.SliceStable(issues, func(i, j int) bool {
		return support.SeverityRank(issues[i].Severity) < support.SeverityRank(issues[j].Severity)
	})

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", *scenario),
			fmt.Sprintf("Open issues: %d", len(issues)),
		},
		ResultsHeading: "Issues",
		Results:        issueRows(issues),
		RetrievalHints: issueHints(*scenario),
	}

	if stale, err := stalenessInfo(core, *scenario); err == nil && stale != nil && stale.IsStale {
		report.RetrievalHints = append([]string{fmt.Sprintf("Stale data: %s", stale.StaleReason)}, report.RetrievalHints...)
		if stale.RescanCommand != "" {
			report.RetrievalHints = append(report.RetrievalHints, stale.RescanCommand)
		}
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string, status string) error {
	fs := support.NewFlagSet("issues update")
	notes := fs.String("notes", "", "Resolution notes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: issues %s <issue-id> [--notes ...]", status)
	}
	issueID := fs.Arg(0)

	body, err := core.Request("PATCH", "/agent/issues/"+issueID, nil, map[string]interface{}{
		"status":           status,
		"resolution_notes": *notes,
	})
	if err != nil {
		return err
	}

	var response support.IssueUpdateResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Issue %d updated", response.ID),
			fmt.Sprintf("New status: %s", response.Status),
		},
		Changes: []string{
			fmt.Sprintf("Updated at: %s", response.UpdatedAt),
		},
		NextCommand: []string{
			fmt.Sprintf("%s issues list --scenario <scenario>", cliName),
		},
	}
	if strings.TrimSpace(*notes) != "" {
		report.Changes = append(report.Changes, "Notes: "+strings.TrimSpace(*notes))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func issueRows(issues []support.Issue) []string {
	if len(issues) == 0 {
		return []string{"No open issues found"}
	}
	rows := make([]string, 0, len(issues))
	for _, issue := range issues {
		location := issue.FilePath
		if issue.LineNumber != nil {
			location += fmt.Sprintf(":%d", *issue.LineNumber)
		}
		rows = append(rows, fmt.Sprintf("#%d [%s] %s | %s | %s", issue.ID, strings.ToUpper(issue.Severity), location, issue.Category, issue.Title))
	}
	return rows
}

func issueHints(scenario string) []string {
	return []string{
		fmt.Sprintf("%s issues resolve <issue-id> --notes \"...\"", cliName),
		fmt.Sprintf("%s scan %s --type light", cliName, scenario),
	}
}

func stalenessInfo(core *cliapp.ScenarioApp, scenario string) (*support.StalenessInfo, error) {
	query := url.Values{}
	query.Set("scenario", scenario)
	body, err := core.Get("/agent/staleness", query)
	if err != nil {
		return nil, err
	}
	var info support.StalenessInfo
	if err := support.Decode(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
