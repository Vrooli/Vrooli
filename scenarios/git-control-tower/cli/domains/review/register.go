package review

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "review",
		Description: "Run and inspect scenario readiness reviews",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "summary", NeedsAPI: true, Description: "Show readiness review for a scenario", Run: func(args []string) error { return runSummary(core, args) }},
			{Name: "run", NeedsAPI: true, Description: "Run readiness checks and show results", Run: func(args []string) error { return runRun(core, args) }},
			{Name: "status", NeedsAPI: true, Description: "Check status of a review run job", Run: func(args []string) error { return runStatus(core, args) }},
		},
	}
}

func runSummary(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("review summary", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	details := fs.Int("details", 5, "Number of detail items per dimension (0 to disable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: review summary <scenario> [--details=N] [--json]")
	}
	scenario := fs.Arg(0)
	query := url.Values{}
	query.Set("scenarioName", scenario)
	if *details > 0 {
		query.Set("details", fmt.Sprintf("%d", *details))
	}
	body, err := core.Get("/review/summary", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp summaryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printSummary(&resp)
	return nil
}

func runRun(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("review run", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	checks := fs.String("checks", "", "Comma-separated checks to run (tidiness,tests,rules)")
	details := fs.Int("details", 5, "Number of detail items per dimension")
	noWait := fs.Bool("no-wait", false, "Return job ID immediately without waiting")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: review run <scenario> [--checks=LIST] [--details=N] [--no-wait] [--json]")
	}
	scenario := fs.Arg(0)
	req := runRequest{ScenarioName: scenario, Details: *details}
	if *checks != "" {
		req.Checks = strings.Split(*checks, ",")
	}
	body, err := core.Request("POST", "/review/run", nil, req)
	if err != nil {
		return err
	}
	var runResp runResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	if *noWait {
		if *jsonOutput {
			cliutil.PrintJSON(body)
		} else {
			fmt.Printf("Job started: %s\n", runResp.JobID)
		}
		return nil
	}
	jobStatus, err := pollJob(core, runResp.JobID, scenario)
	if err != nil {
		return err
	}
	return printRunResult(jobStatus, *jsonOutput)
}

func pollJob(core *cliapp.ScenarioApp, jobID, scenario string) (*jobStatusResponse, error) {
	fmt.Printf("Running review checks for %s...\n", scenario)
	var jobStatus jobStatusResponse
	for {
		time.Sleep(3 * time.Second)
		statusBody, pollErr := core.Get("/review/run/"+jobID, nil)
		if pollErr != nil {
			return nil, fmt.Errorf("polling job status: %w", pollErr)
		}
		if err := json.Unmarshal(statusBody, &jobStatus); err != nil {
			cliutil.PrintJSON(statusBody)
			return nil, nil
		}
		if jobStatus.Status == "completed" || jobStatus.Status == "failed" {
			break
		}
		printPollProgress(jobStatus.Checks)
	}
	return &jobStatus, nil
}

func printPollProgress(checks map[string]string) {
	var progress []string
	for check, status := range checks {
		progress = append(progress, fmt.Sprintf("%s: %s", check, status))
	}
	fmt.Printf("  %s\n", strings.Join(progress, ", "))
}

func printRunResult(jobStatus *jobStatusResponse, jsonOutput bool) error {
	if jobStatus == nil {
		return nil
	}
	if jsonOutput {
		out, _ := json.Marshal(jobStatus)
		cliutil.PrintJSON(out)
		return nil
	}
	if jobStatus.Status == "failed" {
		report := cliapp.OperationalReport{
			Status: []string{
				fmt.Sprintf("Job: %s", jobStatus.JobID),
				fmt.Sprintf("Status: %s", strings.ToUpper(jobStatus.Status)),
			},
		}
		if jobStatus.Error != "" {
			report.Triage = []cliapp.TriageGroup{{
				Heading: "Failure",
				Items:   []string{jobStatus.Error},
			}}
		}
		return cliapp.RenderOperationalReport(os.Stdout, report)
	}
	if jobStatus.Summary != nil {
		return renderSummary(os.Stdout, jobStatus.Summary)
	}
	return nil
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("review status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: review status <jobId> [--json]")
	}
	jobID := fs.Arg(0)
	body, err := core.Get("/review/run/"+jobID, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp jobStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Job: %s", resp.JobID),
			fmt.Sprintf("Status: %s", strings.ToUpper(resp.Status)),
		},
	}
	if strings.TrimSpace(resp.StartedAt) != "" {
		report.Status = append(report.Status, fmt.Sprintf("Started: %s", resp.StartedAt))
	}
	if len(resp.Checks) > 0 {
		checkNames := make([]string, 0, len(resp.Checks))
		for check := range resp.Checks {
			checkNames = append(checkNames, check)
		}
		sort.Strings(checkNames)
		items := make([]string, 0, len(checkNames))
		for _, check := range checkNames {
			items = append(items, fmt.Sprintf("%s: %s", check, resp.Checks[check]))
		}
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Checks", Items: items})
	}
	if resp.Error != "" {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Error", Items: []string{resp.Error}})
	}
	if err := cliapp.RenderOperationalReport(os.Stdout, report); err != nil {
		return err
	}
	if resp.Summary != nil {
		fmt.Fprintln(os.Stdout)
		return renderSummary(os.Stdout, resp.Summary)
	}
	return nil
}
