package orchestrator

import (
	"fmt"
	"os"
	"strings"

	"maintenance-orchestrator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `orchestrator` subcommand group covering orchestrator-wide
// operations that are distinct from the root `/health` probe handled by
// cli-core's built-in `status` command. The `overview` subcommand wraps
// `/api/v1/status` (counts + recent activity log); `stop-all` wraps the
// emergency-stop endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "orchestrator",
		Description: "Orchestrator-wide status and controls",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "overview", Aliases: []string{"status"}, Description: "Show orchestrator totals, active count, and recent activity", Run: func(args []string) error { return runOverview(core, args) }},
			{Name: "stop-all", Description: "Deactivate all maintenance scenarios (emergency stop)", Run: func(args []string) error { return runStopAll(core, args) }},
		},
	}
}

func runOverview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("orchestrator overview")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/status", nil)
	if err != nil {
		return err
	}
	var resp support.StatusResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Health: %s", resp.Health),
		fmt.Sprintf("Maintenance state: %s", resp.MaintenanceState),
		fmt.Sprintf("Scenarios: %d total, %d active, %d inactive",
			resp.TotalScenarios, resp.ActiveScenarios, resp.InactiveScenarios),
		fmt.Sprintf("Uptime: %.1fs", resp.Uptime),
	}

	results := make([]string, 0, len(resp.RecentActivity))
	for _, entry := range resp.RecentActivity {
		parts := []string{support.FormatTime(entry.Timestamp), entry.Action}
		if entry.Scenario != "" {
			parts = append(parts, "scenario="+entry.Scenario)
		}
		if entry.Preset != "" {
			parts = append(parts, "preset="+entry.Preset)
		}
		if entry.Message != "" {
			parts = append(parts, entry.Message)
		}
		results = append(results, strings.Join(parts, " | "))
	}
	if len(results) == 0 {
		results = []string{"(no recent activity)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Recent activity",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s scenario list", support.CLIName),
			fmt.Sprintf("%s status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStopAll(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("orchestrator stop-all")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/stop-all", nil, nil)
	if err != nil {
		return err
	}
	var resp support.StopAllResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("Deactivated: %d scenarios", len(resp.Deactivated))}
	if len(resp.Deactivated) > 0 {
		changes = append(changes, "  "+strings.Join(resp.Deactivated, ", "))
	}

	report := cliapp.MutationReport{
		Result:      []string{"Emergency stop complete"},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s orchestrator overview", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
