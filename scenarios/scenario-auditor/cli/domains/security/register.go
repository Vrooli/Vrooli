package security

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"scenario-auditor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "security",
		Description: "Security scans, findings, and job inspection",
		Subcommands: []cliapp.Command{
			{Name: "scan", NeedsAPI: true, Description: "Start a security scan", Run: func(args []string) error { return runScan(core, args) }},
			{Name: "status", NeedsAPI: true, Description: "Get security scan job status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "summary", NeedsAPI: true, Description: "Get security scan summary", Run: func(args []string) error { return runSummary(core, args) }},
			{Name: "cancel", NeedsAPI: true, Description: "Cancel a security scan job", Run: func(args []string) error { return runCancel(core, args) }},
			{Name: "vulnerabilities", NeedsAPI: true, Description: "List cached vulnerabilities", Run: func(args []string) error { return runVulnerabilities(core, args) }},
		},
	}
}

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("security scan", flag.ContinueOnError)
	scanType := fs.String("type", "quick", "Scan type: quick, full, or targeted")
	var checks cliutil.StringList
	fs.Var(&checks, "check", "Scanner/rule to target (repeatable)")
	checkCSV := fs.String("checks", "", "Comma-separated targeted checks")
	includeUnstable := fs.Bool("include-unstable", false, "Include unstable checks")
	skipValidation := fs.Bool("skip-test-validation", false, "Skip scanner test validation")
	wait := fs.Bool("wait", false, "Wait for scan completion")
	interval := fs.Duration("interval", 2*time.Second, "Polling interval when --wait is set")
	timeout := fs.Duration("timeout", 10*time.Minute, "Maximum wait time when --wait is set")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	scenarioName := "all"
	if fs.NArg() > 0 {
		scenarioName = fs.Arg(0)
	}
	payload := map[string]any{
		"type":                 *scanType,
		"targeted_checks":      support.ParseMultiValue(*checkCSV, checks.Values()),
		"include_unstable":     *includeUnstable,
		"skip_test_validation": *skipValidation,
	}

	var startResp map[string]any
	if err := support.RequestJSON(core, "POST", "/scenarios/"+scenarioName+"/scan", nil, payload, &startResp); err != nil {
		return err
	}

	if !*wait {
		report := cliapp.MutationReport{
			Result: []string{"Security scan started", "Job ID: " + support.StringValue(startResp["job_id"])},
			Changes: []string{
				fmt.Sprintf("Scenario: %s", scenarioName),
				fmt.Sprintf("Type: %s", *scanType),
			},
			NextCommand: []string{
				"scenario-auditor security status " + support.StringValue(startResp["job_id"]),
				"scenario-auditor security summary " + support.StringValue(startResp["job_id"]),
			},
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, startResp)
		}
		return support.PrintMutation(false, report)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	jobID := support.StringValue(startResp["job_id"])
	finalResp, err := support.WaitForStatus(ctx, *interval, func() (map[string]any, error) {
		var status map[string]any
		err := support.GetJSON(core, "/scenarios/scan/jobs/"+jobID, nil, &status)
		return status, err
	}, "status")
	if err != nil {
		return err
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, finalResp)
	}
	return support.PrintOperational(false, jobStatusReport("Security scan", finalResp))
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("security status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: security status <job-id> [--json]")
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/scenarios/scan/jobs/"+fs.Arg(0), nil, &resp); err != nil {
		return err
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, jobStatusReport("Security scan", resp))
}

func runSummary(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("security summary", flag.ContinueOnError)
	limit := fs.Int("limit", 20, "Maximum findings to include")
	minSeverity := fs.String("min-severity", "info", "Minimum severity filter")
	groupBy := fs.String("group-by", "", "Optional grouping (rule)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: security summary <job-id> [--limit N] [--min-severity LEVEL] [--group-by rule] [--json]")
	}

	query := map[string][]string{
		"limit":        {fmt.Sprintf("%d", *limit)},
		"min_severity": {*minSeverity},
	}
	if *groupBy != "" {
		query["group_by"] = []string{*groupBy}
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/scenarios/scan/jobs/"+fs.Arg(0)+"/summary", query, &resp); err != nil {
		return err
	}

	summary := support.MapValue(resp["summary"])
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Job ID: %s", fs.Arg(0))},
		ResultsHeading: "Summary",
		Results:        support.PrettifyMapLines(summary),
		RetrievalHints: []string{"scenario-auditor security status " + fs.Arg(0), "scenario-auditor security vulnerabilities"},
	}
	if groups := support.SliceMaps(resp["groups"]); len(groups) > 0 {
		for _, group := range groups {
			report.Results = append(report.Results, support.PrettifyMapLines(group)...)
		}
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runCancel(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("security cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: security cancel <job-id> [--json]")
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/scenarios/scan/jobs/"+fs.Arg(0)+"/cancel", nil, map[string]any{}, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Security scan cancellation requested"},
		Changes:     support.PrettifyMapLines(resp),
		NextCommand: []string{"scenario-auditor security status " + fs.Arg(0)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func runVulnerabilities(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("security vulnerabilities", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario to filter by")
	includeStats := fs.Bool("include-stats", false, "Include aggregate statistics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := map[string][]string{}
	if *scenario != "" {
		query["scenario"] = []string{*scenario}
	}
	if *includeStats {
		query["include_stats"] = []string{"true"}
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/vulnerabilities", query, &resp); err != nil {
		return err
	}

	vulns := support.SliceMaps(resp["vulnerabilities"])
	results := make([]string, 0, len(vulns))
	for _, vuln := range vulns {
		results = append(results, fmt.Sprintf("%s [%s] %s:%d - %s", support.StringValue(vuln["rule_id"]), support.StringValue(vuln["severity"]), support.StringValue(vuln["file_path"]), support.IntValue(vuln["line_number"]), support.StringValue(vuln["title"])))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Vulnerabilities: %d", len(vulns)),
		},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor security scan all --wait", "scenario-auditor system status"},
	}
	if stats := support.MapValue(resp["stats"]); stats != nil {
		report.Summary = append(report.Summary, support.PrettifyMapLines(stats)...)
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func jobStatusReport(label string, resp map[string]any) cliapp.OperationalReport {
	return cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("%s job: %s", label, support.StringValue(resp["id"])),
			fmt.Sprintf("Status: %s", support.StringValue(resp["status"])),
			fmt.Sprintf("Scenario: %s", support.StringValue(resp["scenario"])),
			fmt.Sprintf("Processed scenarios: %d/%d", support.IntValue(resp["processed_scenarios"]), support.IntValue(resp["total_scenarios"])),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Progress",
				Items: support.FormatKV(resp,
					"scan_type",
					"processed_files",
					"total_files",
					"current_scenario",
					"current_scanner",
					"message",
					"error",
				),
			},
		},
		NextSteps: []string{
			"scenario-auditor security summary " + support.StringValue(resp["id"]),
			"scenario-auditor security cancel " + support.StringValue(resp["id"]),
		},
	}
}
