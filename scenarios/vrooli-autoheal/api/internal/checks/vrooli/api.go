// Package vrooli provides Vrooli-specific health checks
// [REQ:VROOLI-API-001] [REQ:HEAL-ACTION-001]
package vrooli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// APICheck monitors the main Vrooli API health endpoint.
type APICheck struct {
	url     string
	timeout time.Duration
}

// APICheckOption configures an APICheck.
type APICheckOption func(*APICheck)

// WithAPIURL sets the API health endpoint URL.
func WithAPIURL(url string) APICheckOption {
	return func(c *APICheck) {
		c.url = url
	}
}

// WithAPITimeout sets the HTTP request timeout.
func WithAPITimeout(timeout time.Duration) APICheckOption {
	return func(c *APICheck) {
		c.timeout = timeout
	}
}

// NewAPICheck creates a Vrooli API health check.
// Default URL: http://127.0.0.1:8092/health
// Default timeout: 5 seconds
func NewAPICheck(opts ...APICheckOption) *APICheck {
	c := &APICheck{
		url:     "http://127.0.0.1:8092/health",
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *APICheck) ID() string          { return "vrooli-api" }
func (c *APICheck) Title() string       { return "Vrooli API" }
func (c *APICheck) Description() string { return "Checks the main Vrooli API health endpoint" }
func (c *APICheck) Importance() string {
	return "The Vrooli API is the central orchestration layer for all scenarios and resources"
}
func (c *APICheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *APICheck) IntervalSeconds() int       { return 60 }
func (c *APICheck) Platforms() []platform.Type { return nil } // all platforms

func (c *APICheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{
			"url":     c.url,
			"timeout": c.timeout.String(),
		},
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: c.timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to create request"
		result.Details["error"] = err.Error()
		return result
	}

	// Execute request
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	result.Details["responseTimeMs"] = elapsed.Milliseconds()

	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Vrooli API is not responding"
		result.Details["error"] = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.Details["statusCode"] = resp.StatusCode

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read API response"
		result.Details["error"] = err.Error()
		return result
	}

	// Parse JSON response
	var healthResponse struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(body, &healthResponse); err != nil {
		// If not JSON, check HTTP status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Status = checks.StatusOK
			result.Message = "Vrooli API is responding (non-JSON response)"
			return result
		}
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Vrooli API returned HTTP %d", resp.StatusCode)
		return result
	}

	result.Details["apiStatus"] = healthResponse.Status

	// Calculate health score based on response time
	score := 100
	if elapsed > 3*time.Second {
		score = 50
	} else if elapsed > 1*time.Second {
		score = 75
	}

	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "api-response",
				Passed: healthResponse.Status == "ok" || healthResponse.Status == "healthy",
				Detail: fmt.Sprintf("Status: %s, Response time: %dms", healthResponse.Status, elapsed.Milliseconds()),
			},
		},
	}

	// Interpret health status
	switch healthResponse.Status {
	case "ok", "healthy":
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Vrooli API healthy (response time: %dms)", elapsed.Milliseconds())
	case "degraded":
		result.Status = checks.StatusWarning
		result.Message = "Vrooli API is degraded"
		if healthResponse.Message != "" {
			result.Details["degradedReason"] = healthResponse.Message
		}
	case "unhealthy", "failed", "error":
		result.Status = checks.StatusCritical
		result.Message = "Vrooli API reports unhealthy status"
		if healthResponse.Message != "" {
			result.Details["errorMessage"] = healthResponse.Message
		}
	default:
		// Unknown status - assume OK if HTTP 2xx
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Status = checks.StatusOK
			result.Message = fmt.Sprintf("Vrooli API responding (status: %s)", healthResponse.Status)
		} else {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("Vrooli API returned unexpected status: %s", healthResponse.Status)
		}
	}

	return result
}

