package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const UsageLine = "test-genie status"

// HelpText returns the framework-rendered help body for the status command.
func HelpText() string {
	return `Show Test Genie health and the latest execution summary.`
}

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
			"test-genie execute <scenario>",
			"test-genie remediate <scenario> --execution <uuid> --findings <stable-id> --role <role-ref>",
		},
	})
}

func defaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}
