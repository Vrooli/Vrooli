package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Settings Command Dispatcher
// =============================================================================

func (a *App) cmdSettings(args []string) error {
	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "investigation":
		return a.settingsInvestigation(args[1:])
	case "orchestration":
		return a.settingsOrchestration(args[1:])
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown settings subcommand: %s\n\nRun 'agent-manager settings help' for usage", args[0])
	}
}

func (a *App) settingsOrchestration(args []string) error {
	if len(args) == 0 {
		return a.settingsOrchestrationGet(nil)
	}
	switch args[0] {
	case "get":
		return a.settingsOrchestrationGet(args[1:])
	case "update":
		return a.settingsOrchestrationUpdate(args[1:])
	case "reset":
		return a.settingsOrchestrationReset(args[1:])
	default:
		return fmt.Errorf("unknown orchestration subcommand: %s", args[0])
	}
}

func (a *App) settingsOrchestrationGet(args []string) error {
	fs := flag.NewFlagSet("settings orchestration get", flag.ContinueOnError)
	_ = cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Settings.api.Get("/api/v1/orchestration-settings", nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) settingsOrchestrationUpdate(args []string) error {
	fs := flag.NewFlagSet("settings orchestration update", flag.ContinueOnError)
	file := fs.String("file", "", "JSON settings file")
	_ = cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("--file is required")
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("read orchestration settings: %w", err)
	}
	if !json.Valid(data) {
		return fmt.Errorf("orchestration settings file must contain valid JSON")
	}
	body, err := a.services.Settings.api.Request("PUT", "/api/v1/orchestration-settings", nil, json.RawMessage(data))
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) settingsOrchestrationReset(args []string) error {
	fs := flag.NewFlagSet("settings orchestration reset", flag.ContinueOnError)
	_ = cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Settings.api.Request("POST", "/api/v1/orchestration-settings/reset", nil, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

// =============================================================================
// Settings Investigation
// =============================================================================

func (a *App) settingsInvestigation(args []string) error {
	if len(args) == 0 {
		return a.settingsInvestigationGet(nil)
	}

	switch args[0] {
	case "get":
		return a.settingsInvestigationGet(args[1:])
	case "update":
		return a.settingsInvestigationUpdate(args[1:])
	case "reset":
		return a.settingsInvestigationReset(args[1:])
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown investigation subcommand: %s", args[0])
	}
}

// =============================================================================
// Settings Investigation Get
// =============================================================================

func (a *App) settingsInvestigationGet(args []string) error {
	fs := flag.NewFlagSet("settings investigation get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if args != nil {
		if err := cliutil.ParseInterspersed(fs, args); err != nil {
			return err
		}
	}

	body, err := a.services.Settings.GetInvestigation()
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print the JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(body, &prettyJSON); err == nil {
		formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Println(string(formatted))
	} else {
		cliutil.PrintJSON(body)
	}

	return nil
}

// =============================================================================
// Settings Investigation Update
// =============================================================================

func (a *App) settingsInvestigationUpdate(args []string) error {
	fs := flag.NewFlagSet("settings investigation update", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	filePath := fs.String("file", "", "Path to JSON file containing settings (required)")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	var settings json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("invalid JSON in file: %w", err)
	}

	body, err := a.services.Settings.UpdateInvestigation(settings)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Investigation settings updated successfully")
	return nil
}

// =============================================================================
// Settings Investigation Reset
// =============================================================================

func (a *App) settingsInvestigationReset(args []string) error {
	fs := flag.NewFlagSet("settings investigation reset", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	force := fs.Bool("force", false, "Skip confirmation")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if !*force {
		fmt.Print("Reset investigation settings to defaults? [y/N]: ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" && confirm != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	body, err := a.services.Settings.ResetInvestigation()
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Investigation settings reset to defaults")
	return nil
}
