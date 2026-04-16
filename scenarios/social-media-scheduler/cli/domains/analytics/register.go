package analytics

import (
	"fmt"
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wraps /api/v1/analytics/*. Responses are scenario-specific maps;
// we decode them as generic maps and render via support.MapRows rather than
// locking the CLI to a shape the API may still iterate on.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analytics",
		Description: "Read analytics and engagement metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "overview", Description: "Account-level analytics overview", Run: func(args []string) error {
				return runDateFiltered(core, args, "analytics overview", "/analytics/overview", "Overview")
			}},
			{Name: "platforms", Description: "Per-platform analytics", Run: func(args []string) error {
				return runDateFiltered(core, args, "analytics platforms", "/analytics/platforms", "Platforms")
			}},
			{Name: "post-metrics", Description: "Metrics for a specific post", Run: func(args []string) error { return runPostMetrics(core, args) }},
			{Name: "optimal-times", Description: "Suggested posting times", Run: func(args []string) error { return runOptimalTimes(core, args) }},
		},
	}
}

func runDateFiltered(core *cliapp.ScenarioApp, args []string, cmdName, path, heading string) error {
	fs := support.NewFlagSet(cmdName)
	startDate := fs.String("start-date", "", "Start date filter (YYYY-MM-DD or RFC3339)")
	endDate := fs.String("end-date", "", "End date filter (YYYY-MM-DD or RFC3339)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"start_date": *startDate,
		"end_date":   *endDate,
	})
	body, err := core.Get(path, query)
	if err != nil {
		return err
	}

	var generic map[string]interface{}
	_ = support.Decode(body, &generic)

	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: heading,
		Results:        support.MapRows(generic),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPostMetrics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analytics post-metrics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: analytics post-metrics <post-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/analytics/posts/"+id+"/metrics", nil)
	if err != nil {
		return err
	}
	var generic map[string]interface{}
	_ = support.Decode(body, &generic)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Metrics for post %s", id)},
		ResultsHeading: "Metrics",
		Results:        support.MapRows(generic),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runOptimalTimes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analytics optimal-times")
	platform := fs.String("platform", "", "Platform filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"platform": *platform})
	body, err := core.Get("/analytics/optimal-times", query)
	if err != nil {
		return err
	}
	var generic map[string]interface{}
	_ = support.Decode(body, &generic)

	report := cliapp.ListReport{
		Summary:        []string{"Optimal posting times"},
		ResultsHeading: "Times",
		Results:        support.MapRows(generic),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
