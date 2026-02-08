package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/vps/portparse"
	"scenario-to-cloud/vps/systemmetrics"
)

// LiveStateRequest is the input for the live state endpoint.
type LiveStateRequest struct {
	DeploymentID string `json:"-"` // Set from URL path
}

// sshCommand represents a command to execute via SSH with its result destination.
type sshCommand struct {
	id      string
	command string
}

// sshCommandResult holds the result of an SSH command.
type sshCommandResult struct {
	id         string
	result     ssh.Result
	err        error
	durationMs int64
}

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

// RunLiveStateInspection executes SSH commands to gather live state from a VPS.
func RunLiveStateInspection(ctx context.Context, manifest domain.CloudManifest, sshRunner ssh.Runner) domain.LiveStateResult {
	start := time.Now()
	cfg := ssh.ConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	// Get the public key fingerprint for SSH key verification
	keyPath := cfg.KeyPath
	pubKeyFingerprint := getPublicKeyFingerprint(keyPath)

	// Build directory check command for expected processes
	dirCheckCmd := buildDirCheckCommand(workdir, manifest)

	// Determine the best system metrics collector based on OS identity.
	collector := systemmetrics.CollectorForOS("linux")
	if osReleaseRes, err := sshRunner.Run(ctx, cfg, "cat /etc/os-release 2>/dev/null || true", ssh.DefaultRunOptions()); err == nil {
		if osID, _ := systemmetrics.ParseOSRelease(osReleaseRes.Stdout); osID != "" {
			collector = systemmetrics.CollectorForOS(osID)
		}
	}

	// Define all commands to execute
	commands := []sshCommand{
		{id: "ps", command: "ps aux --no-headers"},
		{id: "ss", command: "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null"},
		{id: "ssh_ping", command: "echo ok"},
		{id: "scenario_status", command: shellutil.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario status %s --json 2>/dev/null", shellutil.QuoteSingle(targetScenario)))},
		{id: "resource_status", command: shellutil.VrooliCommand(workdir, "vrooli resource status --json 2>/dev/null")},
		{id: "caddy_config", command: "cat /etc/caddy/Caddyfile 2>/dev/null || echo ''"},
		{id: "caddy_running", command: "pgrep -x caddy >/dev/null 2>&1 && echo 'running' || echo 'stopped'"},
		// SSH health: check if our public key is in authorized_keys
		{id: "ssh_key_check", command: "cat ~/.ssh/authorized_keys 2>/dev/null || echo ''"},
		// Check directory existence for expected processes
		{id: "dir_check", command: dirCheckCmd},
	}
	for _, spec := range collector.SystemCommands() {
		commands = append(commands, sshCommand{id: spec.ID, command: spec.Command})
	}

	// Execute commands in parallel
	results := make(map[string]sshCommandResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, cmd := range commands {
		wg.Add(1)
		go func(c sshCommand) {
			defer wg.Done()
			cmdStart := time.Now()
			res, err := sshRunner.Run(ctx, cfg, c.command, ssh.DefaultRunOptions())
			mu.Lock()
			results[c.id] = sshCommandResult{
				id:         c.id,
				result:     res,
				err:        err,
				durationMs: time.Since(cmdStart).Milliseconds(),
			}
			mu.Unlock()
		}(cmd)
	}

	wg.Wait()

	// Check for context cancellation
	if ctx.Err() != nil {
		return domain.LiveStateResult{
			OK:        false,
			Error:     "context cancelled: " + ctx.Err().Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Parse results
	liveState := domain.LiveStateResult{
		OK:             true,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		SyncDurationMs: time.Since(start).Milliseconds(),
	}

	// Parse system info (including SSH health)
	systemState := parseSystemState(results, keyPath, pubKeyFingerprint, collector)
	liveState.System = &systemState

	// Parse port bindings
	ports := ParseSSOutput(results["ss"].result.Stdout)
	liveState.Ports = ports

	// Parse process information
	processes := parsePSOutput(results["ps"].result.Stdout)

	// Get expected resources from manifest
	expectedResources := make(map[string]bool)
	for _, res := range manifest.Dependencies.Resources {
		expectedResources[res] = true
	}

	// Build process state with scenario and resource status
	processState := buildProcessState(
		processes,
		ports,
		results["scenario_status"].result.Stdout,
		results["resource_status"].result.Stdout,
		targetScenario,
		expectedResources,
	)
	liveState.Processes = &processState

	// Build expected processes list from manifest
	dirCheckOutput := results["dir_check"].result.Stdout
	liveState.Expected = buildExpectedProcesses(manifest, processState, dirCheckOutput)

	// Parse Caddy state
	caddyState := parseCaddyState(
		results["caddy_config"].result.Stdout,
		results["caddy_running"].result.Stdout,
		manifest.Edge.Domain,
	)
	liveState.Caddy = &caddyState

	// Categorize ports with manifest knowledge
	liveState.Ports = categorizePortsWithManifest(ports, processState, manifest)

	return liveState
}

// buildProcessState constructs domain.ProcessState from raw data.
func buildProcessState(
	processes []ProcessInfo,
	ports []domain.PortBinding,
	scenarioStatusJSON string,
	resourceStatusJSON string,
	targetScenario string,
	expectedResources map[string]bool,
) domain.ProcessState {
	state := domain.ProcessState{
		Scenarios:  []domain.ScenarioProcess{},
		Resources:  []domain.ResourceProcess{},
		Unexpected: []domain.UnexpectedProcess{},
	}

	// Parse vrooli scenario status output
	var scenarioStatus map[string]interface{}
	if scenarioStatusJSON != "" {
		_ = json.Unmarshal([]byte(scenarioStatusJSON), &scenarioStatus)
	}

	// Parse vrooli resource status output
	var resourceStatus map[string]interface{}
	if resourceStatusJSON != "" {
		_ = json.Unmarshal([]byte(resourceStatusJSON), &resourceStatus)
	}

	// Build a map of PID to process for quick lookup
	pidToProcess := make(map[int]ProcessInfo)
	for _, p := range processes {
		pidToProcess[p.PID] = p
	}

	// Build a map of port to process name
	portToProcess := make(map[int]string)
	for _, port := range ports {
		portToProcess[port.Port] = port.Process
	}

	// Known system processes to exclude
	systemProcesses := map[string]bool{
		"sshd": true, "systemd": true, "init": true, "agetty": true,
		"cron": true, "rsyslogd": true, "dbus-daemon": true,
		"NetworkManager": true, "polkitd": true, "udevd": true,
	}

	// Known resource processes
	knownResources := map[string]string{
		"postgres":     "postgres",
		"redis-server": "redis",
		"redis":        "redis",
		"qdrant":       "qdrant",
		"ollama":       "ollama",
		"browserless":  "browserless",
		"minio":        "minio",
	}

	// Track which PIDs we've classified
	classifiedPIDs := make(map[int]bool)

	// Find the main scenario process
	for _, p := range processes {
		if containsSubstring(p.Command, targetScenario) || containsSubstring(p.Command, "scenario-") {
			var vrooliStatusRaw json.RawMessage
			if scenarioStatusJSON != "" {
				vrooliStatusRaw = json.RawMessage(scenarioStatusJSON)
			}

			scenarioProc := domain.ScenarioProcess{
				ID:            targetScenario,
				Status:        "running",
				PID:           p.PID,
				UptimeSeconds: 0, // Would need more info to calculate
				Resources: domain.ProcessResources{
					CPUPercent:    p.CPUPercent,
					MemoryMB:      int(p.MemoryMB),
					MemoryPercent: p.MemoryPercent,
				},
				VrooliStatus: vrooliStatusRaw,
			}

			// Find ports for this scenario
			for port, procName := range portToProcess {
				if containsSubstring(procName, "scenario") || containsSubstring(procName, targetScenario) {
					scenarioProc.Ports = append(scenarioProc.Ports, domain.ProcessPort{
						Port:   port,
						Status: "listening",
					})
				}
			}

			state.Scenarios = append(state.Scenarios, scenarioProc)
			classifiedPIDs[p.PID] = true
			break
		}
	}

	// Find resource processes
	for _, p := range processes {
		if classifiedPIDs[p.PID] {
			continue
		}

		for procPattern, resourceName := range knownResources {
			if containsSubstring(p.Command, procPattern) {
				if _, expected := expectedResources[resourceName]; expected {
					var vrooliStatusRaw json.RawMessage
					if resourceStatusJSON != "" {
						vrooliStatusRaw = json.RawMessage(resourceStatusJSON)
					}

					resourceProc := domain.ResourceProcess{
						ID:            resourceName,
						Status:        "running",
						PID:           p.PID,
						UptimeSeconds: 0,
						VrooliStatus:  vrooliStatusRaw,
					}

					// Find port for this resource
					for port, procName := range portToProcess {
						if containsSubstring(procName, procPattern) {
							resourceProc.Port = port
							break
						}
					}

					state.Resources = append(state.Resources, resourceProc)
					classifiedPIDs[p.PID] = true
					break
				}
			}
		}
	}

	// Find unexpected processes (non-system, non-classified)
	for _, p := range processes {
		if classifiedPIDs[p.PID] {
			continue
		}

		// Skip system processes
		isSystem := false
		for sysProc := range systemProcesses {
			if containsSubstring(p.Command, sysProc) {
				isSystem = true
				break
			}
		}
		if isSystem {
			continue
		}

		// Skip caddy (it's the edge proxy)
		if containsSubstring(p.Command, "caddy") {
			continue
		}

		// Check if this process has a bound port
		var boundPort int
		for port, procName := range portToProcess {
			if p.PID > 0 && containsSubstring(procName, fmt.Sprintf("pid=%d", p.PID)) {
				boundPort = port
				break
			}
		}

		// Only report if it has a bound port (more relevant for debugging)
		if boundPort > 0 {
			state.Unexpected = append(state.Unexpected, domain.UnexpectedProcess{
				PID:     p.PID,
				Command: p.Command,
				Port:    boundPort,
				User:    p.User,
			})
		}
	}

	return state
}

// categorizePortsWithManifest adds manifest-aware categorization to ports.
func categorizePortsWithManifest(ports []domain.PortBinding, processState domain.ProcessState, manifest domain.CloudManifest) []domain.PortBinding {
	result := make([]domain.PortBinding, 0, len(ports))

	// Known system ports
	systemPorts := map[int]bool{22: true}

	// Edge proxy ports
	edgePorts := map[int]bool{80: true, 443: true}

	// Build expected scenario ports from manifest
	// This would need manifest port config - for now infer from process state
	scenarioPorts := make(map[int]bool)
	for _, s := range processState.Scenarios {
		for _, p := range s.Ports {
			scenarioPorts[p.Port] = true
		}
	}

	// Resource ports
	resourcePorts := make(map[int]bool)
	for _, r := range processState.Resources {
		if r.Port > 0 {
			resourcePorts[r.Port] = true
		}
	}

	for _, port := range ports {
		categorized := port

		if systemPorts[port.Port] {
			categorized.Type = "system"
		} else if edgePorts[port.Port] {
			categorized.Type = "edge"
		} else if scenarioPorts[port.Port] {
			categorized.Type = "scenario"
			matchesManifest := true
			categorized.MatchesManifest = &matchesManifest
		} else if resourcePorts[port.Port] {
			categorized.Type = "resource"
		} else {
			categorized.Type = "unexpected"
		}

		result = append(result, categorized)
	}

	return result
}

// containsSubstring is a helper to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getPublicKeyFingerprint reads the public key content from the key file.
// Returns the public key content (not fingerprint) for matching in authorized_keys.
func getPublicKeyFingerprint(keyPath string) string {
	// Expand ~ in path
	if len(keyPath) > 0 && keyPath[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			keyPath = home + keyPath[1:]
		}
	}

	// Read public key file (.pub extension)
	pubKeyPath := keyPath + ".pub"
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return ""
	}

	// Return the key content for matching
	return strings.TrimSpace(string(content))
}

// buildDirCheckCommand creates a shell command to check if directories exist for expected processes.
// Output format: "type:id:exists" or "type:id:missing" per line
func buildDirCheckCommand(workdir string, manifest domain.CloudManifest) string {
	var checks []string

	// Check target scenario directory
	scenarioDir := fmt.Sprintf("%s/scenarios/%s", workdir, manifest.Scenario.ID)
	checks = append(checks, fmt.Sprintf("test -d %s && echo 'scenario:%s:exists' || echo 'scenario:%s:missing'",
		shellutil.QuoteSingle(scenarioDir), manifest.Scenario.ID, manifest.Scenario.ID))

	// Check dependent scenarios
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue // Already checked
		}
		scenarioDir := fmt.Sprintf("%s/scenarios/%s", workdir, scenarioID)
		checks = append(checks, fmt.Sprintf("test -d %s && echo 'scenario:%s:exists' || echo 'scenario:%s:missing'",
			shellutil.QuoteSingle(scenarioDir), scenarioID, scenarioID))
	}

	// Check resource directories
	for _, resourceID := range manifest.Dependencies.Resources {
		resourceDir := fmt.Sprintf("%s/resources/%s", workdir, resourceID)
		checks = append(checks, fmt.Sprintf("test -d %s && echo 'resource:%s:exists' || echo 'resource:%s:missing'",
			shellutil.QuoteSingle(resourceDir), resourceID, resourceID))
	}

	if len(checks) == 0 {
		return "echo 'no_expected_processes'"
	}

	return strings.Join(checks, "; ")
}

