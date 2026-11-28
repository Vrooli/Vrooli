package fixes

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
		Name:        "fixes",
		Description: "Automated fix configuration and job control",
		Subcommands: []cliapp.Command{
			{Name: "config", NeedsAPI: true, Description: "Show automated fix configuration", Run: func(args []string) error { return runConfig(core, args) }},
			{Name: "enable", NeedsAPI: true, Description: "Enable automated fixes", Run: func(args []string) error { return runEnable(core, args) }},
			{Name: "disable", NeedsAPI: true, Description: "Disable automated fixes", Run: func(args []string) error { return runDisable(core, args) }},
			{Name: "apply", NeedsAPI: true, Description: "Start an automated fix job for one scenario", Run: func(args []string) error { return runApply(core, args) }},
			{Name: "jobs", NeedsAPI: true, Description: "List active automated fix jobs", Run: func(args []string) error { return runJobs(core, args) }},
			{Name: "job", NeedsAPI: true, Description: "Get one automated fix job", Run: func(args []string) error { return runJob(core, args) }},
			{Name: "cancel", NeedsAPI: true, Description: "Cancel an automated fix job", Run: func(args []string) error { return runCancel(core, args) }},
			{Name: "history", NeedsAPI: true, Description: "Show automated fix history", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "rollback", NeedsAPI: true, Description: "Mark a fix as rolled back", Run: func(args []string) error { return runRollback(core, args) }},
		},
	}
}

func runConfig(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes config", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/fix/config", nil, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Automated fixes enabled: %t", support.BoolValue(resp["enabled"])),
			fmt.Sprintf("Strategy: %s", support.StringValue(resp["strategy"])),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Configuration", Items: support.PrettifyMapLines(resp)},
		},
		NextSteps: []string{
			"scenario-auditor fixes enable",
			"scenario-auditor fixes jobs",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintOperational(false, report)
}

func runEnable(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes enable", flag.ContinueOnError)
	types := fs.String("types", "", "Comma-separated violation types")
	severities := fs.String("severities", "", "Comma-separated severities")
	strategy := fs.String("strategy", "", "Automation strategy")
	loopDelay := fs.Int("loop-delay", 0, "Loop delay in seconds")
	timeout := fs.Int("timeout", 0, "Timeout in seconds")
	maxFixes := fs.Int("max-fixes", 0, "Maximum fixes to apply")
	model := fs.String("model", "", "Model override")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	payload := map[string]any{}
	if values := cliutil.ParseCSV(*types); len(values) > 0 {
		payload["violation_types"] = values
	}
	if values := cliutil.ParseCSV(*severities); len(values) > 0 {
		payload["severities"] = values
	}
	if *strategy != "" {
		payload["strategy"] = *strategy
	}
	if *loopDelay > 0 {
		payload["loop_delay_seconds"] = *loopDelay
	}
	if *timeout > 0 {
		payload["timeout_seconds"] = *timeout
	}
	if *maxFixes > 0 {
		payload["max_fixes"] = *maxFixes
	}
	if *model != "" {
		payload["model"] = *model
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/fix/config/enable", nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Automated fixes enabled"},
		Changes: []string{
			fmt.Sprintf("Enabled: %t", support.BoolValue(resp["enabled"])),
			fmt.Sprintf("Strategy: %s", support.StringValue(resp["strategy"])),
		},
		NextCommand: []string{"scenario-auditor fixes config", "scenario-auditor fixes jobs"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func runDisable(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes disable", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/fix/config/disable", nil, map[string]any{}, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Automated fixes disabled"},
		Changes:     support.PrettifyMapLines(resp),
		NextCommand: []string{"scenario-auditor fixes config"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func runApply(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes apply", flag.ContinueOnError)
	types := fs.String("types", "", "Comma-separated violation types override")
	severities := fs.String("severities", "", "Comma-separated severities override")
	wait := fs.Bool("wait", false, "Wait for job completion")
	interval := fs.Duration("interval", 2*time.Second, "Polling interval when --wait is set")
	timeout := fs.Duration("timeout", 20*time.Minute, "Maximum wait time when --wait is set")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: fixes apply <scenario> [--types csv] [--severities csv] [--wait] [--json]")
	}

	scenario := fs.Arg(0)
	payload := map[string]any{}
	if values := cliutil.ParseCSV(*types); len(values) > 0 {
		payload["violation_types"] = values
	}
	if values := cliutil.ParseCSV(*severities); len(values) > 0 {
		payload["severities"] = values
	}

	var startResp map[string]any
	if err := support.RequestJSON(core, "POST", "/fix/apply/"+scenario, nil, payload, &startResp); err != nil {
		return err
	}

	jobID := support.StringValue(startResp["job_id"])
	if !*wait {
		report := cliapp.MutationReport{
			Result: []string{"Automated fix job started", "Job ID: " + jobID},
			Changes: []string{
				fmt.Sprintf("Scenario: %s", scenario),
			},
			NextCommand: []string{
				"scenario-auditor fixes job " + jobID,
				"scenario-auditor fixes cancel " + jobID,
			},
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, startResp)
		}
		return support.PrintMutation(false, report)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	finalResp, err := support.WaitForStatus(ctx, *interval, func() (map[string]any, error) {
		var jobResp map[string]any
		err := support.GetJSON(core, "/fix/jobs/"+jobID, nil, &jobResp)
		return support.MapValue(jobResp["job"]), err
	}, "status")
	if err != nil {
		return err
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, finalResp)
	}
	return support.PrintOperational(false, jobStatusReport(finalResp))
}

func runJobs(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes jobs", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/fix/jobs", nil, &resp); err != nil {
		return err
	}

	jobs := support.SliceMaps(resp["jobs"])
	results := make([]string, 0, len(jobs))
	for _, job := range jobs {
		results = append(results, fmt.Sprintf("%s [%s] %s - loops=%d attempted=%d", support.StringValue(job["id"]), support.StringValue(job["status"]), support.StringValue(job["scenario"]), support.IntValue(job["loops_completed"]), support.IntValue(job["issues_attempted"])))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active jobs: %d", support.IntValue(resp["count"]))},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor fixes job <job-id>", "scenario-auditor fixes cancel <job-id>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runJob(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes job", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: fixes job <job-id> [--json]")
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/fix/jobs/"+fs.Arg(0), nil, &resp); err != nil {
		return err
	}
	job := support.MapValue(resp["job"])
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, job)
	}
	return support.PrintOperational(false, jobStatusReport(job))
}

