package manifest

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes manifest subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "validate":
		return runValidate(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud manifest help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud manifest <command> [arguments]

Commands:
  validate <manifest.json>    Validate a cloud manifest JSON file

Run 'scenario-to-cloud manifest <command> -h' for command-specific options.`)
	return nil
}

func runValidate(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud manifest validate <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.Validate(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
