package scenarios

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "Scenario inventory available to secrets-manager",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List scenarios known to the platform", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenarios list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp support.ScenarioListResponse
	if err := support.GetJSON(core, "/scenarios", nil, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Scenarios))
	for _, scenario := range resp.Scenarios {
		results = append(results, fmt.Sprintf("%s | %s | %s", scenario.Name, support.Fallback(scenario.Status, "unknown"), support.Fallback(scenario.Description, "no description")))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenarios: %d", resp.Count)},
		ResultsHeading: "Scenario Inventory",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " deployment readiness --scenario <scenario>", support.CLIName + " campaigns list --scenario <scenario> --include-readiness"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}
