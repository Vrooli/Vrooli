package stats

import (
	"fmt"
	"os"

	"app-issue-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `app-issue-tracker stats` as a flat command since the API
// surface is a single analytics endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Analytics",
		Commands: []cliapp.Command{
			{
				Name:        "stats",
				Description: "Comprehensive issue analytics and metrics",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runStats(core, args) },
			},
		},
	}
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/stats", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	stats, _ := payload["stats"].(map[string]interface{})
	results := support.MapRows(stats)

	report := cliapp.ListReport{
		Summary:        []string{"Issue tracker statistics"},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s app list", support.CLIName),
			fmt.Sprintf("%s issue list --status open", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
