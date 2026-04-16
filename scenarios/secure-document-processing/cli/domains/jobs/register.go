package jobs

import (
	"fmt"
	"os"
	"strings"

	"secure-document-processing/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `job` subcommand group over GET /api/jobs. The API
// currently exposes a single list endpoint; submit/cancel/get are deferred
// until the API adds those routes.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "job",
		Description: "List document processing jobs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List processing jobs", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("job list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/jobs", nil)
	if err != nil {
		return err
	}
	var jobs []support.ProcessingJob
	if err := support.Decode(body, &jobs); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Processing jobs: %d", len(jobs))},
		ResultsHeading: "Jobs",
		Results:        jobRows(jobs),
		RetrievalHints: []string{fmt.Sprintf("%s job list --json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func jobRows(jobs []support.ProcessingJob) []string {
	if len(jobs) == 0 {
		return []string{"No processing jobs found"}
	}
	rows := make([]string, 0, len(jobs))
	for _, j := range jobs {
		docs := "-"
		if len(j.Documents) > 0 {
			docs = strings.Join(j.Documents, ",")
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | docs=%s | created=%s",
			support.ShortID(j.ID),
			j.JobName,
			j.Status,
			docs,
			support.FormatTimeValue(j.Created)))
	}
	return rows
}
