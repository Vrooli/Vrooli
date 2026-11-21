package inspect

import (
	"fmt"
	"io"
	"os"
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
	case "metrics":
		return runMetrics(client, args[1:])
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
  metrics <deployment-id>   Debug VPS system metrics collection
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
	id, jsonOutput, showHelp := parseDeploymentArgs(args)
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud inspect live <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
		return nil
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
	writeLiveState(os.Stdout, resp)
	return nil
}

func writeLiveState(w io.Writer, resp LiveStateResponse) {
	fmt.Fprintf(w, "Live State for Deployment: %s\n", resp.DeploymentID)
	fmt.Fprintln(w, strings.Repeat("-", 50))
	s := resp.State
	statusIcon, healthIcon := liveStatusIcons(s)
	fmt.Fprintf(w, "Status:   [%s] Running: %v  [%s] Healthy: %v\n", statusIcon, s.Running, healthIcon, s.Healthy)
	writeLiveSummary(w, s)
	writeLiveProcesses(w, resp.Processes)
	writeLiveResources(w, resp.Resources)
}

func liveStatusIcons(state LiveState) (statusIcon, healthIcon string) {
	statusIcon, healthIcon = "X", "!"
	if state.Running {
		statusIcon = "R"
	}
	if state.Healthy {
		healthIcon = "H"
	}
	return statusIcon, healthIcon
}

func writeLiveSummary(w io.Writer, state LiveState) {
	if state.Uptime != "" {
		fmt.Fprintf(w, "Uptime:   %s\n", state.Uptime)
	}
	if state.PublicIP != "" {
		fmt.Fprintf(w, "Public IP: %s\n", state.PublicIP)
	}
	if state.InternalIP != "" {
		fmt.Fprintf(w, "Internal IP: %s\n", state.InternalIP)
	}
	if state.CPUPercent != "" || state.MemoryPercent != "" || state.DiskUsedGB != "" {
		fmt.Fprintf(w, "Resources: CPU=%s  Memory=%s  Disk=%s\n", state.CPUPercent, state.MemoryPercent, state.DiskUsedGB)
	}
	if state.ErrorMessage != "" {
		fmt.Fprintf(w, "Error:    %s\n", state.ErrorMessage)
	}
}

func writeLiveProcesses(w io.Writer, processes []ProcessInfo) {
	if len(processes) == 0 {
		return
	}
	fmt.Fprintln(w, "\nProcesses:")
	fmt.Fprintf(w, "  %-8s %-20s %-10s %-8s %s\n", "PID", "NAME", "STATUS", "CPU", "MEMORY")
	for _, process := range processes {
		fmt.Fprintf(w, "  %-8d %-20s %-10s %-8s %s\n", process.PID, truncate(process.Name, 20), process.Status, process.CPUPercent, process.MemoryMB)
	}
}

func writeLiveResources(w io.Writer, resources []ResourceStatus) {
	if len(resources) == 0 {
		return
	}
	fmt.Fprintln(w, "\nResources:")
	fmt.Fprintf(w, "  %-20s %-15s %-10s %-7s %s\n", "NAME", "TYPE", "STATUS", "PORT", "HEALTHY")
	for _, resource := range resources {
		health := "no"
		if resource.Healthy {
			health = "yes"
		}
		port := "-"
		if resource.Port > 0 {
			port = strconv.Itoa(resource.Port)
		}
		fmt.Fprintf(w, "  %-20s %-15s %-10s %-7s %s\n", truncate(resource.Name, 20), resource.Type, resource.Status, port, health)
	}
}

func runDrift(client *Client, args []string) error {
	id, jsonOutput, showHelp := parseDeploymentArgs(args)
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud inspect drift <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
		return nil
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
	writeDrift(os.Stdout, resp)
	return nil
}

func parseDeploymentArgs(args []string) (id string, jsonOutput, showHelp bool) {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return id, jsonOutput, true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(arg, "-") && id == "" {
				id = arg
			}
		}
	}
	return id, jsonOutput, false
}

