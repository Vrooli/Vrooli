package apps

import (
	"fmt"
	"os"

	"task-planner/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `app` subcommand group. The API only exposes a list
// endpoint (GET /api/apps); other verbs from the bash CLI referenced endpoints
// that do not exist and have been dropped.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "app",
		Description: "List applications known to the task planner",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List applications", Run: func(args []string) error { return runList(core, args) }},
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
	var resp support.AppsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Apps: %d", resp.Count)},
		ResultsHeading: "Apps",
		Results:        appRows(resp.Apps),
		RetrievalHints: []string{
			fmt.Sprintf("%s task list --app-id <app-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func appRows(apps []support.App) []string {
	if len(apps) == 0 {
		return []string{"No applications registered"}
	}
	rows := make([]string, 0, len(apps))
	for _, a := range apps {
		rows = append(rows, fmt.Sprintf("%s (%s) | display=%s | tasks=%d/%d completed",
			a.Name, support.ShortID(a.ID), a.DisplayName, a.CompletedTasks, a.TotalTasks))
	}
	return rows
}
