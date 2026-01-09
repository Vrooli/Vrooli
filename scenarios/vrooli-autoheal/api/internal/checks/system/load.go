// Package system provides system-level health checks
// [REQ:SYSTEM-LOAD-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// LoadInfo contains system load average information
type LoadInfo struct {
	Load1           float64 `json:"load1"`           // 1-minute load average
	Load5           float64 `json:"load5"`           // 5-minute load average
	Load15          float64 `json:"load15"`          // 15-minute load average
	RunningProcs    int     `json:"runningProcs"`    // Currently running processes
	TotalProcs      int     `json:"totalProcs"`      // Total processes
	LastPID         int     `json:"lastPid"`         // Last assigned PID
	NumCPUs         int     `json:"numCpus"`         // Number of CPU cores
	NormalizedLoad1 float64 `json:"normalizedLoad1"` // Load1 per CPU
	NormalizedLoad5 float64 `json:"normalizedLoad5"` // Load5 per CPU
}

// LoadReader abstracts load average reading for testability.
// [REQ:TEST-SEAM-001]
type LoadReader interface {
	ReadLoadAvg() (*LoadInfo, error)
}

// RealLoadReader is the production implementation of LoadReader.
type RealLoadReader struct{}

// ReadLoadAvg reads load averages from /proc/loadavg.
func (r *RealLoadReader) ReadLoadAvg() (*LoadInfo, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read /proc/loadavg")
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return nil, fmt.Errorf("unexpected /proc/loadavg format: %s", line)
	}

	info := &LoadInfo{
		NumCPUs: runtime.NumCPU(),
	}

	// Parse load averages
	if v, err := strconv.ParseFloat(fields[0], 64); err == nil {
		info.Load1 = v
	}
	if v, err := strconv.ParseFloat(fields[1], 64); err == nil {
		info.Load5 = v
	}
	if v, err := strconv.ParseFloat(fields[2], 64); err == nil {
		info.Load15 = v
	}

	// Parse running/total processes (format: "running/total")
	procParts := strings.Split(fields[3], "/")
	if len(procParts) == 2 {
		if v, err := strconv.Atoi(procParts[0]); err == nil {
			info.RunningProcs = v
		}
		if v, err := strconv.Atoi(procParts[1]); err == nil {
			info.TotalProcs = v
		}
	}

	// Parse last PID
	if v, err := strconv.Atoi(fields[4]); err == nil {
		info.LastPID = v
	}

	// Calculate normalized load (per CPU)
	if info.NumCPUs > 0 {
		info.NormalizedLoad1 = info.Load1 / float64(info.NumCPUs)
		info.NormalizedLoad5 = info.Load5 / float64(info.NumCPUs)
	}

	return info, nil
}

// DefaultLoadReader is the global load reader used when none is injected.
var DefaultLoadReader LoadReader = &RealLoadReader{}

// LoadCheck monitors system load average.
// Load average indicates the number of processes waiting for CPU time.
// A load of 1.0 per CPU core means 100% utilization.
type LoadCheck struct {
	warningThreshold  float64 // Normalized load (per CPU) to warn
	criticalThreshold float64 // Normalized load (per CPU) to go critical
	loadReader        LoadReader
	executor          checks.CommandExecutor
}

// LoadCheckOption configures a LoadCheck.
type LoadCheckOption func(*LoadCheck)

