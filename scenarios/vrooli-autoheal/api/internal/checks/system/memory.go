// Package system provides system-level health checks
// [REQ:SYSTEM-MEMORY-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// MemoryCheck monitors RAM usage and detects memory pressure.
// It uses MemAvailable when available (Linux 3.14+) for accurate available memory,
// falling back to MemFree + Buffers + Cached on older systems.
type MemoryCheck struct {
	warningThreshold  int // percentage used
	criticalThreshold int // percentage used
	procReader        checks.ProcReader
	executor          checks.CommandExecutor
}

// MemoryCheckOption configures a MemoryCheck.
type MemoryCheckOption func(*MemoryCheck)

// WithMemoryThresholds sets warning and critical thresholds (percentages of memory used).
func WithMemoryThresholds(warning, critical int) MemoryCheckOption {
	return func(c *MemoryCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithMemoryProcReader sets the proc reader (for testing).
// [REQ:TEST-SEAM-001]
func WithMemoryProcReader(reader checks.ProcReader) MemoryCheckOption {
	return func(c *MemoryCheck) {
		c.procReader = reader
	}
}

// WithMemoryExecutor sets the command executor (for testing and recovery actions).
// [REQ:TEST-SEAM-001]
func WithMemoryExecutor(executor checks.CommandExecutor) MemoryCheckOption {
	return func(c *MemoryCheck) {
		c.executor = executor
	}
}

// NewMemoryCheck creates a memory usage check.
// Default thresholds: warning at 80%, critical at 90%
func NewMemoryCheck(opts ...MemoryCheckOption) *MemoryCheck {
	c := &MemoryCheck{
		warningThreshold:  80,
		criticalThreshold: 90,
		procReader:        checks.DefaultProcReader,
		executor:          checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *MemoryCheck) ID() string    { return "system-memory" }
func (c *MemoryCheck) Title() string { return "Memory Usage" }
func (c *MemoryCheck) Description() string {
	return "Monitors RAM usage and detects memory pressure before OOM conditions"
}
func (c *MemoryCheck) Importance() string {
	return "High memory usage leads to swap thrashing, OOM kills, and service instability"
}
func (c *MemoryCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *MemoryCheck) IntervalSeconds() int       { return 60 } // Check every minute
func (c *MemoryCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *MemoryCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if runtime.GOOS == "windows" {
		result.Status = checks.StatusWarning
		result.Message = "Memory check not yet implemented for Windows"
		result.Details["platform"] = "windows"
		return result
	}

	// Read memory information via injected reader
	memInfo, err := c.procReader.ReadMeminfo()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read memory information"
		result.Details["error"] = err.Error()
		return result
	}

	memTotal := memInfo.MemTotal
	if memTotal == 0 {
		result.Status = checks.StatusCritical
		result.Message = "Invalid memory information: total is 0"
		return result
	}

	// Calculate available memory
	// Prefer MemAvailable (Linux 3.14+) as it's more accurate
	// Fall back to MemFree + Buffers + Cached for older systems
	var memAvailable uint64
	if memInfo.MemAvailable > 0 {
		memAvailable = memInfo.MemAvailable
		result.Details["availableSource"] = "MemAvailable"
	} else {
		memAvailable = memInfo.MemFree + memInfo.Buffers + memInfo.Cached
		result.Details["availableSource"] = "calculated (Free+Buffers+Cached)"
	}

	memUsed := memTotal - memAvailable
	usedPercent := int((memUsed * 100) / memTotal)

	// Store all values in KB and bytes for flexibility
	result.Details["memTotalKB"] = memTotal
	result.Details["memTotalBytes"] = memTotal * 1024
	result.Details["memFreeKB"] = memInfo.MemFree
	result.Details["memFreeBytes"] = memInfo.MemFree * 1024
	result.Details["memAvailableKB"] = memAvailable
	result.Details["memAvailableBytes"] = memAvailable * 1024
	result.Details["memUsedKB"] = memUsed
	result.Details["memUsedBytes"] = memUsed * 1024
	result.Details["buffersKB"] = memInfo.Buffers
	result.Details["cachedKB"] = memInfo.Cached
	result.Details["usedPercent"] = usedPercent
	result.Details["availablePercent"] = 100 - usedPercent
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Calculate score (inverse of usage)
	score := 100 - usedPercent
	if score < 0 {
		score = 0
	}

	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "memory-usage",
				Passed: usedPercent < c.criticalThreshold,
				Detail: fmt.Sprintf("%d%% used (%s available of %s)",
					usedPercent,
					formatBytes(memAvailable*1024),
					formatBytes(memTotal*1024)),
			},
		},
	}

	switch {
	case usedPercent >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Memory usage critical: %d%% used (%s available)",
			usedPercent, formatBytes(memAvailable*1024))
	case usedPercent >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Memory usage warning: %d%% used - system under memory pressure",
			usedPercent)
	default:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Memory usage healthy: %d%% used (%s available)",
			usedPercent, formatBytes(memAvailable*1024))
	}

	return result
}