// buildExpectedProcesses constructs the expected processes list by comparing manifest with running state.
func buildExpectedProcesses(manifest domain.CloudManifest, processState domain.ProcessState, dirCheckOutput string) []domain.ExpectedProcess {
	expected := []domain.ExpectedProcess{}

	// Parse directory check output
	dirExists := make(map[string]bool)
	for _, line := range strings.Split(dirCheckOutput, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ":")
		if len(parts) == 3 {
			key := parts[0] + ":" + parts[1] // "scenario:id" or "resource:id"
			dirExists[key] = parts[2] == "exists"
		}
	}

	// Build map of running processes for quick lookup
	runningScenarios := make(map[string]bool)
	for _, s := range processState.Scenarios {
		if s.Status == "running" {
			runningScenarios[s.ID] = true
		}
	}

	runningResources := make(map[string]bool)
	for _, r := range processState.Resources {
		if r.Status == "running" {
			runningResources[r.ID] = true
		}
	}

	// Check target scenario
	targetKey := "scenario:" + manifest.Scenario.ID
	targetDirExists := dirExists[targetKey]
	targetRunning := runningScenarios[manifest.Scenario.ID]

	var targetState string
	if targetRunning {
		targetState = "running"
	} else if targetDirExists {
		targetState = "stopped"
	} else {
		targetState = "needs_setup"
	}

	expected = append(expected, domain.ExpectedProcess{
		ID:              manifest.Scenario.ID,
		Type:            "scenario",
		State:           targetState,
		DirectoryExists: targetDirExists,
	})

	// Check dependent scenarios
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue // Already added
		}

		key := "scenario:" + scenarioID
		scenarioDirExists := dirExists[key]
		scenarioRunning := runningScenarios[scenarioID]

		var state string
		if scenarioRunning {
			state = "running"
		} else if scenarioDirExists {
			state = "stopped"
		} else {
			state = "needs_setup"
		}

		expected = append(expected, domain.ExpectedProcess{
			ID:              scenarioID,
			Type:            "scenario",
			State:           state,
			DirectoryExists: scenarioDirExists,
		})
	}

	// Check resources
	for _, resourceID := range manifest.Dependencies.Resources {
		key := "resource:" + resourceID
		resourceDirExists := dirExists[key]
		resourceRunning := runningResources[resourceID]

		var state string
		if resourceRunning {
			state = "running"
		} else if resourceDirExists {
			state = "stopped"
		} else {
			state = "needs_setup"
		}

		expected = append(expected, domain.ExpectedProcess{
			ID:              resourceID,
			Type:            "resource",
			State:           state,
			DirectoryExists: resourceDirExists,
		})
	}

	return expected
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

