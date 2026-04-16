package jobs

import (
	"fmt"
	"os"

	"data-structurer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `jobs` subcommand group wrapping /api/v1/jobs. Jobs are
// populated by the API when processing requests are queued; the CLI is a
// read-only observer.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "jobs",
		Description: "Inspect processing jobs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List recent processing jobs", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one processing job", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("jobs list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/jobs", nil)
	if err != nil {
		return err
	}
	var resp support.JobListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recent jobs: %d", resp.Count)},
		ResultsHeading: "Jobs",
		Results:        jobRows(resp.Jobs),
		RetrievalHints: []string{
			fmt.Sprintf("%s jobs get <job-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("jobs get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: jobs get <job-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/jobs/"+id, nil)
	if err != nil {
		return err
	}
	var job support.Job
	if err := support.Decode(body, &job); err != nil {
		return err
	}

	priority := string(job.Priority)
	if priority == "" {
		priority = "normal"
	}

	results := []string{
		fmt.Sprintf("ID: %s", job.ID),
		fmt.Sprintf("Schema ID: %s", job.SchemaID),
		fmt.Sprintf("Status: %s", job.Status),
		fmt.Sprintf("Priority: %s", priority),
		fmt.Sprintf("Input type: %s", job.InputType),
		fmt.Sprintf("Batch mode: %t", job.BatchMode),
		fmt.Sprintf("Progress: %d/%d items (failed: %d)", job.ProcessedItems, job.TotalItems, job.FailedItems),
		fmt.Sprintf("Created: %s", support.FormatTimePtr(job.CreatedAt)),
		fmt.Sprintf("Started: %s", support.FormatTimePtr(job.StartedAt)),
		fmt.Sprintf("Completed: %s", support.FormatTimePtr(job.CompletedAt)),
	}
	if job.ErrorMessage != "" {
		results = append(results, fmt.Sprintf("Error: %s", job.ErrorMessage))
	}
	if len(job.ErrorDetails) > 0 {
		results = append(results, "Error details:")
		results = append(results, support.MapRows(job.ErrorDetails)...)
	}
	if len(job.ResultSummary) > 0 {
		results = append(results, "Result summary:")
		results = append(results, support.MapRows(job.ResultSummary)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Job %s (%s)", job.ID, job.Status)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func jobRows(jobs []support.Job) []string {
	if len(jobs) == 0 {
		return []string{"No jobs"}
	}
	rows := make([]string, 0, len(jobs))
	for _, j := range jobs {
		priority := string(j.Priority)
		if priority == "" {
			priority = "normal"
		}
		rows = append(rows, fmt.Sprintf("%s | %s | priority=%s | progress=%d/%d | %s",
			support.ShortID(j.ID), j.Status, priority, j.ProcessedItems, j.TotalItems,
			support.FormatTimePtr(j.CreatedAt)))
	}
	return rows
}
