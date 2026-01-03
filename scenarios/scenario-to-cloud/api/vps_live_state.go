package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"scenario-to-cloud/domain"
)

// Type aliases for backward compatibility and shorter references within main package.
type (
	LiveStateResult   = domain.LiveStateResult
	ExpectedProcess   = domain.ExpectedProcess
	ProcessState      = domain.ProcessState
	ScenarioProcess   = domain.ScenarioProcess
	ResourceProcess   = domain.ResourceProcess
	UnexpectedProcess = domain.UnexpectedProcess
	ProcessPort       = domain.ProcessPort
	ProcessResources  = domain.ProcessResources
	PortBinding       = domain.PortBinding
	CaddyState        = domain.CaddyState
	TLSInfo           = domain.TLSInfo
	CaddyRoute        = domain.CaddyRoute
	SystemState       = domain.SystemState
	SSHHealth         = domain.SSHHealth
	CPUInfo           = domain.CPUInfo
	MemoryInfo        = domain.MemoryInfo
	DiskInfo          = domain.DiskInfo
	SwapInfo          = domain.SwapInfo
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
	id     string
	result SSHResult
	err    error
}

// RunLiveStateInspection executes SSH commands to gather live state from a VPS.
func RunLiveStateInspection(ctx context.Context, manifest CloudManifest, sshRunner SSHRunner) LiveStateResult {
	start := time.Now()
	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	// Get the public key fingerprint for SSH key verification
	keyPath := cfg.KeyPath
	pubKeyFingerprint := getPublicKeyFingerprint(keyPath)

	// Build directory check command for expected processes
	dirCheckCmd := buildDirCheckCommand(workdir, manifest)

	// Define all commands to execute
	commands := []sshCommand{
		{id: "ps", command: "ps aux --no-headers"},
		{id: "ss", command: "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null"},
		{id: "df", command: "df -h / 2>/dev/null | tail -1"},
		{id: "free", command: "free -m 2>/dev/null | grep -E '^Mem:|^Swap:'"},
		{id: "loadavg", command: "cat /proc/loadavg 2>/dev/null"},
		{id: "uptime", command: "cat /proc/uptime 2>/dev/null"},
		{id: "cpuinfo", command: "grep -c processor /proc/cpuinfo 2>/dev/null"},
		{id: "cpumodel", command: "grep 'model name' /proc/cpuinfo 2>/dev/null | head -1"},
		// Get CPU usage by sampling /proc/stat twice with 1 second delay
		// This gives accurate current CPU usage, not since-boot average
		{id: "cpuusage", command: "cat /proc/stat | head -1; sleep 1; cat /proc/stat | head -1"},
		{id: "scenario_status", command: vrooliCommand(workdir, fmt.Sprintf("vrooli scenario status %s --json 2>/dev/null", shellQuoteSingle(targetScenario)))},
		{id: "resource_status", command: vrooliCommand(workdir, "vrooli resource status --json 2>/dev/null")},
		{id: "caddy_config", command: "cat /etc/caddy/Caddyfile 2>/dev/null || echo ''"},
		{id: "caddy_running", command: "pgrep -x caddy >/dev/null 2>&1 && echo 'running' || echo 'stopped'"},
		// SSH health: check if our public key is in authorized_keys
		{id: "ssh_key_check", command: "cat ~/.ssh/authorized_keys 2>/dev/null || echo ''"},
		// Check directory existence for expected processes
		{id: "dir_check", command: dirCheckCmd},
	}

	// Execute commands in parallel
	results := make(map[string]sshCommandResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, cmd := range commands {
		wg.Add(1)
		go func(c sshCommand) {
			defer wg.Done()
			res, err := sshRunner.Run(ctx, cfg, c.command)
			mu.Lock()
			results[c.id] = sshCommandResult{id: c.id, result: res, err: err}
			mu.Unlock()
		}(cmd)
	}

	wg.Wait()

	// Check for context cancellation
	if ctx.Err() != nil {
		return LiveStateResult{
			OK:        false,
			Error:     "context cancelled: " + ctx.Err().Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Parse results
	liveState := LiveStateResult{
		OK:             true,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		SyncDurationMs: time.Since(start).Milliseconds(),
	}

	// Parse system info (including SSH health)
	systemState := parseSystemState(results, keyPath, pubKeyFingerprint, time.Since(start).Milliseconds())
	liveState.System = &systemState

	// Parse port bindings
	ports := parseSSOutput(results["ss"].result.Stdout)
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

// buildProcessState constructs ProcessState from raw data.
func buildProcessState(
	processes []ProcessInfo,
	ports []PortBinding,
	scenarioStatusJSON string,
	resourceStatusJSON string,
	targetScenario string,
	expectedResources map[string]bool,
) ProcessState {
	state := ProcessState{
		Scenarios:  []ScenarioProcess{},
		Resources:  []ResourceProcess{},
		Unexpected: []UnexpectedProcess{},
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
		if containsString(p.Command, targetScenario) || containsString(p.Command, "scenario-") {
			var vrooliStatusRaw json.RawMessage
			if scenarioStatusJSON != "" {
				vrooliStatusRaw = json.RawMessage(scenarioStatusJSON)
			}

			scenarioProc := ScenarioProcess{
				ID:            targetScenario,
				Status:        "running",
				PID:           p.PID,
				UptimeSeconds: 0, // Would need more info to calculate
				Resources: ProcessResources{
					CPUPercent:    p.CPUPercent,
					MemoryMB:      int(p.MemoryMB),
					MemoryPercent: p.MemoryPercent,
				},
				VrooliStatus: vrooliStatusRaw,
			}

			// Find ports for this scenario
			for port, procName := range portToProcess {
				if containsString(procName, "scenario") || containsString(procName, targetScenario) {
					scenarioProc.Ports = append(scenarioProc.Ports, ProcessPort{
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
			if containsString(p.Command, procPattern) {
				if _, expected := expectedResources[resourceName]; expected || true {
					var vrooliStatusRaw json.RawMessage
					if resourceStatusJSON != "" {
						vrooliStatusRaw = json.RawMessage(resourceStatusJSON)
					}

					resourceProc := ResourceProcess{
						ID:            resourceName,
						Status:        "running",
						PID:           p.PID,
						UptimeSeconds: 0,
						VrooliStatus:  vrooliStatusRaw,
					}

					// Find port for this resource
					for port, procName := range portToProcess {
						if containsString(procName, procPattern) {
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
			if containsString(p.Command, sysProc) {
				isSystem = true
				break
			}
		}
		if isSystem {
			continue
		}

		// Skip caddy (it's the edge proxy)
		if containsString(p.Command, "caddy") {
			continue
		}

		// Check if this process has a bound port
		var boundPort int
		for port, procName := range portToProcess {
			if p.PID > 0 && containsString(procName, fmt.Sprintf("pid=%d", p.PID)) {
				boundPort = port
				break
			}
		}

		// Only report if it has a bound port (more relevant for debugging)
		if boundPort > 0 {
			state.Unexpected = append(state.Unexpected, UnexpectedProcess{
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
func categorizePortsWithManifest(ports []PortBinding, processState ProcessState, manifest CloudManifest) []PortBinding {
	result := make([]PortBinding, 0, len(ports))

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

// containsString is a helper to check if a string contains a substring.
func containsString(s, substr string) bool {
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
func buildDirCheckCommand(workdir string, manifest CloudManifest) string {
	var checks []string

	// Check target scenario directory
	scenarioDir := fmt.Sprintf("%s/scenarios/%s", workdir, manifest.Scenario.ID)
	checks = append(checks, fmt.Sprintf("test -d %s && echo 'scenario:%s:exists' || echo 'scenario:%s:missing'",
		shellQuoteSingle(scenarioDir), manifest.Scenario.ID, manifest.Scenario.ID))

	// Check dependent scenarios
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue // Already checked
		}
		scenarioDir := fmt.Sprintf("%s/scenarios/%s", workdir, scenarioID)
		checks = append(checks, fmt.Sprintf("test -d %s && echo 'scenario:%s:exists' || echo 'scenario:%s:missing'",
			shellQuoteSingle(scenarioDir), scenarioID, scenarioID))
	}

	// Check resource directories
	for _, resourceID := range manifest.Dependencies.Resources {
		resourceDir := fmt.Sprintf("%s/resources/%s", workdir, resourceID)
		checks = append(checks, fmt.Sprintf("test -d %s && echo 'resource:%s:exists' || echo 'resource:%s:missing'",
			shellQuoteSingle(resourceDir), resourceID, resourceID))
	}

	if len(checks) == 0 {
		return "echo 'no_expected_processes'"
	}

	return strings.Join(checks, "; ")
}

// buildExpectedProcesses constructs the expected processes list by comparing manifest with running state.
func buildExpectedProcesses(manifest CloudManifest, processState ProcessState, dirCheckOutput string) []ExpectedProcess {
	expected := []ExpectedProcess{}

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

	expected = append(expected, ExpectedProcess{
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

		expected = append(expected, ExpectedProcess{
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

		expected = append(expected, ExpectedProcess{
			ID:              resourceID,
			Type:            "resource",
			State:           state,
			DirectoryExists: resourceDirExists,
		})
	}

	return expected
}
