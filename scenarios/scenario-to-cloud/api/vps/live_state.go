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
	cmd, err := reach.NewObservation(program, args...)
	if err != nil {
		// Probe definitions are static and must be reviewed with the owner
		// mapping. Keep an invalid definition visible to the caller rather than
		// silently falling back to a host program.
		return probe{id: id, cmd: reach.Command{Verb: "cloud-target host observe", Args: []string{"--kind", "invalid"}, RequiredScope: "vrooli:read", Timeout: DefaultProbeTimeout}}
	}
	cmd.Timeout = DefaultProbeTimeout
	return probe{id: id, cmd: cmd}
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
	scenarioStatusRaw := validRawJSON(scenarioStatusJSON)
	resourceStatusRaw := validRawJSON(resourceStatusJSON)
	classifiedPIDs := make(map[int]bool)
	portToProcess := portProcessNames(ports)
	resourceIndexByID := make(map[string]int)
	appendScenarioProcess(&state, processes, portToProcess, targetScenario, scenarioStatusRaw, classifiedPIDs)
	appendResourceProcesses(&state, processes, portToProcess, expectedResources, resourceStatusRaw, classifiedPIDs, resourceIndexByID)
	appendTypedResourceStatuses(&state, typedResourceStatuses(resourceStatusJSON), resourceStatusRaw, expectedResources, resourceIndexByID)
	appendUnexpectedProcesses(&state, processes, portToProcess, classifiedPIDs)
	return state
}

func portProcessNames(ports []domain.PortBinding) map[int]string {
	result := make(map[int]string, len(ports))
	for _, port := range ports {
		result[port.Port] = port.Process
	}
	return result
}

func appendScenarioProcess(state *domain.ProcessState, processes []ProcessInfo, portToProcess map[int]string, targetScenario string, status json.RawMessage, classified map[int]bool) {
	for _, p := range processes {
		if !containsSubstring(p.Command, targetScenario) && !containsSubstring(p.Command, "scenario-") {
			continue
		}
		scenarioProc := domain.ScenarioProcess{
			ID: targetScenario, Status: "running", PID: p.PID, UptimeSeconds: 0,
			Resources:    domain.ProcessResources{CPUPercent: p.CPUPercent, MemoryMB: int(p.MemoryMB), MemoryPercent: p.MemoryPercent},
			VrooliStatus: status,
		}
		for port, procName := range portToProcess {
			if containsSubstring(procName, "scenario") || containsSubstring(procName, targetScenario) {
				scenarioProc.Ports = append(scenarioProc.Ports, domain.ProcessPort{Port: port, Status: "listening"})
			}
		}
		state.Scenarios = append(state.Scenarios, scenarioProc)
		classified[p.PID] = true
		break
	}
}

func appendResourceProcesses(state *domain.ProcessState, processes []ProcessInfo, portToProcess map[int]string, expected map[string]bool, status json.RawMessage, classified map[int]bool, indexes map[string]int) {
	for _, p := range processes {
		if classified[p.PID] {
			continue
		}
		resourceProc, ok := resourceProcessFor(p, portToProcess, expected, status)
		if !ok {
			continue
		}
		if idx, exists := indexes[resourceProc.ID]; exists {
			existing := state.Resources[idx]
			if existing.PID <= 0 || (resourceProc.PID > 0 && resourceProc.PID < existing.PID) {
				resourceProc.Port = chooseResourcePort(existing.Port, resourceProc.Port)
				state.Resources[idx] = resourceProc
			}
		} else {
			indexes[resourceProc.ID] = len(state.Resources)
			state.Resources = append(state.Resources, resourceProc)
		}
		classified[p.PID] = true
	}
}

func resourceProcessFor(p ProcessInfo, portToProcess map[int]string, expected map[string]bool, status json.RawMessage) (domain.ResourceProcess, bool) {
	for pattern, resourceName := range knownResourceProcesses {
		if !containsSubstring(p.Command, pattern) || !expected[resourceName] {
			continue
		}
		resourceProc := domain.ResourceProcess{ID: resourceName, Status: "running", PID: p.PID, UptimeSeconds: 0, VrooliStatus: status}
		for port, procName := range portToProcess {
			if containsSubstring(procName, pattern) {
				resourceProc.Port = port
				break
			}
		}
		return resourceProc, true
	}
	return domain.ResourceProcess{}, false
}

