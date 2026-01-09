package preflight

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

// ufwAllowsPort checks if a UFW rule line allows the specified port.
func ufwAllowsPort(line string, port int) bool {
	pattern := fmt.Sprintf(`(^|[^0-9])%d([^0-9]|$)`, port)
	re := regexp.MustCompile(pattern)
	return re.MatchString(line)
}

// parseOSRelease extracts ID and VERSION_ID from /etc/os-release content.
func parseOSRelease(contents string) (id, versionID string) {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			versionID = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
		}
	}
	return strings.ToLower(id), versionID
}

// EdgePortBinding represents a port binding on edge ports (80/443).
type EdgePortBinding struct {
	Port    int    `json:"port"`
	Process string `json:"process,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Service string `json:"service,omitempty"`
}

// parsePortBindings parses ss output to extract port bindings on 80/443.
func parsePortBindings(ssOutput string) []EdgePortBinding {
	bindings := ParseSSOutput(ssOutput)
	filtered := make([]EdgePortBinding, 0, len(bindings))

	for _, binding := range bindings {
		if binding.Port != 80 && binding.Port != 443 {
			continue
		}
		edge := EdgePortBinding{
			Port:    binding.Port,
			Process: binding.Process,
		}
		if binding.PID != nil {
			edge.PID = *binding.PID
		}
		if edge.Process != "" {
			edge.Service = serviceFromProcess(edge.Process)
		}
		filtered = append(filtered, edge)
	}

	if len(filtered) == 0 {
		filtered = append(filtered, fallbackEdgeBindings(ssOutput)...)
	}

	return filtered
}

// fallbackEdgeBindings extracts edge port bindings from ss output when process parsing fails.
func fallbackEdgeBindings(ssOutput string) []EdgePortBinding {
	var bindings []EdgePortBinding
	seen := map[int]bool{}

	for _, line := range strings.Split(ssOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, ":80") && !seen[80] {
			seen[80] = true
			bindings = append(bindings, EdgePortBinding{Port: 80})
		}
		if strings.Contains(line, ":443") && !seen[443] {
			seen[443] = true
			bindings = append(bindings, EdgePortBinding{Port: 443})
		}
	}

	return bindings
}

// formatPortBindings formats port bindings for human-readable display.
func formatPortBindings(bindings []EdgePortBinding) string {
	var parts []string
	for _, binding := range bindings {
		part := fmt.Sprintf("port %d", binding.Port)
		if binding.Process != "" {
			part = fmt.Sprintf("%s: %s", part, binding.Process)
		}
		if binding.PID > 0 {
			part = fmt.Sprintf("%s (pid %d)", part, binding.PID)
		}
		if binding.Service != "" && binding.Service != binding.Process {
			part = fmt.Sprintf("%s [service %s]", part, binding.Service)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | ")
}

// portBindingProcessList returns a list of process descriptions from bindings.
func portBindingProcessList(bindings []EdgePortBinding) []string {
	var processes []string
	for _, binding := range bindings {
		if binding.Process != "" {
			if binding.PID > 0 {
				processes = append(processes, fmt.Sprintf("%s (pid %d)", binding.Process, binding.PID))
			} else {
				processes = append(processes, binding.Process)
			}
			continue
		}
		processes = append(processes, fmt.Sprintf("port %d", binding.Port))
	}
	return processes
}

// portBindingPorts returns unique port numbers as strings.
func portBindingPorts(bindings []EdgePortBinding) []string {
	seen := map[int]bool{}
	var ports []string
	for _, binding := range bindings {
		if seen[binding.Port] {
			continue
		}
		seen[binding.Port] = true
		ports = append(ports, strconv.Itoa(binding.Port))
	}
	return ports
}

// serviceFromProcess maps process names to systemd service names.
func serviceFromProcess(process string) string {
	switch strings.ToLower(process) {
	case "nginx":
		return "nginx"
	case "apache2", "httpd":
		return "apache2"
	case "caddy":
		return "caddy"
	default:
		return ""
	}
}

// formatBytes converts kilobytes to human-readable format.
func formatBytes(kb int64) string {
	if kb >= 1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(kb)/(1024*1024))
	} else if kb >= 1024 {
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	}
	return fmt.Sprintf("%d KB", kb)
}

// checkStaleScenarioProcesses detects running processes for the target scenario.
// This catches cases where old processes are still running, which would prevent
// the newly deployed version from starting correctly.
func checkStaleScenarioProcesses(
	ctx context.Context,
	cfg ssh.Config,
	sshRunner ssh.Runner,
	manifest domain.CloudManifest,
	warn func(id, title, details, hint string, data map[string]string),
	pass func(id, title, details string, data map[string]string),
) {
	scenarioID := manifest.Scenario.ID
	if scenarioID == "" {
		return // Skip if no scenario specified
	}

	// Find processes matching scenario name
	// Look for actual runtime processes (node, python, go, java, pm2) that contain the scenario ID
	// Exclude: grep itself, editors, log viewers, shell sessions
	cmd := fmt.Sprintf(
		`ps aux --no-headers | grep -E '%s' | grep -v -E '(grep|vim|nano|less|tail|cat|ssh|sshd)' || true`,
		scenarioID,
	)
	result, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		warn("stale_processes", "Stale process check",
			"Unable to check for running processes",
			"SSH command failed - check connectivity",
			nil)
		return
	}

	// Parse the process output
	stdout := strings.TrimSpace(result.Stdout)
	if stdout == "" {
		pass("stale_processes", "Stale process check",
			"No existing scenario processes found",
			nil)
		return
	}

	// Count and describe processes
	lines := strings.Split(stdout, "\n")
	var processInfos []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Extract PID and command from ps aux output
		// Format: USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND
		fields := strings.Fields(line)
		if len(fields) >= 11 {
			pid := fields[1]
			// Command is everything from field 10 onwards
			command := strings.Join(fields[10:], " ")
			// Truncate long commands
			if len(command) > 80 {
				command = command[:80] + "..."
			}
			processInfos = append(processInfos, fmt.Sprintf("PID %s: %s", pid, command))
		}
	}

	if len(processInfos) == 0 {
		pass("stale_processes", "Stale process check",
			"No existing scenario processes found",
			nil)
		return
	}

	// Include process details in the visible message
	detailsMsg := fmt.Sprintf("Found %d running process(es) for %s: %s",
		len(processInfos), scenarioID, strings.Join(processInfos, " | "))

	warn("stale_processes", "Stale process check",
		detailsMsg,
		"Stop existing processes to ensure the updated version deploys correctly",
		map[string]string{
			"count":   fmt.Sprintf("%d", len(processInfos)),
			"details": strings.Join(processInfos, "; "),
		})
}

// ParseSSOutput parses the output of `ss -tlnp`.
// Format: State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
// Example: LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1234,fd=3))
func ParseSSOutput(output string) []domain.PortBinding {
	var ports []domain.PortBinding
	lines := strings.Split(output, "\n")

	// Regex to extract port from address
	portRegex := regexp.MustCompile(`:(\d+)$`)
	// Regex to extract process info from users:((..."name",pid=123,...))
	processRegex := regexp.MustCompile(`\(\("([^"]+)",pid=(\d+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header line
		if strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Netid") {
			continue
		}

		// Skip non-LISTEN lines
		if !strings.HasPrefix(line, "LISTEN") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Extract port from local address (field 3 or 4 depending on format)
		localAddr := fields[3]
		portMatch := portRegex.FindStringSubmatch(localAddr)
		if len(portMatch) < 2 {
			continue
		}

		port, err := strconv.Atoi(portMatch[1])
		if err != nil {
			continue
		}

		// Extract process name and PID
		var processName string
		var pid int
		if len(fields) >= 6 {
			processInfo := fields[5]
			if len(fields) > 6 {
				processInfo = strings.Join(fields[5:], " ")
			}
			procMatch := processRegex.FindStringSubmatch(processInfo)
			if len(procMatch) >= 3 {
				processName = procMatch[1]
				pid, _ = strconv.Atoi(procMatch[2])
			}
		}

		binding := domain.PortBinding{
			Port:    port,
			Process: processName,
		}
		if pid > 0 {
			binding.PID = &pid
		}

		ports = append(ports, binding)
	}

	return ports
}

// ExtractPIDsFromSS parses PIDs from ss -ltnp output.
// Example: LISTEN 0 4096 *:80 *:* users:(("nginx",pid=1234,fd=6))
func ExtractPIDsFromSS(output string) []string {
	var pids []string
	seen := make(map[string]bool)

	pidRegex := regexp.MustCompile(`pid=(\d+)`)

	for _, line := range strings.Split(output, "\n") {
		matches := pidRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pid := match[1]
				if !seen[pid] {
					seen[pid] = true
					pids = append(pids, pid)
				}
			}
		}
	}

	return pids
}
