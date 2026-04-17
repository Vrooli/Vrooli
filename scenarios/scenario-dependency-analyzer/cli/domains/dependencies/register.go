package dependencies

import (
	"fmt"
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Dependencies",
		Commands: []cliapp.Command{
			{
				Name:        "list",
				Aliases:     []string{"dependencies"},
				Description: "List dependencies for one scenario",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("list")
	var depType string
	var jsonOutput bool
	fs.StringVar(&depType, "type", "", "Filter by resources, scenarios, or workflows")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s list <scenario> [--type resources|scenarios|workflows] [--json]", support.AppName)
	}
	scenario := positionals[0]
	query := support.BuildQuery(map[string]string{"type": depType})
	body, err := core.Get("/scenarios/"+scenario+"/dependencies", query)
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
	results := make([]string, 0)
	for _, group := range []struct {
		label string
		key   string
	}{
		{label: "Resources", key: "resources"},
		{label: "Scenarios", key: "scenarios"},
		{label: "Shared workflows", key: "shared_workflows"},
	} {
		for _, item := range support.Maps(resp[group.key]) {
			results = append(results, fmt.Sprintf("%s - %s", group.label, dependencyLine(item)))
		}
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", scenario),
			fmt.Sprintf("Filter: %s", firstNonEmpty(depType, "all")),
			fmt.Sprintf("Transitive depth: %d", support.Int(resp["transitive_depth"])),
		},
		ResultsHeading: "Dependencies",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s analyze %s", support.AppName, scenario),
			fmt.Sprintf("%s deployment %s", support.AppName, scenario),
		},
	}
	return support.PrintList(false, report, nil)
}

func dependencyLine(item map[string]interface{}) string {
	name := firstNonEmpty(support.String(item["dependency_name"]), support.String(item["name"]))
	kind := firstNonEmpty(support.String(item["dependency_type"]), support.String(item["type"]))
	line := name
	if kind != "" {
		line += " [" + kind + "]"
	}
	if support.Bool(item["required"]) {
		line += " required"
	}
	if purpose := support.String(item["purpose"]); purpose != "" {
		line += " - " + purpose
	}
	return line
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
