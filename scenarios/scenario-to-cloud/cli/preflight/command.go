package preflight

import (
	"fmt"
	"strconv"
	"strings"

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
  fix-ports              Fix port conflicts
  fix-firewall           Open required firewall ports
  fix-processes          Stop conflicting processes
  disk-usage             Show disk usage breakdown
  disk-cleanup           Clean up disk space

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

func runFixPorts(client *Client, args []string) error {
	req := FixPortsRequest{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight fix-ports [flags]

Fixes port conflicts by stopping processes using required ports.

Flags:
  --port <n>    Fix specific port (can be repeated)
  --json        Output raw JSON`)
			return nil
		case "--port":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.Ports = append(req.Ports, n)
				}
			}
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.FixPorts(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Println("Port conflicts fixed successfully.")
		if len(resp.Fixed) > 0 {
			fmt.Println("Fixed:")
			for _, f := range resp.Fixed {
				fmt.Printf("  - %s\n", f)
			}
		}
	} else {
		fmt.Printf("Failed to fix port conflicts: %s\n", resp.Message)
		if len(resp.Failed) > 0 {
			fmt.Println("Failed to fix:")
			for _, f := range resp.Failed {
				fmt.Printf("  - %s\n", f)
			}
		}
	}

	return nil
}

func runFixFirewall(client *Client, args []string) error {
	req := FixFirewallRequest{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight fix-firewall [flags]

Opens required firewall ports.

Flags:
  --port <n>       Open specific port (can be repeated)
  --protocol <p>   Protocol: tcp, udp, both (default: tcp)
  --json           Output raw JSON`)
			return nil
		case "--port":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.Ports = append(req.Ports, n)
				}
			}
		case "--protocol":
			if i+1 < len(args) {
				i++
				req.Protocol = args[i]
			}
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.FixFirewall(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Println("Firewall rules updated successfully.")
		if len(resp.Fixed) > 0 {
			fmt.Println("Opened:")
			for _, f := range resp.Fixed {
				fmt.Printf("  - %s\n", f)
			}
		}
	} else {
		fmt.Printf("Failed to update firewall: %s\n", resp.Message)
	}

	return nil
}

func runFixProcesses(client *Client, args []string) error {
	req := FixProcessesRequest{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight fix-processes [flags]

Stops conflicting processes.

Flags:
  --pid <n>     Stop specific PID (can be repeated)
  --port <n>    Stop process using port (can be repeated)
  --force       Use SIGKILL instead of SIGTERM
  --json        Output raw JSON`)
			return nil
		case "--pid":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.PIDs = append(req.PIDs, n)
				}
			}
		case "--port":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.Ports = append(req.Ports, n)
				}
			}
		case "--force":
			req.Force = true
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.FixProcesses(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Println("Processes stopped successfully.")
		if len(resp.Fixed) > 0 {
			fmt.Println("Stopped:")
			for _, f := range resp.Fixed {
				fmt.Printf("  - %s\n", f)
			}
		}
	} else {
		fmt.Printf("Failed to stop processes: %s\n", resp.Message)
	}

	return nil
}

func runDiskUsage(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight disk-usage [flags]

Shows disk usage breakdown.

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.DiskUsage()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Println("Disk Usage")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Total:     %s\n", formatSize(resp.TotalBytes))
	fmt.Printf("Used:      %s (%.1f%%)\n", formatSize(resp.UsedBytes), resp.UsedPercent)
	fmt.Printf("Available: %s\n", formatSize(resp.AvailableBytes))

	if len(resp.Breakdown) > 0 {
		fmt.Println("\nBreakdown:")
		fmt.Printf("  %-15s %-12s %s\n", "CATEGORY", "SIZE", "PATH")
		for _, b := range resp.Breakdown {
			category := b.Category
			if category == "" {
				category = "-"
			}
			fmt.Printf("  %-15s %-12s %s\n", category, formatSize(b.SizeBytes), b.Path)
		}
	}

	return nil
}

func runDiskCleanup(client *Client, args []string) error {
	req := DiskCleanupRequest{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight disk-cleanup [flags]

Cleans up disk space by removing old files.

Flags:
  --bundles          Clean old bundles
  --logs             Clean old logs
  --tmp              Clean temp files
  --older-than <n>   Only clean files older than n days
  --dry-run          Just report, don't delete
  --json             Output raw JSON`)
			return nil
		case "--bundles":
			req.CleanBundles = true
		case "--logs":
			req.CleanLogs = true
		case "--tmp":
			req.CleanTmp = true
		case "--older-than":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.OlderThanDays = n
				}
			}
		case "--dry-run":
			req.DryRun = true
		case "--json":
			jsonOutput = true
		}
	}

	// Default to cleaning all if nothing specified
	if !req.CleanBundles && !req.CleanLogs && !req.CleanTmp {
		req.CleanBundles = true
		req.CleanLogs = true
		req.CleanTmp = true
	}

	body, resp, err := client.DiskCleanup(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	action := "Cleaned up"
	if resp.DryRun {
		action = "Would clean up"
	}

	if resp.Success {
		fmt.Printf("%s %d file(s), freed %s\n", action, resp.RemovedFiles, formatSize(resp.FreedBytes))
		if len(resp.Removed) > 0 && len(resp.Removed) <= 10 {
			fmt.Println("Removed:")
			for _, r := range resp.Removed {
				fmt.Printf("  - %s\n", r)
			}
		}
	} else {
		fmt.Printf("Cleanup failed: %s\n", resp.Message)
	}

	return nil
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