func writeDrift(w io.Writer, resp DriftResponse) {
	fmt.Fprintf(w, "Drift Detection for Deployment: %s\n", resp.DeploymentID)
	fmt.Fprintln(w, strings.Repeat("-", 50))
	if !resp.HasDrift {
		fmt.Fprintln(w, "No drift detected. Configuration matches expected state.")
		fmt.Fprintf(w, "Checked at: %s\n", resp.CheckedAt)
		return
	}
	fmt.Fprintf(w, "DRIFT DETECTED: %d item(s)\n\n", len(resp.DriftItems))
	for i, item := range resp.DriftItems {
		writeDriftItem(w, i, item)
	}
	fmt.Fprintf(w, "\nChecked at: %s\n", resp.CheckedAt)
}

func writeDriftItem(w io.Writer, index int, item DriftItem) {
	fmt.Fprintf(w, "%d. [%s] %s (%s)\n", index+1, driftSeverityIcon(item.Severity), item.Path, item.Type)
	if item.Expected != "" {
		fmt.Fprintf(w, "   Expected: %s\n", item.Expected)
	}
	if item.Actual != "" {
		fmt.Fprintf(w, "   Actual:   %s\n", item.Actual)
	}
	if item.Message != "" {
		fmt.Fprintf(w, "   Message:  %s\n", item.Message)
	}
}

func driftSeverityIcon(severity string) string {
	switch severity {
	case "error":
		return "E"
	case "warning":
		return "W"
	case "info":
		return "I"
	default:
		return "?"
	}
}

func runMetrics(client *Client, args []string) error {
	id, jsonOutput, showHelp := parseDeploymentArgs(args)
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud inspect metrics <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
		return nil
	}
	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud inspect metrics <deployment-id>")
	}

	body, resp, err := client.MetricsDebug(id)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	writeMetrics(os.Stdout, resp)
	return nil
}

func writeMetrics(w io.Writer, resp MetricsDebugResponse) {
	result := resp.Result
	fmt.Fprintf(w, "Metrics Debug for Deployment: %s\n", resp.DeploymentID)
	fmt.Fprintln(w, strings.Repeat("-", 70))
	writeMetricsSystem(w, result)
	writeMetricsCommands(w, result.Commands)
}

func writeMetricsSystem(w io.Writer, result MetricsDebugResult) {
	keyAuthState := defaultIfBlank(result.System.SSH.VerificationState, "unknown")
	authMode := defaultIfBlank(result.System.SSH.AuthMode, "unknown")
	fmt.Fprintf(w, "Collector: %s", result.Collector)
	if result.OSID != "" {
		fmt.Fprintf(w, "  OS: %s", result.OSID)
		if result.OSVersion != "" {
			fmt.Fprintf(w, " %s", result.OSVersion)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "SSH: connected=%v latency=%dms auth_mode=%s key_auth=%s\n", result.System.SSH.Connected, result.System.SSH.LatencyMs, authMode, keyAuthState)
	fmt.Fprintf(w, "CPU: usage=%.1f%% cores=%d load=[%.2f %.2f %.2f]\n",
		result.System.CPU.UsagePercent,
		result.System.CPU.Cores,
		safeLoad(result.System.CPU.LoadAverage, 0),
		safeLoad(result.System.CPU.LoadAverage, 1),
		safeLoad(result.System.CPU.LoadAverage, 2),
	)
	fmt.Fprintf(w, "Memory: used=%dMB total=%dMB (%.1f%%)\n", result.System.Memory.UsedMB, result.System.Memory.TotalMB, result.System.Memory.UsagePercent)
	fmt.Fprintf(w, "Disk: used=%dGB total=%dGB (%.1f%%)\n", result.System.Disk.UsedGB, result.System.Disk.TotalGB, result.System.Disk.UsagePercent)
	fmt.Fprintf(w, "       bytes: memory=%d/%d disk=%d/%d\n", result.System.Memory.UsedBytes, result.System.Memory.TotalBytes, result.System.Disk.UsedBytes, result.System.Disk.TotalBytes)
	fmt.Fprintf(w, "Swap: used=%dMB total=%dMB (%.1f%%)\n", result.System.Swap.UsedMB, result.System.Swap.TotalMB, result.System.Swap.UsagePercent)
	if result.Error != "" {
		fmt.Fprintf(w, "Error: %s\n", result.Error)
	}
}

func writeMetricsCommands(w io.Writer, commands []MetricsDebugCommand) {
	fmt.Fprintln(w, "\nCommands:")
	fmt.Fprintf(w, "  %-15s %-6s %-8s %s\n", "ID", "EXIT", "TIME", "COMMAND")
	for _, command := range commands {
		status := strconv.Itoa(command.ExitCode)
		if command.Error != "" {
			status = "ERR"
		}
		fmt.Fprintf(w, "  %-15s %-6s %-8d %s\n", truncate(command.ID, 15), status, command.DurationMs, truncate(command.Command, 64))
		if command.Error != "" {
			fmt.Fprintf(w, "    error: %s\n", command.Error)
		}
	}
}

func defaultIfBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// Log retrieval bounds. The API caps tail at MaxLogTail and redacts
// credential-sensitive content before it leaves the target owner; the CLI
// only bounds what it prints and never scans for secrets itself.
const (
	MaxLogTail        = 2000
	DefaultLogTail    = 100
	DefaultLogMaxByte = 1 << 20
)

func runLogs(client *Client, args []string) error {
	id, opts, jsonOutput, maxBytes, showHelp, err := parseLogsArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		printLogsUsage()
		return nil
	}
	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud inspect logs <deployment-id>")
	}

	body, resp, err := client.Logs(id, opts)
	if err != nil {
		return err
	}

	if jsonOutput {
		if len(body) > maxBytes {
			return fmt.Errorf("log response is %d bytes, above --max-bytes %d; narrow --tail, --source or --since", len(body), maxBytes)
		}
		cliutil.PrintJSON(body)
		return nil
	}
	return writeLogs(os.Stdout, resp, maxBytes)
}

