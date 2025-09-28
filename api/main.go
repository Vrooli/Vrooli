// Vrooli Unified API - Single HTTP server for all Vrooli operations
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// === Common Types ===
type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// === Vrooli Configuration ===
var (
	vrooliRoot = getVrooliRoot()
)

// === Native Scenario Management (Replaces Python Orchestrator) ===

type RunningScenario struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Processes int            `json:"processes"`
	StartedAt *time.Time     `json:"started_at"`
	Runtime   string         `json:"runtime"`
	Ports     map[string]int `json:"ports"`
}

type processTableEntry struct {
	PID     int
	PPID    int
	State   string
	Command string
}

type trackedProcessStats struct {
	trackedPIDs    map[int]struct{}
	trackedCount   int
	runningTracked int
}

type ProcessHealthSnapshot struct {
	ZombieCount   int
	ZombieStatus  string
	ZombieEmoji   string
	OrphanCount   int
	OrphanStatus  string
	OrphanEmoji   string
	OverallStatus string
}

var orphanCommandPattern = regexp.MustCompile(`(vrooli|/scenarios/.*/(api|ui)|node_modules/.bin/vite|ecosystem-manager|picker-wheel)`)

// Helper function to check if a scenario name corresponds to a valid scenario directory
func isValidScenario(name string) bool {
	// Check if scenario directory exists
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		return false
	}

	// Check for valid scenario name pattern
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

// Fork bomb detection - simple process count check
func checkForkBomb() error {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Count(string(output), "\n")
	if lines > 2000 { // Same limit as Python version
		return fmt.Errorf("system overload: %d processes (limit: 2000)", lines)
	}
	return nil
}

// Build process table using a single ps invocation for efficient lookups
func buildProcessTable() (map[int]processTableEntry, error) {
	cmd := exec.Command("ps", "-eo", "pid,ppid,state,cmd")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect process table: %w", err)
	}

	processTable := make(map[int]processTableEntry)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if lineNum == 0 {
			lineNum += 1
			continue // Skip header
		}
		lineNum += 1
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			ppid = 0
		}

		state := fields[2]
		command := strings.Join(fields[3:], " ")

		processTable[pid] = processTableEntry{
			PID:     pid,
			PPID:    ppid,
			State:   state,
			Command: command,
		}
	}

	return processTable, nil
}

// Load tracked process metadata (PIDs/PGIDs) from lifecycle process files once
func loadTrackedProcessStats(processTable map[int]processTableEntry) trackedProcessStats {
	stats := trackedProcessStats{
		trackedPIDs: make(map[int]struct{}),
	}

	processesDir := filepath.Join(os.Getenv("HOME"), ".vrooli/processes/scenarios")
	if _, err := os.Stat(processesDir); os.IsNotExist(err) {
		return stats
	}

	_ = filepath.Walk(processesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var processInfo map[string]interface{}
		if err := json.Unmarshal(data, &processInfo); err != nil {
			return nil
		}

		if pidFloat, ok := processInfo["pid"].(float64); ok {
			pid := int(pidFloat)
			if pid > 0 {
				stats.trackedPIDs[pid] = struct{}{}
				stats.trackedCount += 1
				if _, running := processTable[pid]; running {
					stats.runningTracked += 1
				}
			}
		}

		if pgidFloat, ok := processInfo["pgid"].(float64); ok {
			pgid := int(pgidFloat)
			if pgid > 0 {
				stats.trackedPIDs[pgid] = struct{}{}
			}
		}

		return nil
	})

	return stats
}

func interpretZombieStatus(count int) (string, string) {
	switch {
	case count == 0:
		return "healthy", "✅"
	case count <= 5:
		return "normal", "✅"
	case count <= 20:
		return "warning", "⚠️"
	default:
		return "critical", "🔴"
	}
}

func interpretOrphanStatus(count int) (string, string) {
	switch {
	case count == 0:
		return "healthy", "✅"
	case count <= 3:
		return "normal", "✅"
	case count <= 10:
		return "warning", "⚠️"
	default:
		return "critical", "🔴"
	}
}

func isTrackedOrAncestorTracked(pid int, tracked map[int]struct{}, processTable map[int]processTableEntry, memo map[int]bool, visiting map[int]bool) bool {
	if _, ok := tracked[pid]; ok {
		memo[pid] = true
		return true
	}

	if val, ok := memo[pid]; ok {
		return val
	}

	entry, ok := processTable[pid]
	if !ok {
		memo[pid] = false
		return false
	}

	if entry.PPID == 0 || entry.PPID == 1 {
		memo[pid] = false
		return false
	}

	if visiting[pid] {
		memo[pid] = false
		return false
	}
	visiting[pid] = true

	trackedAncestor := isTrackedOrAncestorTracked(entry.PPID, tracked, processTable, memo, visiting)
	visiting[pid] = false
	if trackedAncestor {
		memo[pid] = true
		return true
	}

	memo[pid] = false
	return false
}

func countOrphanProcessesFast(processTable map[int]processTableEntry, tracked map[int]struct{}) int {
	orphanCount := 0
	memo := make(map[int]bool)
	visiting := make(map[int]bool)
	for pid, entry := range processTable {
		if !orphanCommandPattern.MatchString(entry.Command) {
			continue
		}
		if strings.Contains(entry.Command, "./vrooli-api") || strings.Contains(entry.Command, "vrooli-api-new") {
			continue
		}
		if isTrackedOrAncestorTracked(pid, tracked, processTable, memo, visiting) {
			continue
		}
		orphanCount += 1
	}
	return orphanCount
}

