package scenario

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes scenario subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "list":
		return runList(client, args[1:])
	case "ports":
		return runPorts(client, args[1:])
	case "deps", "dependencies":
		return runDeps(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud scenario help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud scenario <command> [arguments]

Commands:
  list                    List all available scenarios
  ports <scenario-id>     Show port allocations for a scenario
  deps <scenario-id>      Show dependencies for a scenario

Run 'scenario-to-cloud scenario <command> -h' for command-specific options.`)
	return nil
}

func runList(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud scenario list [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.List()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if len(resp.Scenarios) == 0 {
		fmt.Println("No scenarios found.")
		return nil
	}

	fmt.Printf("Available Scenarios: %d\n", len(resp.Scenarios))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-25s %-10s %-8s %s\n", "ID", "VERSION", "PARTS", "DESCRIPTION")

	for _, s := range resp.Scenarios {
		parts := []string{}
		if s.HasAPI {
			parts = append(parts, "API")
		}
		if s.HasUI {
			parts = append(parts, "UI")
		}
		if s.HasCLI {
			parts = append(parts, "CLI")
		}
		partsStr := strings.Join(parts, ",")
		if partsStr == "" {
			partsStr = "-"
		}
		desc := s.Description
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}
		fmt.Printf("%-25s %-10s %-8s %s\n", truncate(s.ID, 25), s.Version, partsStr, desc)
	}

	return nil
}

func runPorts(client *Client, args []string) error {
	var scenarioID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud scenario ports <scenario-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && scenarioID == "" {
				scenarioID = args[i]
			}
		}
	}

	if scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud scenario ports <scenario-id>")
	}

	body, resp, err := client.Ports(scenarioID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Port Allocations for: %s\n", resp.ScenarioID)
	fmt.Println(strings.Repeat("-", 60))

	if len(resp.Ports) == 0 {
		fmt.Println("No ports allocated.")
		return nil
	}

	fmt.Printf("%-20s %-8s %-10s %-8s %s\n", "SERVICE", "PORT", "PROTOCOL", "PUBLIC", "PATH")
	for _, p := range resp.Ports {
		protocol := p.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		public := "no"
		if p.Public {
			public = "yes"
		}
		path := p.Path
		if path == "" {
			path = "-"
		}
		fmt.Printf("%-20s %-8d %-10s %-8s %s\n", truncate(p.Service, 20), p.Port, protocol, public, path)
	}

	return nil
}

func runDeps(client *Client, args []string) error {
	var scenarioID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud scenario deps <scenario-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && scenarioID == "" {
				scenarioID = args[i]
			}
		}
	}

	if scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud scenario deps <scenario-id>")
	}

	body, resp, err := client.Dependencies(scenarioID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Dependencies for: %s\n", resp.ScenarioID)
	fmt.Println(strings.Repeat("-", 70))

	if len(resp.Dependencies) == 0 {
		fmt.Println("No dependencies.")
		return nil
	}

	fmt.Printf("%-12s %-20s %-12s %-10s %s\n", "TYPE", "NAME", "VERSION", "REQUIRED", "STATUS")
	for _, d := range resp.Dependencies {
		required := "no"
		if d.Required {
			required = "yes"
		}
		status := d.Status
		if status == "" {
			status = "-"
		}
		fmt.Printf("%-12s %-20s %-12s %-10s %s\n", d.Type, truncate(d.Name, 20), d.Version, required, status)
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
