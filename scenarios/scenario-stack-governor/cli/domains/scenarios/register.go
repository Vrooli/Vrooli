package scenarios

import (
	"fmt"
	"sort"

	"scenario-stack-governor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "Scenario discovery for targeted rule runs and fixes",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List scenarios visible to the governor", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenarios list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp support.ScenariosResponse
	if err := support.GetJSON(core, "/scenarios", nil, &resp); err != nil {
		return err
	}
	sort.Strings(resp.Scenarios)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenarios discovered: %d", len(resp.Scenarios))},
		Results:        resp.Scenarios,
		RetrievalHints: []string{"scenario-stack-governor run --scenario <scenario>", "scenario-stack-governor fix --scenario <scenario> --dry-run"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}
