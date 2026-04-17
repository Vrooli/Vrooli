package analyze

import (
	"fmt"
	"scenario-dependency-analyzer/cli/internal/support"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Analysis",
		Commands: []cliapp.Command{
			{
				Name:        "analyze",
				Description: "Analyze dependencies for one scenario or all scenarios",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze")
	var includeTransitive bool
	var jsonOutput bool
	var verbose bool
	var outputFormat string
	fs.BoolVar(&includeTransitive, "transitive", false, "Include transitive dependencies")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	fs.BoolVar(&verbose, "verbose", false, "Show detailed analysis")
	fs.StringVar(&outputFormat, "output", "", "Output format (json)")
	fs.StringVar(&outputFormat, "o", "", "Output format (json)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if outputFormat != "" {
		if outputFormat != "json" {
			return fmt.Errorf("unsupported output format %q; supported: json", outputFormat)
		}
		jsonOutput = true
	}

	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s analyze <scenario|all> [--transitive] [--json] [--verbose]", support.AppName)
	}
	scenario := positionals[0]
	query := support.BuildQuery(map[string]string{
		"include_transitive": support.BoolWord(includeTransitive, "true", "false"),
	})
	body, err := core.Get("/analyze/"+scenario, query)
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

	resources := support.Maps(resp["resources"])
	scenarios := support.Maps(resp["scenarios"])
	workflows := support.Maps(resp["shared_workflows"])
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", firstNonEmpty(support.String(resp["scenario"]), scenario)),
			fmt.Sprintf("Resources: %d", len(resources)),
			fmt.Sprintf("Scenarios: %d", len(scenarios)),
			fmt.Sprintf("Shared workflows: %d", len(workflows)),
			fmt.Sprintf("Transitive analysis: %t", includeTransitive),
		},
		NextSteps: []string{
			fmt.Sprintf("%s list %s", support.AppName, scenario),
			fmt.Sprintf("%s deployment %s", support.AppName, scenario),
			fmt.Sprintf("%s scan %s --apply", support.AppName, scenario),
		},
	}

	if verbose {
		report.Triage = append(report.Triage,
			triageGroup("Resources", resources, func(item map[string]interface{}) string {
				return dependencyLine(item)
			}),
			triageGroup("Scenarios", scenarios, func(item map[string]interface{}) string {
				return dependencyLine(item)
			}),
			triageGroup("Shared Workflows", workflows, func(item map[string]interface{}) string {
				return dependencyLine(item)
			}),
		)
	}

	if diff := support.Map(resp["resource_diff"]); diff != nil {
		report.Triage = append(report.Triage, driftGroup("Resource drift", diff))
	}
	if diff := support.Map(resp["scenario_diff"]); diff != nil {
		report.Triage = append(report.Triage, driftGroup("Scenario drift", diff))
	}

	return support.PrintOperational(false, report, nil)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func triageGroup(heading string, items []map[string]interface{}, format func(map[string]interface{}) string) cliapp.TriageGroup {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, format(item))
	}
	return cliapp.TriageGroup{Heading: heading, Items: lines}
}

func dependencyLine(item map[string]interface{}) string {
	name := firstNonEmpty(support.String(item["dependency_name"]), support.String(item["name"]))
	kind := firstNonEmpty(support.String(item["dependency_type"]), support.String(item["type"]))
	required := support.Bool(item["required"])
	purpose := support.String(item["purpose"])
	line := name
	if kind != "" {
		line += " [" + kind + "]"
	}
	if required {
		line += " required"
	}
	if purpose != "" {
		line += " - " + purpose
	}
	return line
}

func driftGroup(heading string, diff map[string]interface{}) cliapp.TriageGroup {
	items := make([]string, 0, 2)
	missing := support.Strings(diff["missing"])
	extra := support.Strings(diff["extra"])
	if len(missing) > 0 {
		items = append(items, "Missing: "+joinList(missing))
	}
	if len(extra) > 0 {
		items = append(items, "Extra: "+joinList(extra))
	}
	return cliapp.TriageGroup{Heading: heading, Items: items}
}

func joinList(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}
