package status

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes the status command.
func Run(client *Client) error {
	body, resp, err := client.Check()
	if err != nil {
		return err
	}

	fmt.Printf("Status: %s\n", defaultValue(resp.Status, "unknown"))
	if resp.Service != "" {
		fmt.Printf("Service: %s\n", resp.Service)
	}
	if resp.Version != "" {
		fmt.Printf("Version: %s\n", resp.Version)
	}

	if resp.Operations.LastExecution != nil {
		icon := "✓"
		if !resp.Operations.LastExecution.Success {
			icon = "✗"
		}
		fmt.Printf("Last execution: %s %s (phases=%d failed=%d) completed %s\n",
			icon,
			resp.Operations.LastExecution.Scenario,
			resp.Operations.LastExecution.PhaseSummary.Total,
			resp.Operations.LastExecution.PhaseSummary.Failed,
			defaultValue(resp.Operations.LastExecution.CompletedAt, "n/a"),
		)
	}

	q := resp.Operations.Queue
	fmt.Printf("Queue: pending=%d queued=%d delegated=%d running=%d failed=%d\n", q.Pending, q.Queued, q.Delegated, q.Running, q.Failed)
	if q.OldestQueuedAgeSecs > 0 {
		fmt.Printf("       oldest queued: %ds\n", q.OldestQueuedAgeSecs)
	}

	if len(resp.Dependencies) > 0 {
		fmt.Println("Dependencies:")
		cliutil.PrintJSONMap(resp.Dependencies, 2)
	}

	if resp.Status == "" && len(body) > 0 {
		cliutil.PrintJSON(body)
	}
	return nil
}

func defaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}
