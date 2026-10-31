package system

import (
	"flag"
	"fmt"
	"os"

	"scenario-auditor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "system",
		Description: "System-wide scenario-auditor operations",
		Subcommands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Show overall auditor system status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "discover", NeedsAPI: true, Description: "Discover scenarios visible to the auditor", Run: func(args []string) error { return runDiscover(core, args) }},
			{Name: "validate-lifecycle", NeedsAPI: true, Description: "Validate lifecycle protection wiring", Run: func(args []string) error { return runValidateLifecycle(core, args) }},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("system status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/system/status", nil, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Overall status: %s", support.StringValue(resp["status"])),
			fmt.Sprintf("Scenarios: %d", support.IntValue(resp["scenarios"])),
			fmt.Sprintf("Vulnerabilities: %d", support.IntValue(resp["vulnerabilities"])),
			fmt.Sprintf("Standards violations: %d", support.IntValue(resp["standards_violations"])),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Scenario Inventory",
				Items:   support.PrettifyMapLines(support.MapValue(resp["scenarios_detail"])),
			},
			{
				Heading: "Vulnerability Posture",
				Items:   support.PrettifyMapLines(support.MapValue(resp["vulnerabilities_detail"])),
			},
			{
				Heading: "Standards Posture",
				Items:   support.PrettifyMapLines(support.MapValue(resp["standards_violations_detail"])),
			},
			{
				Heading: "Scan Status",
				Items:   support.PrettifyMapLines(support.MapValue(resp["scan_status"])),
			},
		},
		NextSteps: []string{
			"scenario-auditor scenarios list",
			"scenario-auditor standards scan all --wait",
			"scenario-auditor security scan all --wait",
		},
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, report)
}

func runDiscover(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("system discover", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp struct {
		Scenarios []map[string]any `json:"scenarios"`
		Count     int              `json:"count"`
	}
	if err := support.RequestJSON(core, "POST", "/system/discover", nil, map[string]any{}, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Scenarios))
	for _, scenario := range resp.Scenarios {
		results = append(results, fmt.Sprintf("%s [%s] - %d endpoints", support.StringValue(scenario["name"]), support.StringValue(scenario["status"]), support.IntValue(scenario["endpoint_count"])))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Discovered scenarios: %d", resp.Count)},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor scenarios get <scenario>", "scenario-auditor scenarios health <scenario>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runValidateLifecycle(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("system validate-lifecycle", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/system/validate-lifecycle", nil, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Lifecycle protection valid: %t", support.BoolValue(resp["valid"])),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Details",
				Items:   []string{support.StringValue(resp["message"])},
			},
		},
		NextSteps: []string{
			"scenario-auditor system status",
			"scenario-auditor scenarios list",
		},
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, report)
}
