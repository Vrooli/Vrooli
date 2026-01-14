package preflight

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes preflight subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "run":
		return runPreflight(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		// For backward compatibility, treat unknown args as manifest path
		// if it looks like a file path (contains . or /)
		if len(args) == 1 && (containsAny(args[0], "./")) {
			return runPreflight(client, args)
		}
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud preflight help' for usage", args[0])
	}
}

func containsAny(s string, chars string) bool {
	for _, c := range chars {
		for _, sc := range s {
			if c == sc {
				return true
			}
		}
	}
	return false
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud preflight <command> [arguments]

Commands:
  run <manifest.json>    Run VPS preflight checks for a cloud manifest

Run 'scenario-to-cloud preflight <command> -h' for command-specific options.`)
	return nil
}

func runPreflight(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud preflight run <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.Run(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