var knownResourceProcesses = map[string]string{
	"postgres": "postgres", "redis-server": "redis", "redis": "redis",
	"qdrant": "qdrant", "ollama": "ollama", "minio": "minio",
}

func appendTypedResourceStatuses(state *domain.ProcessState, statuses []typedResourceStatus, raw json.RawMessage, expected map[string]bool, indexes map[string]int) {
	// Control-plane resource state covers supervisors and containers whose process
	// names do not identify the resource. PID zero keeps this weaker evidence clear.
	for _, status := range statuses {
		if !expected[status.ID] {
			continue
		}
		if _, exists := indexes[status.ID]; exists {
			continue
		}
		indexes[status.ID] = len(state.Resources)
		state.Resources = append(state.Resources, domain.ResourceProcess{ID: status.ID, Status: status.State, VrooliStatus: raw})
	}
}

var systemProcessNames = map[string]bool{
	"sshd": true, "systemd": true, "init": true, "agetty": true,
	"cron": true, "rsyslogd": true, "dbus-daemon": true,
	"NetworkManager": true, "polkitd": true, "udevd": true,
}

func appendUnexpectedProcesses(state *domain.ProcessState, processes []ProcessInfo, portToProcess map[int]string, classified map[int]bool) {
	for _, p := range processes {
		if classified[p.PID] || isIgnoredProcess(p.Command) {
			continue
		}
		if port := boundProcessPort(portToProcess, p.PID); port > 0 {
			state.Unexpected = append(state.Unexpected, domain.UnexpectedProcess{PID: p.PID, Command: p.Command, Port: port, User: p.User})
		}
	}
}

func isIgnoredProcess(command string) bool {
	for name := range systemProcessNames {
		if containsSubstring(command, name) {
			return true
		}
	}
	return containsSubstring(command, "caddy")
}

func boundProcessPort(portToProcess map[int]string, pid int) int {
	if pid <= 0 {
		return 0
	}
	needle := fmt.Sprintf("pid=%d", pid)
	for port, procName := range portToProcess {
		if containsSubstring(procName, needle) {
			return port
		}
	}
	return 0
}

type typedResourceStatus struct {
	ID    string
	State string
}

type typedResourcesEnvelope struct {
	Resources []typedResourceEnvelopeItem `json:"resources"`
}

type typedResourceEnvelopeItem struct {
	Resource struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"resource"`
	Name    string `json:"name"`
	ID      string `json:"id"`
	Running bool   `json:"running"`
	Status  string `json:"status"`
}

// typedResourceStatuses accepts both the fleet envelope emitted by the CLI
// and the older map-shaped status payload. Keeping this adapter at the edge
// lets the health model depend on explicit lifecycle fields rather than on
// driver-specific process names.
func typedResourceStatuses(raw string) []typedResourceStatus {
	var envelope typedResourcesEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil && envelope.Resources != nil {
		return typedEnvelopeResourceStatuses(envelope.Resources)
	}
	return legacyResourceStatuses(raw)
}

func typedEnvelopeResourceStatuses(items []typedResourceEnvelopeItem) []typedResourceStatus {
	result := make([]typedResourceStatus, 0, len(items))
	for _, item := range items {
		id := envelopeResourceID(item)
		state := envelopeResourceState(item)
		if id != "" && state != "" {
			result = append(result, typedResourceStatus{ID: id, State: state})
		}
	}
	return result
}

func envelopeResourceID(item typedResourceEnvelopeItem) string {
	for _, id := range []string{item.Name, item.ID, item.Resource.Name, item.Resource.ID} {
		if id != "" {
			return id
		}
	}
	return ""
}

func envelopeResourceState(item typedResourceEnvelopeItem) string {
	state := strings.ToLower(strings.TrimSpace(item.Status))
	if item.Running || state == "running" || state == "healthy" || state == "ready" {
		return "running"
	}
	return state
}

func legacyResourceStatuses(raw string) []typedResourceStatus {
	var byID map[string]struct {
		Running bool   `json:"running"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal([]byte(raw), &byID); err != nil {
		return nil
	}
	result := make([]typedResourceStatus, 0, len(byID))
	for id, item := range byID {
		state := strings.ToLower(strings.TrimSpace(item.Status))
		if item.Running {
			state = "running"
		}
		if state != "" {
			result = append(result, typedResourceStatus{ID: id, State: state})
		}
	}
	return result
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
	runningScenarios := runningScenarioIDs(processState.Scenarios)
	runningResources := runningResourceIDs(processState.Resources)
	expected := []domain.ExpectedProcess{expectedProcess("scenario", manifest.Scenario.ID, runningScenarios[manifest.Scenario.ID], dirExists)}
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue
		}
		expected = append(expected, expectedProcess("scenario", scenarioID, runningScenarios[scenarioID], dirExists))
	}
	for _, resourceID := range manifest.Dependencies.Resources {
		expected = append(expected, expectedProcess("resource", resourceID, runningResources[resourceID], dirExists))
	}
	return expected
}

