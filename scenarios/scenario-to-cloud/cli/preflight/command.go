package preflight

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/internal/flagutil"
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
	case "requirements":
		return runRequirements(client, args[1:])
	case "fix-ports":
		return runFixPorts(client, args[1:])
	case "fix-firewall":
		return runFixFirewall(client, args[1:])
	case "fix-processes":
		return runFixProcesses(client, args[1:])
	case "disk-usage":
		return runDiskUsage(client, args[1:])
	case "disk-cleanup":
		return runDiskCleanup(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud preflight help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud preflight <command> [arguments]

Commands:
  run <manifest.json>    Run VPS preflight checks for a cloud manifest
  requirements           Show canonical VPS requirements/policy
  fix-ports              Stop services/processes on conflicting ports
  fix-firewall           Open required firewall ports
  fix-processes          Stop stale scenario processes on target VPS
  disk-usage             Show disk usage breakdown
  disk-cleanup           Clean up disk space

Selector flags for target-dependent commands:
  --host <host> | --domain <domain> | --target <domain-or-host>
  [--scenario <id>] [--user <ssh-user>] [--key-path <path>] [--ssh-port <n>]

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

func runRequirements(client *Client, args []string) error {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight requirements [flags]

Show canonical VPS requirements used by preflight/runtime checks.

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag: %s", arg)
			}
			return fmt.Errorf("usage: scenario-to-cloud preflight requirements")
		}
	}

	body, resp, err := client.Requirements()
	if err != nil {
		return err
	}
	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("VPS Requirements")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("OS: %s %s (compatible: %s)\n",
		resp.VPS.OS.RequiredID,
		resp.VPS.OS.RecommendedVersion,
		strings.Join(resp.VPS.OS.CompatibleVersions, ", "),
	)
	fmt.Printf("RAM: min %s, recommended %s\n",
		formatSize(resp.VPS.Resources.MinRAMBytes),
		formatSize(resp.VPS.Resources.RecommendedRAMBytes),
	)
	fmt.Printf("Disk: min free %s\n", formatSize(resp.VPS.Resources.MinDiskFreeBytes))
	fmt.Printf("Inbound ports: %s\n", joinPorts(resp.VPS.Network.RequiredInboundPorts))
	fmt.Printf("Auth: %s (bootstrap: %s)\n",
		resp.VPS.Authentication.RequiredMethod,
		resp.VPS.Authentication.BootstrapFlow,
	)
	return nil
}

func runFixPorts(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight fix-ports", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	var ports intListFlag
	var pids intListFlag
	var services cliutil.StringList
	preferServiceStop := fs.Bool("prefer-service-stop", true, "Prefer stopping owning systemd service before PID kill")
	fs.Var(&ports, "port", "Fix specific port (repeatable)")
	fs.Var(&pids, "pid", "Stop specific PID (repeatable)")
	fs.Var(&services, "service", "Stop specific service (repeatable)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud preflight fix-ports --domain <domain>|--host <host>|--target <target> [--scenario <id>] [--port <n>] [--pid <n>] [--service <name>] [--json]")
	}

	target, err := targetFlags.resolve(client)
	if err != nil {
		return err
	}

	req := FixPortsRequest{
		Host:              target.Host,
		Port:              target.Port,
		User:              target.User,
		KeyPath:           target.KeyPath,
		Ports:             ports.Values,
		PIDs:              pids.Values,
		Services:          services.Values(),
		PreferServiceStop: preferServiceStop,
	}

	body, resp, err := client.FixPorts(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Println("Port conflict actions completed.")
		if len(resp.Stopped) > 0 {
			fmt.Println("Stopped:")
			for _, item := range resp.Stopped {
				fmt.Printf("  - %s\n", item)
			}
		}
		if len(resp.Failed) > 0 {
			fmt.Println("Failed:")
			for _, item := range resp.Failed {
				fmt.Printf("  - %s\n", item)
			}
		}
		return nil
	}

	fmt.Printf("Port conflict fix failed: %s\n", resp.Message)
	if len(resp.Failed) > 0 {
		fmt.Println("Failed:")
		for _, item := range resp.Failed {
			fmt.Printf("  - %s\n", item)
		}
	}
	return nil
}

func runFixFirewall(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight fix-firewall", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	var ports intListFlag
	fs.Var(&ports, "port", "Open specific port (repeatable; default 80,443)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud preflight fix-firewall --domain <domain>|--host <host>|--target <target> [--scenario <id>] [--port <n>] [--json]")
	}

	target, err := targetFlags.resolve(client)
	if err != nil {
		return err
	}

	req := FixFirewallRequest{
		Host:    target.Host,
		Port:    target.Port,
		User:    target.User,
		KeyPath: target.KeyPath,
		Ports:   ports.Values,
	}

	body, resp, err := client.FixFirewall(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Println("Firewall rules updated successfully.")
		fmt.Printf("Ports: %s\n", joinPorts(resp.Ports))
		if strings.TrimSpace(resp.Status) != "" {
			fmt.Printf("Status:\n%s\n", strings.TrimSpace(resp.Status))
		}
		return nil
	}

	fmt.Printf("Failed to update firewall: %s\n", resp.Message)
	return nil
}

