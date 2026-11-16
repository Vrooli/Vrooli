package process

import (
	"fmt"
	"io"
	"os"
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
  vps-action <deployment-id> <action>       VPS-level actions (cleanup, stop_vrooli)

Run 'scenario-to-cloud process <command> -h' for command-specific options.`)
	return nil
}

func runKill(client *Client, args []string) error {
	parsed, showHelp, err := parseKillArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud process kill <deployment-id> <pid> [flags]

Flags:
  --signal <sig>   Signal to send (default: SIGTERM). Options: SIGTERM, SIGKILL, SIGHUP
  --json           Output raw JSON`)
		return nil
	}
	if err := validateKillArgs(parsed); err != nil {
		return err
	}
	body, resp, err := client.Kill(parsed.deploymentID, KillRequest{PID: parsed.pid, Signal: parsed.signal})
	if err != nil {
		return err
	}
	if parsed.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	writeKillResponse(os.Stdout, resp)
	return nil
}

type killArgs struct {
	deploymentID string
	pid          int
	signal       string
	jsonOutput   bool
}

func parseKillArgs(args []string) (killArgs, bool, error) {
	parsed := killArgs{signal: "SIGTERM"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			return parsed, true, nil
		case "--signal":
			if i+1 < len(args) {
				i++
				parsed.signal = args[i]
			}
		case "--json":
			parsed.jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if parsed.deploymentID == "" {
					parsed.deploymentID = args[i]
				} else if parsed.pid == 0 {
					pid, err := strconv.Atoi(args[i])
					if err != nil {
						return killArgs{}, false, fmt.Errorf("invalid PID: %s", args[i])
					}
					parsed.pid = pid
				}
			}
		}
	}
	return parsed, false, nil
}

func validateKillArgs(parsed killArgs) error {
	if parsed.deploymentID == "" || parsed.pid == 0 {
		return fmt.Errorf("usage: scenario-to-cloud process kill <deployment-id> <pid>")
	}
	return nil
}

func writeKillResponse(w io.Writer, resp KillResponse) {
	if resp.Success {
		fmt.Fprintf(w, "Successfully sent %s to PID %d\n", resp.Signal, resp.PID)
	} else {
		fmt.Fprintf(w, "Failed to kill PID %d: %s\n", resp.PID, resp.Message)
	}
}

func runRestart(client *Client, args []string) error {
	parsed, showHelp := parseRestartArgs(args)
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud process restart <deployment-id> <type> <name>

Arguments:
  type    Type of process: scenario, resource
  name    Name of the scenario or resource to restart

Flags:
  --json  Output raw JSON`)
		return nil
	}
	if err := validateRestartArgs(parsed); err != nil {
		return err
	}
	body, resp, err := client.Restart(parsed.deploymentID, RestartRequest{Type: parsed.typeName, Name: parsed.name})
	if err != nil {
		return err
	}
	if parsed.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	writeRestartResponse(os.Stdout, resp)
	return nil
}

type restartArgs struct {
	deploymentID string
	typeName     string
	name         string
	jsonOutput   bool
}

func parseRestartArgs(args []string) (restartArgs, bool) {
	var parsed restartArgs
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return parsed, true
		case "--json":
			parsed.jsonOutput = true
		default:
			if !strings.HasPrefix(arg, "-") {
				assignRestartPositional(&parsed, arg)
			}
		}
	}
	return parsed, false
}

func assignRestartPositional(parsed *restartArgs, value string) {
	switch {
	case parsed.deploymentID == "":
		parsed.deploymentID = value
	case parsed.typeName == "":
		parsed.typeName = value
	case parsed.name == "":
		parsed.name = value
	}
}

func validateRestartArgs(parsed restartArgs) error {
	if parsed.deploymentID == "" || parsed.typeName == "" || parsed.name == "" {
		return fmt.Errorf("usage: scenario-to-cloud process restart <deployment-id> <type> <name>")
	}
	if parsed.typeName != "scenario" && parsed.typeName != "resource" {
		return fmt.Errorf("type must be 'scenario' or 'resource', got: %s", parsed.typeName)
	}
	return nil
}

func writeRestartResponse(w io.Writer, resp RestartResponse) {
	if resp.Success {
		if resp.NewPID > 0 {
			fmt.Fprintf(w, "Successfully restarted %s '%s' (new PID: %d)\n", resp.Type, resp.Name, resp.NewPID)
		} else {
			fmt.Fprintf(w, "Successfully restarted %s '%s'\n", resp.Type, resp.Name)
		}
	} else {
		fmt.Fprintf(w, "Failed to restart %s '%s': %s\n", resp.Type, resp.Name, resp.Message)
	}
}

func runControl(client *Client, args []string) error {
	parsed, showHelp := parseControlArgs(args)
	if showHelp {
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
	}
	if err := validateControlArgs(&parsed); err != nil {
		return err
	}
	body, resp, err := client.Control(parsed.deploymentID, parsed.request)
	if err != nil {
		return err
	}
	if parsed.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	writeControlOutcome(os.Stdout, resp)
	return nil
}

type controlArgs struct {
	deploymentID string
	request      ControlRequest
	jsonOutput   bool
}

func parseControlArgs(args []string) (controlArgs, bool) {
	var parsed controlArgs
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return parsed, true
		case "--json":
			parsed.jsonOutput = true
		default:
			if !strings.HasPrefix(arg, "-") {
				assignControlPositional(&parsed, arg)
			}
		}
	}
	return parsed, false
}

func assignControlPositional(parsed *controlArgs, value string) {
	switch {
	case parsed.deploymentID == "":
		parsed.deploymentID = value
	case parsed.request.Action == "":
		parsed.request.Action = value
	case parsed.request.Type == "":
		parsed.request.Type = value
	case parsed.request.Name == "":
		parsed.request.Name = value
	}
}

func validateControlArgs(parsed *controlArgs) error {
	if parsed.deploymentID == "" || parsed.request.Action == "" {
		return fmt.Errorf("usage: scenario-to-cloud process control <deployment-id> <action> [type] [name]")
	}
	switch parsed.request.Action {
	case "start", "stop", "restart":
	default:
		return fmt.Errorf("action must be 'start', 'stop', or 'restart', got: %s", parsed.request.Action)
	}
	if parsed.request.Type == "" {
		parsed.request.Type = "all"
	}
	return nil
}

func writeControlOutcome(w io.Writer, resp ControlResponse) {
	if resp.Success {
		fmt.Fprintf(w, "Action '%s' completed successfully\n", resp.Action)
	} else {
		fmt.Fprintf(w, "Action '%s' failed: %s\n", resp.Action, resp.Message)
	}
	writeControlResults(w, resp.Results)
}

func writeControlResults(w io.Writer, results []ProcessResult) {
	if len(results) == 0 {
		return
	}
	fmt.Fprintln(w, "\nResults:")
	for _, result := range results {
		status := "OK"
		if !result.Success {
			status = "FAILED"
		}
		pid := ""
		if result.PID > 0 {
			pid = fmt.Sprintf(" (PID: %d)", result.PID)
		}
		detail := ""
		if result.Error != "" {
			detail = fmt.Sprintf(" - %s", result.Error)
		}
		fmt.Fprintf(w, "  [%s] %s %s%s%s\n", status, result.Type, result.Name, pid, detail)
	}
}

func runVPSAction(client *Client, args []string) error {
	parsed, showHelp, err := parseVPSActionArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud process vps-action <deployment-id> <action>

Arguments:
  action  VPS action: reboot, shutdown, start, stop_vrooli, cleanup

Flags:
  --cleanup-level <1-5>  Required for cleanup
  --confirmation <text>  Typed confirmation for effectful actions
  --json                 Output raw JSON

WARNING: These actions affect the entire VPS!`)
		return nil
	}
	if err := validateVPSActionArgs(parsed); err != nil {
		return err
	}
	body, resp, err := client.VPSAction(parsed.deploymentID, VPSActionRequest{Action: parsed.action, CleanupLevel: parsed.cleanupLevel, Confirmation: parsed.confirmation})
	if err != nil {
		return err
	}
	if parsed.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	writeVPSActionResponse(os.Stdout, resp)
	return nil
}

