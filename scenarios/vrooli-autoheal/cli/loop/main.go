// Vrooli Autoheal Loop - Cross-platform boot recovery watchdog
// This Go binary manages the vrooli-autoheal scenario lifecycle,
// ensuring it starts on boot and recovers from crashes.
//
// Build: go build -o vrooli-autoheal-loop ./cli/loop
// Usage: vrooli-autoheal-loop [--interval SECONDS] [--api-url URL]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Config holds loop configuration
type Config struct {
	APIPort             string
	TickInterval        time.Duration
	MaxFailures         int
	StartupGrace        time.Duration
	StartupTimeout      time.Duration
	HealthCheckInterval time.Duration
	VrooliRoot          string
	ScenarioName        string
	HealthEndpoint      string
	TickEndpoint        string
	ManageAPILifecycle  bool
	LastKnownPort       string
	VrooliCmdPath       string
}

type scenarioStatusResponse struct {
	Success  bool `json:"success"`
	Scenario struct {
		Ports map[string]int `json:"ports"`
	} `json:"scenario"`
	Runtime struct {
		Ports map[string]int `json:"ports"`
	} `json:"runtime"`
}

// TickResponse represents the API response from /tick
type TickResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Summary struct {
		Total    int `json:"total"`
		OK       int `json:"ok"`
		Warning  int `json:"warning"`
		Critical int `json:"critical"`
	} `json:"summary"`
}

