package vps

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes VPS subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "setup":
		return runSetup(client, args[1:])
	case "deploy":
		return runDeploy(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud vps help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud vps <command> [arguments]

Commands:
  setup plan <manifest.json> <bundle.tar.gz>    Generate VPS setup plan
  setup apply <manifest.json> <bundle.tar.gz>   Execute VPS setup
  deploy plan <manifest.json>                   Generate VPS deploy plan
  deploy apply <manifest.json>                  Execute VPS deploy

Run 'scenario-to-cloud vps <command> -h' for command-specific options.`)
	return nil
}

func runSetup(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: scenario-to-cloud vps setup <plan|apply> <manifest.json> <bundle.tar.gz>")
	}

	switch args[0] {
	case "plan":
		return runSetupPlan(client, args[1:])
	case "apply":
		return runSetupApply(client, args[1:])
	default:
		return fmt.Errorf("unknown setup subcommand: %s\n\nUsage: scenario-to-cloud vps setup <plan|apply>", args[0])
	}
}

func runSetupPlan(client *Client, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: scenario-to-cloud vps setup plan <manifest.json> <bundle.tar.gz>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.SetupPlan(manifest, args[1])
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runSetupApply(client *Client, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: scenario-to-cloud vps setup apply <manifest.json> <bundle.tar.gz>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.SetupApply(manifest, args[1])
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runDeploy(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: scenario-to-cloud vps deploy <plan|apply> <manifest.json>")
	}

	switch args[0] {
	case "plan":
		return runDeployPlan(client, args[1:])
	case "apply":
		return runDeployApply(client, args[1:])
	default:
		return fmt.Errorf("unknown deploy subcommand: %s\n\nUsage: scenario-to-cloud vps deploy <plan|apply>", args[0])
	}
}

func runDeployPlan(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud vps deploy plan <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.DeployPlan(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runDeployApply(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud vps deploy apply <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.DeployApply(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
