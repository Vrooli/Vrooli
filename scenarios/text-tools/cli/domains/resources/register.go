package resources

import (
	"fmt"
	"os"

	"text-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `text-tools resources` as a flat command. The API serves the
// scenario-specific /resources endpoint at the root (not under /api/v1), so we
// use core.GetRoot and render the extended resource/metrics payload that the
// built-in `status` command does not surface.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Resources",
		Commands: []cliapp.Command{
			{
				Name:        "resources",
				Description: "Show status and metrics for dependent resources",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.GetRoot("/resources", nil)
	if err != nil {
		return err
	}
	var status support.ResourcesStatus
	if err := support.Decode(body, &status); err != nil {
		return err
	}

	results := []string{}
	if len(status.Resources) == 0 {
		results = append(results, "(no resources reported)")
	} else {
		results = append(results, "=== Resources ===")
		results = append(results, support.MapRows(status.Resources)...)
	}
	if len(status.Metrics) > 0 {
		results = append(results, "=== Metrics ===")
		results = append(results, support.MapRows(status.Metrics)...)
	}

	summary := []string{fmt.Sprintf("Resources reported: %d", len(status.Resources))}
	if status.Timestamp > 0 {
		summary = append(summary, fmt.Sprintf("Timestamp: %d", status.Timestamp))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Resource status",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