func runningScenarioIDs(scenarios []domain.ScenarioProcess) map[string]bool {
	running := make(map[string]bool, len(scenarios))
	for _, scenario := range scenarios {
		if scenario.Status == "running" {
			running[scenario.ID] = true
		}
	}
	return running
}

func runningResourceIDs(resources []domain.ResourceProcess) map[string]bool {
	running := make(map[string]bool, len(resources))
	for _, resource := range resources {
		if resource.Status == "running" {
			running[resource.ID] = true
		}
	}
	return running
}

func expectedProcess(kind, id string, running bool, directories map[string]bool) domain.ExpectedProcess {
	key := kind + ":" + id
	directoryExists := directories[key]
	state := expectedProcessState(running, directoryExists)
	return domain.ExpectedProcess{ID: id, Type: kind, State: state, DirectoryExists: directoryExists}
}

func expectedProcessState(running, directoryExists bool) string {
	if running {
		return "running"
	}
	if directoryExists {
		return "stopped"
	}
	return "needs_setup"
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
	identity = normalizedSSHIdentity(identity)
	state := collector.ParseSystemState(systemMetricResults(results))
	state.SSH = sshHealthFromResults(results, identity)
	verifyExplicitSSHKey(results, identity, pubKeyContent, &state.SSH)
	return state
}

func normalizedSSHIdentity(identity sshidentity.DeploymentSSHIdentity) sshidentity.DeploymentSSHIdentity {
	if identity.AuthMode == "" {
		identity.AuthMode = sshidentity.AuthModeUnknown
	}
	if identity.VerificationState == "" {
		identity.VerificationState = sshidentity.VerificationUnknown
	}
	return identity
}

func systemMetricResults(results map[string]probeResult) map[string]systemmetrics.CommandResult {
	converted := make(map[string]systemmetrics.CommandResult, len(results))
	for id, result := range results {
		converted[id] = systemmetrics.CommandResult{Stdout: result.result.Stdout, Stderr: result.result.Stderr, ExitCode: result.result.ExitCode}
	}
	return converted
}

func sshHealthFromResults(results map[string]probeResult, identity sshidentity.DeploymentSSHIdentity) domain.SSHHealth {
	var connected bool
	var latencyMs int64
	if ping, ok := results["ssh_ping"]; ok {
		connected = ping.err == nil && ping.result.ExitCode == 0
		latencyMs = ping.durationMs
	}
	verification := identity.VerificationState
	if identity.AuthMode == sshidentity.AuthModeExplicitKey {
		verification = sshidentity.VerificationUnknown
	}
	return domain.SSHHealth{Connected: connected, LatencyMs: latencyMs, KeyPath: identity.KeyPath, AuthMode: string(identity.AuthMode),
		VerificationState: string(verification), PublicKeyFingerprint: identity.PublicKeyFingerprint, LastVerifiedAt: identity.LastVerifiedAt}
}

func verifyExplicitSSHKey(results map[string]probeResult, identity sshidentity.DeploymentSSHIdentity, pubKeyContent string, health *domain.SSHHealth) {
	if identity.AuthMode != sshidentity.AuthModeExplicitKey || pubKeyContent == "" {
		return
	}
	keyResult, ok := results["ssh_key_check"]
	if !ok || keyResult.err != nil {
		return
	}
	parts := strings.Fields(pubKeyContent)
	if len(parts) < 2 {
		return
	}
	key := parts[0] + " " + parts[1]
	if strings.Contains(keyResult.result.Stdout, key) {
		health.VerificationState = string(sshidentity.VerificationAuthorized)
		return
	}
	health.VerificationState = string(sshidentity.VerificationUnauthorized)
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
