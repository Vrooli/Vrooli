package executions

import (
	"fmt"
	"net/url"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

func runList(ctx *appctx.Context, args []string) error {
	jsonOutput := false
	workflowFilter := ""
	statusFilter := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--workflow":
			if i+1 >= len(args) {
				return fmt.Errorf("--workflow requires a value")
			}
			workflowFilter = args[i+1]
			i++
		case "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			statusFilter = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	query := url.Values{}
	if workflowFilter != "" {
		query.Set("workflow_id", workflowFilter)
	}
	if statusFilter != "" {
		query.Set("status", statusFilter)
	}

	executions, raw, err := listExecutions(ctx, query)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Println("Recent Executions")
	fmt.Println("==================")
	if len(executions) == 0 {
		fmt.Println("No executions found")
		return nil
	}

	for _, exec := range executions {
		date := strings.Split(exec.StartedAt, "T")
		dateLabel := ""
		if len(date) > 0 {
			dateLabel = date[0]
		}
		fmt.Printf("  %s [%s] - %s\n", fallback(exec.ID, "unknown"), fallback(exec.Status, "unknown"), dateLabel)
	}
	return nil
}

func fallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