func printLogsUsage() {
	fmt.Println(`Usage: scenario-to-cloud inspect logs <deployment-id> [flags]

Bounded, redacted log retrieval: the API caps --tail at 2000 lines and
redacts credential-sensitive content; --max-bytes bounds the printed output.

Flags:
  --source <name>   Filter by source (scenario name, resource name)
  --level <level>   Filter by level (debug, info, warn, error)
  --search <text>   Search for text in log messages
  --tail <n>        Number of lines to fetch (default: 100, max: 2000)
  --since <time>    Fetch logs since (timestamp or duration like "1h")
  --max-bytes <n>   Maximum bytes to print (default: 1048576)
  --json            Output raw JSON`)
}

func parseLogsArgs(args []string) (id string, opts LogsOptions, jsonOutput bool, maxBytes int, showHelp bool, err error) {
	opts.Tail = DefaultLogTail
	maxBytes = DefaultLogMaxByte
	for i := 0; i < len(args); i++ {
		if next, handled, optionErr := parseLogQueryOption(args, i, &opts); handled {
			i = next
			if optionErr != nil {
				return "", opts, false, maxBytes, false, optionErr
			}
			continue
		}
		next, handled, help := parseLogOutputOption(args, i, &jsonOutput, &maxBytes)
		if help {
			return id, opts, jsonOutput, maxBytes, true, nil
		}
		if handled {
			i = next
			continue
		}
		if !strings.HasPrefix(args[i], "-") && id == "" {
			id = args[i]
		}
	}
	return id, opts, jsonOutput, maxBytes, showHelp, nil
}

func parseLogQueryOption(args []string, index int, opts *LogsOptions) (next int, handled bool, err error) {
	flag := args[index]
	switch flag {
	case "--source":
		return parseLogStringOption(args, index, &opts.Source)
	case "--level":
		return parseLogStringOption(args, index, &opts.Level)
	case "--search":
		return parseLogStringOption(args, index, &opts.Search)
	case "--since":
		return parseLogStringOption(args, index, &opts.Since)
	case "--tail":
		return parseLogTailOption(args, index, opts)
	default:
		return index, false, nil
	}
}

func parseLogStringOption(args []string, index int, destination *string) (next int, handled bool, err error) {
	if index+1 >= len(args) {
		return index, true, nil
	}
	*destination = args[index+1]
	return index + 1, true, nil
}

func parseLogTailOption(args []string, index int, opts *LogsOptions) (next int, handled bool, err error) {
	if index+1 >= len(args) {
		return index, true, nil
	}
	value := args[index+1]
	tail, parseErr := strconv.Atoi(value)
	if parseErr != nil || tail <= 0 {
		return index, true, fmt.Errorf("--tail must be a positive integer, got %q", value)
	}
	if tail > MaxLogTail {
		return index, true, fmt.Errorf("--tail %d exceeds the API bound of %d lines", tail, MaxLogTail)
	}
	opts.Tail = tail
	return index + 1, true, nil
}

