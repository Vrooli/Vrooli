package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Run executes the status command.
func Run(client *Client) error {
	body, resp, err := client.Check()
	if err != nil {
		return err
	}

	if resp.Status == "" && len(body) > 0 {
		cliutil.PrintJSON(body)
		return nil
	}

	statusLines := []string{fmt.Sprintf("Service health: %s", defaultValue(resp.Status, "unknown"))}
	if resp.Service != "" {
		statusLines = append(statusLines, fmt.Sprintf("Service: %s", resp.Service))
	}
	if resp.Version != "" {
		statusLines = append(statusLines, fmt.Sprintf("Version: %s", resp.Version))
	}

	triage := []cliapp.TriageGroup{}
	if resp.Operations.LastExecution != nil {
		icon := "✓"
		if !resp.Operations.LastExecution.Success {
			icon = "✗"
		}
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Last Execution",
			Items: []string{fmt.Sprintf("%s %s (phases=%d failed=%d) completed %s",
				icon,
				resp.Operations.LastExecution.Scenario,
				resp.Operations.LastExecution.PhaseSummary.Total,
				resp.Operations.LastExecution.PhaseSummary.Failed,
				defaultValue(resp.Operations.LastExecution.CompletedAt, "n/a"),
			)},
		})
	}

	q := resp.Operations.Queue
	queueItems := []string{
		fmt.Sprintf("pending=%d queued=%d delegated=%d stale=%d running=%d failed=%d", q.Pending, q.Queued, q.Delegated, q.Stale, q.Running, q.Failed),
	}
	if q.OldestQueuedAgeSecs > 0 {
		queueItems = append(queueItems, fmt.Sprintf("oldest queued: %ds", q.OldestQueuedAgeSecs))
	}
	triage = append(triage, cliapp.TriageGroup{Heading: "Queue", Items: queueItems})

	if len(resp.Dependencies) > 0 {
		dependencyItems := make([]string, 0, len(resp.Dependencies))
		for key, value := range resp.Dependencies {
			dependencyItems = append(dependencyItems, fmt.Sprintf("%s: %v", key, value))
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "Dependencies", Items: dependencyItems})
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: statusLines,
		Triage: triage,
		NextSteps: []string{
			"test-genie generate <scenario>",
			"test-genie execute <scenario>",
		},
	})
}

func defaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}
