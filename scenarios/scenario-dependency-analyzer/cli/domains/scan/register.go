package scan

import (
	"fmt"
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Scanning",
		Commands: []cliapp.Command{
			{
				Name:        "scan",
				Description: "Scan one scenario and optionally apply inferred dependencies",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scan")
	var apply bool
	var applyResources bool
	var applyScenarios bool
	var jsonOutput bool
	fs.BoolVar(&apply, "apply", false, "Apply inferred resources and scenarios")
	fs.BoolVar(&applyResources, "apply-resources", false, "Apply inferred resources")
	fs.BoolVar(&applyScenarios, "apply-scenarios", false, "Apply inferred scenarios")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s scan <scenario> [--apply] [--apply-resources] [--apply-scenarios] [--json]", support.AppName)
	}
	scenario := positionals[0]
	body, err := core.Request("POST", "/scenarios/"+scenario+"/scan", nil, map[string]interface{}{
		"apply":           apply,
		"apply_resources": applyResources,
		"apply_scenarios": applyScenarios,
	})
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

	applied := support.Bool(resp["applied"])
	applySummary := support.Map(resp["apply_summary"])
	resourcesAdded := support.Strings(applySummary["resources_added"])
	scenariosAdded := support.Strings(applySummary["scenarios_added"])

	if applied {
		report := cliapp.MutationReport{
			Result: []string{
				fmt.Sprintf("Applied dependency updates for %s.", scenario),
			},
			Changes: []string{
				fmt.Sprintf("Resources added: %d", len(resourcesAdded)),
				fmt.Sprintf("Scenarios added: %d", len(scenariosAdded)),
			},
			NextCommand: []string{
				fmt.Sprintf("git diff scenarios/%s/.vrooli/service.json", scenario),
				fmt.Sprintf("%s analyze %s --verbose", support.AppName, scenario),
			},
		}
		return support.PrintMutation(false, report, nil)
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scanned scenario: %s", scenario),
			"No dependency updates were written.",
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Detected additions", Items: []string{
				fmt.Sprintf("Resources: %v", resourcesAdded),
				fmt.Sprintf("Scenarios: %v", scenariosAdded),
			}},
		},
		NextSteps: []string{
			fmt.Sprintf("%s scan %s --apply", support.AppName, scenario),
			fmt.Sprintf("%s analyze %s --verbose", support.AppName, scenario),
		},
	}
	return support.PrintOperational(false, report, nil)
}
