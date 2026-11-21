// Package system provides system-level health checks
// [REQ:SYSTEM-MEMORY-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// MemoryCheck monitors RAM usage and detects memory pressure.
// It uses MemAvailable when available (Linux 3.14+) for accurate available memory,
// falling back to MemFree + Buffers + Cached on older systems.
type MemoryCheck struct {
	warningThreshold  int // percentage used
	criticalThreshold int // percentage used
	procReader        checks.ProcReader
	hostCollector     hostSnapshotCollector
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

func WithMemoryHostCollector(collector hostSnapshotCollector) MemoryCheckOption {
	return func(c *MemoryCheck) {
		c.hostCollector = collector
		c.procReader = nil
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
		hostCollector:     defaultHostSnapshotCollector{},
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

	if checkOS != "linux" {
		result.Status = checks.StatusNotApplicable
		result.Message = "Memory check is not implemented on this platform"
		result.Details["platform"] = checkOS
		return result
	}

	memInfo, err := c.readMemoryInfo(ctx)
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

func (c *MemoryCheck) readMemoryInfo(ctx context.Context) (*checks.MemInfo, error) {
	if c.procReader != nil {
		return c.procReader.ReadMeminfo()
	}
	collector := c.hostCollector
	if collector == nil {
		collector = defaultHostSnapshotCollector{}
	}
	snap, err := collector.Collect(ctx)
	if err != nil {
		return nil, err
	}
	return memInfoFromSnapshot(snap), nil
}

func memInfoFromSnapshot(snap sharedhost.Snapshot) *checks.MemInfo {
	return &checks.MemInfo{
		MemTotal:     snap.Memory.TotalBytes / 1024,
		MemAvailable: snap.Memory.AvailableBytes / 1024,
		Buffers:      snap.Memory.BuffersBytes / 1024,
		Cached:       snap.Memory.CachedBytes / 1024,
		SwapTotal:    snap.Swap.TotalBytes / 1024,
		SwapFree:     snap.Swap.FreeBytes / 1024,
	}
}

// RecoveryActions returns available recovery actions for memory issues.
// Note: Memory issues often can't be "healed" automatically, so actions focus on
// diagnostics and identifying memory hogs.
// [REQ:HEAL-ACTION-001]
func (c *MemoryCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"
	_ = lastResult

	return []checks.RecoveryAction{
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
	var failures []string

	// Basic inventory and privileged slab accounting both belong to the native
	// control plane. Keep useful partial output, but never call it complete.
	outputBuilder.WriteString("=== Memory Overview ===\n")
	memInfo, err := c.readMemoryInfo(ctx)
	if err != nil {
		failures = append(failures, fmt.Sprintf("memory inventory unavailable: %v", err))
	} else if memInfo == nil || memInfo.MemTotal == 0 {
		failures = append(failures, "memory inventory is empty")
	} else {
		fmt.Fprintf(&outputBuilder, "MemTotal: %d kB\n", memInfo.MemTotal)
		fmt.Fprintf(&outputBuilder, "MemAvailable: %d kB\n", memInfo.MemAvailable)
		fmt.Fprintf(&outputBuilder, "Buffers: %d kB\n", memInfo.Buffers)
		fmt.Fprintf(&outputBuilder, "Cached: %d kB\n", memInfo.Cached)
		fmt.Fprintf(&outputBuilder, "SwapTotal: %d kB\n", memInfo.SwapTotal)
		fmt.Fprintf(&outputBuilder, "SwapFree: %d kB\n", memInfo.SwapFree)
	}

	outputBuilder.WriteString("\n=== Kernel Slab Memory ===\n")
	kernelOutput, err := c.executor.Output(ctx, "vrooli", "host", "kernel-memory", "--json")
	if err != nil {
		failures = append(failures, fmt.Sprintf("kernel memory diagnostic unavailable: %v", err))
	} else {
		var report sharedhost.KernelMemoryReport
		if err := json.Unmarshal(kernelOutput, &report); err != nil {
			failures = append(failures, fmt.Sprintf("invalid kernel memory diagnostic: %v", err))
		} else if report.Source == "" || report.CacheCount < len(report.Caches) || len(report.Caches) == 0 {
			failures = append(failures, "kernel memory diagnostic is empty or incomplete")
		} else {
			fmt.Fprintf(&outputBuilder, "Source: %s\nAllocated: %d bytes across %d caches\n", report.Source, report.AllocatedBytes, report.CacheCount)
			for _, cache := range report.Caches {
				reclaimable := "unknown"
				if cache.Reclaimable != nil {
					reclaimable = fmt.Sprint(*cache.Reclaimable)
				}
				fmt.Fprintf(&outputBuilder, "%s: %d bytes; reclaimable=%s\n", cache.Name, cache.AllocatedBytes, reclaimable)
			}
			if report.CacheCount > len(report.Caches) {
				fmt.Fprintf(&outputBuilder, "Showing the largest %d caches.\n", len(report.Caches))
			}
		}
	}

	result.Duration = time.Since(start)
	result.Success = len(failures) == 0
	if result.Success {
		result.Message = "Memory summary retrieved"
	} else {
		result.Message = "Memory summary incomplete"
		result.Error = strings.Join(failures, "; ")
		fmt.Fprintf(&outputBuilder, "\nUnavailable diagnostics: %s\n", result.Error)
	}
	result.Output = outputBuilder.String()
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
