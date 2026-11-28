package scenarios

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"scenario-auditor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "Scenario inventory and scenario-level health",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List scenarios known to scenario-auditor", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get one scenario by name", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "health", NeedsAPI: true, Description: "Show health for one scenario", Run: func(args []string) error { return runHealth(core, args) }},
			{Name: "alerts", NeedsAPI: true, Description: "List active health alerts", Run: func(args []string) error { return runAlerts(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scenarios list", flag.ContinueOnError)
	statusFilter := fs.String("status", "", "Filter by scenario status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp struct {
		Scenarios []map[string]any `json:"scenarios"`
		Count     int              `json:"count"`
	}
	if err := support.GetJSON(core, "/scenarios", nil, &resp); err != nil {
		return err
	}

	filtered := make([]map[string]any, 0, len(resp.Scenarios))
	running := 0
	for _, scenario := range resp.Scenarios {
		status := support.StringValue(scenario["status"])
		if strings.EqualFold(status, "running") {
			running++
		}
		if *statusFilter != "" && !strings.EqualFold(status, *statusFilter) {
			continue
		}
		filtered = append(filtered, scenario)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return support.StringValue(filtered[i]["name"]) < support.StringValue(filtered[j]["name"])
	})

	results := make([]string, 0, len(filtered))
	for _, scenario := range filtered {
		results = append(results, fmt.Sprintf("%s [%s] - %d endpoints - %s", support.StringValue(scenario["name"]), support.StringValue(scenario["status"]), support.IntValue(scenario["endpoint_count"]), support.StringValue(scenario["path"])))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios returned: %d", len(filtered)),
			fmt.Sprintf("Running scenarios: %d", running),
		},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor scenarios get <scenario>", "scenario-auditor scenarios health <scenario>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scenarios get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenarios get <name> [--json]")
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/scenarios/"+fs.Arg(0), nil, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario: %s", support.StringValue(resp["name"])), fmt.Sprintf("Status: %s", support.StringValue(resp["status"]))},
		ResultsHeading: "Details",
		Results:        support.PrettifyMapLines(resp),
		RetrievalHints: []string{"scenario-auditor scenarios health " + fs.Arg(0), "scenario-auditor security scan " + fs.Arg(0) + " --wait"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runHealth(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scenarios health", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenarios health <name> [--json]")
	}

	name := fs.Arg(0)
	var resp map[string]any
	if err := support.GetJSON(core, "/scenarios/"+name+"/health", nil, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", support.StringValue(resp["scenario"])),
			fmt.Sprintf("Health score: %.1f", support.FloatValue(resp["health_score"])),
			fmt.Sprintf("Critical vulnerabilities: %d", support.IntValue(resp["critical_vulns"])),
			fmt.Sprintf("Total vulnerabilities: %d", support.IntValue(resp["vulnerabilities"])),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Scenario Health",
				Items: support.FormatKV(resp,
					"status",
					"last_health_check",
				),
			},
		},
		NextSteps: []string{
			"scenario-auditor standards scan " + name + " --wait",
			"scenario-auditor security scan " + name + " --wait",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, report)
}

func runAlerts(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scenarios alerts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp struct {
		Alerts []map[string]any `json:"alerts"`
		Count  int              `json:"count"`
	}
	if err := support.GetJSON(core, "/health/alerts", nil, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Alerts))
	for _, alert := range resp.Alerts {
		results = append(results, fmt.Sprintf("%s [%s/%s] - %s", support.StringValue(alert["title"]), support.StringValue(alert["level"]), support.StringValue(alert["category"]), support.StringValue(alert["action"])))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active alerts: %d", len(resp.Alerts))},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor system status", "scenario-auditor security vulnerabilities"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}
