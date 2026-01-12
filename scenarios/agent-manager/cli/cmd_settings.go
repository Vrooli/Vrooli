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
		return a.settingsHelp()
	}

	switch args[0] {
	case "investigation":
		return a.settingsInvestigation(args[1:])
	case "help", "-h", "--help":
		return a.settingsHelp()
	default:
		return fmt.Errorf("unknown settings subcommand: %s\n\nRun 'agent-manager settings help' for usage", args[0])
	}
}

func (a *App) settingsHelp() error {
	fmt.Println(`Usage: agent-manager settings <subcommand> [options]

Subcommands:
  investigation     Manage investigation settings

Investigation Subcommands:
  investigation get     Get current investigation settings
  investigation update  Update investigation settings from JSON file
  investigation reset   Reset investigation settings to defaults

Options:
  --json            Output raw JSON

Examples:
  agent-manager settings investigation get
  agent-manager settings investigation update --file settings.json
  agent-manager settings investigation reset`)
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
		return a.settingsInvestigationHelp()
	default:
		return fmt.Errorf("unknown investigation subcommand: %s", args[0])
	}
}

func (a *App) settingsInvestigationHelp() error {
	fmt.Println(`Usage: agent-manager settings investigation <subcommand> [options]

Subcommands:
  get       Get current investigation settings
  update    Update investigation settings from JSON file
  reset     Reset investigation settings to defaults

Update File Format:
  {
    "promptTemplate": "...",
    "applyPromptTemplate": "...",
    "defaultDepth": "quick|standard|deep",
    "defaultContext": { ... },
    "investigationTagAllowlist": [ ... ]
  }

Examples:
  agent-manager settings investigation get
  agent-manager settings investigation update --file settings.json
  agent-manager settings investigation reset`)
	return nil
}

// =============================================================================
// Settings Investigation Get
// =============================================================================

func (a *App) settingsInvestigationGet(args []string) error {
	fs := flag.NewFlagSet("settings investigation get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if args != nil {
		if err := fs.Parse(args); err != nil {
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

	if err := fs.Parse(args); err != nil {
		return err
	}

	if !*force {
		fmt.Print("Reset investigation settings to defaults? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)
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