func runFixProcesses(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight fix-processes", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	scenarioID := fs.String("scenario-id", "", "Override scenario ID for targeted stop")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud preflight fix-processes --domain <domain>|--host <host>|--target <target> [--scenario <id>] [--scenario-id <id>] [--workdir <path>] [--json]")
	}

	target, err := targetFlags.resolve(client)
	if err != nil {
		return err
	}

	req := FixProcessesRequest{
		Host:       target.Host,
		Port:       target.Port,
		User:       target.User,
		KeyPath:    target.KeyPath,
		Workdir:    target.Workdir,
		ScenarioID: target.ScenarioID,
	}
	if v := strings.TrimSpace(*scenarioID); v != "" {
		req.ScenarioID = v
	}

	body, resp, err := client.FixProcesses(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Printf("Process cleanup succeeded (%s).\n", resp.Action)
		if strings.TrimSpace(resp.Output) != "" {
			fmt.Printf("%s\n", strings.TrimSpace(resp.Output))
		}
		return nil
	}

	fmt.Printf("Process cleanup failed (%s): %s\n", resp.Action, resp.Message)
	if strings.TrimSpace(resp.Output) != "" {
		fmt.Printf("%s\n", strings.TrimSpace(resp.Output))
	}
	return nil
}

func runDiskUsage(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight disk-usage", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud preflight disk-usage --domain <domain>|--host <host>|--target <target> [--scenario <id>] [--json]")
	}

	target, err := targetFlags.resolve(client)
	if err != nil {
		return err
	}

	body, resp, err := client.DiskUsage(DiskUsageRequest{
		Host:    target.Host,
		Port:    target.Port,
		User:    target.User,
		KeyPath: target.KeyPath,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Disk Usage")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Total:     %s\n", resp.TotalSpace)
	fmt.Printf("Used:      %d%%\n", resp.UsedPercent)
	fmt.Printf("Available: %s\n", resp.FreeSpace)

	if len(resp.LargestDirs) > 0 {
		fmt.Println("\nLargest Directories:")
		fmt.Printf("  %-12s %s\n", "SIZE", "PATH")
		for _, d := range resp.LargestDirs {
			fmt.Printf("  %-12s %s\n", d.Size, d.Path)
		}
	}

	return nil
}

func runDiskCleanup(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight disk-cleanup", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	var actions cliutil.StringList
	fs.Var(&actions, "action", "Cleanup action (repeatable): apt_clean, journal_vacuum, docker_prune, tmp_clean")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud preflight disk-cleanup --domain <domain>|--host <host>|--target <target> [--scenario <id>] [--action <name>] [--json]")
	}

	target, err := targetFlags.resolve(client)
	if err != nil {
		return err
	}

	requestedActions := actions.Values()
	if len(requestedActions) == 0 {
		requestedActions = []string{"apt_clean", "journal_vacuum"}
	}

	body, resp, err := client.DiskCleanup(DiskCleanupRequest{
		Host:    target.Host,
		Port:    target.Port,
		User:    target.User,
		KeyPath: target.KeyPath,
		Actions: requestedActions,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Printf("Disk cleanup completed: freed %s\n", resp.SpaceFreed)
	} else {
		fmt.Printf("Disk cleanup completed with failures: freed %s\n", resp.SpaceFreed)
	}
	if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("%s\n", strings.TrimSpace(resp.Message))
	}
	if len(resp.ActionsRun) > 0 {
		fmt.Printf("Actions run: %s\n", strings.Join(resp.ActionsRun, ", "))
	}

	if len(resp.ActionResults) > 0 {
		fmt.Println("Action results:")
		for _, result := range resp.ActionResults {
			status := "ok"
			if !result.OK {
				status = "failed"
			}
			fmt.Printf("  - %s: %s", result.Action, status)
			if result.ExitCode != 0 {
				fmt.Printf(" (exit %d)", result.ExitCode)
			}
			fmt.Println()
			if strings.TrimSpace(result.Summary) != "" {
				fmt.Printf("    summary: %s\n", result.Summary)
			}
			if strings.TrimSpace(result.Hint) != "" {
				fmt.Printf("    hint: %s\n", result.Hint)
			}
		}
	}

	return nil
}

func joinPorts(ports []int) string {
	if len(ports) == 0 {
		return "-"
	}
	values := make([]string, 0, len(ports))
	for _, p := range ports {
		values = append(values, strconv.Itoa(p))
	}
	return strings.Join(values, ", ")
}

// formatSize formats bytes into a human-readable string.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

type intListFlag struct {
	Values []int
}

func (f *intListFlag) String() string {
	if len(f.Values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f.Values))
	for _, v := range f.Values {
		parts = append(parts, strconv.Itoa(v))
	}
	return strings.Join(parts, ",")
}

func (f *intListFlag) Set(value string) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("invalid integer %q", value)
	}
	f.Values = append(f.Values, n)
	return nil
}