// WithLoadThresholds sets warning and critical thresholds for normalized load.
// For example, 0.8 means 80% of CPUs busy, 2.0 means 2x overloaded.
func WithLoadThresholds(warning, critical float64) LoadCheckOption {
	return func(c *LoadCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithLoadReader sets the load reader (for testing).
// [REQ:TEST-SEAM-001]
func WithLoadReader(reader LoadReader) LoadCheckOption {
	return func(c *LoadCheck) {
		c.loadReader = reader
	}
}

// WithLoadExecutor sets the command executor (for recovery actions).
// [REQ:TEST-SEAM-001]
func WithLoadExecutor(executor checks.CommandExecutor) LoadCheckOption {
	return func(c *LoadCheck) {
		c.executor = executor
	}
}

// NewLoadCheck creates a system load check.
// Default thresholds: warning at 1.5x CPUs, critical at 3.0x CPUs
func NewLoadCheck(opts ...LoadCheckOption) *LoadCheck {
	c := &LoadCheck{
		warningThreshold:  1.5, // Load per CPU
		criticalThreshold: 3.0, // Load per CPU
		loadReader:        DefaultLoadReader,
		executor:          checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *LoadCheck) ID() string    { return "system-load" }
func (c *LoadCheck) Title() string { return "System Load Average" }
func (c *LoadCheck) Description() string {
	return "Monitors CPU load average to detect system overload conditions"
}
func (c *LoadCheck) Importance() string {
	return "High load indicates CPU contention - processes wait longer, response times increase"
}
func (c *LoadCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *LoadCheck) IntervalSeconds() int       { return 30 } // Check every 30 seconds
func (c *LoadCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *LoadCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Read load average
	loadInfo, err := c.loadReader.ReadLoadAvg()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read system load"
		result.Details["error"] = err.Error()
		return result
	}

	// Store all values
	result.Details["load1"] = loadInfo.Load1
	result.Details["load5"] = loadInfo.Load5
	result.Details["load15"] = loadInfo.Load15
	result.Details["runningProcesses"] = loadInfo.RunningProcs
	result.Details["totalProcesses"] = loadInfo.TotalProcs
	result.Details["cpuCores"] = loadInfo.NumCPUs
	result.Details["normalizedLoad1"] = loadInfo.NormalizedLoad1
	result.Details["normalizedLoad5"] = loadInfo.NormalizedLoad5
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Use 5-minute average for status determination (more stable than 1-min)
	normalizedLoad := loadInfo.NormalizedLoad5

	// Calculate score (inverse of normalized load, capped at 0-100)
	score := int(100 - (normalizedLoad * 50)) // 0 load = 100, 2x load = 0
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "load-1min",
				Passed: loadInfo.NormalizedLoad1 < c.criticalThreshold,
				Detail: fmt.Sprintf("%.2f (%.2f per CPU)", loadInfo.Load1, loadInfo.NormalizedLoad1),
			},
			{
				Name:   "load-5min",
				Passed: loadInfo.NormalizedLoad5 < c.criticalThreshold,
				Detail: fmt.Sprintf("%.2f (%.2f per CPU)", loadInfo.Load5, loadInfo.NormalizedLoad5),
			},
			{
				Name:   "load-15min",
				Passed: true, // 15-min is informational only
				Detail: fmt.Sprintf("%.2f", loadInfo.Load15),
			},
		},
	}

	// Determine status based on normalized load
	switch {
	case normalizedLoad >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("System overloaded: load %.2f (%.1fx CPUs) - processes severely delayed",
			loadInfo.Load5, normalizedLoad)
	case normalizedLoad >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("System load elevated: load %.2f (%.1fx CPUs) - some process delays expected",
			loadInfo.Load5, normalizedLoad)
	default:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("System load healthy: %.2f (%.1fx CPUs)",
			loadInfo.Load5, normalizedLoad)
	}

	return result
}

