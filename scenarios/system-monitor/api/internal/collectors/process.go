package collectors

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// ProcessCollector collects process metrics
type ProcessCollector struct {
	BaseCollector
	forkRate forkRateTracker
}

var (
	topProcessSamplerMu sync.Mutex
	topProcessSampler   procsampler.Sampler = procsampler.NewSampler()
)

func SetTopProcessSampler(sampler procsampler.Sampler) {
	topProcessSamplerMu.Lock()
	defer topProcessSamplerMu.Unlock()
	if sampler == nil {
		topProcessSampler = procsampler.NewSampler()
		return
	}
	topProcessSampler = sampler
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
	if collectorOS != runtime.GOOS {
		return unsupportedMetricData(c.GetName(), "process"), nil
	}
	samples, err := currentProcessSamples(ctx)
	if err != nil {
		if err == procsampler.ErrUnsupported {
			return unsupportedMetricData(c.GetName(), "process"), nil
		}
		return &MetricData{
			CollectorName: c.GetName(),
			Timestamp:     time.Now(),
			Type:          "process",
			Values: map[string]interface{}{
				"status": "failed",
				"reason": "native process table unavailable: " + err.Error(),
			},
		}, nil
	}
	totalProcesses := len(samples)
	zombieProcesses := zombieProcessesFromSamples(samples)
	highThreadProcesses := highThreadProcessesFromSamples(samples)
	topProcesses, _ := GetTopProcessesByCPU(10)

	now := time.Now()
	values := map[string]interface{}{
		"total_count":       totalProcesses,
		"zombie_processes":  zombieProcesses,
		"high_thread_count": highThreadProcesses,
		"top_by_cpu":        topProcesses,
		"process_health":    c.processHealth(ctx, zombieProcesses, highThreadProcesses),
	}
	// Process-creation rate. A fork storm is invisible in total_count because
	// the processes are short-lived: the population stays flat while the host
	// burns its CPU in the kernel creating and reaping them.
	for key, value := range forkRateValues(&c.forkRate, readForkRate(), now) {
		values[key] = value
	}

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     now,
		Type:          "process",
		Values:        values,
	}, nil
}

func currentProcessSamples(ctx context.Context) ([]procsampler.ProcessSample, error) {
	topProcessSamplerMu.Lock()
	defer topProcessSamplerMu.Unlock()
	return topProcessSampler.Sample(ctx)
}

func zombieProcessesFromSamples(samples []procsampler.ProcessSample) []map[string]interface{} {
	zombies := make([]map[string]interface{}, 0)
	for _, sample := range samples {
		if sample.State != "Z" && sample.State != "z" {
			continue
		}
		zombies = append(zombies, map[string]interface{}{
			"pid":    sample.PID,
			"name":   sample.Comm,
			"status": "zombie",
		})
		if len(zombies) >= 10 {
			break
		}
	}
	return zombies
}

func highThreadProcessesFromSamples(samples []procsampler.ProcessSample) []map[string]interface{} {
	ordered := append([]procsampler.ProcessSample(nil), samples...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Threads > ordered[j].Threads })
	processes := make([]map[string]interface{}, 0)
	for _, sample := range ordered {
		if sample.Threads <= 20 {
			continue
		}
		processes = append(processes, map[string]interface{}{
			"pid":     sample.PID,
			"name":    sample.Comm,
			"threads": sample.Threads,
		})
		if len(processes) >= 5 {
			break
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

// checkCriticalProcesses reports presence of each critical process using one
// native /proc scan rather than one pgrep fork per name.
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
// single /proc scan. On non-Linux platforms it returns nil and callers treat
// every critical process as present (the prior behavior).
func (c *ProcessCollector) runningCommandSet(ctx context.Context) map[string]struct{} {
	samples, err := currentProcessSamples(ctx)
	if err != nil {
		return map[string]struct{}{}
	}
	set := map[string]struct{}{}
	for _, sample := range samples {
		if sample.Comm != "" {
			set[sample.Comm] = struct{}{}
		}
	}
	return set
}

// runningContains reports whether any running command name contains the target
// substring (matching pgrep -f's loose semantics for names like "redis-server"
// that ps may truncate).
func runningContains(set map[string]struct{}, target string) bool {
	for name := range set {
		if name == target || strings.Contains(name, target) || strings.Contains(target, name) {
			return true
		}
	}
	return false
}

// GetProcessFileDescriptors returns file descriptor count for a process
func GetProcessFileDescriptors(pid int) int {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return 0
	}

	return len(entries)
}

// GetResourceLeakCandidates identifies processes that might have resource leaks
func GetResourceLeakCandidates() []map[string]interface{} {
	// Leak candidates require historical process observations. Do not invent
	// candidates when that evidence has not been collected.
	return nil
}

func topProcessSamples(limit int, less func(a, b procsampler.ProcessSample) bool) ([]procsampler.ProcessSample, error) {
	if limit <= 0 {
		return nil, nil
	}
	topProcessSamplerMu.Lock()
	defer topProcessSamplerMu.Unlock()
	samples, err := topProcessSampler.Sample(context.Background())
	if err != nil {
		if err == procsampler.ErrUnsupported {
			return []procsampler.ProcessSample{}, nil
		}
		return nil, err
	}
	filtered := samples[:0]
	for _, sample := range samples {
		if sample.MetricsStatus != "unsupported" {
			filtered = append(filtered, sample)
		}
	}
	samples = filtered
	sort.SliceStable(samples, func(i, j int) bool {
		return less(samples[i], samples[j])
	})
	if len(samples) > limit {
		samples = samples[:limit]
	}
	return samples, nil
}

func totalMemoryKB() int64 {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		n, _ := strconv.ParseInt(fields[1], 10, 64)
		return n
	}
	return 0
}

func memoryPercent(rssKB, totalKB int64) float64 {
	if rssKB <= 0 || totalKB <= 0 {
		return 0
	}
	return float64(rssKB) / float64(totalKB) * 100
}
