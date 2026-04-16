package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdAgentManagerStatus(args []string) error {
	fs := flag.NewFlagSet("agent-manager status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Get("/agent-manager/status", nil)
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

func (a *App) cmdAgentManagerRunGet(args []string) error {
	fs := flag.NewFlagSet("agent-manager run-get", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Run ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: agent-manager run-get --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.core.Get("/agent-manager/runs/"+id, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[AgentManagerRunResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	fmt.Printf("  Run %s (%s)\n", response.RunID, response.Status)

	printSection("Details")
	fmt.Printf("  Run ID: %s\n", response.RunID)
	if response.TaskID != "" {
		fmt.Printf("  Task ID: %s\n", response.TaskID)
	}
	fmt.Printf("  Status: %s\n", response.Status)
	fmt.Printf("  Active: %v\n", response.Active)
	if response.StartedAt != "" {
		fmt.Printf("  Started: %s\n", response.StartedAt)
	}
	if response.FinishedAt != "" {
		fmt.Printf("  Finished: %s\n", response.FinishedAt)
	}
	if response.DurationSeconds > 0 {
		fmt.Printf("  Duration: %.1fs\n", response.DurationSeconds)
	}
	if response.ErrorMessage != "" {
		fmt.Printf("  Error: %s\n", response.ErrorMessage)
	}

	if response.Active {
		printCommandListSection("Next Steps", []string{
			cliCommand("agent-manager", "run-stop", "--id", response.RunID),
		})
	}
	return nil
}

func (a *App) cmdAgentManagerRunStop(args []string) error {
	fs := flag.NewFlagSet("agent-manager run-stop", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Run ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: agent-manager run-stop --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.core.Request("POST", "/agent-manager/runs/"+id+"/stop", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[AgentManagerStopResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Run %s: %s\n", response.RunID, response.Status)
	if response.Stopped {
		fmt.Println("  Stop requested successfully.")
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("agent-manager", "run-get", "--id", response.RunID),
	})
	return nil
}