// RecoveryActions returns available recovery actions for memory issues.
// Note: Memory issues often can't be "healed" automatically, so actions focus on
// diagnostics and identifying memory hogs.
// [REQ:HEAL-ACTION-001]
func (c *MemoryCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"
	isCritical := lastResult != nil && lastResult.Status == checks.StatusCritical

	return []checks.RecoveryAction{
		{
			ID:          "drop-caches",
			Name:        "Drop Caches",
			Description: "Drop filesystem caches to free memory (safe operation)",
			Dangerous:   false,
			Available:   isLinux && isCritical,
		},
		{
			ID:          "top-memory",
			Name:        "Top Memory Users",
			Description: "Show processes using the most memory",
			Dangerous:   false,
			Available:   isLinux,
		},
		{
			ID:          "memory-summary",
			Name:        "Memory Summary",
			Description: "Show detailed memory usage breakdown",
			Dangerous:   false,
			Available:   isLinux,
		},
		{
			ID:          "oom-candidates",
			Name:        "OOM Candidates",
			Description: "Show processes most likely to be killed by OOM killer",
			Dangerous:   false,
			Available:   isLinux,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *MemoryCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "drop-caches":
		return c.executeDropCaches(ctx, start)

	case "top-memory":
		return c.executeTopMemory(ctx, start)

	case "memory-summary":
		return c.executeMemorySummary(ctx, start)

	case "oom-candidates":
		return c.executeOOMCandidates(ctx, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeDropCaches drops filesystem caches to free memory
func (c *MemoryCheck) executeDropCaches(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "drop-caches",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Get memory before
	outputBuilder.WriteString("=== Memory Before ===\n")
	beforeOutput, _ := c.executor.Output(ctx, "free", "-h")
	outputBuilder.Write(beforeOutput)
	outputBuilder.WriteString("\n")

	// Sync filesystem to disk first
	outputBuilder.WriteString("=== Syncing Filesystem ===\n")
	if err := c.executor.Run(ctx, "sync"); err != nil {
		outputBuilder.WriteString(fmt.Sprintf("Warning: sync failed: %v\n", err))
	} else {
		outputBuilder.WriteString("Sync completed\n")
	}

	// Drop caches (3 = drop page cache, dentries, and inodes)
	outputBuilder.WriteString("\n=== Dropping Caches ===\n")
	output, err := c.executor.CombinedOutput(ctx, "sudo", "sh", "-c", "echo 3 > /proc/sys/vm/drop_caches")
	if err != nil {
		result.Duration = time.Since(start)
		result.Output = outputBuilder.String()
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to drop caches (may require sudo)"
		return result
	}
	outputBuilder.Write(output)
	outputBuilder.WriteString("Caches dropped successfully\n")

	// Get memory after
	outputBuilder.WriteString("\n=== Memory After ===\n")
	afterOutput, _ := c.executor.Output(ctx, "free", "-h")
	outputBuilder.Write(afterOutput)

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Filesystem caches dropped successfully"
	return result
}

// executeTopMemory shows the top memory-consuming processes
func (c *MemoryCheck) executeTopMemory(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "top-memory",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Top 20 Memory Users ===\n\n")

	// Use ps to get memory usage sorted by RSS
	output, err := c.executor.Output(ctx, "ps", "aux", "--sort=-rss")
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

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Listed top 20 memory-consuming processes"
	return result
}

// executeMemorySummary shows detailed memory breakdown
func (c *MemoryCheck) executeMemorySummary(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "memory-summary",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Overall memory with human-readable format
	outputBuilder.WriteString("=== Memory Overview ===\n")
	freeOutput, _ := c.executor.Output(ctx, "free", "-h")
	outputBuilder.Write(freeOutput)
	outputBuilder.WriteString("\n")

	// Detailed /proc/meminfo
	outputBuilder.WriteString("=== Detailed Memory Info ===\n")
	meminfoOutput, _ := c.executor.Output(ctx, "cat", "/proc/meminfo")
	outputBuilder.Write(meminfoOutput)
	outputBuilder.WriteString("\n")

	// Memory by type (slab, etc.)
	outputBuilder.WriteString("=== Slab Memory ===\n")
	slabOutput, _ := c.executor.Output(ctx, "cat", "/proc/slabinfo")
	// Only show header and top entries
	slabLines := strings.Split(string(slabOutput), "\n")
	if len(slabLines) > 20 {
		slabLines = slabLines[:20]
	}
	outputBuilder.WriteString(strings.Join(slabLines, "\n"))
	outputBuilder.WriteString("\n... (truncated)\n")

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Memory summary retrieved"
	return result
}

// executeOOMCandidates shows processes most likely to be killed by OOM
func (c *MemoryCheck) executeOOMCandidates(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "oom-candidates",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== OOM Kill Candidates ===\n")
	outputBuilder.WriteString("Higher oom_score = more likely to be killed\n\n")

	// Get all processes with their oom_score
	output, err := c.executor.CombinedOutput(ctx, "bash", "-c",
		`for pid in /proc/[0-9]*; do
			p=$(basename $pid)
			if [ -f "$pid/oom_score" ] && [ -f "$pid/cmdline" ]; then
				score=$(cat "$pid/oom_score" 2>/dev/null || echo 0)
				cmd=$(tr '\0' ' ' < "$pid/cmdline" 2>/dev/null | head -c 50)
				if [ -n "$cmd" ] && [ "$score" -gt 0 ]; then
					echo "$score $p $cmd"
				fi
			fi
		done | sort -rn | head -20`)

	if err != nil {
		// Fallback to simpler approach
		outputBuilder.WriteString("Using fallback method...\n\n")
		fallbackOutput, _ := c.executor.Output(ctx, "ps", "aux", "--sort=-rss")
		outputBuilder.Write(fallbackOutput)
	} else {
		outputBuilder.WriteString("OOM_SCORE  PID    COMMAND\n")
		outputBuilder.WriteString("---------- ------ --------------------------------------------------\n")
		for _, line := range strings.Split(string(output), "\n") {
			if line != "" {
				parts := strings.SplitN(line, " ", 3)
				if len(parts) == 3 {
					outputBuilder.WriteString(fmt.Sprintf("%-10s %-6s %s\n", parts[0], parts[1], parts[2]))
				}
			}
		}
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Listed processes most likely to be OOM killed"
	return result
}

// Ensure MemoryCheck implements HealableCheck
var _ checks.HealableCheck = (*MemoryCheck)(nil)
