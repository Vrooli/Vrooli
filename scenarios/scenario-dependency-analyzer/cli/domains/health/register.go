package health

import (
	"fmt"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{
				Name:        "health",
				Description: "Check service health and analysis capabilities",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("health")
	var jsonOutput bool
	var detailed bool
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	fs.BoolVar(&detailed, "detailed", false, "Include analysis capability health")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	rootBody, err := core.GetRoot("/health", nil)
	if err != nil {
		return err
	}
	if jsonOutput && !detailed {
		return support.PrintAPIJSON(rootBody)
	}

	var rootResp map[string]interface{}
	if err := support.Decode(rootBody, &rootResp); err != nil {
		return err
	}

	raw := map[string]interface{}{"basic": rootResp}
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Service status: %s", support.String(rootResp["status"])),
			fmt.Sprintf("Ready: %t", support.Bool(rootResp["readiness"])),
			fmt.Sprintf("Service: %s", support.String(rootResp["service"])),
		},
		NextSteps: []string{
			fmt.Sprintf("%s status", support.AppName),
		},
	}

	if detailed {
		analysisBody, err := core.Get("/health/analysis", nil)
		if err != nil {
			return err
		}
		var analysisResp map[string]interface{}
		if err := support.Decode(analysisBody, &analysisResp); err != nil {
			return err
		}
		raw["analysis"] = analysisResp
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Analysis capabilities",
			Items: []string{
				fmt.Sprintf("Status: %s", support.String(analysisResp["status"])),
				fmt.Sprintf("Capabilities: %v", support.Strings(analysisResp["capabilities"])),
				fmt.Sprintf("Scenarios found: %d", support.Int(analysisResp["scenarios_found"])),
				fmt.Sprintf("Resources available: %d", support.Int(analysisResp["resources_available"])),
				fmt.Sprintf("Database status: %s", support.String(analysisResp["database_status"])),
			},
		})
		report.NextSteps = append(report.NextSteps, fmt.Sprintf("%s analyze all --json", support.AppName))
	}

	return support.PrintOperational(jsonOutput, report, raw)
}
