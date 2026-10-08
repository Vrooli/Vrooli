package metrics

import (
	"fmt"
	"os"

	"graph-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `graph-studio metrics` as a flat command. Metrics is a
// single read-only surface (`GET /api/v1/metrics`) whose payload shape is
// scenario-managed, so results are rendered as sorted key: value pairs.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Metrics",
		Commands: []cliapp.Command{
			{
				Name:        "metrics",
				Description: "Show system metrics and analytics",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runMetrics(core, args) },
			},
		},
	}
}

func runMetrics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/metrics", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"System metrics"},
		ResultsHeading: "Metrics",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
