package stats

import (
	"fmt"
	"os"

	"graph-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes two flat observability commands that wrap scenario-specific
// endpoints distinct from the root /health probe (which cli-core's built-in
// `status` command already covers):
//   - `stats`          → GET /api/v1/stats           (dashboard statistics)
//   - `health-detailed` → GET /api/v1/health/detailed (capability probe)
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Observability",
		Commands: []cliapp.Command{
			{
				Name:        "stats",
				Description: "Show dashboard statistics",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runStats(core, args) },
			},
			{
				Name:        "health-detailed",
				Description: "Show detailed service health probe",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runDetailedHealth(core, args) },
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
	var stats support.DashboardStats
	if err := support.Decode(body, &stats); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Graph Studio dashboard statistics"},
		ResultsHeading: "Stats",
		Results: []string{
			fmt.Sprintf("Total graphs: %d", stats.TotalGraphs),
			fmt.Sprintf("Conversions today: %d", stats.ConversionsToday),
			fmt.Sprintf("Active users: %d", stats.ActiveUsers),
		},
		RetrievalHints: []string{fmt.Sprintf("%s graphs list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDetailedHealth(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("health-detailed")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/health/detailed", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{"Graph Studio detailed health"},
		Triage: []cliapp.TriageGroup{
			{Heading: "Probes", Items: support.MapRows(data)},
		},
		NextSteps: []string{
			fmt.Sprintf("%s status", support.CLIName),
			fmt.Sprintf("%s stats", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
