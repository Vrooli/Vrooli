package inspect

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// DefaultTailLines is the default number of log lines to fetch.
const DefaultTailLines = 200

// Run executes inspect subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "plan":
		return runPlan(client, args[1:])
	case "status", "apply":
		return runApply(client, args[1:])
	case "live":
		return runLive(client, args[1:])
	case "drift":
		return runDrift(client, args[1:])
	case "logs":
		return runLogs(client, args[1:])
	case "files":
		return runFiles(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud inspect help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud inspect <command> [arguments]

Commands:
  plan <manifest.json>      Generate inspection plan
  status <manifest.json>    Fetch remote status and logs
  live <deployment-id>      Get live state of a deployment
  drift <deployment-id>     Detect configuration drift
  logs <deployment-id>      View aggregated logs
  files <deployment-id>     Browse or read files on the VPS

Run 'scenario-to-cloud inspect <command> -h' for command-specific options.`)
	return nil
}

func runPlan(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud inspect plan <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	opts := Options{TailLines: DefaultTailLines}
	body, _, err := client.Plan(manifest, opts)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runApply(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud inspect status <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	opts := Options{TailLines: DefaultTailLines}
	body, _, err := client.Apply(manifest, opts)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runLive(client *Client, args []string) error {
	var id string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud inspect live <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && id == "" {
				id = args[i]
			}
		}
	}

	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud inspect live <deployment-id>")
	}

	body, resp, err := client.LiveState(id)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Live State for Deployment: %s\n", resp.DeploymentID)
	fmt.Println(strings.Repeat("-", 50))

	s := resp.State
	statusIcon := "X"
	if s.Running {
		statusIcon = "R"
	}
	healthIcon := "!"
	if s.Healthy {
		healthIcon = "H"
	}

	fmt.Printf("Status:   [%s] Running: %v  [%s] Healthy: %v\n", statusIcon, s.Running, healthIcon, s.Healthy)
	if s.Uptime != "" {
		fmt.Printf("Uptime:   %s\n", s.Uptime)
	}
	if s.PublicIP != "" {
		fmt.Printf("Public IP: %s\n", s.PublicIP)
	}
	if s.InternalIP != "" {
		fmt.Printf("Internal IP: %s\n", s.InternalIP)
	}
	if s.CPUPercent != "" || s.MemoryPercent != "" || s.DiskUsedGB != "" {
		fmt.Printf("Resources: CPU=%s  Memory=%s  Disk=%s\n", s.CPUPercent, s.MemoryPercent, s.DiskUsedGB)
	}
	if s.ErrorMessage != "" {
		fmt.Printf("Error:    %s\n", s.ErrorMessage)
	}

	if len(resp.Processes) > 0 {
		fmt.Println("\nProcesses:")
		fmt.Printf("  %-8s %-20s %-10s %-8s %s\n", "PID", "NAME", "STATUS", "CPU", "MEMORY")
		for _, p := range resp.Processes {
			fmt.Printf("  %-8d %-20s %-10s %-8s %s\n", p.PID, truncate(p.Name, 20), p.Status, p.CPUPercent, p.MemoryMB)
		}
	}

	if len(resp.Resources) > 0 {
		fmt.Println("\nResources:")
		fmt.Printf("  %-20s %-15s %-10s %-7s %s\n", "NAME", "TYPE", "STATUS", "PORT", "HEALTHY")
		for _, r := range resp.Resources {
			healthStr := "no"
			if r.Healthy {
				healthStr = "yes"
			}
			portStr := "-"
			if r.Port > 0 {
				portStr = strconv.Itoa(r.Port)
			}
			fmt.Printf("  %-20s %-15s %-10s %-7s %s\n", truncate(r.Name, 20), r.Type, r.Status, portStr, healthStr)
		}
	}

	return nil
}

func runDrift(client *Client, args []string) error {
	var id string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud inspect drift <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && id == "" {
				id = args[i]
			}
		}
	}

	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud inspect drift <deployment-id>")
	}

	body, resp, err := client.Drift(id)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Drift Detection for Deployment: %s\n", resp.DeploymentID)
	fmt.Println(strings.Repeat("-", 50))

	if !resp.HasDrift {
		fmt.Println("No drift detected. Configuration matches expected state.")
		fmt.Printf("Checked at: %s\n", resp.CheckedAt)
		return nil
	}

	fmt.Printf("DRIFT DETECTED: %d item(s)\n\n", len(resp.DriftItems))

	for i, item := range resp.DriftItems {
		severityIcon := "?"
		switch item.Severity {
		case "error":
			severityIcon = "E"
		case "warning":
			severityIcon = "W"
		case "info":
			severityIcon = "I"
		}
		fmt.Printf("%d. [%s] %s (%s)\n", i+1, severityIcon, item.Path, item.Type)
		if item.Expected != "" {
			fmt.Printf("   Expected: %s\n", item.Expected)
		}
		if item.Actual != "" {
			fmt.Printf("   Actual:   %s\n", item.Actual)
		}
		if item.Message != "" {
			fmt.Printf("   Message:  %s\n", item.Message)
		}
	}

	fmt.Printf("\nChecked at: %s\n", resp.CheckedAt)
	return nil
}

func runLogs(client *Client, args []string) error {
	var id string
	opts := LogsOptions{Tail: 100}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud inspect logs <deployment-id> [flags]

Flags:
  --source <name>   Filter by source (scenario name, resource name)
  --level <level>   Filter by level (debug, info, warn, error)
  --search <text>   Search for text in log messages
  --tail <n>        Number of lines to fetch (default: 100)
  --since <time>    Fetch logs since (timestamp or duration like "1h")
  --json            Output raw JSON`)
			return nil
		case "--source":
			if i+1 < len(args) {
				i++
				opts.Source = args[i]
			}
		case "--level":
			if i+1 < len(args) {
				i++
				opts.Level = args[i]
			}
		case "--search":
			if i+1 < len(args) {
				i++
				opts.Search = args[i]
			}
		case "--tail":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					opts.Tail = n
				}
			}
		case "--since":
			if i+1 < len(args) {
				i++
				opts.Since = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && id == "" {
				id = args[i]
			}
		}
	}

	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud inspect logs <deployment-id>")
	}

	body, resp, err := client.Logs(id, opts)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if len(resp.Logs) == 0 {
		fmt.Println("No logs found matching the criteria.")
		return nil
	}

	fmt.Printf("Logs for Deployment: %s (%d entries)\n", resp.DeploymentID, resp.TotalCount)
	fmt.Println(strings.Repeat("-", 80))

	for _, log := range resp.Logs {
		levelIcon := " "
		switch log.Level {
		case "error":
			levelIcon = "E"
		case "warn", "warning":
			levelIcon = "W"
		case "info":
			levelIcon = "I"
		case "debug":
			levelIcon = "D"
		}
		fmt.Printf("[%s] %s [%-15s] %s\n", levelIcon, log.Timestamp, truncate(log.Source, 15), log.Message)
	}

	if resp.HasMore {
		fmt.Printf("\n... %d more entries available (use --tail to fetch more)\n", resp.TotalCount-len(resp.Logs))
	}

	return nil
}

