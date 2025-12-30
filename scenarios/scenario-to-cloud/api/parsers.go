package main

import (
	"regexp"
	"strconv"
	"strings"
)

// ProcessInfo represents a parsed process from ps output.
type ProcessInfo struct {
	User          string
	PID           int
	CPUPercent    float64
	MemoryPercent float64
	MemoryMB      float64
	VSZ           int64 // Virtual memory size in KB
	RSS           int64 // Resident set size in KB
	TTY           string
	Stat          string
	Start         string
	Time          string
	Command       string
}

// parsePSOutput parses the output of `ps aux --no-headers`.
// Format: USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND
func parsePSOutput(output string) []ProcessInfo {
	var processes []ProcessInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by whitespace, but command can contain spaces
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, _ := strconv.Atoi(fields[1])
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		memPercent, _ := strconv.ParseFloat(fields[3], 64)
		vsz, _ := strconv.ParseInt(fields[4], 10, 64)
		rss, _ := strconv.ParseInt(fields[5], 10, 64)

		// Command is everything from field 10 onwards
		command := strings.Join(fields[10:], " ")

		processes = append(processes, ProcessInfo{
			User:          fields[0],
			PID:           pid,
			CPUPercent:    cpuPercent,
			MemoryPercent: memPercent,
			VSZ:           vsz,
			RSS:           rss,
			MemoryMB:      float64(rss) / 1024.0,
			TTY:           fields[6],
			Stat:          fields[7],
			Start:         fields[8],
			Time:          fields[9],
			Command:       command,
		})
	}

	return processes
}