func collectProcessHealthSnapshot() ProcessHealthSnapshot {
	processTable, err := buildProcessTable()
	if err != nil {
		log.Printf("Failed to build process table: %v", err)
		return ProcessHealthSnapshot{
			ZombieStatus:  "unknown",
			ZombieEmoji:   "❔",
			OrphanStatus:  "unknown",
			OrphanEmoji:   "❔",
			OverallStatus: "unknown",
		}
	}

	snapshot, _ := computeProcessSnapshot(processTable)
	return snapshot
}

func computeProcessSnapshot(processTable map[int]processTableEntry) (ProcessHealthSnapshot, trackedProcessStats) {
	stats := loadTrackedProcessStats(processTable)

	zombieCount := 0
	for _, entry := range processTable {
		if strings.HasPrefix(entry.State, "Z") {
			zombieCount += 1
		}
	}

	orphanCount := countOrphanProcessesFast(processTable, stats.trackedPIDs)
	zombieStatus, zombieEmoji := interpretZombieStatus(zombieCount)
	orphanStatus, orphanEmoji := interpretOrphanStatus(orphanCount)

	overallStatus := "healthy"
	switch {
	case zombieStatus == "critical" || orphanStatus == "critical":
		overallStatus = "critical"
	case zombieStatus == "warning" || orphanStatus == "warning":
		overallStatus = "warning"
	case zombieStatus == "normal" || orphanStatus == "normal":
		overallStatus = "normal"
	}

	snapshot := ProcessHealthSnapshot{
		ZombieCount:   zombieCount,
		ZombieStatus:  zombieStatus,
		ZombieEmoji:   zombieEmoji,
		OrphanCount:   orphanCount,
		OrphanStatus:  orphanStatus,
		OrphanEmoji:   orphanEmoji,
		OverallStatus: overallStatus,
	}

	return snapshot, stats
}

func getEnhancedProcessMetrics() map[string]interface{} {
	processTable, err := buildProcessTable()
	if err != nil {
		log.Printf("Failed to build process table for metrics: %v", err)
		return map[string]interface{}{
			"tracked_processes": 0,
			"running_tracked":   0,
			"child_processes":   0,
			"total_processes":   0,
			"zombie_processes":  0,
			"orphan_processes":  0,
		}
	}

	snapshot, stats := computeProcessSnapshot(processTable)

	totalProcesses := len(processTable)
	childProcesses := totalProcesses - stats.runningTracked
	if _, exists := processTable[os.Getpid()]; exists {
		childProcesses -= 1 // Exclude API process
	}
	if childProcesses < 0 {
		childProcesses = 0
	}

	metrics := map[string]interface{}{
		"tracked_processes": stats.trackedCount,
		"running_tracked":   stats.runningTracked,
		"child_processes":   childProcesses,
		"total_processes":   totalProcesses,
		"zombie_processes":  snapshot.ZombieCount,
		"orphan_processes":  snapshot.OrphanCount,
	}

	return metrics
}

// Discover ports for a specific scenario by reading process environment variables
func discoverScenarioPorts(scenarioName string) map[string]int {
	ports := make(map[string]int)

	// First, load the service.json to see what ports are actually defined
	serviceFile := filepath.Join(vrooliRoot, "scenarios", scenarioName, ".vrooli", "service.json")
	serviceData, err := os.ReadFile(serviceFile)
	if err != nil {
		// If we can't read service.json, return empty
		return ports
	}

	// Parse service.json to get defined ports
	var serviceConfig struct {
		Ports map[string]struct {
			EnvVar string `json:"env_var"`
			Range  string `json:"range"`
		} `json:"ports"`
	}

	if err := json.Unmarshal(serviceData, &serviceConfig); err != nil {
		return ports
	}

	// Build a set of valid port environment variables
	validPortVars := make(map[string]bool)
	for _, portConfig := range serviceConfig.Ports {
		if portConfig.EnvVar != "" {
			validPortVars[portConfig.EnvVar] = true
		}
	}

	// If no ports defined, return empty
	if len(validPortVars) == 0 {
		return ports
	}

	// Get PIDs from the scenario's process directory
	processesDir := filepath.Join(os.Getenv("HOME"), ".vrooli/processes/scenarios", scenarioName)

	if _, err := os.Stat(processesDir); os.IsNotExist(err) {
		return ports
	}

	// Read all .json files to get PIDs
	processFiles, _ := filepath.Glob(filepath.Join(processesDir, "*.json"))
	for _, file := range processFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var processInfo ProcessInfo
		if err := json.Unmarshal(data, &processInfo); err != nil {
			continue
		}

		// Check if process is actually running
		if !isPidRunning(processInfo.PID) {
			continue
		}

		// Read environment variables from /proc/PID/environ
		envFile := fmt.Sprintf("/proc/%d/environ", processInfo.PID)
		envData, err := os.ReadFile(envFile)
		if err != nil {
			continue
		}

		// Parse environment variables (they're null-separated)
		envVars := strings.Split(string(envData), "\x00")
		for _, envVar := range envVars {
			if strings.Contains(envVar, "=") {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) == 2 {
					varName := parts[0]
					varValue := parts[1]

					// Only capture ports that are defined in service.json
					if validPortVars[varName] {
						if port, err := strconv.Atoi(varValue); err == nil {
							ports[varName] = port
						}
					}
				}
			}
		}
	}

	return ports
}

// ProcessInfo represents a single tracked process
type ProcessInfo struct {
	PID        int       `json:"pid"`
	ProcessID  string    `json:"process_id"`
	Phase      string    `json:"phase"`
	Scenario   string    `json:"scenario"`
	Step       string    `json:"step"`
	Command    string    `json:"command"`
	WorkingDir string    `json:"working_dir"`
	LogFile    string    `json:"log_file"`
	StartedAt  time.Time `json:"started_at"`
	Status     string    `json:"status"`
}

