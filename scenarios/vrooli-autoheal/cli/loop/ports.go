package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/repo-contract-go/cliinvoke"

	repocontract "github.com/vrooli/repo-contract-go"
)

// detection is what detectAPIPort found. Verified is a port on which the
// autoheal API identified itself. Pending is a port the process registry
// names but on which nothing answered yet; it is where waitForAPIHealthy
// looks first, and it is never adopted without passing the identity check.
type detection struct {
	Verified string
	Pending  string
}

// adoptPort makes port the loop's target, but only after the process there
// has identified itself as the autoheal API. This is the single adoption
// path: every detection strategy ends here, so no strategy can adopt a
// stranger the way the pre-identity code adopted two orphaned mock servers
// on 2026-09-01.
func (config *Config) adoptPort(ctx context.Context, port string) bool {
	port = strings.TrimSpace(port)
	if port == "" || !isAutohealAPI(ctx, port) {
		return false
	}
	changed := port != config.APIPort
	config.APIPort = port
	config.LastKnownPort = port
	config.setBaseURL("http://localhost:" + port)
	if changed {
		log.Printf("adopted autoheal API on port %s", port)
	}
	return true
}

// dropPort forgets the current port after the API stopped answering there,
// keeping it as the last-known candidate for the next detection.
func (config *Config) dropPort() {
	if config.APIPort != "" {
		config.LastKnownPort = config.APIPort
	}
	config.APIPort = ""
}

// detectAPIPort finds where the autoheal API listens. The strategies are
// ordered from most to least authoritative, and every one of them passes
// through the identity check before a port is returned as verified. Strategy
// 2 alone may surface an unverified port, and only as Pending.
func detectAPIPort(ctx context.Context, config *Config) detection {
	// Strategy 1: the environment names a port.
	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" && isAutohealAPI(ctx, port) {
		return detection{Verified: port}
	}

	// Strategy 2: the process registry's port file. A file with nothing
	// answering behind it usually means the scenario is still starting, so
	// the port is worth waiting on but not worth adopting.
	var pending string
	for _, portFile := range processRegistryPaths(config, "port") {
		data, err := os.ReadFile(portFile)
		if err != nil {
			continue
		}
		port := strings.TrimSpace(string(data))
		if port == "" {
			continue
		}
		if isAutohealAPI(ctx, port) {
			return detection{Verified: port}
		}
		if pending == "" {
			pending = port
		}
	}

	// Strategy 3: ask the CLI.
	if config.VrooliCmdPath != "" {
		for _, port := range []string{getPortFromScenarioStatus(ctx, config), getPortFromVrooliCLI(ctx, config)} {
			if port != "" && isAutohealAPI(ctx, port) {
				return detection{Verified: port, Pending: pending}
			}
		}
	}

	// Strategy 4: process metadata.
	for _, metaFile := range processRegistryPaths(config, "metadata.json") {
		data, err := os.ReadFile(metaFile)
		if err != nil {
			continue
		}
		var meta struct {
			APIPort string            `json:"api_port"`
			Port    string            `json:"port"`
			Ports   map[string]string `json:"ports"`
		}
		if json.Unmarshal(data, &meta) != nil {
			continue
		}
		for _, port := range []string{meta.APIPort, meta.Port, meta.Ports["api"]} {
			if port != "" && isAutohealAPI(ctx, port) {
				return detection{Verified: port, Pending: pending}
			}
		}
	}

	// Strategy 5: the last port the API answered on. The lifecycle can
	// reassign a port to another scenario, so this is a guess like any
	// other.
	if config.LastKnownPort != "" && isAutohealAPI(ctx, config.LastKnownPort) {
		return detection{Verified: config.LastKnownPort, Pending: pending}
	}

	// Strategy 6: the historical allocations. A bare 200 here proves only
	// that SOMETHING listens; two orphaned test fixtures on 15000/15001 once
	// convinced the pre-identity loop that a stopped autoheal was alive.
	for _, port := range config.ProbePorts {
		candidate := strconv.Itoa(port)
		if isAutohealAPI(ctx, candidate) {
			return detection{Verified: candidate, Pending: pending}
		}
	}

	return detection{Pending: pending}
}

// processRegistryPaths lists the two places the lifecycle engine records a
// scenario's process entry: the repo-local registry and the runtime home's.
func processRegistryPaths(config *Config, name string) []string {
	paths := []string{
		filepath.Join(config.VrooliRoot, ".vrooli", "processes", "scenarios", config.ScenarioName, name),
	}
	if processRoot := runtimeHomeEntry(repocontract.HomeKeyProcesses); processRoot != "" {
		paths = append(paths, filepath.Join(processRoot, "scenarios", config.ScenarioName, name))
	}
	return paths
}

// getPortFromVrooliCLI reads `vrooli scenario port <name> API_PORT`.
func getPortFromVrooliCLI(ctx context.Context, config *Config) string {
	res := invokeVrooli(ctx, config, nil, nil, cliinvoke.ScenarioPort(config.ScenarioName, "API_PORT")...)
	if res.Class != cliinvoke.OK {
		return ""
	}
	port := strings.TrimSpace(string(res.Stdout))
	if _, err := strconv.Atoi(port); err != nil {
		return ""
	}
	return port
}

// scenarioStatusResponse is the typed shape of `vrooli scenario status
// <name> --json` the loop depends on. The preflight's cli-contract check
// parses through the same shape.
type scenarioStatusResponse struct {
	Success  bool `json:"success"`
	Scenario struct {
		Name  string         `json:"name"`
		Ports map[string]int `json:"ports"`
	} `json:"scenario"`
	Runtime struct {
		Ports map[string]int `json:"ports"`
	} `json:"runtime"`
}

func getPortFromScenarioStatus(ctx context.Context, config *Config) string {
	res := invokeVrooli(ctx, config, nil, nil, cliinvoke.ScenarioStatusJSON(config.ScenarioName)...)
	if res.Class != cliinvoke.OK {
		return ""
	}
	var status scenarioStatusResponse
	if err := json.Unmarshal(res.Stdout, &status); err != nil || !status.Success {
		return ""
	}
	for _, ports := range []map[string]int{status.Runtime.Ports, status.Scenario.Ports} {
		if port := apiPortFromMap(ports); port != "" {
			return port
		}
	}
	return ""
}

func apiPortFromMap(ports map[string]int) string {
	for _, key := range []string{"API_PORT", "api", "api_port"} {
		if port, ok := ports[key]; ok && port > 0 {
			return strconv.Itoa(port)
		}
	}
	return ""
}