func main() {
	// Parse command line flags
	interval := flag.Int("interval", 60, "Tick interval in seconds")
	apiURL := flag.String("api-url", "", "API base URL (auto-detected if not specified)")
	maxFailures := flag.Int("max-failures", 3, "Max consecutive failures before restart")
	noManageAPI := flag.Bool("no-manage-api", false, "Disable API lifecycle management")
	vrooliBin := flag.String("vrooli-bin", "", "Path to the vrooli CLI to use for lifecycle operations")
	flag.Parse()

	config := &Config{
		TickInterval:        time.Duration(*interval) * time.Second,
		MaxFailures:         *maxFailures,
		StartupGrace:        30 * time.Second,
		StartupTimeout:      120 * time.Second,
		HealthCheckInterval: 5 * time.Second,
		ScenarioName:        "vrooli-autoheal",
		ManageAPILifecycle:  !*noManageAPI,
	}

	// Detect VROOLI_ROOT
	config.VrooliRoot = resolveVrooliRoot()
	if config.VrooliRoot == "" {
		log.Fatalf("Failed to resolve VROOLI_ROOT")
	}

	// Find vrooli command
	config.VrooliCmdPath = findVrooliCommand(config, strings.TrimSpace(*vrooliBin))

	log.Printf("Vrooli Autoheal Loop starting...")
	log.Printf("  VROOLI_ROOT: %s", config.VrooliRoot)
	log.Printf("  Interval: %v", config.TickInterval)
	log.Printf("  Max failures: %d", config.MaxFailures)
	log.Printf("  API lifecycle management: %v", config.ManageAPILifecycle)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// If managing API lifecycle, ensure API is running before starting loop
	if config.ManageAPILifecycle {
		log.Printf("Ensuring API is running...")
		if err := ensureAPIRunning(config); err != nil {
			log.Printf("Warning: Failed to ensure API is running: %v", err)
			log.Printf("Will retry in main loop...")
		}
	}

	// Set up API endpoints
	if *apiURL != "" {
		config.HealthEndpoint = *apiURL + "/health"
		config.TickEndpoint = *apiURL + "/api/v1/tick"
	} else {
		// Detect API port - may change between iterations
		port := detectAPIPort(config)
		if port != "" {
			config.APIPort = port
			config.LastKnownPort = port
			baseURL := fmt.Sprintf("http://localhost:%s", port)
			config.HealthEndpoint = baseURL + "/health"
			config.TickEndpoint = baseURL + "/api/v1/tick"
			log.Printf("  API detected at: %s", config.HealthEndpoint)
		} else {
			log.Printf("  API port not yet detected (will retry)")
		}
	}

	// Wait for startup grace period
	log.Printf("Waiting %v for system to stabilize...", config.StartupGrace)
	time.Sleep(config.StartupGrace)

	// Main loop
	ticker := time.NewTicker(config.TickInterval)
	defer ticker.Stop()

	consecutiveFailures := 0
	tickCount := 0

	for {
		select {
		case <-sigChan:
			log.Println("Received shutdown signal, exiting...")
			return
		case <-ticker.C:
			tickCount++
			log.Printf("[Tick %d] Running health check...", tickCount)

			// Re-detect port if we don't have one or if we're recovering
			if config.APIPort == "" || consecutiveFailures > 0 {
				port := detectAPIPort(config)
				if port != "" && port != config.APIPort {
					config.APIPort = port
					config.LastKnownPort = port
					baseURL := fmt.Sprintf("http://localhost:%s", port)
					config.HealthEndpoint = baseURL + "/health"
					config.TickEndpoint = baseURL + "/api/v1/tick"
					log.Printf("[Tick %d] API port updated to: %s", tickCount, port)
				}
			}

			// Check if we can reach the API
			if config.APIPort == "" {
				consecutiveFailures++
				log.Printf("[Tick %d] No API port detected (failures: %d/%d)",
					tickCount, consecutiveFailures, config.MaxFailures)

				if config.ManageAPILifecycle && consecutiveFailures >= config.MaxFailures {
					log.Printf("Max failures reached, attempting to start/restart API...")
					if err := ensureAPIRunning(config); err != nil {
						log.Printf("Failed to start API: %v", err)
					} else {
						log.Printf("API start initiated, waiting for stabilization...")
						time.Sleep(config.StartupGrace)
						// Re-detect port after restart
						if port := detectAPIPort(config); port != "" {
							config.APIPort = port
							config.LastKnownPort = port
							baseURL := fmt.Sprintf("http://localhost:%s", port)
							config.HealthEndpoint = baseURL + "/health"
							config.TickEndpoint = baseURL + "/api/v1/tick"
						}
					}
					consecutiveFailures = 0
				}
				continue
			}

			result, err := runTick(config)
			if err != nil {
				consecutiveFailures++
				log.Printf("[Tick %d] Error: %v (failures: %d/%d)",
					tickCount, err, consecutiveFailures, config.MaxFailures)

				// A failing tick is not by itself evidence that the API needs
				// restarting. Ticks also fail when the API is alive and its
				// database is busy — and restarting into that condition aborts
				// whatever was making progress and guarantees the next attempt
				// meets the same state. Confirm the process has actually stopped
				// answering before reaching for the biggest hammer available.
				if config.ManageAPILifecycle && consecutiveFailures >= config.MaxFailures && apiIsAlive(config.APIPort) {
					log.Printf("[Tick %d] %d consecutive tick failures, but the API is still answering on port %s; not restarting. Ticks fail this way while the database is busy, and a restart would only interrupt it.",
						tickCount, consecutiveFailures, config.APIPort)
					// Reset so the next sustained outage gets a fresh count
					// rather than restarting on the first failure after this.
					consecutiveFailures = 0
				} else if config.ManageAPILifecycle && consecutiveFailures >= config.MaxFailures {
					log.Printf("Max failures reached and the API is not answering, attempting to restart API...")
					if restartErr := restartAPI(config); restartErr != nil {
						log.Printf("Restart failed: %v, trying full start...", restartErr)
						if startErr := ensureAPIRunning(config); startErr != nil {
							log.Printf("Failed to start API: %v", startErr)
						}
					} else {
						log.Printf("API restart initiated, waiting for stabilization...")
						time.Sleep(config.StartupGrace)
					}
					// Re-detect port after restart
					if port := detectAPIPort(config); port != "" {
						config.APIPort = port
						config.LastKnownPort = port
						baseURL := fmt.Sprintf("http://localhost:%s", port)
						config.HealthEndpoint = baseURL + "/health"
						config.TickEndpoint = baseURL + "/api/v1/tick"
					}
					consecutiveFailures = 0
				}
			} else {
				if consecutiveFailures > 0 {
					log.Printf("[Tick %d] Recovered after %d failures", tickCount, consecutiveFailures)
				}
				consecutiveFailures = 0
				log.Printf("[Tick %d] Status: %s (OK: %d, Warn: %d, Crit: %d)",
					tickCount, result.Status, result.Summary.OK,
					result.Summary.Warning, result.Summary.Critical)
			}
		}
	}
}

func resolveVrooliRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

// findVrooliCommand locates the vrooli CLI
func findVrooliCommand(config *Config, explicitPath string) string {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			return explicitPath
		}
		log.Printf("Configured vrooli binary not found: %s", explicitPath)
	}
	if envPath := strings.TrimSpace(os.Getenv("VROOLI_BIN")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
		log.Printf("VROOLI_BIN points to a missing file: %s", envPath)
	}
	switch runtime.GOOS {
	case "windows":
		for _, candidate := range []string{
			filepath.Join(config.VrooliRoot, ".vrooli", "build", "vrooli.exe"),
			filepath.Join(config.VrooliRoot, "vrooli.exe"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		// Check for vrooli.bat in the CLI directory
		batPath := filepath.Join(config.VrooliRoot, "cli", "vrooli.bat")
		if _, err := os.Stat(batPath); err == nil {
			return batPath
		}
		// Fall back to vrooli in PATH
		if path, err := exec.LookPath("vrooli"); err == nil {
			return path
		}
		return ""
	default:
		for _, candidate := range []string{
			filepath.Join(config.VrooliRoot, ".vrooli", "build", "vrooli"),
			filepath.Join(config.VrooliRoot, "vrooli"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		if path, err := exec.LookPath("vrooli"); err == nil {
			return path
		}
		return ""
	}
}

// detectAPIPort attempts to find the API port using multiple strategies
func detectAPIPort(config *Config) string {
	// Strategy 1: Check environment variable
	if port := os.Getenv("API_PORT"); port != "" {
		if isPortHealthy(port) {
			return port
		}
	}

	// Strategy 2: Read from vrooli process registry (most reliable for Vrooli-managed scenarios)
	homeDir, _ := os.UserHomeDir()
	registryPaths := []string{
		filepath.Join(config.VrooliRoot, ".vrooli", "processes", "scenarios", config.ScenarioName, "port"),
		filepath.Join(homeDir, ".vrooli", "processes", "scenarios", config.ScenarioName, "port"),
	}

	for _, portFile := range registryPaths {
		if data, err := os.ReadFile(portFile); err == nil {
			port := strings.TrimSpace(string(data))
			if port != "" {
				// Validate the port is actually responding
				if isPortHealthy(port) {
					return port
				}
				// Port file exists but API isn't responding - might be starting up
				// Return port anyway so we know where to look
				return port
			}
		}
	}

	// Strategy 3: Try to get port from vrooli CLI
	if config.VrooliCmdPath != "" {
		if port := getPortFromScenarioStatus(config); port != "" {
			return port
		}
		if port := getPortFromVrooliCLI(config); port != "" {
			return port
		}
	}

	// Strategy 4: Check process metadata
	metadataPaths := []string{
		filepath.Join(config.VrooliRoot, ".vrooli", "processes", "scenarios", config.ScenarioName, "metadata.json"),
		filepath.Join(homeDir, ".vrooli", "processes", "scenarios", config.ScenarioName, "metadata.json"),
	}

	for _, metaFile := range metadataPaths {
		if data, err := os.ReadFile(metaFile); err == nil {
			var meta struct {
				APIPort string            `json:"api_port"`
				Port    string            `json:"port"`
				Ports   map[string]string `json:"ports"`
			}
			if json.Unmarshal(data, &meta) == nil {
				if meta.APIPort != "" {
					return meta.APIPort
				}
				if meta.Port != "" {
					return meta.Port
				}
				if p, ok := meta.Ports["api"]; ok {
					return p
				}
			}
		}
	}

	// Strategy 5: Use last known port if we have one
	if config.LastKnownPort != "" && isPortHealthy(config.LastKnownPort) {
		return config.LastKnownPort
	}

	// Strategy 6: Probe common port ranges
	// Vrooli allocates API ports from 15000-19999 for scenarios
	// Start with the most common allocations
	probePorts := []int{
		19761, 19762, 19763, 19764, 19765, // Historical defaults
		15000, 15001, 15002, 15003, 15004, // Start of range
		18000, 18001, 18002, 18003, 18004, // Middle of range
	}

	for _, port := range probePorts {
		if isPortHealthy(strconv.Itoa(port)) {
			return strconv.Itoa(port)
		}
	}

	return ""
}

// buildVrooliCmd constructs an *exec.Cmd that invokes the project vrooli CLI
// with --no-stale-check forced as the first global flag. This prevents the
// loop from ever triggering a Go rebuild of the project CLI during recovery —
// a broken top-level go.mod would otherwise cascade through the stale-check
// rebuild path and lock the host out of autoheal.
func buildVrooliCmd(vrooliCmdPath string, subArgs ...string) *exec.Cmd {
	args := append([]string{"--no-stale-check"}, subArgs...)
	if runtime.GOOS == "windows" && !strings.HasSuffix(vrooliCmdPath, ".bat") {
		psCmd := fmt.Sprintf("& '%s'", vrooliCmdPath)
		for _, a := range args {
			psCmd += " " + a
		}
		return exec.Command("powershell", "-Command", psCmd)
	}
	return exec.Command(vrooliCmdPath, args...)
}

// vrooliCommandTimeout bounds any single vrooli CLI invocation the loop makes.
//
// Lifecycle commands are the slowest thing here — a start does setup, develop,
// and a health wait — so this is generous. It is a backstop against hanging
// forever, not a latency target.
const vrooliCommandTimeout = 10 * time.Minute

// vrooliOutputGrace is how long we keep reading a finished command's output
// before giving up on descendants that inherited the pipe.
const vrooliOutputGrace = 5 * time.Second

// runVrooliCommand runs a vrooli CLI command and returns its combined output,
// and it is the ONLY way this loop should invoke that CLI.
//
// The plain exec.Cmd.CombinedOutput it replaces cannot be used here, and the
// reason is the 2026-08-01 outage. CombinedOutput hands the child a PIPE and
// then reads it until EOF. EOF arrives when every write end is closed — not when
// the child exits. `vrooli scenario start` spawns the long-lived runtime
// supervisor, which inherits that pipe as its stderr and holds it open for as
// long as it runs, which is forever. So the loop's very first action on boot —
// ensuring the API is up — blocked in Wait permanently. It never reached its
// tick loop, never ran a single health check, and never started any of the
// scenarios it exists to start. Observed directly: the loop parked in
// futex_do_wait holding the read end of pipe:[623670] while the supervisor held
// the write end as fd 2.
//
// Two things fix it, and both are here because either alone leaves a hole:
//
//   - WaitDelay bounds how long Wait will keep reading after the process itself
//     has exited, so an inherited pipe delays the result by seconds instead of
//     ending the loop's useful life.
//   - The context timeout bounds the command itself, for the different failure
//     where the CLI genuinely hangs rather than exiting and leaking a pipe.
//
// A supervisor that can be blocked forever by the thing it supervises is not a
// supervisor.
func runVrooliCommand(config *Config, subArgs ...string) ([]byte, error) {
	var combined bytes.Buffer
	err := runVrooliCommandInto(config, &combined, &combined, subArgs...)
	return combined.Bytes(), err
}

// runVrooliCommandStdout is the same, for commands whose stdout is parsed and
// must not have diagnostics interleaved into it.
func runVrooliCommandStdout(config *Config, subArgs ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	err := runVrooliCommandInto(config, &stdout, &stderr, subArgs...)
	return stdout.Bytes(), err
}

// runVrooliCommandInto is the shared body: timeout, WaitDelay, and the
// inherited-pipe tolerance described on runVrooliCommand.
func runVrooliCommandInto(config *Config, stdout, stderr io.Writer, subArgs ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), vrooliCommandTimeout)
	defer cancel()

	cmd := buildVrooliCmd(config.VrooliCmdPath, subArgs...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("VROOLI_ROOT=%s", config.VrooliRoot))

	// Stop waiting on inherited pipes shortly after the direct child is gone.
	cmd.WaitDelay = vrooliOutputGrace
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmdStartWithContext(ctx, cmd); err != nil {
		return err
	}
	err := cmd.Wait()
	// A WaitDelay expiry means the command finished but something it spawned
	// still holds the output pipe. The command's own result is what we asked
	// for, and it is complete; the leftover descendant is expected and is not
	// this call's failure.
	if errors.Is(err, exec.ErrWaitDelay) {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("vrooli %s timed out after %v: %w",
			strings.Join(subArgs, " "), vrooliCommandTimeout, ctx.Err())
	}
	return err
}

// cmdStartWithContext attaches ctx to cmd and starts it. exec.CommandContext
// cannot be used directly because buildVrooliCmd owns command construction
// (including the Windows powershell wrapper), so the context is wired onto the
// already-built command instead.
func cmdStartWithContext(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
	return nil
}

// getPortFromVrooliCLI tries to get the port using vrooli scenario port command
func getPortFromVrooliCLI(config *Config) string {
	output, err := runVrooliCommandStdout(config, "scenario", "port", config.ScenarioName, "API_PORT")
	if err != nil {
		return ""
	}

	port := strings.TrimSpace(string(output))
	// Validate it looks like a port number
	if _, err := strconv.Atoi(port); err == nil {
		return port
	}

	return ""
}

func getPortFromScenarioStatus(config *Config) string {
	output, err := runVrooliCommandStdout(config, "scenario", "status", config.ScenarioName, "--json")
	if err != nil {
		return ""
	}

	var status scenarioStatusResponse
	if err := json.Unmarshal(output, &status); err != nil {
		return ""
	}
	if !status.Success {
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

// isPortHealthy checks if the API is responding on the given port
func isPortHealthy(port string) bool {
	url := fmt.Sprintf("http://localhost:%s/health", port)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// apiIsAlive reports whether the API process is still answering HTTP at all,
// regardless of what it says about its own health.
//
// This is the question a supervisor should ask before restarting something, and
// it is NOT the question isPortHealthy answers. On 2026-08-01 the API was
// serving every request correctly while its database was busy with a retention
// cycle; /health reported unhealthy, ticks timed out behind the same
// contention, and this loop restarted a process with nothing wrong with it. The
// restart aborted the cycle, and RunOnStart began it again — so the supervisor
// was the mechanism that prevented recovery, on every iteration, indefinitely.
//
// A process that completes an HTTP round trip is not the failure a restart
// fixes. Restarting is for a process that has stopped answering.
func apiIsAlive(port string) bool {
	if port == "" {
		return false
	}
	url := fmt.Sprintf("http://localhost:%s/health", port)
	// Generous relative to the health handler's own budget: the point is to
	// find out whether anything is listening and answering, not whether it is
	// answering quickly.
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	// Any status is an answer. A 503 from a service reporting its own degraded
	// dependency is still a live process that a restart cannot improve.
	return true
}

// isScenarioRunning checks if the scenario process is active
func isScenarioRunning(config *Config) bool {
	// Check PID file
	homeDir, _ := os.UserHomeDir()
	pidPaths := []string{
		filepath.Join(config.VrooliRoot, ".vrooli", "processes", "scenarios", config.ScenarioName, "start-api.pid"),
		filepath.Join(homeDir, ".vrooli", "processes", "scenarios", config.ScenarioName, "start-api.pid"),
	}

	for _, pidFile := range pidPaths {
		if data, err := os.ReadFile(pidFile); err == nil {
			pidStr := strings.TrimSpace(string(data))
			if pid, err := strconv.Atoi(pidStr); err == nil {
				// Check if process with this PID exists
				if isProcessRunning(pid) {
					return true
				}
			}
		}
	}

	// Fall back to checking if API is responding
	port := detectAPIPort(config)
	if port != "" && isPortHealthy(port) {
		return true
	}

	return false
}

// isProcessRunning checks if a process with the given PID exists
func isProcessRunning(pid int) bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), strconv.Itoa(pid))
	default:
		// On Unix, sending signal 0 checks if process exists
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}
}

