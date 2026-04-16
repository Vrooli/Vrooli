package agent

import (
	"fmt"
	"os"
	"strings"

	"app-issue-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `agent` subcommand group covering agent discovery and
// runner settings.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "agent",
		Description: "List AI investigation agents and manage runner settings",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available agents", Run: func(args []string) error { return runList(core, args) }},
			{Name: "settings", Description: "Show current agent-manager settings", Run: func(args []string) error { return runSettings(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/agents", nil)
	if err != nil {
		return err
	}
	var data support.AgentsData
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Agents: %d", data.Count)}
	if data.Runner != "" {
		summary = append(summary, fmt.Sprintf("Active runner: %s", data.Runner))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Agents",
		Results:        agentRows(data.Agents),
		RetrievalHints: []string{
			fmt.Sprintf("%s agent settings", support.CLIName),
			fmt.Sprintf("%s issue investigate <issue-id> --agent <agent-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSettings(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agent settings")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/agent/settings", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	settings, _ := payload["settings"].(map[string]interface{})
	results := support.MapRows(settings)

	report := cliapp.ListReport{
		Summary:        []string{"Agent-manager settings"},
		ResultsHeading: "Fields",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s agent list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func agentRows(agents []support.Agent) []string {
	if len(agents) == 0 {
		return []string{"(no agents available)"}
	}
	rows := make([]string, 0, len(agents)*3)
	for _, a := range agents {
		name := strings.TrimSpace(a.DisplayName)
		if name == "" {
			name = a.Name
		}
		rows = append(rows, fmt.Sprintf("%s (%s)", name, a.ID))
		if a.Description != "" {
			rows = append(rows, fmt.Sprintf("  %s", a.Description))
		}
		if len(a.Capabilities) > 0 {
			rows = append(rows, fmt.Sprintf("  capabilities: %s", strings.Join(a.Capabilities, ", ")))
		}
		state := "inactive"
		if a.IsActive {
			state = "active"
		}
		rows = append(rows, fmt.Sprintf("  status: %s | success: %.1f%% (%d/%d)", state, a.SuccessRate, a.SuccessfulRuns, a.TotalRuns))
	}
	return rows
}
