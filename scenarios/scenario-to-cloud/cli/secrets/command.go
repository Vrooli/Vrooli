package secrets

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes secrets subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "get":
		return runGet(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud secrets help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud secrets <command> [arguments]

Commands:
  get <scenario-id>    Get secrets for a scenario

Run 'scenario-to-cloud secrets <command> -h' for command-specific options.`)
	return nil
}

func runGet(client *Client, args []string) error {
	var scenarioID string
	reveal := false
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud secrets get <scenario-id> [flags]

Flags:
  --reveal  Show secret values (default: masked)
  --json    Output raw JSON`)
			return nil
		case "--reveal":
			reveal = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && scenarioID == "" {
				scenarioID = args[i]
			}
		}
	}

	if scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud secrets get <scenario-id>")
	}

	body, resp, err := client.Get(scenarioID, reveal)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Secrets for Scenario: %s\n", resp.ScenarioID)
	fmt.Println(strings.Repeat("-", 70))

	if len(resp.Secrets) == 0 {
		fmt.Println("No secrets configured.")
		return nil
	}

	fmt.Printf("%-30s %-12s %-10s %s\n", "KEY", "SOURCE", "REQUIRED", "VALUE")

	for key, secret := range resp.Secrets {
		required := "no"
		if secret.Required {
			required = "yes"
		}

		value := "(not set)"
		if secret.HasValue {
			if reveal && secret.Value != "" {
				value = secret.Value
			} else if secret.Masked != "" {
				value = secret.Masked
			} else {
				value = "****"
			}
		}

		fmt.Printf("%-30s %-12s %-10s %s\n", truncate(key, 30), secret.Source, required, value)
	}

	if !reveal {
		fmt.Println("\nUse --reveal to show actual secret values.")
	}

	return nil
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
