package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (a *App) cmdReviewSummary(args []string) error {
	fs := flag.NewFlagSet("review-summary", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	details := fs.Int("details", 5, "Number of detail items per dimension (0 to disable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: review-summary <scenario> [--details=N] [--json]")
	}
	scenario := fs.Arg(0)

	query := url.Values{}
	query.Set("scenarioName", scenario)
	if *details > 0 {
		query.Set("details", fmt.Sprintf("%d", *details))
	}

	body, err := a.core.Get("/review/summary", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp reviewSummaryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	printReviewSummary(&resp)
	return nil
}

func (a *App) cmdReviewRun(args []string) error {
	fs := flag.NewFlagSet("review-run", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	checks := fs.String("checks", "", "Comma-separated checks to run (tidiness,tests,rules)")
	details := fs.Int("details", 5, "Number of detail items per dimension")
	noWait := fs.Bool("no-wait", false, "Return job ID immediately without waiting")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: review-run <scenario> [--checks=LIST] [--details=N] [--no-wait] [--json]")
	}
	scenario := fs.Arg(0)

	req := reviewRunRequest{
		ScenarioName: scenario,
		Details:      *details,
	}
	if *checks != "" {
		req.Checks = strings.Split(*checks, ",")
	}

	body, err := a.core.Request("POST", "/review/run", nil, req)
	if err != nil {
		return err
	}

	var runResp reviewRunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if *noWait {
		return printNoWaitResult(body, runResp.JobID, *jsonOutput)
	}

	jobStatus, err := a.pollReviewJob(runResp.JobID, scenario)
	if err != nil {
		return err
	}

	return printReviewRunResult(jobStatus, *jsonOutput)
}

func printNoWaitResult(body []byte, jobID string, jsonOutput bool) error {
	if jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Printf("Job started: %s\n", jobID)
	}
	return nil
}

func (a *App) pollReviewJob(jobID, scenario string) (*reviewJobStatusResponse, error) {
	fmt.Printf("Running review checks for %s...\n", scenario)
	var jobStatus reviewJobStatusResponse
	for {
		time.Sleep(3 * time.Second)

		statusBody, pollErr := a.core.Get("/review/run/"+jobID, nil)
		if pollErr != nil {
			return nil, fmt.Errorf("polling job status: %w", pollErr)
		}

		if err := json.Unmarshal(statusBody, &jobStatus); err != nil {
			cliutil.PrintJSON(statusBody)
			return nil, nil //nolint:nilnil // matches original behavior
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

func printReviewRunResult(jobStatus *reviewJobStatusResponse, jsonOutput bool) error {
	if jobStatus == nil {
		return nil
	}

	if jsonOutput {
		out, _ := json.Marshal(jobStatus)
		cliutil.PrintJSON(out)
		return nil
	}

	if jobStatus.Status == "failed" {
		fmt.Printf("Review run failed: %s\n", jobStatus.Error)
		return nil
	}

	fmt.Println()
	if jobStatus.Summary != nil {
		printReviewSummary(jobStatus.Summary)
	}
	return nil
}

func (a *App) cmdReviewStatus(args []string) error {
	fs := flag.NewFlagSet("review-status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: review-status <jobId> [--json]")
	}
	jobID := fs.Arg(0)

	body, err := a.core.Get("/review/run/"+jobID, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp reviewJobStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Job: %s\n", resp.JobID)
	fmt.Printf("Status: %s\n", strings.ToUpper(resp.Status))
	fmt.Printf("Started: %s\n", resp.StartedAt)

	if len(resp.Checks) > 0 {
		fmt.Println("Checks:")
		for check, status := range resp.Checks {
			fmt.Printf("  %s: %s\n", check, status)
		}
	}

	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}

	if resp.Summary != nil {
		fmt.Println()
		printReviewSummary(resp.Summary)
	}
	return nil
}