func parseLogOutputOption(args []string, index int, jsonOutput *bool, maxBytes *int) (next int, handled bool, help bool) {
	switch args[index] {
	case "-h", "--help":
		return index, true, true
	case "--max-bytes":
		if index+1 >= len(args) {
			return index, true, false
		}
		n, err := strconv.Atoi(args[index+1])
		if err == nil && n > 0 {
			*maxBytes = n
		}
		return index + 1, true, false
	case "--json":
		*jsonOutput = true
		return index, true, false
	default:
		return index, false, false
	}
}

// writeLogs prints entries as returned by the API (already redacted) and
// stops at maxBytes, saying so.
func writeLogs(w io.Writer, resp LogsResponse, maxBytes int) error {
	if len(resp.Logs) == 0 {
		fmt.Fprintln(w, "No logs found matching the criteria.")
		return nil
	}
	written := 0
	emit := func(line string) bool {
		if written+len(line)+1 > maxBytes {
			return false
		}
		fmt.Fprintln(w, line)
		written += len(line) + 1
		return true
	}
	emit(fmt.Sprintf("Logs for Deployment: %s (%d entries)", resp.DeploymentID, resp.TotalCount))
	emit(strings.Repeat("-", 80))
	for i, log := range resp.Logs {
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
		if !emit(fmt.Sprintf("[%s] %s [%-15s] %s", levelIcon, log.Timestamp, truncate(log.Source, 15), log.Message)) {
			fmt.Fprintf(w, "\n... output bounded at %d bytes after %d of %d entries (raise --max-bytes or narrow the query)\n", maxBytes, i, len(resp.Logs))
			return nil
		}
	}
	if resp.HasMore {
		fmt.Fprintf(w, "\n... %d more entries available (use --tail, max %d)\n", resp.TotalCount-len(resp.Logs), MaxLogTail)
	}
	return nil
}

func runFiles(client *Client, args []string) error {
	id, opts, jsonOutput, showHelp := parseFilesArgs(args)
	if showHelp {
		fmt.Println(`Usage: scenario-to-cloud inspect files <deployment-id> [path] [flags]

Flags:
  --content    Read file content instead of listing directory
  --json       Output raw JSON

Examples:
  scenario-to-cloud inspect files abc123                    # List root directory
  scenario-to-cloud inspect files abc123 /var/log          # List /var/log
  scenario-to-cloud inspect files abc123 /etc/nginx.conf --content  # Read file`)
		return nil
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
	writeFiles(os.Stdout, opts, resp)
	return nil
}

func parseFilesArgs(args []string) (id string, opts FilesOptions, jsonOutput, showHelp bool) {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return id, opts, jsonOutput, true
		case "--content":
			opts.Content = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(arg, "-") {
				id, opts.Path = assignFilePositionals(id, opts.Path, arg)
			}
		}
	}
	return id, opts, jsonOutput, false
}

func assignFilePositionals(id, path, value string) (string, string) {
	if id == "" {
		return value, path
	}
	if path == "" {
		return id, value
	}
	return id, path
}

func writeFiles(w io.Writer, opts FilesOptions, resp FilesResponse) {
	if opts.Content && resp.Content != "" {
		fmt.Fprintf(w, "File: %s\n", resp.Path)
		fmt.Fprintln(w, strings.Repeat("-", 50))
		fmt.Fprint(w, resp.Content)
		if !strings.HasSuffix(resp.Content, "\n") {
			fmt.Fprintln(w)
		}
		return
	}
	if len(resp.Files) == 0 {
		fmt.Fprintf(w, "No files found at: %s\n", resp.Path)
		return
	}
	fmt.Fprintf(w, "Directory: %s\n", resp.Path)
	fmt.Fprintln(w, strings.Repeat("-", 80))
	fmt.Fprintf(w, "%-10s %-12s %-20s %s\n", "MODE", "SIZE", "MODIFIED", "NAME")
	for _, file := range resp.Files {
		writeFileRow(w, file)
	}
}

func writeFileRow(w io.Writer, file FileInfo) {
	name := file.Name
	if file.IsDirectory {
		name += "/"
	}
	fmt.Fprintf(w, "%-10s %-12s %-20s %s\n", file.Mode, formatSize(file.Size), file.ModTime, name)
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

func safeLoad(loads []float64, index int) float64 {
	if index < 0 || index >= len(loads) {
		return 0
	}
	return loads[index]
}