// RecoveryActions returns available recovery actions for the Vrooli API check
// [REQ:HEAL-ACTION-001]
func (c *APICheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isResponding := false
	if lastResult != nil && lastResult.Status == checks.StatusOK {
		isResponding = true
	}

	return []checks.RecoveryAction{
		{
			ID:          "restart",
			Name:        "Restart API",
			Description: "Stop and restart the Vrooli API server",
			Dangerous:   true, // Brief downtime while restarting
			Available:   true,
		},
		{
			ID:          "kill-port",
			Name:        "Kill Port Process",
			Description: "Kill any process holding the API port (8092)",
			Dangerous:   true,
			Available:   !isResponding, // Only if not responding
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent Vrooli API logs",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "diagnose",
			Name:        "Diagnose",
			Description: "Get diagnostic information about the Vrooli API",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action for the Vrooli API
// [REQ:HEAL-ACTION-001]
func (c *APICheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "restart":
		return c.executeRestart(ctx, start)
	case "kill-port":
		return c.executeKillPort(ctx, start)
	case "logs":
		return c.executeLogs(ctx, start)
	case "diagnose":
		return c.executeDiagnose(ctx, start)
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeRestart restarts the Vrooli API using the maintenance script or vrooli develop
func (c *APICheck) executeRestart(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "restart",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Try to find the Vrooli root directory
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		// Try common locations
		homeDir, _ := os.UserHomeDir()
		possiblePaths := []string{
			filepath.Join(homeDir, "Vrooli"),
			"/home/matthalloran8/Vrooli",
		}
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				vrooliRoot = path
				break
			}
		}
	}

	// Step 1: Try to stop any running vrooli processes on the API port
	outputBuilder.WriteString("=== Stopping API processes ===\n")
	port := "8092" // Default API port
	if strings.Contains(c.url, ":") {
		// Extract port from URL
		parts := strings.Split(c.url, ":")
		if len(parts) >= 3 {
			portPart := strings.Split(parts[2], "/")[0]
			port = portPart
		}
	}

	// Find and kill processes on the port
	lsofCmd := exec.CommandContext(ctx, "lsof", "-ti", ":"+port)
	pids, _ := lsofCmd.Output()
	if len(strings.TrimSpace(string(pids))) > 0 {
		for _, pid := range strings.Fields(strings.TrimSpace(string(pids))) {
			outputBuilder.WriteString(fmt.Sprintf("Killing PID %s on port %s\n", pid, port))
			killCmd := exec.CommandContext(ctx, "kill", "-9", pid)
			killCmd.Run()
		}
	} else {
		outputBuilder.WriteString("No processes found on port " + port + "\n")
	}

	// Wait a moment for processes to die
	time.Sleep(2 * time.Second)

	// Step 2: Try to restart using vrooli develop or the restart script
	outputBuilder.WriteString("\n=== Starting API ===\n")

	// Try vrooli develop first
	startCmd := exec.CommandContext(ctx, "vrooli", "develop")
	if vrooliRoot != "" {
		startCmd.Dir = vrooliRoot
	}
	startOutput, err := startCmd.CombinedOutput()
	outputBuilder.Write(startOutput)

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()

	if err != nil {
		// Try alternative: restart script
		if vrooliRoot != "" {
			restartScript := filepath.Join(vrooliRoot, "scripts", "maintenance", "restart-vrooli-api.sh")
			if _, statErr := os.Stat(restartScript); statErr == nil {
				outputBuilder.WriteString("\n=== Trying restart script ===\n")
				scriptCmd := exec.CommandContext(ctx, "bash", restartScript)
				scriptOutput, scriptErr := scriptCmd.CombinedOutput()
				outputBuilder.Write(scriptOutput)
				result.Output = outputBuilder.String()

				if scriptErr == nil {
					result.Success = true
					result.Message = "Vrooli API restarted successfully via script"
					return result
				}
			}
		}

		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to restart Vrooli API"
		return result
	}

	result.Success = true
	result.Message = "Vrooli API restart initiated"
	return result
}

// executeKillPort kills processes holding the API port
func (c *APICheck) executeKillPort(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "kill-port",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	port := "8092" // Default API port

	// Extract port from URL
	if strings.Contains(c.url, ":") {
		parts := strings.Split(c.url, ":")
		if len(parts) >= 3 {
			portPart := strings.Split(parts[2], "/")[0]
			port = portPart
		}
	}

	outputBuilder.WriteString(fmt.Sprintf("Looking for processes on port %s...\n", port))

	// Find processes on the port
	lsofCmd := exec.CommandContext(ctx, "lsof", "-ti", ":"+port)
	pids, err := lsofCmd.Output()

	if err != nil || len(strings.TrimSpace(string(pids))) == 0 {
		result.Duration = time.Since(start)
		result.Output = outputBuilder.String() + "No processes found on port " + port
		result.Success = true
		result.Message = "No processes found on port " + port
		return result
	}

	killedCount := 0
	for _, pid := range strings.Fields(strings.TrimSpace(string(pids))) {
		outputBuilder.WriteString(fmt.Sprintf("Killing PID %s... ", pid))

		// Try SIGTERM first
		killCmd := exec.CommandContext(ctx, "kill", pid)
		if err := killCmd.Run(); err != nil {
			// Try SIGKILL
			killCmd = exec.CommandContext(ctx, "kill", "-9", pid)
			if err := killCmd.Run(); err != nil {
				outputBuilder.WriteString(fmt.Sprintf("FAILED: %v\n", err))
				continue
			}
		}
		outputBuilder.WriteString("OK\n")
		killedCount++
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = fmt.Sprintf("Killed %d processes on port %s", killedCount, port)
	return result
}

// executeLogs retrieves Vrooli API logs
func (c *APICheck) executeLogs(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "logs",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Try to find Vrooli logs
	homeDir, _ := os.UserHomeDir()
	logPaths := []string{
		filepath.Join(homeDir, ".vrooli", "logs", "vrooli.log"),
		"/var/log/vrooli.log",
		filepath.Join(homeDir, ".vrooli", "logs", "api.log"),
	}

	foundLogs := false
	for _, logPath := range logPaths {
		if _, err := os.Stat(logPath); err == nil {
			outputBuilder.WriteString(fmt.Sprintf("=== %s (last 100 lines) ===\n", logPath))
			tailCmd := exec.CommandContext(ctx, "tail", "-100", logPath)
			tailOutput, _ := tailCmd.Output()
			outputBuilder.Write(tailOutput)
			outputBuilder.WriteString("\n\n")
			foundLogs = true
		}
	}

	// Also check journalctl for vrooli entries
	outputBuilder.WriteString("=== System logs (journalctl) ===\n")
	journalCmd := exec.CommandContext(ctx, "journalctl", "--no-pager", "-n", "50", "-g", "vrooli")
	journalOutput, _ := journalCmd.CombinedOutput()
	outputBuilder.Write(journalOutput)

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	if foundLogs {
		result.Message = "Retrieved Vrooli API logs"
	} else {
		result.Message = "Retrieved system logs (no Vrooli log files found)"
	}
	return result
}

// executeDiagnose gathers diagnostic information
func (c *APICheck) executeDiagnose(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "diagnose",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Health endpoint
	outputBuilder.WriteString("=== Health Endpoint ===\n")
	outputBuilder.WriteString(fmt.Sprintf("URL: %s\n", c.url))
	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(c.url)
	if err != nil {
		outputBuilder.WriteString(fmt.Sprintf("Error: %v\n", err))
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		outputBuilder.WriteString(fmt.Sprintf("Status: %d\n", resp.StatusCode))
		outputBuilder.WriteString(fmt.Sprintf("Response: %s\n", string(body)))
	}
	outputBuilder.WriteString("\n")

	// Port usage
	outputBuilder.WriteString("=== Port Usage ===\n")
	port := "8092"
	if strings.Contains(c.url, ":") {
		parts := strings.Split(c.url, ":")
		if len(parts) >= 3 {
			port = strings.Split(parts[2], "/")[0]
		}
	}
	lsofCmd := exec.CommandContext(ctx, "lsof", "-i", ":"+port)
	lsofOutput, _ := lsofCmd.CombinedOutput()
	outputBuilder.Write(lsofOutput)
	outputBuilder.WriteString("\n")

	// Vrooli processes
	outputBuilder.WriteString("=== Vrooli Processes ===\n")
	pgrepCmd := exec.CommandContext(ctx, "pgrep", "-af", "vrooli")
	pgrepOutput, _ := pgrepCmd.CombinedOutput()
	outputBuilder.Write(pgrepOutput)
	outputBuilder.WriteString("\n")

	// Resource status
	outputBuilder.WriteString("=== Resource Status ===\n")
	resourceCmd := exec.CommandContext(ctx, "vrooli", "resource", "status")
	resourceOutput, _ := resourceCmd.CombinedOutput()
	outputBuilder.Write(resourceOutput)

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Diagnostic information gathered"
	return result
}

// Ensure APICheck implements HealableCheck
var _ checks.HealableCheck = (*APICheck)(nil)
