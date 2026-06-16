package standards

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"scenario-auditor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "standards",
		Description: "Standards scans, violations, and job inspection",
		Subcommands: []cliapp.Command{
			{Name: "scan", NeedsAPI: true, Description: "Start a standards scan", Run: func(args []string) error { return runScan(core, args) }},
			{Name: "status", NeedsAPI: true, Description: "Get standards scan job status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "summary", NeedsAPI: true, Description: "Get standards scan summary", Run: func(args []string) error { return runSummary(core, args) }},
			{Name: "cancel", NeedsAPI: true, Description: "Cancel a standards scan job", Run: func(args []string) error { return runCancel(core, args) }},
			{Name: "violations", NeedsAPI: true, Description: "List cached standards violations", Run: func(args []string) error { return runViolations(core, args) }},
		},
	}
}

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("standards scan", flag.ContinueOnError)
	scanType := fs.String("type", "full", "Scan type: quick, full, or targeted")
	var rules cliutil.StringList
	fs.Var(&rules, "rule", "Rule ID to include (repeatable)")
	ruleCSV := fs.String("rules", "", "Comma-separated rule IDs")
	forceDisabled := fs.Bool("force-disabled", false, "Include disabled targeted rules")
	wait := fs.Bool("wait", false, "Wait for scan completion")
	interval := fs.Duration("interval", 2*time.Second, "Polling interval when --wait is set")
	timeout := fs.Duration("timeout", 10*time.Minute, "Maximum wait time when --wait is set")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	scenarioName := "all"
	if fs.NArg() > 0 {
		scenarioName = strings.TrimSpace(fs.Arg(0))
	}
	payload := map[string]any{
		"type":           *scanType,
		"standards":      support.ParseMultiValue(*ruleCSV, rules.Values()),
		"force_disabled": *forceDisabled,
	}
	if scenarioName != "all" {
		if resolved := support.ResolveScenarioPath(scenarioName); resolved != "" {
			payload["scenario_path"] = resolved
		}
	}

	var startResp map[string]any
	if err := support.RequestJSON(core, "POST", "/standards/check/"+scenarioName, nil, payload, &startResp); err != nil {
		return err
	}

	if !*wait {
		report := cliapp.MutationReport{
			Result: []string{"Standards scan started", "Job ID: " + support.StringValue(startResp["job_id"])},
			Changes: []string{
				fmt.Sprintf("Scenario: %s", scenarioName),
				fmt.Sprintf("Type: %s", *scanType),
			},
			NextCommand: []string{
				"scenario-auditor standards status " + support.StringValue(startResp["job_id"]),
				"scenario-auditor standards summary " + support.StringValue(startResp["job_id"]),
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
		err := support.GetJSON(core, "/standards/check/jobs/"+jobID, nil, &status)
		return status, err
	}, "status")
	if err != nil {
		return err
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, finalResp)
	}
	return support.PrintOperational(false, jobStatusReport("Standards scan", finalResp))
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("standards status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: standards status <job-id> [--json]")
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/standards/check/jobs/"+fs.Arg(0), nil, &resp); err != nil {
		return err
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, jobStatusReport("Standards scan", resp))
}

func runSummary(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("standards summary", flag.ContinueOnError)
	limit := fs.Int("limit", 20, "Maximum findings to include")
	minSeverity := fs.String("min-severity", "info", "Minimum severity filter")
	groupBy := fs.String("group-by", "", "Optional grouping (rule)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: standards summary <job-id> [--limit N] [--min-severity LEVEL] [--group-by rule] [--json]")
	}

	query := map[string][]string{
		"limit":        {fmt.Sprintf("%d", *limit)},
		"min_severity": {*minSeverity},
	}
	if *groupBy != "" {
		query["group_by"] = []string{*groupBy}
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/standards/check/jobs/"+fs.Arg(0)+"/summary", query, &resp); err != nil {
		return err
	}

	summary := support.MapValue(resp["summary"])
	if assessment := support.MapValue(summary["assessment"]); len(assessment) > 0 {
		if local := support.MapValue(assessment["local"]); len(local) > 0 {
			summary["local_maturity"] = fmt.Sprintf("current=%s next=%s", support.StringValue(local["current_level"]), support.StringValue(local["next_level"]))
		}
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Job ID: %s", fs.Arg(0)),
		},
		ResultsHeading: "Summary",
		Results:        support.PrettifyMapLines(summary),
		RetrievalHints: []string{"scenario-auditor standards status " + fs.Arg(0), "scenario-auditor standards violations"},
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
	fs := flag.NewFlagSet("standards cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: standards cancel <job-id> [--json]")
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/standards/check/jobs/"+fs.Arg(0)+"/cancel", nil, map[string]any{}, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Standards scan cancellation requested"},
		Changes:     support.PrettifyMapLines(resp),
		NextCommand: []string{"scenario-auditor standards status " + fs.Arg(0)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func runViolations(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("standards violations", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario to filter by")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := map[string][]string{}
	if *scenario != "" {
		query["scenario"] = []string{*scenario}
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/standards/violations", query, &resp); err != nil {
		return err
	}

	violations := support.SliceMaps(resp["violations"])
	results := make([]string, 0, len(violations))
	for _, violation := range violations {
		results = append(results, fmt.Sprintf("%s [%s] %s:%d - %s", support.StringValue(violation["type"]), support.StringValue(violation["severity"]), support.StringValue(violation["file_path"]), support.IntValue(violation["line_number"]), support.StringValue(violation["description"])))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Violations: %d", len(violations)), support.StringValue(resp["note"])},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor standards scan all --wait", "scenario-auditor fixes enable"},
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
					"current_file",
					"message",
					"error",
				),
			},
		},
		NextSteps: []string{
			"scenario-auditor standards summary " + support.StringValue(resp["id"]),
			"scenario-auditor standards cancel " + support.StringValue(resp["id"]),
		},
	}
}