// Helper function to check if a PID is running
func isPidRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Discover all running scenarios using PID-based tracking
func discoverRunningScenarios() ([]RunningScenario, error) {

	scenarios := make(map[string]*RunningScenario)
	processesDir := filepath.Join(os.Getenv("HOME"), ".vrooli/processes/scenarios")

	// Check PID files for running processes
	if _, err := os.Stat(processesDir); !os.IsNotExist(err) {
		processDirs, err := os.ReadDir(processesDir)
		if err == nil {
			for _, processDir := range processDirs {
				if !processDir.IsDir() {
					continue
				}

				scenarioName := processDir.Name()

				// Verify scenario exists and has service.json
				if !isValidScenario(scenarioName) {
					continue
				}

				scenarioProcessDir := filepath.Join(processesDir, scenarioName)
				runningProcesses := 0
				var earliestStart *time.Time

				// Check each process file for this scenario
				processFiles, err := filepath.Glob(filepath.Join(scenarioProcessDir, "*.json"))
				if err != nil {
					continue
				}

				for _, processFile := range processFiles {
					// Read process metadata
					data, err := os.ReadFile(processFile)
					if err != nil {
						continue
					}

					var processInfo ProcessInfo
					if err := json.Unmarshal(data, &processInfo); err != nil {
						continue
					}

					// Check if process is actually running
					if isPidRunning(processInfo.PID) {
						runningProcesses++

						// Track earliest start time
						if earliestStart == nil || processInfo.StartedAt.Before(*earliestStart) {
							earliestStart = &processInfo.StartedAt
						}
					} else {
						// Cleanup dead process metadata
						os.Remove(processFile)
						pidFile := strings.Replace(processFile, ".json", ".pid", 1)
						os.Remove(pidFile)
					}
				}

				// Add scenario to results if it has running processes
				if runningProcesses > 0 {
					runtime := "unknown"
					if earliestStart != nil {
						duration := time.Since(*earliestStart)
						if duration < time.Hour {
							runtime = fmt.Sprintf("%.0fm", duration.Minutes())
						} else if duration < 24*time.Hour {
							runtime = fmt.Sprintf("%.1fh", duration.Hours())
						} else {
							runtime = fmt.Sprintf("%.1fd", duration.Hours()/24)
						}
					}

					scenarios[scenarioName] = &RunningScenario{
						Name:      scenarioName,
						Status:    "running",
						Processes: runningProcesses,
						StartedAt: earliestStart,
						Runtime:   runtime,
						Ports:     make(map[string]int),
					}
				}
			}
		}
	}

	// Convert map to slice and discover ports for each scenario
	var result []RunningScenario
	for _, scenario := range scenarios {
		scenario.Ports = discoverScenarioPorts(scenario.Name)
		result = append(result, *scenario)
	}

	return result, nil
}

// Start all scenarios natively
// Start all scenarios with concurrency support
func startAllScenariosNative() (map[string]interface{}, error) {
	// Check fork bomb BEFORE starting anything
	if err := checkForkBomb(); err != nil {
		return nil, err
	}

	// Find all scenarios
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %v", err)
	}

	type startResult struct {
		Name    string
		Success bool
		Error   string
	}

	// Collect valid scenarios first
	var validScenarios []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "templates" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Check if service.json exists
		serviceFile := filepath.Join(scenariosDir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
			continue
		}

		validScenarios = append(validScenarios, entry.Name())
	}

	if len(validScenarios) == 0 {
		return map[string]interface{}{
			"started": []map[string]string{},
			"failed":  []map[string]string{},
			"message": "No valid scenarios found",
		}, nil
	}

	// Use rolling/streaming concurrency instead of batched
	resultChan := make(chan startResult, len(validScenarios))
	concurrentLimit := 20 // Higher limit, smooth processing
	sem := make(chan struct{}, concurrentLimit)
	var wg sync.WaitGroup

	// Start worker pool that processes scenarios as they become available
	scenarioQueue := make(chan string, len(validScenarios))

	// Fill the queue
	for _, scenarioName := range validScenarios {
		scenarioQueue <- scenarioName
	}
	close(scenarioQueue)

	// Launch workers that pull from queue (rolling concurrency)
	for i := 0; i < concurrentLimit && i < len(validScenarios); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for scenarioName := range scenarioQueue {
				sem <- struct{}{} // Acquire semaphore

				err := startScenarioNative(scenarioName)
				if err != nil {
					resultChan <- startResult{
						Name:    scenarioName,
						Success: false,
						Error:   err.Error(),
					}
				} else {
					resultChan <- startResult{
						Name:    scenarioName,
						Success: true,
						Error:   "",
					}
				}

				<-sem // Release semaphore immediately so next scenario can start
			}
		}()
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var started []map[string]string
	var failed []map[string]string

	for result := range resultChan {
		if result.Success {
			started = append(started, map[string]string{
				"name":    result.Name,
				"message": "Started successfully",
			})
		} else {
			failed = append(failed, map[string]string{
				"name":  result.Name,
				"error": result.Error,
			})
		}
	}

	return map[string]interface{}{
		"started": started,
		"failed":  failed,
		"message": fmt.Sprintf("Started %d scenarios, %d failed", len(started), len(failed)),
	}, nil
}

// HealthCheckConfig represents a health check from service.json
type HealthCheckConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Critical bool   `json:"critical"`
	Timeout  int    `json:"timeout"`
	Interval int    `json:"interval"`
}

// ScenarioHealthConfig represents the health section from service.json
type ScenarioHealthConfig struct {
	Description        string              `json:"description"`
	Endpoints          map[string]string   `json:"endpoints"`
	Checks             []HealthCheckConfig `json:"checks"`
	Timeout            int                 `json:"timeout"`
	Interval           int                 `json:"interval"`
	StartupGracePeriod int                 `json:"startup_grace_period"`
}

