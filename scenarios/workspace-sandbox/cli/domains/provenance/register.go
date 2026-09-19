package provenance

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"

	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "provenance",
		Description: "Inspect pending and committed file history across sandboxes",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "pending", Description: "List pending uncommitted changes", Run: func(args []string) error { return runPending(deps, args) }},
			{Name: "history", Description: "Show file change history", Run: func(args []string) error { return runHistory(deps, args) }},
			{Name: "commit", Description: "Commit all pending changes", Run: func(args []string) error { return runCommit(deps, args) }},
		},
	}
}

func runPending(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("provenance pending", flag.ContinueOnError)
	project := fs.String("project", "", "Filter by project root")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *project != "" {
		query.Set("projectRoot", *project)
	}
	body, err := deps.ScenarioApp().Get("/pending", query)
	if err != nil {
		return err
	}

	var resp support.PendingChangesResult
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Pending files: %d", resp.TotalFiles),
			fmt.Sprintf("Sandboxes with pending changes: %d", len(resp.Summaries)),
		},
		Results:        renderPendingRows(resp.Summaries),
		RetrievalHints: []string{support.CLIName + " provenance commit -m \"Commit message\""},
	}
	if *project != "" {
		report.Summary = append(report.Summary, "Project root: "+*project)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runHistory(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("provenance history", flag.ContinueOnError)
	path := fs.String("path", "", "File path to inspect")
	project := fs.String("project", "", "Project root")
	limit := fs.Int("limit", 0, "Maximum number of history rows")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *path == "" && fs.NArg() > 0 {
		*path = fs.Arg(0)
	}
	if *path == "" {
		return fmt.Errorf("usage: %s provenance history --path FILE [--project PATH] [--limit N] [--json]", support.CLIName)
	}

	query := url.Values{}
	query.Set("path", *path)
	if *project != "" {
		query.Set("projectRoot", *project)
	}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}

	body, err := deps.ScenarioApp().Get("/provenance", query)
	if err != nil {
		return err
	}

	var resp support.FileProvenanceResult
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{"File: " + resp.FilePath, fmt.Sprintf("Changes found: %d", len(resp.Changes))},
		Results:        renderHistoryRows(resp.Changes),
		RetrievalHints: []string{support.CLIName + " provenance pending", support.CLIName + " sandbox inspect <sandbox-id>"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCommit(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("provenance commit", flag.ContinueOnError)
	message := fs.String("message", "", "Commit message")
	project := fs.String("project", "", "Project root")
	actor := fs.String("actor", "", "Actor name")
	fs.StringVar(message, "m", "", "Commit message")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	reqBody := map[string]any{}
	if *message != "" {
		reqBody["commitMessage"] = *message
	}
	if *project != "" {
		reqBody["projectRoot"] = *project
	}
	if *actor != "" {
		reqBody["actor"] = *actor
	}

	body, err := deps.ScenarioApp().Request("POST", "/commit-pending", nil, reqBody)
	if err != nil {
		return err
	}

	var resp support.CommitPendingResult
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("commit failed: %s", resp.ErrorMsg)
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Files committed: %d", resp.FilesCommitted),
		},
		Changes:     []string{},
		NextCommand: []string{support.CLIName + " provenance pending"},
	}
	if resp.CommitHash != "" {
		report.Changes = append(report.Changes, "Commit: "+resp.CommitHash)
	}
	if resp.FilesCommitted == 0 {
		report.Result = []string{"No pending changes to commit"}
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderPendingRows(summaries []support.PendingChangesSummary) []string {
	if len(summaries) == 0 {
		return nil
	}
	rows := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, fmt.Sprintf(
			"%s | owner=%s | files=%d | last applied=%s",
			support.TruncateID(summary.SandboxID),
			support.DisplayOwner(summary.SandboxOwner),
			summary.FileCount,
			summary.LatestApplied.Format("2006-01-02 15:04"),
		))
	}
	return rows
}

func renderHistoryRows(changes []support.AppliedChange) []string {
	if len(changes) == 0 {
		return nil
	}
	rows := make([]string, 0, len(changes))
	for _, change := range changes {
		committed := "-"
		if change.CommittedAt != nil {
			committed = support.TruncateHash(change.CommitHash)
		}
		rows = append(rows, fmt.Sprintf(
			"%s | owner=%s | change=%s | applied=%s | committed=%s",
			support.TruncateID(change.SandboxID),
			support.DisplayOwner(change.SandboxOwner),
			change.ChangeType,
			change.AppliedAt.Format("2006-01-02 15:04"),
			committed,
		))
	}
	return rows
}
