package metrics

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `web-console metrics` as a flat read-only command.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Metrics",
		Commands: []cliapp.Command{
			{
				Name:        "metrics",
				Description: "Show runtime metrics (active sessions, totals, TTS/voice counters)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/metrics", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Runtime metrics"},
		ResultsHeading: "Values",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s events", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