// Load health config from scenario service.json (supports both v1 and v2 formats)
func loadHealthConfig(scenarioName string) (*ScenarioHealthConfig, error) {
	serviceFile := filepath.Join(vrooliRoot, "scenarios", scenarioName, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceFile)
	if err != nil {
		return nil, err
	}

	// Try v2.0 format first (health under lifecycle)
	var v2Config struct {
		Lifecycle struct {
			Health *ScenarioHealthConfig `json:"health"`
		} `json:"lifecycle"`
	}

	if err := json.Unmarshal(data, &v2Config); err == nil && v2Config.Lifecycle.Health != nil {
		return v2Config.Lifecycle.Health, nil
	}

	// Fall back to v1 format (health at root level)
	var v1Config struct {
		Health *ScenarioHealthConfig `json:"health"`
	}

	if err := json.Unmarshal(data, &v1Config); err != nil {
		return nil, err
	}

	return v1Config.Health, nil
}

// Expand environment variables in target URL using discovered ports
func expandEnvVars(target string, scenarioName string, ports map[string]int) string {
	expanded := target

	// Replace known port variables with discovered ports
	for varName, port := range ports {
		// Handle both ${VAR} and $VAR syntax
		expanded = strings.ReplaceAll(expanded, "${"+varName+"}", strconv.Itoa(port))
		expanded = strings.ReplaceAll(expanded, "$"+varName, strconv.Itoa(port))
	}

	// Fallback to environment variables if not in discovered ports
	if strings.Contains(expanded, "${API_PORT}") && os.Getenv("API_PORT") != "" {
		expanded = strings.ReplaceAll(expanded, "${API_PORT}", os.Getenv("API_PORT"))
	}
	if strings.Contains(expanded, "${UI_PORT}") && os.Getenv("UI_PORT") != "" {
		expanded = strings.ReplaceAll(expanded, "${UI_PORT}", os.Getenv("UI_PORT"))
	}

	return expanded
}

// Perform a single health check with discovered ports
func performHealthCheck(check HealthCheckConfig, scenarioName string, ports map[string]int) error {
	switch check.Type {
	case "http":
		target := expandEnvVars(check.Target, scenarioName, ports)

		// Parse URL to validate
		_, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("invalid URL: %s", target)
		}

		// Create HTTP client with timeout
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 5 * time.Second
		}

		client := &http.Client{
			Timeout: timeout,
		}

		resp, err := client.Get(target)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Accept 2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		return nil
	default:
		return fmt.Errorf("unsupported health check type: %s", check.Type)
	}
}

// Wait for scenario to become healthy
func waitForScenarioHealth(scenarioName string, healthConfig *ScenarioHealthConfig) error {
	if healthConfig == nil || len(healthConfig.Checks) == 0 {
		// No health checks defined, assume started immediately
		return nil
	}

	// Use startup grace period (default 15 seconds)
	gracePeriod := time.Duration(healthConfig.StartupGracePeriod) * time.Millisecond
	if gracePeriod == 0 {
		gracePeriod = 15 * time.Second
	}

	// Poll interval (default 1 second)
	pollInterval := 1 * time.Second

	deadline := time.Now().Add(gracePeriod)

	for time.Now().Before(deadline) {
		// Discover ports for this scenario (they may not be available immediately)
		ports := discoverScenarioPorts(scenarioName)

		allPassed := true

		for _, check := range healthConfig.Checks {
			if err := performHealthCheck(check, scenarioName, ports); err != nil {
				allPassed = false
				if check.Critical {
					// Don't fail immediately on critical checks during startup
					// Give them the full grace period
					break
				}
			}
		}

		if allPassed {
			return nil // All health checks passed!
		}

		// Wait before next check
		time.Sleep(pollInterval)
	}

	// Grace period expired, do final check with latest port discovery
	ports := discoverScenarioPorts(scenarioName)
	for _, check := range healthConfig.Checks {
		if err := performHealthCheck(check, scenarioName, ports); err != nil {
			if check.Critical {
				return fmt.Errorf("critical health check failed: %s (%s)", check.Name, err)
			}
		}
	}

	return nil // Non-critical failures are acceptable
}

// Check health status of a running scenario (for ongoing monitoring)
func checkScenarioHealth(scenarioName string, healthConfig *ScenarioHealthConfig, ports map[string]int) string {
	if healthConfig == nil || len(healthConfig.Checks) == 0 {
		return "running" // No health checks configured
	}

	criticalFailed := false
	nonCriticalFailed := false

	for _, check := range healthConfig.Checks {
		if err := performHealthCheck(check, scenarioName, ports); err != nil {
			if check.Critical {
				criticalFailed = true
			} else {
				nonCriticalFailed = true
			}
		}
	}

	// Determine overall health status
	if criticalFailed {
		return "unhealthy"
	} else if nonCriticalFailed {
		return "degraded"
	} else {
		return "healthy"
	}
}

