package workflows

import (
	"fmt"
	"strings"
	"time"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

func runList(ctx *appctx.Context, args []string) error {
	folderFilter := ""
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--folder":
			if i+1 >= len(args) {
				return fmt.Errorf("--folder requires a value")
			}
			folderFilter = args[i+1]
			i++
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	workflows, raw, err := listWorkflows(ctx)
	if err != nil {
		return err
	}
	if jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Println("Workflows")
	fmt.Println("=========")

	if len(workflows) == 0 {
		fmt.Println("No workflows found")
		return nil
	}

	for _, workflow := range workflows {
		entry := workflow
		if entry.ID == "" && entry.Workflow != nil {
			entry = *entry.Workflow
		}
		if folderFilter != "" && strings.TrimSpace(folderFilter) != strings.TrimSpace(entry.FolderPath) {
			continue
		}
		created := strings.Split(entry.CreatedAt, "T")
		createdLabel := ""
		if len(created) > 0 && created[0] != "" {
			if _, err := time.Parse("2006-01-02", created[0]); err == nil {
				createdLabel = created[0]
			} else {
				createdLabel = created[0]
			}
		}
		label := entry.Name
		if label == "" {
			label = entry.ID
		}
		if createdLabel != "" {
			fmt.Printf("  %s (%s) - %s\n", label, entry.FolderPath, createdLabel)
		} else {
			fmt.Printf("  %s (%s)\n", label, entry.FolderPath)
		}
	}

	return nil
}