// parseSSOutput parses the output of `ss -tlnp`.
// Format: State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
// Example: LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1234,fd=3))
func parseSSOutput(output string) []PortBinding {
	var ports []PortBinding
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

		binding := PortBinding{
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

// parseSystemState parses various system info commands into SystemState.
func parseSystemState(results map[string]sshCommandResult, keyPath, pubKeyContent string, latencyMs int64) SystemState {
	state := SystemState{}

	// Parse SSH health
	state.SSH = SSHHealth{
		Connected: true, // If we got results, SSH is connected
		LatencyMs: latencyMs,
		KeyPath:   keyPath,
	}

	// Check if our public key is in authorized_keys
	if sshKeyResult, ok := results["ssh_key_check"]; ok && sshKeyResult.err == nil {
		authorizedKeys := sshKeyResult.result.Stdout
		if pubKeyContent != "" {
			// Extract just the key part (type and base64 content, ignore comment)
			pubKeyParts := strings.Fields(pubKeyContent)
			if len(pubKeyParts) >= 2 {
				keyToMatch := pubKeyParts[0] + " " + pubKeyParts[1]
				state.SSH.KeyInAuth = strings.Contains(authorizedKeys, keyToMatch)
			}
		}
	}

	// Parse CPU info
	if coresResult, ok := results["cpuinfo"]; ok && coresResult.err == nil {
		cores, _ := strconv.Atoi(strings.TrimSpace(coresResult.result.Stdout))
		state.CPU.Cores = cores
	}

	// Parse CPU model
	if modelResult, ok := results["cpumodel"]; ok && modelResult.err == nil {
		// Format: model name : Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz
		line := strings.TrimSpace(modelResult.result.Stdout)
		if idx := strings.Index(line, ":"); idx >= 0 {
			state.CPU.Model = strings.TrimSpace(line[idx+1:])
		}
	}

	// Parse load average
	if loadResult, ok := results["loadavg"]; ok && loadResult.err == nil {
		// Format: 1.24 1.18 0.95 1/234 5678
		fields := strings.Fields(loadResult.result.Stdout)
		if len(fields) >= 3 {
			load1, _ := strconv.ParseFloat(fields[0], 64)
			load5, _ := strconv.ParseFloat(fields[1], 64)
			load15, _ := strconv.ParseFloat(fields[2], 64)
			state.CPU.LoadAverage = []float64{load1, load5, load15}
		}
	}

	// Parse actual CPU usage from top output
	// Format varies but typically: %Cpu(s):  1.2 us,  0.8 sy,  0.0 ni, 97.5 id, ...
	// or: Cpu(s):  1.2%us,  0.8%sy,  0.0%ni, 97.5%id, ...
	if cpuUsageResult, ok := results["cpuusage"]; ok && cpuUsageResult.err == nil {
		state.CPU.UsagePercent = parseCPUUsageFromTop(cpuUsageResult.result.Stdout)
	}

	// Fallback: estimate from load average if top parsing failed
	if state.CPU.UsagePercent == 0 && state.CPU.Cores > 0 && len(state.CPU.LoadAverage) > 0 {
		// Use a more conservative formula: load / (cores * 2) to avoid always showing 100%
		// This is still an approximation but won't max out as easily
		state.CPU.UsagePercent = (state.CPU.LoadAverage[0] / float64(state.CPU.Cores*2)) * 100
		if state.CPU.UsagePercent > 100 {
			state.CPU.UsagePercent = 100
		}
	}

	// Parse uptime
	if uptimeResult, ok := results["uptime"]; ok && uptimeResult.err == nil {
		// Format: 389593.24 1558372.96
		fields := strings.Fields(uptimeResult.result.Stdout)
		if len(fields) >= 1 {
			uptime, _ := strconv.ParseFloat(fields[0], 64)
			state.UptimeSeconds = int64(uptime)
		}
	}

	// Parse memory (free -m output)
	if freeResult, ok := results["free"]; ok && freeResult.err == nil {
		lines := strings.Split(freeResult.result.Stdout, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}

			if strings.HasPrefix(fields[0], "Mem:") {
				// Format: Mem: total used free shared buff/cache available
				total, _ := strconv.Atoi(fields[1])
				used, _ := strconv.Atoi(fields[2])
				free, _ := strconv.Atoi(fields[3])

				state.Memory.TotalMB = total
				state.Memory.UsedMB = used
				state.Memory.FreeMB = free
				if total > 0 {
					state.Memory.UsagePercent = float64(used) / float64(total) * 100
				}
			} else if strings.HasPrefix(fields[0], "Swap:") {
				// Format: Swap: total used free
				total, _ := strconv.Atoi(fields[1])
				used, _ := strconv.Atoi(fields[2])

				state.Swap.TotalMB = total
				state.Swap.UsedMB = used
				if total > 0 {
					state.Swap.UsagePercent = float64(used) / float64(total) * 100
				}
			}
		}
	}

	// Parse disk (df -h / output)
	if dfResult, ok := results["df"]; ok && dfResult.err == nil {
		// Format: Filesystem Size Used Avail Use% Mounted
		// Example: /dev/sda1 200G 84G 116G 42% /
		fields := strings.Fields(dfResult.result.Stdout)
		if len(fields) >= 5 {
			// Parse size values (e.g., "200G", "84G")
			state.Disk.TotalGB = parseHumanSize(fields[1])
			state.Disk.UsedGB = parseHumanSize(fields[2])
			state.Disk.FreeGB = parseHumanSize(fields[3])

			// Parse usage percentage (e.g., "42%")
			usageStr := strings.TrimSuffix(fields[4], "%")
			usage, _ := strconv.ParseFloat(usageStr, 64)
			state.Disk.UsagePercent = usage
		}
	}

	return state
}

// parseHumanSize converts human-readable size (e.g., "200G", "84M") to GB.
func parseHumanSize(s string) int {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0
	}

	// Extract numeric part and unit
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	switch unit {
	case 'T', 't':
		return int(num * 1024)
	case 'G', 'g':
		return int(num)
	case 'M', 'm':
		return int(num / 1024)
	case 'K', 'k':
		return int(num / (1024 * 1024))
	default:
		// Assume bytes if no unit
		return int(num / (1024 * 1024 * 1024))
	}
}

// parseCaddyState parses Caddy configuration and status.
func parseCaddyState(caddyfileContent, runningStatus, expectedDomain string) CaddyState {
	state := CaddyState{
		Running: strings.TrimSpace(runningStatus) == "running",
		Domain:  expectedDomain,
		Routes:  []CaddyRoute{},
	}

	// Parse routes from Caddyfile
	// Simple parsing - looks for reverse_proxy directives
	if caddyfileContent != "" {
		state.Routes = parseCaddyRoutes(caddyfileContent)
	}

	// TLS info would require additional commands like openssl s_client
	// For now, just set basic info
	if state.Running {
		state.TLS = TLSInfo{
			Valid: true, // Assume valid if Caddy is running with auto-TLS
		}
	}

	return state
}

