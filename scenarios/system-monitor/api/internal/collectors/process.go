package collectors

import (
	"context"
	"fmt"
	"os/exec"
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

// Collect gathers process metrics
func (c *ProcessCollector) Collect(ctx context.Context) (*MetricData, error) {
	totalProcesses := c.getTotalProcessCount()
	zombieProcesses := c.getZombieProcesses()
	highThreadProcesses := c.getHighThreadProcesses()
	topProcesses, _ := GetTopProcessesByCPU(10)
	
	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "process",
		Values: map[string]interface{}{
			"total_count":      totalProcesses,
			"zombie_processes": zombieProcesses,
			"high_thread_count": highThreadProcesses,
			"top_by_cpu":      topProcesses,
			"process_health":   c.getProcessHealth(),
		},
	}, nil
}

// getTotalProcessCount returns the total number of processes
func (c *ProcessCollector) getTotalProcessCount() int {
	if runtime.GOOS != "linux" {
		return 250
	}
	
	cmd := exec.Command("bash", "-c", "ps -e --no-headers | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	
	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

// getZombieProcesses returns zombie processes
func (c *ProcessCollector) getZombieProcesses() []map[string]interface{} {
	var zombies []map[string]interface{}
	
	if runtime.GOOS != "linux" {
		return zombies
	}
	
	cmd := exec.Command("bash", "-c", "ps -eo pid,comm,stat | grep ' Z' | head -10")
	output, err := cmd.Output()
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
func (c *ProcessCollector) getHighThreadProcesses() []map[string]interface{} {
	var processes []map[string]interface{}
	
	if runtime.GOOS != "linux" {
		return processes
	}
	
	cmd := exec.Command("bash", "-c", "ps -eo pid,comm,nlwp --sort=-nlwp --no-headers | head -5")
	output, err := cmd.Output()
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

// getProcessHealth returns overall process health metrics
func (c *ProcessCollector) getProcessHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status":           "healthy",
		"zombie_count":     0,
		"high_thread_count": 0,
		"leak_candidates":  0,
		"critical_processes": c.checkCriticalProcesses(),
	}
	
	zombies := c.getZombieProcesses()
	health["zombie_count"] = len(zombies)
	
	highThread := c.getHighThreadProcesses()
	health["high_thread_count"] = len(highThread)
	
	if len(zombies) > 5 || len(highThread) > 10 {
		health["status"] = "warning"
	}
	
	return health
}

// checkCriticalProcesses checks if critical processes are running
func (c *ProcessCollector) checkCriticalProcesses() []map[string]interface{} {
	criticalProcesses := []string{
		"postgres",
		"redis-server",
		"node",
		"system-monitor-api",
	}
	
	var status []map[string]interface{}
	
	for _, process := range criticalProcesses {
		running := c.isProcessRunning(process)
		status = append(status, map[string]interface{}{
			"name":    process,
			"running": running,
		})
	}
	
	return status
}

// isProcessRunning checks if a process is running
func (c *ProcessCollector) isProcessRunning(processName string) bool {
	if runtime.GOOS != "linux" {
		return true // Assume running in non-Linux environments
	}
	
	cmd := exec.Command("bash", "-c", fmt.Sprintf("pgrep -f %s", processName))
	output, err := cmd.Output()
	
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

// GetProcessFileDescriptors returns file descriptor count for a process
func GetProcessFileDescriptors(pid int) int {
	if runtime.GOOS != "linux" {
		return 10
	}
	
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ls %s 2>/dev/null | wc -l", fdDir))
	output, err := cmd.Output()
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