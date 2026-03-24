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
// Response types (mirrors review_model.go from the API)
// ---------------------------------------------------------------------------

type reviewSummaryResponse struct {
	ScenarioName string           `json:"scenarioName"`
	Readiness    string           `json:"readiness"`
	Dimensions   reviewDimensions `json:"dimensions"`
	Capabilities map[string]bool  `json:"capabilities"`
	Timestamp    string           `json:"timestamp"`
}

type reviewDimensions struct {
	CodeQuality *codeQualityDimension `json:"codeQuality,omitempty"`
	Tests       *testsDimension       `json:"tests,omitempty"`
	Standards   *standardsDimension   `json:"standards,omitempty"`
	Visual      *visualDimension      `json:"visual,omitempty"`
	Provenance  *provenanceDimension  `json:"provenance,omitempty"`
}

type codeQualityIssue struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type codeQualityDimension struct {
	Available  bool               `json:"available"`
	Score      float64            `json:"score"`
	Violations int                `json:"violations"`
	Stale      bool               `json:"stale"`
	LastScan   string             `json:"lastScan,omitempty"`
	TopIssues  []codeQualityIssue `json:"topIssues,omitempty"`
}

type testFailure struct {
	Phase          string `json:"phase"`
	Error          string `json:"error,omitempty"`
	Classification string `json:"classification,omitempty"`
	Remediation    string `json:"remediation,omitempty"`
}

type testsDimension struct {
	Available   bool          `json:"available"`
	Passed      bool          `json:"passed"`
	Total       int           `json:"total"`
	PassedCount int           `json:"passedCount"`
	FailedCount int           `json:"failedCount"`
	LastRun     string        `json:"lastRun,omitempty"`
	Failures    []testFailure `json:"failures,omitempty"`
}

type standardsViolationDetail struct {
	FilePath       string `json:"filePath"`
	LineNumber     int    `json:"lineNumber"`
	Title          string `json:"title"`
	Severity       string `json:"severity"`
	Recommendation string `json:"recommendation,omitempty"`
}

type standardsDimension struct {
	Available          bool                       `json:"available"`
	BlockingViolations int                        `json:"blockingViolations"`
	Warnings           int                        `json:"warnings"`
	TotalViolations    int                        `json:"totalViolations"`
	TopViolations      []standardsViolationDetail `json:"topViolations,omitempty"`
}

type visualCaptureMeta struct {
	CapturedAt      string `json:"capturedAt"`
	CommitHash      string `json:"commitHash,omitempty"`
	ScreenshotCount int    `json:"screenshotCount"`
}

type visualDimension struct {
	Available       bool               `json:"available"`
	ScreenshotCount int                `json:"screenshotCount"`
	Stale           bool               `json:"stale"`
	LatestCapture   *visualCaptureMeta `json:"latestCapture,omitempty"`
}

type provenanceDimension struct {
	Available     bool     `json:"available"`
	TracedFiles   int      `json:"tracedFiles"`
	UntracedFiles []string `json:"untracedFiles,omitempty"`
}

type reviewRunRequest struct {
	ScenarioName string   `json:"scenarioName"`
	Checks       []string `json:"checks,omitempty"`
	Details      int      `json:"details,omitempty"`
}

type reviewRunResponse struct {
	JobID string `json:"jobId"`
}

