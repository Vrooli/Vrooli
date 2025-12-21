package playbooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"browser-automation-studio/cli/internal/appctx"
	"browser-automation-studio/cli/internal/output"
	"browser-automation-studio/cli/internal/playbooks"
)

func runOrder(ctx *appctx.Context, args []string) error {
	scenarioDir := ctx.ScenarioRoot
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scenario":
			if i+1 >= len(args) {
				return fmt.Errorf("--scenario requires a path")
			}
			scenarioDir = args[i+1]
			i++
		case "--json":
			jsonOutput = true
		case "--help", "-h":
			fmt.Println("Usage: browser-automation-studio playbooks order [--scenario <dir>] [--json]")
			fmt.Println("  --scenario <dir>  Override scenario directory (defaults to CLI parent)")
			fmt.Println("  --json            Print the processed playbook list as JSON")
			return nil
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if scenarioDir == "" {
		return fmt.Errorf("scenario root not resolved")
	}

	registryPath := filepath.Join(scenarioDir, "test", "playbooks", "registry.json")
	registry, err := playbooks.LoadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("registry not found at %s", registryPath)
	}

	rows := make([]output.PlaybookOrderRow, 0, len(registry.Playbooks))
	for _, entry := range registry.Playbooks {
		reset := entry.Reset
		if reset == "" {
			reset = "project"
		}
		rows = append(rows, output.PlaybookOrderRow{
			Order:        entry.Order,
			Reset:        reset,
			Requirements: len(entry.Requirements),
			File:         entry.File,
			Description:  entry.Description,
		})
	}

	if jsonOutput {
		payload := map[string]any{"playbooks": rows}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	output.RenderPlaybookOrder(os.Stdout, rows)
	return nil
}
