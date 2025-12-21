package workflows

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

type versionsResponse struct {
	Versions []struct {
		Version           int    `json:"version"`
		ChangeDescription string `json:"change_description"`
	} `json:"versions"`
}

func runVersions(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: browser-automation-studio workflow versions <list|get|restore>")
	}

	sub := args[0]
	switch sub {
	case "list":
		return runVersionsList(ctx, args[1:])
	case "get":
		return runVersionsGet(ctx, args[1:])
	case "restore":
		return runVersionsRestore(ctx, args[1:])
	default:
		return fmt.Errorf("unknown workflow versions command: %s", sub)
	}
}

func runVersionsList(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("workflow ID is required")
	}
	workflowID := args[0]
	args = args[1:]

	limit := 50
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 >= len(args) {
				return fmt.Errorf("--limit requires a value")
			}
			fmt.Sscanf(args[i+1], "%d", &limit)
			i++
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	body, err := listWorkflowVersions(ctx, workflowID, limit)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var parsed versionsResponse
	if json.Unmarshal(body, &parsed) != nil {
		fmt.Println(string(body))
		return nil
	}

	fmt.Println("Workflow Versions")
	fmt.Println("=================")
	for _, version := range parsed.Versions {
		label := strings.TrimSpace(version.ChangeDescription)
		if label != "" {
			fmt.Printf("  v%d - %s\n", version.Version, label)
		} else {
			fmt.Printf("  v%d\n", version.Version)
		}
	}
	return nil
}

func runVersionsGet(ctx *appctx.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: browser-automation-studio workflow versions get <workflow-id> <version>")
	}
	workflowID := args[0]
	version := args[1]

	body, err := getWorkflowVersion(ctx, workflowID, version)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runVersionsRestore(ctx *appctx.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: browser-automation-studio workflow versions restore <workflow-id> <version> [--change-description <text>]")
	}
	workflowID := args[0]
	version := args[1]
	args = args[2:]

	changeDescription := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--change-description":
			if i+1 >= len(args) {
				return fmt.Errorf("--change-description requires a value")
			}
			changeDescription = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	payload := map[string]any{}
	if strings.TrimSpace(changeDescription) != "" {
		payload["change_description"] = changeDescription
	}

	body, err := restoreWorkflowVersion(ctx, workflowID, version, payload)
	if err != nil {
		return err
	}

	fmt.Printf("OK: Workflow restored to version %s\n", version)
	cliutil.PrintJSON(body)
	return nil
}
