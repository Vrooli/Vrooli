package impact

import (
	"fmt"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Analysis",
		Commands: []cliapp.Command{
			{
				Name:        "impact",
				Description: "Analyze the impact of removing one dependency",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("impact")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s impact <dependency> [--json]", support.AppName)
	}
	dependency := positionals[0]
	body, err := core.Get("/dependencies/"+dependency+"/impact", nil)
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
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Dependency: %s", dependency),
			fmt.Sprintf("Severity: %s", support.String(resp["severity"])),
			fmt.Sprintf("Total affected: %d", support.Int(resp["total_affected"])),
			support.String(resp["impact_summary"]),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Direct dependents", Items: dependentLines(support.Maps(resp["direct_dependents"]))},
			{Heading: "Indirect dependents", Items: dependentLines(support.Maps(resp["indirect_dependents"]))},
			{Heading: "Recommendations", Items: support.Strings(resp["recommendations"])},
		},
		NextSteps: []string{
			fmt.Sprintf("%s list %s", support.AppName, dependency),
			fmt.Sprintf("%s graph combined", support.AppName),
		},
	}
	return support.PrintOperational(false, report, nil)
}

func dependentLines(items []map[string]interface{}) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		name := support.String(item["scenario_name"])
		line := name
		if support.Bool(item["required"]) {
			line += " required"
		}
		if purpose := support.String(item["purpose"]); purpose != "" {
			line += " - " + purpose
		}
		out = append(out, line)
	}
	return out
}
