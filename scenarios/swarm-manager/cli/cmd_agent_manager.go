package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdAgentManagerStatus(args []string) error {
	fs := flag.NewFlagSet("agent-manager status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := a.getV1("/agent-manager/status", nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response AgentManagerStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	printSection("Summary")
	fmt.Printf("  Enabled: %v\n", response.Enabled)
	fmt.Printf("  Available: %v\n", response.Available)
	if response.ProfileID != nil && *response.ProfileID != "" {
		fmt.Printf("  Profile ID: %s\n", *response.ProfileID)
	}
	if response.URL != nil && *response.URL != "" {
		fmt.Printf("  URL: %s\n", *response.URL)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("status"),
		cliCommand("execution", "create", "<backlog-kind>", "<backlog-name>"),
	})
	return nil
}