// ParseSSOutput parses the output of `ss -tlnp`.
// Format: State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
// Example: LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1234,fd=3))
func ParseSSOutput(output string) []domain.PortBinding {
	return portparse.ParseSSOutput(output)
}

// parseSystemState parses command output into domain.SystemState.
func parseSystemState(
	results map[string]sshCommandResult,
	keyPath, pubKeyContent string,
	collector systemmetrics.Collector,
) domain.SystemState {
	metricResults := make(map[string]systemmetrics.CommandResult, len(results))
	for id, res := range results {
		metricResults[id] = systemmetrics.CommandResult{
			Stdout:   res.result.Stdout,
			Stderr:   res.result.Stderr,
			ExitCode: res.result.ExitCode,
		}
	}

	state := collector.ParseSystemState(metricResults)

	connected := false
	latencyMs := int64(0)
	if pingResult, ok := results["ssh_ping"]; ok {
		connected = pingResult.err == nil && pingResult.result.ExitCode == 0
		latencyMs = pingResult.durationMs
	}
	state.SSH = domain.SSHHealth{
		Connected:      connected,
		LatencyMs:      latencyMs,
		KeyInAuthState: "unknown",
		KeyPath:        keyPath,
	}

	// Check if our public key is in authorized_keys.
	if sshKeyResult, ok := results["ssh_key_check"]; ok && sshKeyResult.err == nil {
		authorizedKeys := sshKeyResult.result.Stdout
		if pubKeyContent != "" {
			pubKeyParts := strings.Fields(pubKeyContent)
			if len(pubKeyParts) >= 2 {
				keyToMatch := pubKeyParts[0] + " " + pubKeyParts[1]
				state.SSH.KeyInAuth = strings.Contains(authorizedKeys, keyToMatch)
				if state.SSH.KeyInAuth {
					state.SSH.KeyInAuthState = "authorized"
				} else {
					state.SSH.KeyInAuthState = "unauthorized"
				}
			}
		}
	}

	return state
}