// parseCPUUsageFromTop parses CPU usage from /proc/stat samples.
// We sample /proc/stat twice with 1 second delay to get accurate current usage.
// /proc/stat format:
//
//	cpu  10132153 290696 3084719 46828483 16683 0 25195 0 0 0
//
// Columns: user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice
// CPU usage = (total_diff - idle_diff) / total_diff * 100
func parseCPUUsageFromTop(output string) float64 {
	output = strings.TrimSpace(output)
	if output == "" {
		return 0
	}

	// Split into lines - we expect two /proc/stat samples
	lines := strings.Split(output, "\n")
	var cpuLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "cpu ") {
			cpuLines = append(cpuLines, line)
		}
	}

	if len(cpuLines) >= 2 {
		// Parse both samples
		stats1 := parseProcStatLine(cpuLines[0])
		stats2 := parseProcStatLine(cpuLines[1])

		if stats1 != nil && stats2 != nil {
			// Calculate differences
			total1 := stats1.user + stats1.nice + stats1.system + stats1.idle + stats1.iowait + stats1.irq + stats1.softirq + stats1.steal
			total2 := stats2.user + stats2.nice + stats2.system + stats2.idle + stats2.iowait + stats2.irq + stats2.softirq + stats2.steal
			idle1 := stats1.idle + stats1.iowait
			idle2 := stats2.idle + stats2.iowait

			totalDiff := total2 - total1
			idleDiff := idle2 - idle1

			if totalDiff > 0 {
				usage := float64(totalDiff-idleDiff) / float64(totalDiff) * 100
				if usage < 0 {
					usage = 0
				}
				if usage > 100 {
					usage = 100
				}
				return usage
			}
		}
	}

	// Fallback for single sample - calculate since-boot average (less accurate)
	if len(cpuLines) >= 1 {
		stats := parseProcStatLine(cpuLines[0])
		if stats != nil {
			total := stats.user + stats.nice + stats.system + stats.idle + stats.iowait + stats.irq + stats.softirq + stats.steal
			idle := stats.idle + stats.iowait
			if total > 0 {
				usage := float64(total-idle) / float64(total) * 100
				if usage < 0 {
					usage = 0
				}
				if usage > 100 {
					usage = 100
				}
				return usage
			}
		}
	}

	return 0
}

// cpuStats holds parsed /proc/stat CPU values
type cpuStats struct {
	user, nice, system, idle, iowait, irq, softirq, steal int64
}

// parseProcStatLine parses a single "cpu ..." line from /proc/stat
func parseProcStatLine(line string) *cpuStats {
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return nil
	}

	// Parse at least user, nice, system, idle
	// Optional: iowait, irq, softirq, steal
	stats := &cpuStats{}
	if len(fields) > 1 {
		stats.user, _ = strconv.ParseInt(fields[1], 10, 64)
	}
	if len(fields) > 2 {
		stats.nice, _ = strconv.ParseInt(fields[2], 10, 64)
	}
	if len(fields) > 3 {
		stats.system, _ = strconv.ParseInt(fields[3], 10, 64)
	}
	if len(fields) > 4 {
		stats.idle, _ = strconv.ParseInt(fields[4], 10, 64)
	}
	if len(fields) > 5 {
		stats.iowait, _ = strconv.ParseInt(fields[5], 10, 64)
	}
	if len(fields) > 6 {
		stats.irq, _ = strconv.ParseInt(fields[6], 10, 64)
	}
	if len(fields) > 7 {
		stats.softirq, _ = strconv.ParseInt(fields[7], 10, 64)
	}
	if len(fields) > 8 {
		stats.steal, _ = strconv.ParseInt(fields[8], 10, 64)
	}

	return stats
}

// parseCaddyRoutes extracts routes from a Caddyfile.
func parseCaddyRoutes(caddyfile string) []CaddyRoute {
	var routes []CaddyRoute
	lines := strings.Split(caddyfile, "\n")

	// Simple regex patterns for common Caddyfile patterns
	reverseProxyRegex := regexp.MustCompile(`reverse_proxy\s+([^\s{]+)`)
	handlePathRegex := regexp.MustCompile(`handle_path\s+([^\s{]+)`)
	handleRegex := regexp.MustCompile(`handle\s+([^\s{]+)`)

	currentPath := "/"

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for handle_path or handle directives
		if match := handlePathRegex.FindStringSubmatch(line); len(match) >= 2 {
			currentPath = match[1]
		} else if match := handleRegex.FindStringSubmatch(line); len(match) >= 2 {
			currentPath = match[1]
		}

		// Check for reverse_proxy
		if match := reverseProxyRegex.FindStringSubmatch(line); len(match) >= 2 {
			upstream := match[1]
			routes = append(routes, CaddyRoute{
				Path:     currentPath,
				Upstream: upstream,
			})
		}
	}

	// If no routes found but there's content, add a default route indicator
	if len(routes) == 0 && len(caddyfile) > 0 {
		routes = append(routes, CaddyRoute{
			Path:     "/",
			Upstream: "(parsed from Caddyfile)",
		})
	}

	return routes
}
