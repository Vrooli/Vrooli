package dashboard

import (
	"fmt"
	"os"

	"product-manager-agent/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `product-manager-agent dashboard` as a flat command since the
// dashboard is a single read-only aggregate endpoint (GET /api/dashboard).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Dashboard",
		Commands: []cliapp.Command{
			{
				Name:        "dashboard",
				Description: "Show product dashboard metrics and recent activity",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runDashboard(core, args) },
			},
		},
	}
}

func runDashboard(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("dashboard")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/dashboard", nil)
	if err != nil {
		return err
	}
	var dash support.Dashboard
	if err := support.Decode(body, &dash); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Active features: %d", dash.Metrics.ActiveFeatures),
		fmt.Sprintf("Sprint progress: %d%%", dash.Metrics.SprintProgress),
		fmt.Sprintf("Team velocity: %g", dash.Metrics.TeamVelocity),
		fmt.Sprintf("Customer NPS: %d", dash.Metrics.CustomerNPS),
		fmt.Sprintf("Completed tasks: %d", dash.Metrics.CompletedTasks),
		fmt.Sprintf("Pending decisions: %d", dash.Metrics.PendingDecisions),
	}
	if len(dash.RecentFeatures) > 0 {
		results = append(results, "", "Recent features:")
		for _, f := range dash.RecentFeatures {
			results = append(results, fmt.Sprintf("  %s (%s) score=%.2f priority=%s",
				f.Name, support.ShortID(f.ID), f.Score, f.Priority))
		}
	}
	if dash.CurrentSprint != nil {
		results = append(results, "", fmt.Sprintf("Current sprint: #%d capacity=%d effort=%d risk=%s",
			dash.CurrentSprint.SprintNumber, dash.CurrentSprint.Capacity,
			dash.CurrentSprint.TotalEffort, dash.CurrentSprint.RiskLevel))
	}
	if dash.Roadmap != nil {
		results = append(results, fmt.Sprintf("Roadmap: %s (v%d) %d features",
			dash.Roadmap.Name, dash.Roadmap.Version, len(dash.Roadmap.Features)))
	}

	report := cliapp.ListReport{
		Summary:        []string{"Product Manager dashboard"},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s features list", support.CLIName),
			fmt.Sprintf("%s sprint current", support.CLIName),
			fmt.Sprintf("%s roadmap get", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