// Start a specific scenario natively with health checking
func startScenarioNative(name string) error {
	// Check if already running using our PID system
	processesDir := filepath.Join(os.Getenv("HOME"), ".vrooli/processes/scenarios", name)
	if _, err := os.Stat(processesDir); !os.IsNotExist(err) {
		// Check if any processes are actually running
		files, _ := filepath.Glob(filepath.Join(processesDir, "*.json"))
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			var processInfo ProcessInfo
			if err := json.Unmarshal(data, &processInfo); err != nil {
				continue
			}

			if isPidRunning(processInfo.PID) {
				return fmt.Errorf("scenario %s is already running (PID: %d)", name, processInfo.PID)
			}
		}
	}

	// Check fork bomb
	if err := checkForkBomb(); err != nil {
		return err
	}

	// Load health configuration
	healthConfig, err := loadHealthConfig(name)
	if err != nil {
		// Health config is optional, continue without it
		healthConfig = nil
	}

	// Execute scenario lifecycle
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", name)
	cmd := exec.Command("bash",
		filepath.Join(vrooliRoot, "scripts/lib/utils/lifecycle.sh"),
		name, "develop")
	cmd.Dir = scenarioDir
	cmd.Env = os.Environ()

	// Start the process
	if err := cmd.Start(); err != nil {
		return err
	}

	// CRITICAL: Wait for process in background to prevent zombies
	// The lifecycle.sh script starts background processes and exits,
	// we must reap it when it finishes to prevent it becoming a zombie
	go func() {
		if err := cmd.Wait(); err != nil {
			// Log but don't fail - lifecycle script exit is expected
			log.Printf("Scenario %s lifecycle process exited: %v", name, err)
		} else {
			log.Printf("Scenario %s lifecycle process completed successfully", name)
		}
	}()

	// Wait for health checks to pass (if configured)
	if err := waitForScenarioHealth(name, healthConfig); err != nil {
		// Health checks failed, but process might still be running
		// This is a soft failure - we report it but don't kill the process
		return fmt.Errorf("scenario started but health checks failed: %v", err)
	}

	return nil
}

// Stop a specific scenario
func stopScenarioNative(name string) error {
	// Use lifecycle stop command
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", name)
	cmd := exec.Command("bash",
		filepath.Join(vrooliRoot, "scripts/lib/utils/lifecycle.sh"),
		name, "stop")
	cmd.Dir = scenarioDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop scenario %s: %v - %s", name, err, string(output))
	}
	return nil
}

// Stop all running scenarios with concurrency support
func stopAllScenariosNative() (map[string]interface{}, error) {
	// First get all running scenarios
	runningScenarios, err := discoverRunningScenarios()
	if err != nil {
		return nil, fmt.Errorf("failed to discover running scenarios: %v", err)
	}

	if len(runningScenarios) == 0 {
		return map[string]interface{}{
			"stopped": []map[string]string{},
			"failed":  []map[string]string{},
			"message": "No running scenarios to stop",
		}, nil
	}

	type stopResult struct {
		Name    string
		Success bool
		Error   string
	}

	// Use channels for concurrent stops
	resultChan := make(chan stopResult, len(runningScenarios))
	var wg sync.WaitGroup
	concurrentLimit := 10 // Can stop more concurrently than start
	sem := make(chan struct{}, concurrentLimit)

	for _, scenario := range runningScenarios {
		if scenario.Status != "running" {
			continue
		}

		wg.Add(1)
		go func(scenarioName string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			err := stopScenarioNative(scenarioName)
			if err != nil {
				resultChan <- stopResult{
					Name:    scenarioName,
					Success: false,
					Error:   err.Error(),
				}
			} else {
				resultChan <- stopResult{
					Name:    scenarioName,
					Success: true,
					Error:   "",
				}
			}
		}(scenario.Name)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var stopped []map[string]string
	var failed []map[string]string

	for result := range resultChan {
		if result.Success {
			stopped = append(stopped, map[string]string{
				"name":    result.Name,
				"message": "Stopped successfully",
			})
		} else {
			failed = append(failed, map[string]string{
				"name":  result.Name,
				"error": result.Error,
			})
		}
	}

	return map[string]interface{}{
		"stopped": stopped,
		"failed":  failed,
		"message": fmt.Sprintf("Stopped %d scenarios, %d failed", len(stopped), len(failed)),
	}, nil
}

func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

// === App Management ===
type App struct {
	Name          string                 `json:"name"`
	Path          string                 `json:"path"`
	Protected     bool                   `json:"protected"`
	HasGit        bool                   `json:"has_git"`
	Customized    bool                   `json:"customized"`
	Modified      time.Time              `json:"modified"`
	RuntimeStatus string                 `json:"runtime_status,omitempty"` // From orchestrator
	Ports         map[string]interface{} `json:"ports,omitempty"`          // From orchestrator
	PID           *int                   `json:"pid,omitempty"`            // From orchestrator
}

var appsDir = getAppsDir()

func getAppsDir() string {
	// Now using scenarios directory directly
	return filepath.Join(vrooliRoot, "scenarios")
}

func isCustomized(path string) bool {
	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		return false
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	out, _ := cmd.Output()
	if len(out) > 0 {
		return true
	}
	cmd = exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = path
	out, _ = cmd.Output()
	count := strings.TrimSpace(string(out))
	return count != "0" && count != "1"
}

func isProtected(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".vrooli", ".protected"))
	return err == nil
}

func hasGit(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// App handlers
func listApps(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Cannot read apps directory"})
		return
	}

	apps := []App{}

	// Get runtime status from native scenario discovery
	scenarios, _ := discoverRunningScenarios()
	scenarioMap := make(map[string]interface{})
	for _, scenario := range scenarios {
		scenarioMap[scenario.Name] = scenario
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".backups" {
			continue
		}
		appPath := filepath.Join(appsDir, entry.Name())
		info, _ := entry.Info()

		app := App{
			Name:       entry.Name(),
			Path:       appPath,
			Protected:  isProtected(appPath),
			HasGit:     hasGit(appPath),
			Customized: isCustomized(appPath),
			Modified:   info.ModTime(),
		}

		// Enhance with scenario data if available
		if scenarioData, ok := scenarioMap[entry.Name()].(RunningScenario); ok {
			if scenarioData.Status == "running" {
				app.RuntimeStatus = "running"
				// Convert scenario ports to interface map
				ports := make(map[string]interface{})
				for k, v := range scenarioData.Ports {
					ports[k] = v
				}
				app.Ports = ports
				// We don't track individual PIDs in our simplified implementation
				app.PID = nil
			} else {
				app.RuntimeStatus = "stopped"
			}
		} else {
			app.RuntimeStatus = "stopped"
		}

		apps = append(apps, app)
	}
	json.NewEncoder(w).Encode(Response{Success: true, Data: apps})
}

func protectApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	appPath := filepath.Join(appsDir, name)

	if _, err := os.Stat(appPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "App not found"})
		return
	}

	protectDir := filepath.Join(appPath, ".vrooli")
	os.MkdirAll(protectDir, 0755)
	protectFile := filepath.Join(protectDir, ".protected")
	content := fmt.Sprintf("Protected on %s\n", time.Now().UTC().Format(time.RFC3339))
	os.WriteFile(protectFile, []byte(content), 0644)

	json.NewEncoder(w).Encode(Response{Success: true})
}

// Start an app via orchestrator or fallback to process manager
func startApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)

	// Validate scenario exists
	if _, err := os.Stat(scenarioPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}

	// Use lifecycle develop event directly - much simpler and more reliable
	// The develop phase includes auto-setup, so no need for separate setup step
	startCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s && %s/scripts/lib/utils/lifecycle.sh %s develop",
			scenarioPath, vrooliRoot, name))

	output, err := startCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to start scenario %s: %s", name, string(output)),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("Scenario %s started successfully", name),
			"output":  string(output),
		},
	})
}

// Stop an app via orchestrator or fallback to process manager
func stopApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	// Use lifecycle stop event directly - much simpler and more reliable
	stopCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s/scenarios/%s && %s/scripts/lib/utils/lifecycle.sh %s stop",
			vrooliRoot, name, vrooliRoot, name))

	output, err := stopCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to stop app %s: %s", name, string(output)),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("App %s stopped successfully", name),
			"output":  string(output),
		},
	})
}

// Restart an app
func restartApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)

	// Validate scenario exists
	if _, err := os.Stat(scenarioPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}

	// Stop first using lifecycle stop
	stopCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s && %s/scripts/lib/utils/lifecycle.sh %s stop",
			scenarioPath, vrooliRoot, name))

	stopOutput, _ := stopCmd.CombinedOutput() // Ignore stop errors - might not be running

	// Brief pause to ensure processes are fully stopped
	time.Sleep(2 * time.Second)

	// Start using lifecycle develop
	startCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s && %s/scripts/lib/utils/lifecycle.sh %s develop",
			scenarioPath, vrooliRoot, name))

	startOutput, err := startCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to restart scenario %s: %s", name, string(startOutput)),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message":      fmt.Sprintf("Scenario %s restarted successfully", name),
			"stop_output":  string(stopOutput),
			"start_output": string(startOutput),
		},
	})
}

// Get app logs
func getAppLogs(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "50"
	}

	// Get logs using process manager
	logsCmd := exec.Command("bash", "-c",
		fmt.Sprintf("source %s/scripts/lib/process-manager.sh && pm::logs 'vrooli.develop.%s' %s",
			vrooliRoot, name, lines))

	output, err := logsCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to get logs: %s", string(output)),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"logs": string(output),
		},
	})
}

// === Scenario Management ===
type Scenario struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Template    bool   `json:"template"`
}

// Scenario handlers
func listScenarios(w http.ResponseWriter, r *http.Request) {
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")

	// Read all directories in scenarios folder
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to read scenarios directory"})
		return
	}

	scenarios := []Scenario{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if .vrooli/service.json exists
		serviceFile := filepath.Join(scenariosDir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
			continue
		}

		// Read and parse service.json
		data, err := os.ReadFile(serviceFile)
		if err != nil {
			log.Printf("Failed to read service.json for %s: %v", entry.Name(), err)
			continue
		}

		var serviceConfig struct {
			Service struct {
				Name        string   `json:"name"`
				DisplayName string   `json:"displayName"`
				Description string   `json:"description"`
				Tags        []string `json:"tags"`
			} `json:"service"`
		}

		if err := json.Unmarshal(data, &serviceConfig); err != nil {
			log.Printf("Failed to parse service.json for %s: %v", entry.Name(), err)
			continue
		}

		// Determine category from tags if available
		category := ""
		for _, tag := range serviceConfig.Service.Tags {
			if tag == "vrooli-helpers" || tag == "saas-applications" || tag == "automation" || tag == "technical-reference" {
				category = tag
				break
			}
		}

		scenarios = append(scenarios, Scenario{
			Name:        serviceConfig.Service.Name,
			Location:    entry.Name(),
			Category:    category,
			Description: serviceConfig.Service.Description,
			Enabled:     true, // Default to enabled if it has a service.json
			Template:    false,
		})
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: scenarios})
}

func getScenarioStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	// Check if scenario exists
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	if _, err := os.Stat(scenarioPath); err == nil {
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: map[string]interface{}{
				"exists":  true,
				"message": "Scenario exists and ready to run",
				"path":    scenarioPath,
			},
		})
	} else {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Scenario not found",
		})
	}
}

// === Native Scenario API Endpoints (Replaces Python Orchestrator) ===

