package drift

import (
	"fmt"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Dependency Drift",
		Commands: []cliapp.Command{
			{
				Name:        "drift",
				Description: "Report declared-vs-actual scenario dependency drift",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("drift")
	var scenario string
	var jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Limit drift detection to one scenario")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 1 {
		return fmt.Errorf("usage: %s drift [scenario] [--scenario name] [--json]", support.AppName)
	}
	if len(positionals) == 1 {
		if strings.TrimSpace(scenario) != "" {
			return fmt.Errorf("provide scenario either positionally or with --scenario, not both")
		}
		scenario = positionals[0]
	}

	body, err := core.Get("/drift", support.BuildQuery(map[string]string{"scenarios": scenario}))
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
	findings := support.Maps(resp["findings"])
	results := make([]string, 0, len(findings))
	warnings := 0
	infos := 0
	for _, finding := range findings {
		severity := support.String(finding["severity"])
		switch severity {
		case "WARNING":
			warnings++
		case "INFO":
			infos++
		}
		results = append(results, fmt.Sprintf(
			"%s %s -> %s [%s]",
			severity,
			support.String(finding["scenario"]),
			support.String(finding["dependency"]),
			support.String(finding["kind"]),
		))
	}

	scope := "fleet"
	if strings.TrimSpace(scenario) != "" {
		scope = scenario
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scope: %s", scope),
			fmt.Sprintf("Findings: %d", len(findings)),
			fmt.Sprintf("Warnings: %d", warnings),
			fmt.Sprintf("Infos: %d", infos),
		},
		ResultsHeading: "Drift Findings",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s drift %s --json", support.AppName, strings.TrimSpace(scenario)),
			fmt.Sprintf("%s graph actual --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}
