package rules

import (
	"fmt"
	"os"
	"strings"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `app-monitor rules` as a flat command since rules is a
// single, read-only surface in the API (`GET /api/v1/rules`). Filter flags
// mirror the bash CLI's behavior: --scenario, --tech-stack, --severity.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Rules",
		Commands: []cliapp.Command{
			{
				Name:        "rules",
				Description: "List interop rules, grouped by severity",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("rules")
	scenario := fs.String("scenario", "", "Filter by scenario (detects tech stack)")
	techStack := fs.String("tech-stack", "", "Explicit tech-stack filter (comma-separated)")
	severity := fs.String("severity", "", "Severity filter (comma-separated)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"scenario":   *scenario,
		"tech_stack": *techStack,
		"severity":   *severity,
	})
	body, err := core.Get("/rules", query)
	if err != nil {
		return err
	}
	var resp support.RulesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if resp.ScenarioName != "" {
		summary = append(summary, fmt.Sprintf("Scenario: %s", resp.ScenarioName))
	}
	if len(resp.TechStack) > 0 {
		summary = append(summary, fmt.Sprintf("Tech stack: %s", strings.Join(resp.TechStack, ", ")))
	}
	summary = append(summary, fmt.Sprintf("Showing %d of %d rules", resp.FilteredCount, resp.TotalCount))

	grouped := groupBySeverity(resp.Rules)
	results := make([]string, 0, len(resp.Rules)*2)
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		items, ok := grouped[sev]
		if !ok {
			continue
		}
		results = append(results, fmt.Sprintf("=== %s ===", strings.ToUpper(sev)))
		results = append(results, items...)
	}
	if len(results) == 0 {
		results = []string{"(no rules matched)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Rules",
		Results:        results,
		RetrievalHints: []string{
			"prompt-manager skill read ui-health",
			fmt.Sprintf("%s rules --severity critical,high", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func groupBySeverity(rules []support.Rule) map[string][]string {
	grouped := map[string][]string{}
	for _, r := range rules {
		sev := r.Severity
		if sev == "" {
			sev = "medium"
		}
		line := fmt.Sprintf("%s [%s] -> %s", r.Name, r.Slot, r.Recommendation)
		if r.SlotFile != "" {
			line += fmt.Sprintf(" (%s)", r.SlotFile)
		}
		grouped[sev] = append(grouped[sev], line)
	}
	return grouped
}