// List running scenarios with status and ports (replaces Python /scenarios)
func listScenariosNative(w http.ResponseWriter, r *http.Request) {
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")

	// Read all directories in scenarios folder
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to read scenarios directory"})
		return
	}

	// Get running scenarios for status info
	runningScenarios, _ := discoverRunningScenarios()
	runningMap := make(map[string]*RunningScenario)
	for i, scenario := range runningScenarios {
		runningMap[scenario.Name] = &runningScenarios[i]
	}

	// Get process health information for system warnings
	healthSnapshot := collectProcessHealthSnapshot()
	zombieCount := healthSnapshot.ZombieCount
	zombieStatus := healthSnapshot.ZombieStatus
	zombieEmoji := healthSnapshot.ZombieEmoji
	orphanCount := healthSnapshot.OrphanCount
	orphanStatus := healthSnapshot.OrphanStatus
	orphanEmoji := healthSnapshot.OrphanEmoji
	processStatus := healthSnapshot.OverallStatus

	scenarios := []map[string]interface{}{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if .vrooli/service.json exists
		serviceFile := filepath.Join(scenariosDir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
			continue
		}

		// Read and parse service.json
		data, err := os.ReadFile(serviceFile)
		if err != nil {
			log.Printf("Failed to read service.json for %s: %v", entry.Name(), err)
			continue
		}

		var serviceConfig struct {
			Service struct {
				Name        string   `json:"name"`
				DisplayName string   `json:"displayName"`
				Description string   `json:"description"`
				Tags        []string `json:"tags"`
			} `json:"service"`
		}

		if err := json.Unmarshal(data, &serviceConfig); err != nil {
			log.Printf("Failed to parse service.json for %s: %v", entry.Name(), err)
			continue
		}

		// Create scenario info
		scenario := map[string]interface{}{
			"name":         entry.Name(),
			"display_name": serviceConfig.Service.DisplayName,
			"description":  serviceConfig.Service.Description,
			"tags":         serviceConfig.Service.Tags,
			"status":       "stopped",
			"processes":    0,
			"ports":        map[string]int{},
			"runtime":      "N/A",
		}

		// Update with running info if available
		if running, exists := runningMap[entry.Name()]; exists {
			scenario["status"] = running.Status
			scenario["processes"] = running.Processes
			scenario["ports"] = running.Ports
			scenario["runtime"] = running.Runtime
			if running.StartedAt != nil {
				scenario["started_at"] = running.StartedAt.Format(time.RFC3339)
			}

			// Perform health checks for running scenarios using discovered ports
			healthStatus := "running" // Default if no health checks configured
			if healthConfig, err := loadHealthConfig(entry.Name()); err == nil && healthConfig != nil && len(healthConfig.Checks) > 0 {
				// Use the discovered ports from the running scenario
				healthStatus = checkScenarioHealth(entry.Name(), healthConfig, running.Ports)
			}
			scenario["health_status"] = healthStatus
		} else {
			// Not running, no health status
			scenario["health_status"] = nil
		}

		scenarios = append(scenarios, scenario)
	}

	// Build response with optional system warnings
	response := map[string]interface{}{
		"success": true,
		"data":    scenarios,
	}

	// Add system warnings if necessary
	var warnings []map[string]interface{}

	if zombieStatus != "healthy" && zombieStatus != "normal" {
		warnings = append(warnings, map[string]interface{}{
			"type":    "zombies",
			"count":   zombieCount,
			"status":  zombieStatus,
			"emoji":   zombieEmoji,
			"message": fmt.Sprintf("System has %d zombie processes %s", zombieCount, zombieEmoji),
		})
	}

	if orphanStatus != "healthy" && orphanStatus != "normal" {
		warnings = append(warnings, map[string]interface{}{
			"type":    "orphans",
			"count":   orphanCount,
			"status":  orphanStatus,
			"emoji":   orphanEmoji,
			"message": fmt.Sprintf("System has %d orphaned processes %s", orphanCount, orphanEmoji),
		})
	}

	if len(warnings) > 0 {
		response["system_warnings"] = warnings
		response["system_health"] = processStatus
	}

	json.NewEncoder(w).Encode(response)
}

// Get detailed scenario status with ports (replaces Python /scenarios/{name})
func getScenarioStatusNative(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	// Check if scenario exists first
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   fmt.Sprintf("Scenario '%s' not found", name),
		})
		return
	}

	// Get running status and ports
	scenarios, err := discoverRunningScenarios()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Find specific scenario in running scenarios
	for _, scenario := range scenarios {
		if scenario.Name == name {
			// Format detailed response like Python orchestrator
			processes := []map[string]interface{}{}
			for i := 0; i < scenario.Processes; i++ {
				processes = append(processes, map[string]interface{}{
					"step_name": fmt.Sprintf("process-%d", i+1),
					"pid":       0, // We don't track individual PIDs
					"status":    "running",
					"ports":     scenario.Ports,
				})
			}

			json.NewEncoder(w).Encode(Response{
				Success: true,
				Data: map[string]interface{}{
					"name":            scenario.Name,
					"status":          scenario.Status,
					"phase":           "develop",
					"processes":       processes,
					"started_at":      scenario.StartedAt,
					"runtime":         scenario.Runtime,
					"allocated_ports": scenario.Ports,
				},
			})
			return
		}
	}

	// Scenario exists but not running - return stopped status like Python orchestrator
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"name":            name,
			"status":          "stopped",
			"phase":           "develop",
			"processes":       []map[string]interface{}{},
			"started_at":      nil,
			"runtime":         "N/A",
			"allocated_ports": map[string]int{},
		},
	})
}

// Start all scenarios (replaces Python /scenarios/start-all)
func startAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	result, err := startAllScenariosNative()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    result,
	})
}

// === Resource Management (placeholder) ===
func listResources(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement resource listing
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": "Resource API coming soon",
		},
	})
}

// === Lifecycle Management (placeholder) ===
func handleLifecycle(w http.ResponseWriter, r *http.Request) {
	action := mux.Vars(r)["action"]
	// TODO: Implement lifecycle actions
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"action":  action,
			"message": "Lifecycle API coming soon",
		},
	})
}