// parseCaddyState parses Caddy configuration and status.
func parseCaddyState(caddyfileContent, runningStatus, expectedDomain string) domain.CaddyState {
	state := domain.CaddyState{
		Running: strings.TrimSpace(runningStatus) == "running",
		Domain:  expectedDomain,
		Routes:  []domain.CaddyRoute{},
	}

	// Parse routes from Caddyfile
	// Simple parsing - looks for reverse_proxy directives
	if caddyfileContent != "" {
		state.Routes = parseCaddyRoutes(caddyfileContent)
	}

	return state
}

// parseCPUUsageFromTop is a compatibility wrapper used by existing tests.
// New code should use systemmetrics.ParseCPUUsageFromProcStat directly.
func parseCPUUsageFromTop(output string) float64 {
	return systemmetrics.ParseCPUUsageFromProcStat(output)
}

// parseHumanSize is a compatibility wrapper used by existing tests.
// New code should use systemmetrics.ParseHumanSizeToGB directly.
func parseHumanSize(s string) int {
	return systemmetrics.ParseHumanSizeToGB(s)
}

// parseCaddyRoutes extracts routes from a Caddyfile.
func parseCaddyRoutes(caddyfile string) []domain.CaddyRoute {
	var routes []domain.CaddyRoute
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
			routes = append(routes, domain.CaddyRoute{
				Path:     currentPath,
				Upstream: upstream,
			})
		}
	}

	// If no routes found but there's content, add a default route indicator
	if len(routes) == 0 && len(caddyfile) > 0 {
		routes = append(routes, domain.CaddyRoute{
			Path:     "/",
			Upstream: "(parsed from Caddyfile)",
		})
	}

	return routes
}
