package rules

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
		Name:        "rules",
		Description: "Rule inventory, toggles, and test execution",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List available rules", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get one rule by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "toggle", NeedsAPI: true, Description: "Enable or disable a rule", Run: func(args []string) error { return runToggle(core, args) }},
			{Name: "test", NeedsAPI: true, Description: "Run embedded tests for a rule", Run: func(args []string) error { return runTest(core, args) }},
			{Name: "scenario-test", NeedsAPI: true, Description: "Run one rule across one or more scenarios", Run: func(args []string) error { return runScenarioTest(core, args) }},
			{Name: "coverage", NeedsAPI: true, Description: "Get rule test coverage summary", Run: func(args []string) error { return runCoverage(core, args) }},
			{Name: "clear-test-cache", NeedsAPI: true, Description: "Clear rule test cache", Run: func(args []string) error { return runClearTestCache(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules list", flag.ContinueOnError)
	category := fs.String("category", "", "Filter by rule category")
	onlyEnabled := fs.Bool("enabled", false, "Show only enabled rules")
	onlyDisabled := fs.Bool("disabled", false, "Show only disabled rules")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := map[string][]string{}
	if *category != "" {
		query["category"] = []string{*category}
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/rules", query, &resp); err != nil {
		return err
	}

	rulesMap := support.MapValue(resp["rules"])
	ids := support.SortedMapKeys(rulesMap)
	results := make([]string, 0, len(ids))
	enabledCount := 0
	for _, id := range ids {
		rule := support.MapValue(rulesMap[id])
		if rule == nil {
			continue
		}
		enabled := support.BoolValue(rule["enabled"])
		if enabled {
			enabledCount++
		}
		if *onlyEnabled && !enabled {
			continue
		}
		if *onlyDisabled && enabled {
			continue
		}

		implementation := support.MapValue(rule["implementation"])
		results = append(results, fmt.Sprintf("%s [%s/%s] enabled=%t implementation=%s", id, support.StringValue(rule["category"]), support.StringValue(rule["severity"]), enabled, support.StringValue(implementation["status"])))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Rules returned: %d", len(results)),
			fmt.Sprintf("Enabled rules in response: %d", enabledCount),
		},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor rules get <rule-id>", "scenario-auditor rules test <rule-id>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: rules get <rule-id> [--json]")
	}

	ruleID := fs.Arg(0)
	var resp map[string]any
	if err := support.GetJSON(core, "/rules/"+ruleID, nil, &resp); err != nil {
		return err
	}

	rule := support.MapValue(resp["rule"])
	execInfo := support.MapValue(resp["execution_info"])

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Rule: %s", support.StringValue(rule["id"])),
			fmt.Sprintf("Enabled: %t", support.BoolValue(rule["enabled"])),
		},
		ResultsHeading: "Details",
		Results: append(
			support.PrettifyMapLines(rule),
			support.PrettifyMapLines(execInfo)...,
		),
		RetrievalHints: []string{
			"scenario-auditor rules test " + ruleID,
			"scenario-auditor rules scenario-test " + ruleID + " --scenario <scenario>",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runToggle(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules toggle", flag.ContinueOnError)
	enable := fs.Bool("enable", false, "Enable the rule")
	disable := fs.Bool("disable", false, "Disable the rule")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: rules toggle <rule-id> --enable|--disable [--json]")
	}
	if *enable == *disable {
		return fmt.Errorf("exactly one of --enable or --disable must be set")
	}

	ruleID := fs.Arg(0)
	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/rules/"+ruleID+"/toggle", nil, map[string]any{"enabled": *enable}, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{support.StringValue(resp["message"])},
		Changes: []string{
			fmt.Sprintf("Rule ID: %s", support.StringValue(resp["rule_id"])),
			fmt.Sprintf("Enabled: %t", support.BoolValue(resp["enabled"])),
		},
		NextCommand: []string{
			"scenario-auditor rules get " + ruleID,
			"scenario-auditor rules list",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func runTest(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules test", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: rules test <rule-id> [--json]")
	}

	ruleID := fs.Arg(0)
	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/rules/"+ruleID+"/test", nil, map[string]any{}, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Rule: %s", support.StringValue(resp["rule_id"])),
			fmt.Sprintf("Passed: %d", support.IntValue(resp["passed"])),
			fmt.Sprintf("Failed: %d", support.IntValue(resp["failed"])),
			fmt.Sprintf("Cached: %t", support.BoolValue(resp["cached"])),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Test Summary",
				Items: append(
					support.FormatKV(resp, "total_tests", "file_hash"),
					support.StringValue(resp["warning"]),
				),
			},
			{
				Heading: "Failing Tests",
				Items:   renderTestFailures(resp),
			},
		},
		NextSteps: []string{
			"scenario-auditor rules get " + ruleID,
			"scenario-auditor rules clear-test-cache " + ruleID,
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, report)
}

