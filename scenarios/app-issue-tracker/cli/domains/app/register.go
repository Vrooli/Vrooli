package app

import (
	"fmt"
	"os"
	"strings"

	"app-issue-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `app` subcommand group. The tracker exposes per-app issue
// rollups via GET /api/v1/apps; this wraps that endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "app",
		Description: "Inspect per-app issue rollups",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List apps with issue counts", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("app list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/apps", nil)
	if err != nil {
		return err
	}
	var data support.AppsData
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Apps: %d", data.Count)},
		ResultsHeading: "App rollups",
		Results:        appRows(data.Apps),
		RetrievalHints: []string{
			fmt.Sprintf("%s issue list --target-id <app-id>", support.CLIName),
			fmt.Sprintf("%s stats", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func appRows(apps []support.App) []string {
	if len(apps) == 0 {
		return []string{"(no apps with tracked issues)"}
	}
	rows := make([]string, 0, len(apps))
	for _, a := range apps {
		label := strings.TrimSpace(a.DisplayName)
		if label == "" {
			label = a.ID
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | total=%d | open=%d | type=%s | status=%s",
			label, a.ID, a.TotalIssues, a.OpenIssues,
			defaultIfEmpty(a.Type, "unknown"),
			defaultIfEmpty(a.Status, "unknown")))
	}
	return rows
}

func defaultIfEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
