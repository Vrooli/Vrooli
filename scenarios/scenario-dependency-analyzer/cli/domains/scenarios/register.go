package scenarios

import (
	"fmt"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "List scenarios and inspect one scenario in detail",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List scenarios known to the analyzer", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Get detailed scenario metadata", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenarios list")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/scenarios", nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}
	var resp []map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	results := make([]string, 0, len(resp))
	for _, item := range resp {
		results = append(results, fmt.Sprintf("%s - %s (resources=%d scenarios=%d)", support.String(item["name"]), support.String(item["display_name"]), support.Int(item["resource_count"]), support.Int(item["scenario_count"])))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios indexed: %d", len(resp)),
		},
		ResultsHeading: "Scenarios",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s scenarios get <scenario>", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenarios get")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s scenarios get <scenario> [--json]", support.AppName)
	}
	scenario := positionals[0]
	body, err := core.Get("/scenarios/"+scenario, nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	deps := support.Map(resp["dependencies"])
	drift := support.Map(resp["drift"])
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", support.String(resp["name"])),
			fmt.Sprintf("Last analyzed: %s", support.String(resp["last_analyzed"])),
			fmt.Sprintf("Declared resources: %d", len(support.Maps(deps["resources"]))),
			fmt.Sprintf("Declared scenarios: %d", len(support.Maps(deps["scenarios"]))),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Resource drift", Items: driftLines(support.Map(drift["resources"]))},
			{Heading: "Scenario drift", Items: driftLines(support.Map(drift["scenarios"]))},
		},
		NextSteps: []string{
			fmt.Sprintf("%s analyze %s --verbose", support.AppName, scenario),
			fmt.Sprintf("%s scan %s --apply", support.AppName, scenario),
		},
	}
	return support.PrintOperational(false, report, nil)
}

func driftLines(drift map[string]interface{}) []string {
	if drift == nil {
		return nil
	}
	return []string{
		fmt.Sprintf("Missing: %v", support.Strings(drift["missing"])),
		fmt.Sprintf("Extra: %v", support.Strings(drift["extra"])),
	}
}