type reviewJobStatusResponse struct {
	JobID     string                 `json:"jobId"`
	Status    string                 `json:"status"`
	Checks    map[string]string      `json:"checks"`
	Summary   *reviewSummaryResponse `json:"summary,omitempty"`
	StartedAt string                 `json:"startedAt"`
	Error     string                 `json:"error,omitempty"`
}

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

	body, err := a.core.APIClient.Get(a.apiPath("/review/summary"), query)
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

	body, err := a.core.APIClient.Request("POST", a.apiPath("/review/run"), nil, req)
	if err != nil {
		return err
	}

	var runResp reviewRunResponse
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

	// Poll until completion.
	fmt.Printf("Running review checks for %s...\n", scenario)
	var jobStatus reviewJobStatusResponse
	for {
		time.Sleep(3 * time.Second)

		statusBody, pollErr := a.core.APIClient.Get(a.apiPath("/review/run/"+runResp.JobID), nil)
		if pollErr != nil {
			return fmt.Errorf("polling job status: %w", pollErr)
		}

		if err := json.Unmarshal(statusBody, &jobStatus); err != nil {
			cliutil.PrintJSON(statusBody)
			return nil
		}

		if jobStatus.Status == "completed" || jobStatus.Status == "failed" {
			break
		}

		// Print progress.
		var progress []string
		for check, status := range jobStatus.Checks {
			progress = append(progress, fmt.Sprintf("%s: %s", check, status))
		}
		fmt.Printf("  %s\n", strings.Join(progress, ", "))
	}

	if *jsonOutput {
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

	body, err := a.core.APIClient.Get(a.apiPath("/review/run/"+jobID), nil)
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

// ---------------------------------------------------------------------------
// Shared output formatting — Operational contract: Status → Triage → Next Steps
// ---------------------------------------------------------------------------

func printReviewSummary(resp *reviewSummaryResponse) {
	// Status
	label := readinessLabel(resp.Readiness)
	fmt.Printf("=== Readiness Review: %s ===\n", resp.ScenarioName)
	fmt.Printf("Status: %s\n", label)
	fmt.Println()

	// Triage: separate needs-attention from passing dimensions.
	var needsAttention []string
	var passing []string

	d := resp.Dimensions

	// Standards
	if d.Standards != nil && d.Standards.Available {
		if d.Standards.BlockingViolations > 0 || d.Standards.Warnings > 0 {
			line := fmt.Sprintf("  Standards — %d blocking, %d warnings (%d total)",
				d.Standards.BlockingViolations, d.Standards.Warnings, d.Standards.TotalViolations)
			for _, v := range d.Standards.TopViolations {
				line += fmt.Sprintf("\n    %s:%d  %s (%s)", v.FilePath, v.LineNumber, v.Title, v.Severity)
				if v.Recommendation != "" {
					line += fmt.Sprintf("\n      -> %s", v.Recommendation)
				}
			}
			needsAttention = append(needsAttention, line)
		} else {
			passing = append(passing, "  Standards — 0 violations")
		}
	}

	// Tests
	if d.Tests != nil && d.Tests.Available {
		if !d.Tests.Passed && d.Tests.Total > 0 {
			line := fmt.Sprintf("  Tests — %d of %d failed", d.Tests.FailedCount, d.Tests.Total)
			for _, f := range d.Tests.Failures {
				line += fmt.Sprintf("\n    %s: %s", f.Phase, f.Error)
				if f.Classification != "" {
					line += fmt.Sprintf(" (classification: %s)", f.Classification)
				}
				if f.Remediation != "" {
					line += fmt.Sprintf("\n      -> %s", f.Remediation)
				}
			}
			needsAttention = append(needsAttention, line)
		} else if d.Tests.Total > 0 {
			line := fmt.Sprintf("  Tests — %d/%d passed", d.Tests.PassedCount, d.Tests.Total)
			if d.Tests.LastRun != "" {
				line += fmt.Sprintf(" (last run: %s)", formatTimestamp(d.Tests.LastRun))
			}
			passing = append(passing, line)
		} else {
			needsAttention = append(needsAttention, "  Tests — no test runs found")
		}
	}

	// Code quality
	if d.CodeQuality != nil && d.CodeQuality.Available {
		if d.CodeQuality.Score < 60 || d.CodeQuality.Stale {
			issue := fmt.Sprintf("  Code quality — %.0f/100", d.CodeQuality.Score)
			if d.CodeQuality.Stale {
				issue += " (stale)"
			}
			for _, i := range d.CodeQuality.TopIssues {
				issue += fmt.Sprintf("\n    %s: %d", i.Category, i.Count)
			}
			needsAttention = append(needsAttention, issue)
		} else {
			line := fmt.Sprintf("  Code quality — %.0f/100", d.CodeQuality.Score)
			if len(d.CodeQuality.TopIssues) > 0 {
				var parts []string
				for _, i := range d.CodeQuality.TopIssues {
					parts = append(parts, fmt.Sprintf("%s: %d", i.Category, i.Count))
				}
				line += " (" + strings.Join(parts, ", ") + ")"
			}
			passing = append(passing, line)
		}
	}

	// Visual
	if d.Visual != nil && d.Visual.Available {
		line := fmt.Sprintf("  Visual — %d screenshots", d.Visual.ScreenshotCount)
		if d.Visual.LatestCapture != nil {
			line += fmt.Sprintf(" (latest: %s", formatTimestamp(d.Visual.LatestCapture.CapturedAt))
			if d.Visual.LatestCapture.CommitHash != "" {
				hash := d.Visual.LatestCapture.CommitHash
				if len(hash) > 7 {
					hash = hash[:7]
				}
				line += fmt.Sprintf(", commit %s", hash)
			}
			line += ")"
		}
		if d.Visual.ScreenshotCount == 0 {
			needsAttention = append(needsAttention, line)
		} else {
			passing = append(passing, line)
		}
	}

	// Provenance
	if d.Provenance != nil && d.Provenance.Available {
		line := fmt.Sprintf("  Provenance — %d traced files", d.Provenance.TracedFiles)
		passing = append(passing, line)
		if len(d.Provenance.UntracedFiles) > 0 {
			needsAttention = append(needsAttention,
				fmt.Sprintf("  Provenance — %d untraced files", len(d.Provenance.UntracedFiles)))
		}
	}

	// Print triage sections
	if len(needsAttention) > 0 {
		fmt.Println("Needs attention:")
		for _, line := range needsAttention {
			fmt.Println(line)
		}
		fmt.Println()
	}

	if len(passing) > 0 {
		fmt.Println("Passing:")
		for _, line := range passing {
			fmt.Println(line)
		}
		fmt.Println()
	}

	// Untraced files detail
	if d.Provenance != nil && len(d.Provenance.UntracedFiles) > 0 {
		fmt.Println("Untraced files:")
		for _, f := range d.Provenance.UntracedFiles {
			fmt.Printf("  %s\n", f)
		}
		fmt.Println()
	}

	// Next steps
	fmt.Println("Next steps:")
	switch resp.Readiness {
	case "green":
		fmt.Println("  All checks pass — scenario appears ready for commit review")
	case "yellow":
		fmt.Printf("  git-control-tower review-run %s    # re-check after fixes\n", resp.ScenarioName)
	case "red":
		fmt.Printf("  git-control-tower review-run %s    # re-check after fixes\n", resp.ScenarioName)
		fmt.Println("  Address the issues above before proceeding")
	}
}

func readinessLabel(r string) string {
	switch r {
	case "green":
		return "GREEN (ready)"
	case "yellow":
		return "YELLOW (needs attention)"
	case "red":
		return "RED (not ready)"
	default:
		return strings.ToUpper(r)
	}
}

func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04")
}