type vpsActionArgs struct {
	deploymentID string
	action       string
	jsonOutput   bool
	cleanupLevel int
	confirmation string
}

func parseVPSActionArgs(args []string) (vpsActionArgs, bool, error) {
	var parsed vpsActionArgs
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			return parsed, true, nil
		case "--json":
			parsed.jsonOutput = true
		case "--cleanup-level":
			if i+1 >= len(args) {
				return vpsActionArgs{}, false, fmt.Errorf("--cleanup-level requires a value")
			}
			i++
			level, err := strconv.Atoi(args[i])
			if err != nil {
				return vpsActionArgs{}, false, fmt.Errorf("invalid cleanup level: %s", args[i])
			}
			parsed.cleanupLevel = level
		case "--confirmation":
			if i+1 >= len(args) {
				return vpsActionArgs{}, false, fmt.Errorf("--confirmation requires a value")
			}
			i++
			parsed.confirmation = args[i]
		default:
			if !strings.HasPrefix(args[i], "-") {
				assignVPSActionPositional(&parsed, args[i])
			}
		}
	}
	return parsed, false, nil
}

func assignVPSActionPositional(parsed *vpsActionArgs, value string) {
	if parsed.deploymentID == "" {
		parsed.deploymentID = value
		return
	}
	if parsed.action == "" {
		parsed.action = value
	}
}

func validateVPSActionArgs(parsed vpsActionArgs) error {
	if parsed.deploymentID == "" || parsed.action == "" {
		return fmt.Errorf("usage: scenario-to-cloud process vps-action <deployment-id> <action>")
	}
	switch parsed.action {
	case "reboot", "shutdown", "start", "stop_vrooli", "cleanup":
		return nil
	default:
		return fmt.Errorf("action must be 'reboot', 'shutdown', 'start', 'stop_vrooli', or 'cleanup', got: %s", parsed.action)
	}
}

func writeVPSActionResponse(w io.Writer, resp VPSActionResponse) {
	if resp.Success {
		fmt.Fprintf(w, "VPS action '%s' initiated successfully\n", resp.Action)
		if resp.Message != "" {
			fmt.Fprintf(w, "Message: %s\n", resp.Message)
		}
	} else {
		fmt.Fprintf(w, "VPS action '%s' failed: %s\n", resp.Action, resp.Message)
	}
}
