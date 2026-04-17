package deployment

import (
	"fmt"
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Deployment",
		Commands: []cliapp.Command{
			{
				Name:        "deployment",
				Description: "Show deployment readiness for one scenario",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deployment")
	var jsonOutput bool
	var refresh bool
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	fs.BoolVar(&refresh, "refresh", false, "Refresh deployment report first")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deployment <scenario> [--refresh] [--json]", support.AppName)
	}
	scenario := positionals[0]
	query := support.BuildQuery(map[string]string{
		"refresh": support.BoolWord(refresh, "true", ""),
	})
	body, err := core.Get("/scenarios/"+scenario+"/deployment", query)
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
			fmt.Sprintf("Scenario: %s", firstNonEmpty(support.String(resp["scenario"]), scenario)),
			fmt.Sprintf("Generated at: %s", support.String(resp["generated_at"])),
			fmt.Sprintf("Dependency count: %d", len(support.Maps(resp["dependencies"]))),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Tier readiness", Items: tierLines(support.Map(resp["aggregates"]))},
			{Heading: "Bundle dependencies", Items: bundleDependencyLines(support.Map(resp["bundle_manifest"]))},
			{Heading: "Bundle files", Items: bundleFileLines(support.Map(resp["bundle_manifest"]))},
		},
		NextSteps: []string{
			fmt.Sprintf("%s dag export %s", support.AppName, scenario),
			fmt.Sprintf("%s bundle manifest %s", support.AppName, scenario),
			fmt.Sprintf("%s scan %s --apply", support.AppName, scenario),
		},
	}
	if gaps := support.Map(resp["metadata_gaps"]); gaps != nil && support.Int(gaps["total_gaps"]) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Metadata gaps",
			Items: append([]string{
				fmt.Sprintf("Total gaps: %d", support.Int(gaps["total_gaps"])),
			}, support.Strings(gaps["recommendations"])...),
		})
	}
	return support.PrintOperational(false, report, nil)
}

func tierLines(aggregates map[string]interface{}) []string {
	lines := make([]string, 0, len(aggregates))
	for _, tier := range support.KeysSorted(aggregates) {
		item := support.Map(aggregates[tier])
		if item == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s - fitness %.0f%%, dependencies %d, blockers %v", tier, support.Float(item["fitness_score"])*100, support.Int(item["dependency_count"]), support.Strings(item["blocking_dependencies"])))
	}
	return lines
}

func bundleDependencyLines(bundle map[string]interface{}) []string {
	lines := []string{}
	for _, dep := range support.Maps(bundle["dependencies"]) {
		lines = append(lines, fmt.Sprintf("%s :: %s", support.String(dep["type"]), support.String(dep["name"])))
	}
	return lines
}

func bundleFileLines(bundle map[string]interface{}) []string {
	lines := []string{}
	for _, file := range support.Maps(bundle["files"]) {
		status := "missing"
		if support.Bool(file["exists"]) {
			status = "present"
		}
		lines = append(lines, fmt.Sprintf("%s - %s (%s)", support.String(file["type"]), support.String(file["path"]), status))
	}
	return lines
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