func runFiles(client *Client, args []string) error {
	var id string
	opts := FilesOptions{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud inspect files <deployment-id> [path] [flags]

Flags:
  --content    Read file content instead of listing directory
  --json       Output raw JSON

Examples:
  scenario-to-cloud inspect files abc123                    # List root directory
  scenario-to-cloud inspect files abc123 /var/log          # List /var/log
  scenario-to-cloud inspect files abc123 /etc/nginx.conf --content  # Read file`)
			return nil
		case "--content":
			opts.Content = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if id == "" {
					id = args[i]
				} else if opts.Path == "" {
					opts.Path = args[i]
				}
			}
		}
	}

	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud inspect files <deployment-id> [path]")
	}

	body, resp, err := client.Files(id, opts)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if opts.Content && resp.Content != "" {
		fmt.Printf("File: %s\n", resp.Path)
		fmt.Println(strings.Repeat("-", 50))
		fmt.Print(resp.Content)
		if !strings.HasSuffix(resp.Content, "\n") {
			fmt.Println()
		}
		return nil
	}

	if len(resp.Files) == 0 {
		fmt.Printf("No files found at: %s\n", resp.Path)
		return nil
	}

	fmt.Printf("Directory: %s\n", resp.Path)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-10s %-12s %-20s %s\n", "MODE", "SIZE", "MODIFIED", "NAME")

	for _, f := range resp.Files {
		sizeStr := formatSize(f.Size)
		name := f.Name
		if f.IsDirectory {
			name = name + "/"
		}
		fmt.Printf("%-10s %-12s %-20s %s\n", f.Mode, sizeStr, f.ModTime, name)
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