func runCancel(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: fixes cancel <job-id> [--json]")
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/fix/jobs/"+fs.Arg(0)+"/cancel", nil, map[string]any{}, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{support.StringValue(resp["message"])},
		Changes:     support.PrettifyMapLines(resp),
		NextCommand: []string{"scenario-auditor fixes job " + fs.Arg(0)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes history", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp map[string]any
	if err := support.GetJSON(core, "/fix/history", nil, &resp); err != nil {
		return err
	}

	fixes := support.SliceMaps(resp["fixes"])
	results := make([]string, 0, len(fixes))
	for _, item := range fixes {
		results = append(results, fmt.Sprintf("%s [%s] %s - %s", support.StringValue(item["id"]), support.StringValue(item["status"]), support.StringValue(item["scenario"]), support.StringValue(item["summary"])))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Historical fixes: %d", len(fixes))},
		Results:        results,
		RetrievalHints: []string{"scenario-auditor fixes rollback <fix-id>", "scenario-auditor fixes jobs"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintList(false, report)
}

func runRollback(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("fixes rollback", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: fixes rollback <fix-id> [--json]")
	}

	var resp map[string]any
	if err := support.RequestJSON(core, "POST", "/fix/rollback/"+fs.Arg(0), nil, map[string]any{}, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{support.StringValue(resp["message"])},
		Changes:     support.PrettifyMapLines(support.MapValue(resp["fix"])),
		NextCommand: []string{"scenario-auditor fixes history"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}
	return support.PrintMutation(false, report)
}

func jobStatusReport(job map[string]any) cliapp.OperationalReport {
	return cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Job: %s", support.StringValue(job["id"])),
			fmt.Sprintf("Status: %s", support.StringValue(job["status"])),
			fmt.Sprintf("Scenario: %s", support.StringValue(job["scenario"])),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Automation",
				Items: support.FormatKV(job,
					"strategy",
					"loops_completed",
					"issues_attempted",
					"max_fixes",
					"model",
					"message",
					"error",
				),
			},
		},
		NextSteps: []string{
			"scenario-auditor fixes jobs",
			"scenario-auditor fixes cancel " + support.StringValue(job["id"]),
		},
	}
}
