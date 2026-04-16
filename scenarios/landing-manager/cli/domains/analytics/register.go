package analytics

import (
	"fmt"
	"os"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `analytics` subcommand group for generation analytics:
// `GET /api/v1/analytics/summary` and `GET /api/v1/analytics/events`.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analytics",
		Description: "Generation analytics and event history",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "summary", Description: "Show aggregate generation analytics", Run: func(args []string) error { return runSummary(core, args) }},
			{Name: "events", Description: "Show detailed generation event history", Run: func(args []string) error { return runEvents(core, args) }},
		},
	}
}

func runSummary(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analytics summary")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/analytics/summary", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Generation analytics summary"},
		ResultsHeading: "Summary",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{
			fmt.Sprintf("%s analytics events", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runEvents(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analytics events")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/analytics/events", nil)
	if err != nil {
		return err
	}
	var events []map[string]interface{}
	if err := support.Decode(body, &events); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Events: %d", len(events))},
		ResultsHeading: "Events",
		Results:        eventRows(events),
		RetrievalHints: []string{
			fmt.Sprintf("%s analytics summary", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func eventRows(events []map[string]interface{}) []string {
	if len(events) == 0 {
		return []string{"(no events recorded)"}
	}
	rows := make([]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, support.RenderValue(e))
	}
	return rows
}
