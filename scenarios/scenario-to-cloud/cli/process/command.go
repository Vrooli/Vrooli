package process

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes process subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "kill":
		return runKill(client, args[1:])
	case "restart":
		return runRestart(client, args[1:])
	case "control":
		return runControl(client, args[1:])
	case "vps-action":
		return runVPSAction(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud process help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud process <command> [arguments]

Commands:
  kill <deployment-id> <pid>                Kill a process by PID
  restart <deployment-id> <type> <name>     Restart a scenario or resource
  control <deployment-id> <action> [type]   Start/stop/restart processes
  vps-action <deployment-id> <action>       VPS-level actions (reboot, shutdown)

Run 'scenario-to-cloud process <command> -h' for command-specific options.`)
	return nil
}

func runKill(client *Client, args []string) error {
	var deploymentID string
	var pid int
	signal := "SIGTERM"
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud process kill <deployment-id> <pid> [flags]

Flags:
  --signal <sig>   Signal to send (default: SIGTERM). Options: SIGTERM, SIGKILL, SIGHUP
  --json           Output raw JSON`)
			return nil
		case "--signal":
			if i+1 < len(args) {
				i++
				signal = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if pid == 0 {
					if n, err := strconv.Atoi(args[i]); err == nil {
						pid = n
					} else {
						return fmt.Errorf("invalid PID: %s", args[i])
					}
				}
			}
		}
	}

	if deploymentID == "" || pid == 0 {
		return fmt.Errorf("usage: scenario-to-cloud process kill <deployment-id> <pid>")
	}

	req := KillRequest{PID: pid, Signal: signal}
	body, resp, err := client.Kill(deploymentID, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Successfully sent %s to PID %d\n", resp.Signal, resp.PID)
	} else {
		fmt.Printf("Failed to kill PID %d: %s\n", resp.PID, resp.Message)
	}
	return nil
}

func runRestart(client *Client, args []string) error {
	var deploymentID, typeName, name string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud process restart <deployment-id> <type> <name>

Arguments:
  type    Type of process: scenario, resource
  name    Name of the scenario or resource to restart

Flags:
  --json  Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if typeName == "" {
					typeName = args[i]
				} else if name == "" {
					name = args[i]
				}
			}
		}
	}

	if deploymentID == "" || typeName == "" || name == "" {
		return fmt.Errorf("usage: scenario-to-cloud process restart <deployment-id> <type> <name>")
	}

	if typeName != "scenario" && typeName != "resource" {
		return fmt.Errorf("type must be 'scenario' or 'resource', got: %s", typeName)
	}

	req := RestartRequest{Type: typeName, Name: name}
	body, resp, err := client.Restart(deploymentID, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		if resp.NewPID > 0 {
			fmt.Printf("Successfully restarted %s '%s' (new PID: %d)\n", resp.Type, resp.Name, resp.NewPID)
		} else {
			fmt.Printf("Successfully restarted %s '%s'\n", resp.Type, resp.Name)
		}
	} else {
		fmt.Printf("Failed to restart %s '%s': %s\n", resp.Type, resp.Name, resp.Message)
	}
	return nil
}

func runControl(client *Client, args []string) error {
	var deploymentID, action, typeName, name string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud process control <deployment-id> <action> [type] [name]

Arguments:
  action  Action to perform: start, stop, restart
  type    Optional type filter: scenario, resource, all (default: all)
  name    Optional name of specific scenario/resource

Flags:
  --json  Output raw JSON

Examples:
  scenario-to-cloud process control abc123 restart                    # Restart all
  scenario-to-cloud process control abc123 stop resource postgres     # Stop postgres
  scenario-to-cloud process control abc123 start scenario my-app      # Start specific scenario`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if action == "" {
					action = args[i]
				} else if typeName == "" {
					typeName = args[i]
				} else if name == "" {
					name = args[i]
				}
			}
		}
	}

	if deploymentID == "" || action == "" {
		return fmt.Errorf("usage: scenario-to-cloud process control <deployment-id> <action> [type] [name]")
	}

	validActions := map[string]bool{"start": true, "stop": true, "restart": true}
	if !validActions[action] {
		return fmt.Errorf("action must be 'start', 'stop', or 'restart', got: %s", action)
	}

	if typeName == "" {
		typeName = "all"
	}

	req := ControlRequest{Action: action, Type: typeName, Name: name}
	body, resp, err := client.Control(deploymentID, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Action '%s' completed successfully\n", resp.Action)
	} else {
		fmt.Printf("Action '%s' failed: %s\n", resp.Action, resp.Message)
	}

	if len(resp.Results) > 0 {
		fmt.Println("\nResults:")
		for _, r := range resp.Results {
			status := "OK"
			if !r.Success {
				status = "FAILED"
			}
			pidStr := ""
			if r.PID > 0 {
				pidStr = fmt.Sprintf(" (PID: %d)", r.PID)
			}
			errStr := ""
			if r.Error != "" {
				errStr = fmt.Sprintf(" - %s", r.Error)
			}
			fmt.Printf("  [%s] %s %s%s%s\n", status, r.Type, r.Name, pidStr, errStr)
		}
	}

	return nil
}

func runVPSAction(client *Client, args []string) error {
	var deploymentID, action string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud process vps-action <deployment-id> <action>

Arguments:
  action  VPS action: reboot, shutdown, start

Flags:
  --json  Output raw JSON

WARNING: These actions affect the entire VPS!`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if action == "" {
					action = args[i]
				}
			}
		}
	}

	if deploymentID == "" || action == "" {
		return fmt.Errorf("usage: scenario-to-cloud process vps-action <deployment-id> <action>")
	}

	validActions := map[string]bool{"reboot": true, "shutdown": true, "start": true}
	if !validActions[action] {
		return fmt.Errorf("action must be 'reboot', 'shutdown', or 'start', got: %s", action)
	}

	req := VPSActionRequest{Action: action}
	body, resp, err := client.VPSAction(deploymentID, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("VPS action '%s' initiated successfully\n", resp.Action)
		if resp.Message != "" {
			fmt.Printf("Message: %s\n", resp.Message)
		}
	} else {
		fmt.Printf("VPS action '%s' failed: %s\n", resp.Action, resp.Message)
	}
	return nil
}
