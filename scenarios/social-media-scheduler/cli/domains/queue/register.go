package queue

import (
	"fmt"
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the scenario-specific /health/queue endpoint as the flat
// `queue` command. It reports background job queue depths and worker status
// and is distinct from cli-core's built-in `status` (which hits /health).
// /health/queue lives at the API root, not under /api/v1, so we call GetRoot.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Queue",
		Commands: []cliapp.Command{
			{
				Name:        "queue",
				Description: "Show background job queue depths and worker status",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runQueue(core, args) },
			},
		},
	}
}

func runQueue(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("queue")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.GetRoot("/health/queue", nil)
	if err != nil {
		return err
	}

	var health support.QueueHealth
	if err := support.Decode(body, &health); err != nil {
		return err
	}

	statusOK := health.Status == "healthy"
	statusLines := []string{fmt.Sprintf("Queue status: %s", health.Status)}
	if health.Timestamp != "" {
		statusLines = append(statusLines, fmt.Sprintf("Timestamp: %s", support.FormatTime(health.Timestamp)))
	}
	if health.WorkerPID != "" {
		statusLines = append(statusLines, fmt.Sprintf("Worker PID: %s", health.WorkerPID.String()))
	}

	triage := []cliapp.TriageGroup{}
	if len(health.Queues) > 0 {
		items := make([]string, 0, len(health.Queues))
		for name, depth := range health.Queues {
			items = append(items, fmt.Sprintf("%s: %s", name, depth.String()))
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "Queue depths", Items: items})
	}

	report := cliapp.OperationalReport{
		Status: statusLines,
		Triage: triage,
	}
	if !statusOK {
		report.NextSteps = []string{fmt.Sprintf("%s status", support.CLIName)}
	}

	if *jsonOutput {
		if err := cliapp.PrintReportJSON(os.Stdout, report); err != nil {
			return err
		}
	} else if err := cliapp.RenderOperationalReport(os.Stdout, report); err != nil {
		return err
	}

	if !statusOK {
		return fmt.Errorf("queue status: %s", health.Status)
	}
	return nil
}
