package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps/portparse"
	"scenario-to-cloud/vps/systemmetrics"
)

// LiveStateRequest is the input for the live state endpoint.
type LiveStateRequest struct {
	DeploymentID string `json:"-"` // Set from URL path
}

// probe is one typed read issued through reach: a vrooli verb or a host
// observation program. There is no shell string anywhere in a probe.
type probe struct {
	id  string
	cmd reach.Command
}

// probeResult holds the answer to one probe.
type probeResult struct {
	id         string
	result     reach.Result
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

func observe(id, program string, args ...string) probe {
	return probe{id: id, cmd: reach.Command{Program: program, Args: args, RequiredScope: "vrooli:read", Timeout: DefaultProbeTimeout}}
}

func verb(id, verb string, args ...string) probe {
	return probe{id: id, cmd: reach.Command{Verb: verb, Args: args, RequiredScope: "vrooli:read", Timeout: DefaultProbeTimeout}}
}

// cpuSampleInterval separates the two /proc/stat samples usage is derived
// from. Tests shorten it.
var cpuSampleInterval = time.Second

// LiveStateProbes lists every read the inspection issues for a manifest;
// exported so the architecture tests can prove none of them carries shell
// syntax and every program is an observation program.
func LiveStateProbes(manifest domain.CloudManifest, user string) []reach.Command {
	probes := inspectionProbes(manifest, user, systemmetrics.CollectorForOS("linux"))
	out := make([]reach.Command, 0, len(probes)+2)
	out = append(out, observe("os_release", "cat", "/etc/os-release").cmd)
	for _, p := range probes {
		out = append(out, p.cmd)
	}
	return out
}

func inspectionProbes(manifest domain.CloudManifest, user string, collector systemmetrics.Collector) []probe {
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID
	probes := []probe{
		observe("ps", "ps", "aux", "--no-headers"),
		observe("ss", "ss", "-tlnp"),
		observe("ssh_ping", "uname", "-s"),
		verb("scenario_status", "scenario status", targetScenario, "--json"),
		verb("resource_status", "resource status", "--json"),
		observe("caddy_config", "cat", "/etc/caddy/Caddyfile"),
		observe("caddy_running", "pgrep", "-x", "caddy"),
		observe("ssh_key_check", "cat", path.Join(HomeDir(user), ".ssh", "authorized_keys")),
		observe("dir_check", "stat", append([]string{"-c", "%n", "--"}, expectedDirectories(workdir, manifest)...)...),
	}
	for _, spec := range collector.SystemCommands() {
		if spec.ID == systemmetrics.CPUUsageProbeID {
			continue
		}
		probes = append(probes, observe(spec.ID, spec.Program, spec.Args...))
	}
	return probes
}

// RunLiveStateInspection gathers the live state of one target through the
// prober: typed vrooli verbs for what the control plane reports and bounded
// observation programs for host facts. Nothing here is effectful.
// DOC: docs/reference/api-endpoints.md#get-deployment-live-state
func RunLiveStateInspection(
	ctx context.Context,
	manifest domain.CloudManifest,
	identity sshidentity.DeploymentSSHIdentity,
	prober Prober,
) domain.LiveStateResult {
	start := time.Now()
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	pubKeyContent := ""
	if identity.AuthMode == sshidentity.AuthModeExplicitKey && identity.KeyPath != "" {
		if content, _, err := sshidentity.ReadPublicKeyAndFingerprint(identity.KeyPath); err == nil {
			pubKeyContent = content
		}
	}

	// Determine the best system metrics collector based on OS identity.
	collector := systemmetrics.CollectorForOS("linux")
	if osReleaseRes, err := prober.Observe(ctx, "cat", "/etc/os-release"); err == nil {
		if osID, _ := systemmetrics.ParseOSRelease(osReleaseRes.Stdout); osID != "" {
			collector = systemmetrics.CollectorForOS(osID)
		}
	}

	probes := inspectionProbes(manifest, prober.Target.Locator.User, collector)
	results := runProbes(ctx, prober, probes)
	// CPU usage sampling runs after the concurrent probes so the probes
	// themselves do not inflate the reading on small hosts.
	results[systemmetrics.CPUUsageProbeID] = sampleCPUUsage(ctx, prober, collector)

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
	systemState := parseSystemState(results, identity, pubKeyContent, collector)
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
	liveState.Expected = buildExpectedProcesses(manifest, processState, parseExistingDirectories(results["dir_check"].result.Stdout, workdir, manifest))

	// Parse Caddy state
	caddyState := parseCaddyState(
		results["caddy_config"].result.Stdout,
		results["caddy_running"],
		manifest.Edge.Domain,
	)
	liveState.Caddy = &caddyState

	// Categorize ports with manifest knowledge
	liveState.Ports = categorizePortsWithManifest(ports, processState, manifest)

	return liveState
}

// runProbes issues every probe concurrently and collects the answers.
func runProbes(ctx context.Context, prober Prober, probes []probe) map[string]probeResult {
	results := make(map[string]probeResult, len(probes)+1)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, p := range probes {
		wg.Add(1)
		go func(p probe) {
			defer wg.Done()
			cmdStart := time.Now()
			res, err := prober.Reach.Exec(ctx, prober.Target, p.cmd)
			mu.Lock()
			results[p.id] = probeResult{id: p.id, result: res, err: err, durationMs: time.Since(cmdStart).Milliseconds()}
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	return results
}

// sampleCPUUsage reads /proc/stat twice, one interval apart, and joins the
// samples so the collector derives usage from the delta.
func sampleCPUUsage(ctx context.Context, prober Prober, collector systemmetrics.Collector) probeResult {
	var spec *systemmetrics.CommandSpec
	for _, candidate := range collector.SystemCommands() {
		if candidate.ID == systemmetrics.CPUUsageProbeID {
			c := candidate
			spec = &c
			break
		}
	}
	if spec == nil {
		return probeResult{id: systemmetrics.CPUUsageProbeID}
	}
	cmdStart := time.Now()
	var samples []string
	var lastErr error
	for i := 0; i < 2; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return probeResult{id: spec.ID, err: ctx.Err(), durationMs: time.Since(cmdStart).Milliseconds()}
			case <-time.After(cpuSampleInterval):
			}
		}
		res, err := prober.Observe(ctx, spec.Program, spec.Args...)
		if err != nil {
			lastErr = err
			continue
		}
		samples = append(samples, strings.TrimSpace(res.Stdout))
	}
	return probeResult{id: spec.ID, result: reach.Result{Stdout: strings.Join(samples, "\n")}, err: lastErr, durationMs: time.Since(cmdStart).Milliseconds()}
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
	scenarioStatusRaw := validRawJSON(scenarioStatusJSON)

	// Parse vrooli resource status output
	var resourceStatus map[string]interface{}
	if resourceStatusJSON != "" {
		_ = json.Unmarshal([]byte(resourceStatusJSON), &resourceStatus)
	}
	resourceStatusRaw := validRawJSON(resourceStatusJSON)

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
		"minio":        "minio",
	}

	// Track which PIDs we've classified
	classifiedPIDs := make(map[int]bool)

	// Find the main scenario process
	for _, p := range processes {
		if containsSubstring(p.Command, targetScenario) || containsSubstring(p.Command, "scenario-") {
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
				VrooliStatus: scenarioStatusRaw,
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
	resourceIndexByID := make(map[string]int)
	for _, p := range processes {
		if classifiedPIDs[p.PID] {
			continue
		}

		for procPattern, resourceName := range knownResources {
			if containsSubstring(p.Command, procPattern) {
				if _, expected := expectedResources[resourceName]; expected {
					resourceProc := domain.ResourceProcess{
						ID:            resourceName,
						Status:        "running",
						PID:           p.PID,
						UptimeSeconds: 0,
						VrooliStatus:  resourceStatusRaw,
					}

					// Find port for this resource
					for port, procName := range portToProcess {
						if containsSubstring(procName, procPattern) {
							resourceProc.Port = port
							break
						}
					}

					if idx, exists := resourceIndexByID[resourceName]; exists {
						existing := state.Resources[idx]
						// Prefer the earliest (typically parent) PID as the canonical row.
						if existing.PID <= 0 || (resourceProc.PID > 0 && resourceProc.PID < existing.PID) {
							resourceProc.Port = chooseResourcePort(existing.Port, resourceProc.Port)
							state.Resources[idx] = resourceProc
						}
					} else {
						resourceIndexByID[resourceName] = len(state.Resources)
						state.Resources = append(state.Resources, resourceProc)
					}
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

func chooseResourcePort(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}

func validRawJSON(s string) json.RawMessage {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	if !json.Valid([]byte(trimmed)) {
		return nil
	}
	return json.RawMessage(trimmed)
}

// expectedDirectories lists the scenario and resource directories the
// manifest expects under the workdir, in a stable order. The stat probe
// reports which of them exist; a missing one is simply absent from stdout.
func expectedDirectories(workdir string, manifest domain.CloudManifest) []string {
	dirs := []string{fmt.Sprintf("%s/scenarios/%s", workdir, manifest.Scenario.ID)}
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue
		}
		dirs = append(dirs, fmt.Sprintf("%s/scenarios/%s", workdir, scenarioID))
	}
	for _, resourceID := range manifest.Dependencies.Resources {
		dirs = append(dirs, fmt.Sprintf("%s/resources/%s", workdir, resourceID))
	}
	return dirs
}

// parseExistingDirectories maps the stat probe output (one existing path
// per line) back to "type:id" keys.
func parseExistingDirectories(output, workdir string, manifest domain.CloudManifest) map[string]bool {
	existing := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		existing[line] = true
	}
	keys := make(map[string]bool)
	mark := func(kind, id, dir string) {
		if existing[dir] {
			keys[kind+":"+id] = true
		}
	}
	mark("scenario", manifest.Scenario.ID, fmt.Sprintf("%s/scenarios/%s", workdir, manifest.Scenario.ID))
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		mark("scenario", scenarioID, fmt.Sprintf("%s/scenarios/%s", workdir, scenarioID))
	}
	for _, resourceID := range manifest.Dependencies.Resources {
		mark("resource", resourceID, fmt.Sprintf("%s/resources/%s", workdir, resourceID))
	}
	return keys
}

// buildExpectedProcesses constructs the expected processes list by comparing manifest with running state.
func buildExpectedProcesses(manifest domain.CloudManifest, processState domain.ProcessState, dirExists map[string]bool) []domain.ExpectedProcess {
	expected := []domain.ExpectedProcess{}

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
	results map[string]probeResult,
	identity sshidentity.DeploymentSSHIdentity,
	pubKeyContent string,
	collector systemmetrics.Collector,
) domain.SystemState {
	if identity.AuthMode == "" {
		identity.AuthMode = sshidentity.AuthModeUnknown
	}
	if identity.VerificationState == "" {
		identity.VerificationState = sshidentity.VerificationUnknown
	}

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
		Connected:            connected,
		LatencyMs:            latencyMs,
		KeyPath:              identity.KeyPath,
		AuthMode:             string(identity.AuthMode),
		VerificationState:    string(identity.VerificationState),
		PublicKeyFingerprint: identity.PublicKeyFingerprint,
		LastVerifiedAt:       identity.LastVerifiedAt,
	}

	// For explicit key auth, verify whether the configured key exists in authorized_keys.
	if identity.AuthMode == sshidentity.AuthModeExplicitKey {
		state.SSH.VerificationState = string(sshidentity.VerificationUnknown)
	}
	if identity.AuthMode == sshidentity.AuthModeExplicitKey && pubKeyContent != "" {
		if sshKeyResult, ok := results["ssh_key_check"]; ok && sshKeyResult.err == nil {
			authorizedKeys := sshKeyResult.result.Stdout
			pubKeyParts := strings.Fields(pubKeyContent)
			if len(pubKeyParts) >= 2 {
				keyToMatch := pubKeyParts[0] + " " + pubKeyParts[1]
				if strings.Contains(authorizedKeys, keyToMatch) {
					state.SSH.VerificationState = string(sshidentity.VerificationAuthorized)
				} else {
					state.SSH.VerificationState = string(sshidentity.VerificationUnauthorized)
				}
			}
		}
	}

	return state
}

// parseCaddyState parses Caddy configuration and status.
func parseCaddyState(caddyfileContent string, running probeResult, expectedDomain string) domain.CaddyState {
	state := domain.CaddyState{
		Running: running.err == nil && running.result.ExitCode == 0 && strings.TrimSpace(running.result.Stdout) != "",
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