func runScenarioTest(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules scenario-test", flag.ContinueOnError)
	var scenarios cliutil.StringList
	fs.Var(&scenarios, "scenario", "Scenario to test against (repeatable)")
	scenarioCSV := fs.String("scenarios", "", "Comma-separated scenarios to test against")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: rules scenario-test <rule-id> --scenario <name> [--scenario <name>] [--json]")
	}

	ruleID := fs.Arg(0)
	targetScenarios := support.ParseMultiValue(*scenarioCSV, scenarios.Values())
	targetScenarios = cliutil.MergeArgs(targetScenarios, fs.Args()[1:])
	if len(targetScenarios) == 0 {
		return fmt.Errorf("at least one scenario must be provided")
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/rules/"+ruleID+"/scenario-test", nil, map[string]any{"scenarios": targetScenarios}, &resp); err != nil {
		return err
	}

	results := support.SliceMaps(resp["results"])
	lines := make([]string, 0, len(results))
	for _, result := range results {
		lines = append(lines, fmt.Sprintf("%s - files=%d violations=%d warning=%s error=%s",
			support.StringValue(result["scenario"]),
			support.IntValue(result["files_scanned"]),
			len(support.SliceValue(result["violations"])),
			support.StringValue(result["warning"]),
			support.StringValue(result["error"]),
		))
	}
	sort.Strings(lines)

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Rule: %s", support.StringValue(resp["rule_id"])),
			fmt.Sprintf("Scenarios evaluated: %d", len(results)),
			fmt.Sprintf("Duration: %s", support.StringValue(resp["duration"])),
		},
		ResultsHeading: "Scenario Results",
		Results:        lines,
		RetrievalHints: []string{"scenario-auditor rules get " + ruleID, "scenario-auditor rules test " + ruleID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runCoverage(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules coverage", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/rules/test-coverage", nil, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        support.PrettifyMapLines(resp),
		ResultsHeading: "Coverage Details",
		Results:        nil,
		RetrievalHints: []string{"scenario-auditor rules list", "scenario-auditor rules test <rule-id>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runClearTestCache(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("rules clear-test-cache", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	path := "/rules/test-cache"
	next := []string{"scenario-auditor rules list"}
	if fs.NArg() > 0 {
		ruleID := fs.Arg(0)
		path = "/rules/" + ruleID + "/test-cache"
		next = []string{"scenario-auditor rules test " + ruleID, "scenario-auditor rules get " + ruleID}
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "DELETE", path, nil, nil, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{support.StringValue(resp["message"])},
		Changes:     support.PrettifyMapLines(resp),
		NextCommand: next,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func renderTestFailures(resp map[string]any) []string {
	tests := support.SliceValue(resp["tests"])
	lines := make([]string, 0)
	for _, item := range tests {
		test := support.MapValue(item)
		if test == nil || support.BoolValue(test["passed"]) {
			continue
		}
		testCase := support.MapValue(test["test_case"])
		line := support.StringValue(testCase["id"])
		if desc := support.StringValue(testCase["description"]); desc != "" {
			line += " - " + desc
		}
		if errText := support.StringValue(test["error"]); errText != "" {
			line += " - " + errText
		}
		lines = append(lines, strings.TrimSpace(line))
	}
	if len(lines) == 0 {
		return []string{"No failing tests reported"}
	}
	sort.Strings(lines)
	return lines
}
