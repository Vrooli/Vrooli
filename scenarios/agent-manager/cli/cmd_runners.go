package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Runner Command Dispatcher
// =============================================================================

func (a *App) cmdRunner(args []string) error {
	if len(args) == 0 {
		return a.runnerHelp()
	}

	switch args[0] {
	case "list":
		return a.runnerList(args[1:])
	case "probe":
		return a.runnerProbe(args[1:])
	case "models":
		return a.runnerModels(args[1:])
	case "models-update":
		return a.runnerModelsUpdate(args[1:])
	case "help", "-h", "--help":
		return a.runnerHelp()
	default:
		return fmt.Errorf("unknown runner subcommand: %s\n\nRun 'agent-manager runner help' for usage", args[0])
	}
}

func (a *App) runnerHelp() error {
	fmt.Println(`Usage: agent-manager runner <subcommand> [options]

Subcommands:
  list              List all runners and their status
  probe <type>      Probe a specific runner to verify it can respond
  models            Get the model registry for all runners
  models-update     Update the model registry from a JSON file

Runner Types:
  claude-code       Claude Code runner
  codex             OpenAI Codex runner
  opencode          OpenCode runner

Options:
  --json            Output raw JSON

Examples:
  agent-manager runner list
  agent-manager runner probe claude-code
  agent-manager runner models
  agent-manager runner models-update --file registry.json`)
	return nil
}

// =============================================================================
// Runner List
// =============================================================================

func (a *App) runnerList(args []string) error {
	fs := flag.NewFlagSet("runner list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	body, runners, err := a.services.Runners.GetStatus()
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if runners == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(runners) == 0 {
		fmt.Println("No runners found")
		return nil
	}

	fmt.Printf("%-12s  %-9s  %-40s\n", "TYPE", "AVAILABLE", "MESSAGE")
	fmt.Printf("%-12s  %-9s  %-40s\n", strings.Repeat("-", 12), strings.Repeat("-", 9), strings.Repeat("-", 40))
	for _, r := range runners {
		runnerType := formatEnumValue(r.RunnerType, "RUNNER_TYPE_", "-")
		available := "no"
		if r.Available {
			available = "yes"
		}
		message := r.Message
		if len(message) > 40 {
			message = message[:37] + "..."
		}
		fmt.Printf("%-12s  %-9s  %-40s\n", runnerType, available, message)
	}

	return nil
}

// =============================================================================
// Runner Probe
// =============================================================================

func (a *App) runnerProbe(args []string) error {
	fs := flag.NewFlagSet("runner probe", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional runner type first
	var runnerType string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		runnerType = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if runnerType == "" {
		return fmt.Errorf("usage: agent-manager runner probe <type>\n\nValid types: claude-code, codex, opencode")
	}

	body, result, err := a.services.Runners.Probe(runnerType)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	status := "FAILED"
	if result.Success {
		status = "SUCCESS"
	}
	fmt.Printf("Runner:   %s\n", runnerType)
	fmt.Printf("Status:   %s\n", status)
	fmt.Printf("Latency:  %dms\n", result.LatencyMs)
	if result.Error != "" {
		fmt.Printf("Error:    %s\n", result.Error)
	}
	if len(result.Details) > 0 {
		fmt.Println("Details:")
		for k, v := range result.Details {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}

// =============================================================================
// Runner Models
// =============================================================================

func (a *App) runnerModels(args []string) error {
	fs := flag.NewFlagSet("runner models", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := a.services.Runners.GetModels()
	if err != nil {
		return err
	}

	// Models are returned as raw JSON, always pretty-print
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		// Pretty print the JSON for human-readable output
		var prettyJSON map[string]interface{}
		if err := json.Unmarshal(body, &prettyJSON); err == nil {
			formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
			fmt.Println(string(formatted))
		} else {
			cliutil.PrintJSON(body)
		}
	}

	return nil
}

// =============================================================================
// Runner Models Update
// =============================================================================

func (a *App) runnerModelsUpdate(args []string) error {
	fs := flag.NewFlagSet("runner models-update", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	filePath := fs.String("file", "", "Path to JSON file containing model registry (required)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *filePath == "" {
		return fmt.Errorf("--file is required")
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Validate it's valid JSON
	var registry json.RawMessage
	if err := json.Unmarshal(data, &registry); err != nil {
		return fmt.Errorf("invalid JSON in file: %w", err)
	}

	body, err := a.services.Runners.UpdateModels(registry)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Model registry updated successfully")
	return nil
}
