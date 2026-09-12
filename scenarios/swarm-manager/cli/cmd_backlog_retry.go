package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// cmdBacklogRetry handles `swarm-manager backlog retry`. Re-dispatches the
// most recent terminal execution for an item as a NEW execution attempt,
// preserving the prior attempt's record. If the item is currently in a
// terminal state (failed/completed/needs_followup), it is reopened to
// in_progress as part of the retry.
func (a *App) cmdBacklogRetry(args []string) error {
	fs := flag.NewFlagSet("backlog retry", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	noteFlag := fs.String("note", "", "Optional informational note (e.g. 'fixed agent-manager bug')")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog retry --kind KIND --name NAME [--note MSG] [--json]\n\n%s", err)
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	payload, err := json.Marshal(map[string]any{
		"note": strings.TrimSpace(*noteFlag),
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/retry", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var response struct {
		NewExecutionID    string `json:"new_execution_id"`
		ParentExecutionID string `json:"parent_execution_id"`
		Status            string `json:"status"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	printSection("Retry Dispatched")
	fmt.Printf("  Item:        %s/%s\n", kind, name)
	fmt.Printf("  Parent:      %s\n", response.ParentExecutionID)
	fmt.Printf("  New attempt: %s (%s)\n", response.NewExecutionID, response.Status)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "get", "--id", response.NewExecutionID),
		cliCommand("execution", "list", "--backlog-kind", kind, "--backlog-name", name),
	})
	return nil
}