// RecoveryActions returns available recovery actions for load issues.
// [REQ:HEAL-ACTION-001]
func (c *LoadCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isElevated := lastResult != nil && (lastResult.Status == checks.StatusWarning || lastResult.Status == checks.StatusCritical)

	return []checks.RecoveryAction{
		{
			ID:          "top-cpu",
			Name:        "Top CPU Users",
			Description: "Show processes consuming the most CPU time",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "load-history",
			Name:        "Load History",
			Description: "Show recent load average history",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "process-tree",
			Name:        "Process Tree",
			Description: "Show process hierarchy to identify load sources",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "runaway-check",
			Name:        "Check Runaway",
			Description: "Identify potential runaway processes (high CPU, long runtime)",
			Dangerous:   false,
			Available:   isElevated,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *LoadCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "top-cpu":
		return c.executeTopCPU(ctx, start)

	case "load-history":
		return c.executeLoadHistory(ctx, start)

	case "process-tree":
		return c.executeProcessTree(ctx, start)

	case "runaway-check":
		return c.executeRunawayCheck(ctx, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeTopCPU shows top CPU-consuming processes
func (c *LoadCheck) executeTopCPU(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "top-cpu",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Top 20 CPU Users ===\n\n")

	// Use ps to get CPU usage sorted by %CPU
	output, err := c.executor.Output(ctx, "ps", "aux", "--sort=-%cpu")
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to list processes"
		return result
	}

	// Take header and top 20 processes
	lines := strings.Split(string(output), "\n")
	var topLines []string
	if len(lines) > 0 {
		topLines = append(topLines, lines[0]) // Header
	}
	for i := 1; i < len(lines) && i <= 20; i++ {
		topLines = append(topLines, lines[i])
	}
	outputBuilder.WriteString(strings.Join(topLines, "\n"))

	// Add current load info
	outputBuilder.WriteString("\n\n=== Current Load ===\n")
	loadOutput, _ := c.executor.Output(ctx, "cat", "/proc/loadavg")
	outputBuilder.Write(loadOutput)

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Listed top 20 CPU-consuming processes"
	return result
}

// executeLoadHistory shows load history
func (c *LoadCheck) executeLoadHistory(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "load-history",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Current Load Average ===\n")

	// Current load
	loadOutput, _ := c.executor.Output(ctx, "cat", "/proc/loadavg")
	outputBuilder.Write(loadOutput)
	outputBuilder.WriteString("\n")

	// Uptime shows load averages with context
	outputBuilder.WriteString("\n=== System Uptime ===\n")
	uptimeOutput, _ := c.executor.Output(ctx, "uptime")
	outputBuilder.Write(uptimeOutput)
	outputBuilder.WriteString("\n")

	// Try to get sar data if available
	outputBuilder.WriteString("\n=== Load History (sar) ===\n")
	sarOutput, err := c.executor.CombinedOutput(ctx, "sar", "-q", "-1")
	if err != nil {
		outputBuilder.WriteString("(sar not available - install sysstat for historical data)\n")
	} else {
		outputBuilder.Write(sarOutput)
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Load history retrieved"
	return result
}

// executeProcessTree shows process hierarchy
func (c *LoadCheck) executeProcessTree(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "process-tree",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Process Tree (Top Level) ===\n\n")

	// Try pstree first
	output, err := c.executor.CombinedOutput(ctx, "pstree", "-p", "-a")
	if err != nil {
		// Fall back to ps forest format
		output, err = c.executor.CombinedOutput(ctx, "ps", "auxf")
		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to get process tree"
			return result
		}
	}

	// Limit output to first 100 lines
	lines := strings.Split(string(output), "\n")
	if len(lines) > 100 {
		lines = lines[:100]
		lines = append(lines, "... (truncated)")
	}
	outputBuilder.WriteString(strings.Join(lines, "\n"))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Process tree retrieved"
	return result
}

// executeRunawayCheck identifies potential runaway processes
func (c *LoadCheck) executeRunawayCheck(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "runaway-check",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Potential Runaway Processes ===\n\n")
	outputBuilder.WriteString("Criteria: >50% CPU for >5 minutes, or D-state (uninterruptible sleep)\n\n")

	// Find high-CPU processes
	outputBuilder.WriteString("=== High CPU (>50%) ===\n")
	output, err := c.executor.CombinedOutput(ctx, "bash", "-c",
		`ps aux --sort=-%cpu | awk 'NR==1 || $3>50 {print}'`)
	if err == nil {
		outputBuilder.Write(output)
	} else {
		outputBuilder.WriteString("(failed to get high CPU processes)\n")
	}

	// Find D-state processes (stuck in uninterruptible sleep)
	outputBuilder.WriteString("\n=== D-State Processes (Stuck I/O) ===\n")
	output, err = c.executor.CombinedOutput(ctx, "bash", "-c",
		`ps aux | awk 'NR==1 || $8~/D/ {print}'`)
	if err == nil {
		outputBuilder.Write(output)
	} else {
		outputBuilder.WriteString("(failed to get D-state processes)\n")
	}

	// Find long-running processes
	outputBuilder.WriteString("\n=== Long-Running (>1 hour CPU time) ===\n")
	output, err = c.executor.CombinedOutput(ctx, "bash", "-c",
		`ps aux --sort=-time | head -15`)
	if err == nil {
		outputBuilder.Write(output)
	} else {
		outputBuilder.WriteString("(failed to get long-running processes)\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Runaway check completed"
	return result
}

// Ensure LoadCheck implements HealableCheck
var _ checks.HealableCheck = (*LoadCheck)(nil)