// === Process Metrics ===
func processMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := getEnhancedProcessMetrics()

	// Add status interpretations
	metrics["status"] = "healthy"
	if zombies, ok := metrics["zombie_processes"].(int); ok && zombies > 5 {
		metrics["status"] = "warning"
	}
	if orphans, ok := metrics["orphan_processes"].(int); ok && orphans > 3 {
		metrics["status"] = "warning"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    metrics,
	})
}

// === Health Check ===
func healthCheck(w http.ResponseWriter, r *http.Request) {
	// Get process health information in a single pass
	healthSnapshot := collectProcessHealthSnapshot()
	zombieCount := healthSnapshot.ZombieCount
	zombieStatus := healthSnapshot.ZombieStatus
	orphanCount := healthSnapshot.OrphanCount
	orphanStatus := healthSnapshot.OrphanStatus
	processStatus := healthSnapshot.OverallStatus

	// Determine overall health based on process status
	overallStatus := "healthy"
	switch processStatus {
	case "critical":
		overallStatus = "unhealthy"
	case "warning":
		overallStatus = "degraded"
	case "unknown":
		overallStatus = "degraded"
	}

	// Set appropriate HTTP status code
	if overallStatus != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      overallStatus,
		"version":     "1.0.0",
		"vrooli_root": vrooliRoot,
		"apps_dir":    appsDir,
		"system": map[string]interface{}{
			"zombie_processes": zombieCount,
			"zombie_status":    zombieStatus,
			"orphan_processes": orphanCount,
			"orphan_status":    orphanStatus,
			"process_health":   processStatus,
		},
		"apis": map[string]bool{
			"apps":      true,
			"scenarios": true,
			"resources": false, // Coming soon
			"lifecycle": false, // Coming soon
		},
	})
}

// === Orchestrator-Specific Endpoints ===

// Get running apps using native scenario discovery
func getRunningApps(w http.ResponseWriter, r *http.Request) {
	scenarios, err := discoverRunningScenarios()
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to get running scenarios"})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: scenarios})
}

// Start all enabled apps using native scenario management
func startAllApps(w http.ResponseWriter, r *http.Request) {
	result, err := startAllScenariosNative()
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to start scenarios: %v", err)})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

// Stop all running apps using native stop implementation
func stopAllApps(w http.ResponseWriter, r *http.Request) {
	result, err := stopAllScenariosNative()
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to stop scenarios: %v", err)})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

// Stop all scenarios endpoint
func stopAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	result, err := stopAllScenariosNative()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    result,
	})
}

// Stop a specific scenario endpoint
func stopScenarioEndpoint(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	err := stopScenarioNative(name)
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("Scenario %s stopped successfully", name),
		},
	})
}

// Get detailed app status using native scenario discovery
func getDetailedAppStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	scenarios, err := discoverRunningScenarios()
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to get scenario status"})
		return
	}

	// Find the specific scenario
	for _, scenario := range scenarios {
		if scenario.Name == name {
			json.NewEncoder(w).Encode(Response{Success: true, Data: scenario})
			return
		}
	}

	// Scenario not running
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"name":      name,
			"status":    "stopped",
			"processes": 0,
			"ports":     map[string]int{},
		},
	})
}

func main() {
	port := os.Getenv("VROOLI_API_PORT")
	if port == "" {
		port = "8092"
	}

	// No longer starting orchestrator - using native Go scenario management
	log.Println("🎛️  Using native Go scenario management")

	// Handle shutdown signals
	go func() {
		// Wait for interrupt signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		<-ch

		log.Println("🛑 Shutting down gracefully...")
		os.Exit(0)
	}()

	r := mux.NewRouter()

	// Health
	r.HandleFunc("/health", healthCheck).Methods("GET")
	r.HandleFunc("/metrics/processes", processMetricsHandler).Methods("GET")

	// Apps API - Enhanced with orchestrator
	r.HandleFunc("/apps", listApps).Methods("GET")
	r.HandleFunc("/apps/running", getRunningApps).Methods("GET")
	r.HandleFunc("/apps/start-all", startAllApps).Methods("POST")
	r.HandleFunc("/apps/stop-all", stopAllApps).Methods("POST")
	r.HandleFunc("/apps/{name}/protect", protectApp).Methods("POST")
	r.HandleFunc("/apps/{name}/start", startApp).Methods("POST")
	r.HandleFunc("/apps/{name}/stop", stopApp).Methods("POST")
	r.HandleFunc("/apps/{name}/restart", restartApp).Methods("POST")
	r.HandleFunc("/apps/{name}/logs", getAppLogs).Methods("GET")
	r.HandleFunc("/apps/{name}/status", getDetailedAppStatus).Methods("GET")

	// Scenarios API - Native Go Implementation (replaces Python orchestrator)
	r.HandleFunc("/scenarios", listScenariosNative).Methods("GET")
	r.HandleFunc("/scenarios/{name}/status", getScenarioStatusNative).Methods("GET")
	r.HandleFunc("/scenarios/{name}/start", startApp).Methods("POST") // Reuse startApp function
	r.HandleFunc("/scenarios/{name}/stop", stopScenarioEndpoint).Methods("POST")
	r.HandleFunc("/scenarios/start-all", startAllScenariosEndpoint).Methods("POST")
	r.HandleFunc("/scenarios/stop-all", stopAllScenariosEndpoint).Methods("POST")

	// Resources API (placeholder)
	r.HandleFunc("/resources", listResources).Methods("GET")

	// Lifecycle API (placeholder)
	r.HandleFunc("/lifecycle/{action}", handleLifecycle).Methods("POST")

	log.Printf("🚀 Vrooli Unified API running on port %s", port)
	log.Printf("   Scenario Management: ✅ native Go implementation")
	log.Fatal(http.ListenAndServe(":"+port, r))
}
