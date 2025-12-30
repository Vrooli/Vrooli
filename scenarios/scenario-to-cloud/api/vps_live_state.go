package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// LiveStateRequest is the input for the live state endpoint.
type LiveStateRequest struct {
	DeploymentID string `json:"-"` // Set from URL path
}

// LiveStateResult contains the comprehensive live state of a VPS.
type LiveStateResult struct {
	OK             bool          `json:"ok"`
	Timestamp      string        `json:"timestamp"`
	SyncDurationMs int64         `json:"sync_duration_ms"`
	Processes      *ProcessState `json:"processes,omitempty"`
	Ports          []PortBinding `json:"ports,omitempty"`
	Caddy          *CaddyState   `json:"caddy,omitempty"`
	System         *SystemState  `json:"system,omitempty"`
	Error          string        `json:"error,omitempty"`
}

// ProcessState contains all process information.
type ProcessState struct {
	Scenarios  []ScenarioProcess   `json:"scenarios"`
	Resources  []ResourceProcess   `json:"resources"`
	Unexpected []UnexpectedProcess `json:"unexpected"`
}

// ScenarioProcess represents a running scenario.
type ScenarioProcess struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"` // running, stopped, failed
	PID           int              `json:"pid"`
	UptimeSeconds int64            `json:"uptime_seconds"`
	LastRestart   string           `json:"last_restart,omitempty"`
	Ports         []ProcessPort    `json:"ports,omitempty"`
	Resources     ProcessResources `json:"resources"`
	VrooliStatus  json.RawMessage  `json:"vrooli_status,omitempty"` // Raw output from vrooli scenario status
}

// ResourceProcess represents a running resource (postgres, redis, etc).
type ResourceProcess struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	PID           int             `json:"pid"`
	Port          int             `json:"port,omitempty"`
	UptimeSeconds int64           `json:"uptime_seconds"`
	Metrics       json.RawMessage `json:"metrics,omitempty"` // Resource-specific metrics
	VrooliStatus  json.RawMessage `json:"vrooli_status,omitempty"`
}

// UnexpectedProcess represents a process that isn't expected from the manifest.
type UnexpectedProcess struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user"`
}

// ProcessPort represents a port a process is listening on.
type ProcessPort struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Status     string `json:"status"`     // listening, responding, not_responding
	Responding bool   `json:"responding"` // true if health check passed
	Clients    int    `json:"clients,omitempty"`
}

// ProcessResources contains resource usage for a process.
type ProcessResources struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      int     `json:"memory_mb"`
	MemoryPercent float64 `json:"memory_percent"`
}

// PortBinding represents a single port binding on the VPS.
type PortBinding struct {
	Port            int    `json:"port"`
	Process         string `json:"process"`
	Type            string `json:"type"` // system, edge, scenario, resource, unexpected
	MatchesManifest *bool  `json:"matches_manifest,omitempty"`
	PID             *int   `json:"pid,omitempty"`
	Command         string `json:"command,omitempty"`
}

// CaddyState contains Caddy/TLS information.
type CaddyState struct {
	Running bool         `json:"running"`
	Domain  string       `json:"domain"`
	TLS     TLSInfo      `json:"tls"`
	Routes  []CaddyRoute `json:"routes"`
}

// TLSInfo contains TLS certificate information.
type TLSInfo struct {
	Valid         bool   `json:"valid"`
	Issuer        string `json:"issuer,omitempty"`
	Expires       string `json:"expires,omitempty"`
	DaysRemaining int    `json:"days_remaining,omitempty"`
	Error         string `json:"error,omitempty"`
}

// CaddyRoute represents a route in the Caddyfile.
type CaddyRoute struct {
	Path     string `json:"path"`
	Upstream string `json:"upstream"`
}

// SystemState contains system resource information.
type SystemState struct {
	CPU           CPUInfo    `json:"cpu"`
	Memory        MemoryInfo `json:"memory"`
	Disk          DiskInfo   `json:"disk"`
	Swap          SwapInfo   `json:"swap"`
	SSH           SSHHealth  `json:"ssh"`
	UptimeSeconds int64      `json:"uptime_seconds"`
}

// SSHHealth contains SSH connectivity status.
type SSHHealth struct {
	Connected    bool   `json:"connected"`
	LatencyMs    int64  `json:"latency_ms"`
	KeyInAuth    bool   `json:"key_in_auth"`    // Is manifest key in authorized_keys?
	KeyPath      string `json:"key_path"`       // Path to the key file used
	Error        string `json:"error,omitempty"`
}

// CPUInfo contains CPU information.
type CPUInfo struct {
	Cores        int       `json:"cores"`
	Model        string    `json:"model,omitempty"`
	UsagePercent float64   `json:"usage_percent"`
	LoadAverage  []float64 `json:"load_average"`
}

// MemoryInfo contains memory information.
type MemoryInfo struct {
	TotalMB      int     `json:"total_mb"`
	UsedMB       int     `json:"used_mb"`
	FreeMB       int     `json:"free_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

// DiskInfo contains disk information.
type DiskInfo struct {
	TotalGB      int     `json:"total_gb"`
	UsedGB       int     `json:"used_gb"`
	FreeGB       int     `json:"free_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

// SwapInfo contains swap information.
type SwapInfo struct {
	TotalMB      int     `json:"total_mb"`
	UsedMB       int     `json:"used_mb"`
	UsagePercent float64 `json:"usage_percent"`
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
		{id: "scenario_status", command: fmt.Sprintf("cd %s && vrooli scenario status %s --json 2>/dev/null", shellQuoteSingle(workdir), shellQuoteSingle(targetScenario))},
		{id: "resource_status", command: fmt.Sprintf("cd %s && vrooli resource status --json 2>/dev/null", shellQuoteSingle(workdir))},
		{id: "caddy_config", command: "cat /etc/caddy/Caddyfile 2>/dev/null || echo ''"},
		{id: "caddy_running", command: "pgrep -x caddy >/dev/null 2>&1 && echo 'running' || echo 'stopped'"},
		// SSH health: check if our public key is in authorized_keys
		{id: "ssh_key_check", command: "cat ~/.ssh/authorized_keys 2>/dev/null || echo ''"},
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