// ensureAPIRunning makes sure the API is running, starting it if necessary
func ensureAPIRunning(config *Config) error {
	// First check if API is already healthy
	port := detectAPIPort(config)
	if port != "" && isPortHealthy(port) {
		log.Printf("API is already running and healthy on port %s", port)
		config.APIPort = port
		config.LastKnownPort = port
		return nil
	}

	// Check if vrooli command is available
	if config.VrooliCmdPath == "" {
		return fmt.Errorf("vrooli command not found - cannot manage API lifecycle")
	}

	// Check if scenario is running but not healthy
	if isScenarioRunning(config) {
		log.Printf("Scenario process exists but API not healthy, restarting...")
		return restartAPI(config)
	}

	// Scenario not running - start it
	log.Printf("Starting scenario via vrooli...")
	if err := startAPI(config); err != nil {
		return fmt.Errorf("failed to start scenario: %w", err)
	}

	// Wait for API to become healthy
	return waitForAPIHealthy(config)
}

// startAPI starts the scenario using vrooli
func startAPI(config *Config) error {
	output, err := runVrooliCommand(config, "scenario", "start", config.ScenarioName, "--best-effort")
	if err != nil {
		return fmt.Errorf("start command failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Start output: %s", strings.TrimSpace(string(output)))
	return nil
}

// restartAPI attempts to restart the autoheal API
func restartAPI(config *Config) error {
	if config.VrooliCmdPath == "" {
		return fmt.Errorf("vrooli command not found")
	}

	output, err := runVrooliCommand(config, "scenario", "restart", config.ScenarioName, "--best-effort")
	if err != nil {
		return fmt.Errorf("restart command failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Restart output: %s", strings.TrimSpace(string(output)))
	return nil
}

// waitForAPIHealthy waits for the API to become healthy
func waitForAPIHealthy(config *Config) error {
	deadline := time.Now().Add(config.StartupTimeout)
	lastError := ""

	for time.Now().Before(deadline) {
		// Try to detect port
		port := detectAPIPort(config)
		if port != "" {
			config.APIPort = port
			config.LastKnownPort = port

			// Check if healthy
			if isPortHealthy(port) {
				baseURL := fmt.Sprintf("http://localhost:%s", port)
				config.HealthEndpoint = baseURL + "/health"
				config.TickEndpoint = baseURL + "/api/v1/tick"
				log.Printf("API is healthy on port %s", port)
				return nil
			}
			lastError = fmt.Sprintf("port %s found but not healthy yet", port)
		} else {
			lastError = "no port detected yet"
		}

		elapsed := time.Since(deadline.Add(-config.StartupTimeout))
		log.Printf("Waiting for API... (%v/%v) - %s", elapsed.Round(time.Second), config.StartupTimeout, lastError)
		time.Sleep(config.HealthCheckInterval)
	}

	return fmt.Errorf("API failed to become healthy within %v: %s", config.StartupTimeout, lastError)
}

// runTick calls the /tick endpoint
func runTick(config *Config) (*TickResponse, error) {
	if config.TickEndpoint == "" {
		return nil, fmt.Errorf("tick endpoint not configured")
	}

	client := &http.Client{Timeout: 5 * time.Minute}

	requestURL, err := url.Parse(config.TickEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid tick endpoint URL: %w", err)
	}
	query := requestURL.Query()
	query.Set("compact", "true")
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequest("POST", requestURL.String(), bytes.NewReader([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 409 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result TickResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
