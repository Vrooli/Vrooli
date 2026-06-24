package collectors

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ProcessCollector collects process metrics
type ProcessCollector struct {
	BaseCollector
}

// NewProcessCollector creates a new process collector
func NewProcessCollector() *ProcessCollector {
	return &ProcessCollector{
		BaseCollector: NewBaseCollector("process", 20*time.Second),
	}
}

// Collect gathers process metrics. Zombie and high-thread results are computed
// once here and reused for the health summary — previously getProcessHealth
// re-shelled both queries, doubling the forks per cycle.
func (c *ProcessCollector) Collect(ctx context.Context) (*MetricData, error) {
	totalProcesses := c.getTotalProcessCount(ctx)
	zombieProcesses := c.getZombieProcesses(ctx)
	highThreadProcesses := c.getHighThreadProcesses(ctx)
	topProcesses, _ := GetTopProcessesByCPU(10)

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "process",
		Values: map[string]interface{}{
			"total_count":       totalProcesses,
			"zombie_processes":  zombieProcesses,
			"high_thread_count": highThreadProcesses,
			"top_by_cpu":        topProcesses,
			"process_health":    c.processHealth(ctx, zombieProcesses, highThreadProcesses),
		},
	}, nil
}

// getTotalProcessCount returns the total number of processes
func (c *ProcessCollector) getTotalProcessCount(ctx context.Context) int {
	if runtime.GOOS != "linux" {
		return 250
	}

	output, err := commandOutput(ctx, 2*time.Second, "bash", "-c", "ps -e --no-headers | wc -l")
	if err != nil {
		return 0
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

// getZombieProcesses returns zombie processes
func (c *ProcessCollector) getZombieProcesses(ctx context.Context) []map[string]interface{} {
	var zombies []map[string]interface{}

	if runtime.GOOS != "linux" {
		return zombies
	}

	output, err := commandOutput(ctx, 2*time.Second, "bash", "-c", "ps -eo pid,comm,stat | grep ' Z' | head -10")
	if err != nil {
		return zombies
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, _ := strconv.Atoi(fields[0])
			zombies = append(zombies, map[string]interface{}{
				"pid":    pid,
				"name":   fields[1],
				"status": "zombie",
			})
		}
	}

	return zombies
}

// getHighThreadProcesses returns processes with high thread counts
func (c *ProcessCollector) getHighThreadProcesses(ctx context.Context) []map[string]interface{} {
	var processes []map[string]interface{}

	if runtime.GOOS != "linux" {
		return processes
	}

	output, err := commandOutput(ctx, 2*time.Second, "bash", "-c", "ps -eo pid,comm,nlwp --sort=-nlwp --no-headers | head -5")
	if err != nil {
		return processes
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, _ := strconv.Atoi(fields[0])
			threads, _ := strconv.Atoi(fields[2])

			if threads > 20 { // Only include processes with >20 threads
				processes = append(processes, map[string]interface{}{
					"pid":     pid,
					"name":    fields[1],
					"threads": threads,
				})
			}
		}
	}

	return processes
}

// processHealth builds the health summary from the zombie and high-thread
// results already computed in Collect (no re-shelling), plus a single
// process-table scan for critical-process presence (replaces 4 pgrep forks).
func (c *ProcessCollector) processHealth(ctx context.Context, zombies, highThread []map[string]interface{}) map[string]interface{} {
	health := map[string]interface{}{
		"status":             "healthy",
		"zombie_count":       len(zombies),
		"high_thread_count":  len(highThread),
		"leak_candidates":    0,
		"critical_processes": c.checkCriticalProcesses(ctx),
	}

	if len(zombies) > 5 || len(highThread) > 10 {
		health["status"] = "warning"
	}

	return health
}

// criticalProcessNames are the processes whose presence is reported in the
// health summary.
var criticalProcessNames = []string{
	"postgres",
	"redis-server",
	"node",
	"system-monitor-api",
}

// checkCriticalProcesses reports presence of each critical process using ONE
// process-table scan (a single fork) rather than one pgrep fork per name.
func (c *ProcessCollector) checkCriticalProcesses(ctx context.Context) []map[string]interface{} {
	running := c.runningCommandSet(ctx)

	status := make([]map[string]interface{}, 0, len(criticalProcessNames))
	for _, process := range criticalProcessNames {
		status = append(status, map[string]interface{}{
			"name":    process,
			"running": runningContains(running, process),
		})
	}
	return status
}

// runningCommandSet returns the set of running process command names from a
// single `ps -eo comm` scan. On non-Linux platforms it returns nil and callers
// treat every critical process as present (the prior behavior).
func (c *ProcessCollector) runningCommandSet(ctx context.Context) map[string]struct{} {
	if runtime.GOOS != "linux" {
		return nil
	}
	output, err := commandOutput(ctx, 2*time.Second, "ps", "-eo", "comm", "--no-headers")
	if err != nil {
		return map[string]struct{}{}
	}
	set := map[string]struct{}{}
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			set[name] = struct{}{}
		}
	}
	return set
}

// runningContains reports whether any running command name contains the target
// substring (matching pgrep -f's loose semantics for names like "redis-server"
// that ps may truncate). A nil set (non-Linux) is treated as "present".
func runningContains(set map[string]struct{}, target string) bool {
	if set == nil {
		return true
	}
	for name := range set {
		if name == target || strings.Contains(name, target) || strings.Contains(target, name) {
			return true
		}
	}
	return false
}

// GetProcessFileDescriptors returns file descriptor count for a process
func GetProcessFileDescriptors(pid int) int {
	if runtime.GOOS != "linux" {
		return 10
	}

	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	output, err := commandOutput(context.Background(), 2*time.Second, "bash", "-c", fmt.Sprintf("ls %s 2>/dev/null | wc -l", fdDir))
	if err != nil {
		return 0
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

// GetResourceLeakCandidates identifies processes that might have resource leaks
func GetResourceLeakCandidates() []map[string]interface{} {
	// This would require historical tracking
	// For now, return mock data
	return []map[string]interface{}{
		{
			"pid":         1234,
			"name":        "scenario-api-1",
			"status":      "fd_leak_risk",
			"fd_count":    512,
			"memory_mb":   1024,
			"description": "High file descriptor count",
		},
	}
}
