package optimize

import (
	"fmt"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Optimization",
		Commands: []cliapp.Command{
			{
				Name:        "optimize",
				Description: "Get optimization recommendations",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("optimize")
	var optType string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&optType, "type", "all", "Optimization type")
	fs.BoolVar(&apply, "apply", false, "Apply safe optimizations")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	scenario := "all"
	if positionals := fs.Args(); len(positionals) > 0 {
		scenario = positionals[0]
	}
	body, err := core.Request("POST", "/optimize", nil, map[string]interface{}{
		"scenario": scenario,
		"type":     optType,
		"apply":    apply,
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
	resultsMap := support.Map(resp["results"])
	lines := make([]string, 0, len(resultsMap))
	for _, scenarioName := range support.KeysSorted(resultsMap) {
		result := support.Map(resultsMap[scenarioName])
		if result == nil {
			continue
		}
		if errText := support.String(result["error"]); errText != "" {
			lines = append(lines, fmt.Sprintf("%s - error: %s", scenarioName, errText))
			continue
		}
		summary := support.Map(result["summary"])
		count := support.Int(summary["recommendation_count"])
		high := support.Int(summary["high_priority"])
		lines = append(lines, fmt.Sprintf("%s - %d recommendations (%d high priority)", scenarioName, count, high))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario target: %s", scenario),
			fmt.Sprintf("Optimization type: %s", optType),
			fmt.Sprintf("Apply safe changes: %t", apply),
		},
		ResultsHeading: "Recommendations",
		Results:        lines,
		RetrievalHints: []string{
			fmt.Sprintf("%s optimize %s --json", support.AppName, scenario),
			fmt.Sprintf("%s deployment %s", support.AppName, scenario),
		},
	}
	return support.PrintList(false, report, nil)
}
