package systemmetrics

import (
	"math"
	"scenario-to-cloud/domain"
	"strconv"
	"strings"
)

type linuxCollector struct{}

func (linuxCollector) Name() string { return "linux" }

// SystemCommands returns Linux-specific commands used to build SystemState.
func (linuxCollector) SystemCommands() []CommandSpec {
	return []CommandSpec{
		{ID: "df_kb", Command: "df -Pk / 2>/dev/null | tail -1"},
		{ID: "meminfo", Command: "cat /proc/meminfo 2>/dev/null"},
		{ID: "loadavg", Command: "cat /proc/loadavg 2>/dev/null"},
		{ID: "uptime", Command: "cat /proc/uptime 2>/dev/null"},
		{ID: "cpuinfo", Command: "grep -c processor /proc/cpuinfo 2>/dev/null"},
		{ID: "cpumodel", Command: "grep 'model name' /proc/cpuinfo 2>/dev/null | head -1"},
		// Sample /proc/stat twice over 1 second to estimate current CPU usage.
		{ID: "cpuusage", Command: "cat /proc/stat | head -1; sleep 1; cat /proc/stat | head -1"},
	}
}

func (linuxCollector) ParseSystemState(results map[string]CommandResult) domain.SystemState {
	state := domain.SystemState{}

	parseCPUMetrics(results, &state)
	parseUptimeMetrics(results, &state)
	parseMemoryMetrics(results, &state)
	parseDiskMetrics(results, &state)

	return state
}

func parseCPUMetrics(results map[string]CommandResult, state *domain.SystemState) {
	if coresResult, ok := results["cpuinfo"]; ok {
		cores, _ := strconv.Atoi(strings.TrimSpace(coresResult.Stdout))
		state.CPU.Cores = cores
	}

	if modelResult, ok := results["cpumodel"]; ok {
		line := strings.TrimSpace(modelResult.Stdout)
		if idx := strings.Index(line, ":"); idx >= 0 {
			state.CPU.Model = strings.TrimSpace(line[idx+1:])
		}
	}

	if loadResult, ok := results["loadavg"]; ok {
		fields := strings.Fields(loadResult.Stdout)
		if len(fields) >= 3 {
			load1, _ := strconv.ParseFloat(fields[0], 64)
			load5, _ := strconv.ParseFloat(fields[1], 64)
			load15, _ := strconv.ParseFloat(fields[2], 64)
			state.CPU.LoadAverage = []float64{load1, load5, load15}
		}
	}

	if cpuUsageResult, ok := results["cpuusage"]; ok {
		state.CPU.UsagePercent = ParseCPUUsageFromProcStat(cpuUsageResult.Stdout)
	}

	if state.CPU.UsagePercent == 0 && state.CPU.Cores > 0 && len(state.CPU.LoadAverage) > 0 {
		state.CPU.UsagePercent = (state.CPU.LoadAverage[0] / float64(state.CPU.Cores*2)) * 100
		if state.CPU.UsagePercent > 100 {
			state.CPU.UsagePercent = 100
		}
	}
}

func parseUptimeMetrics(results map[string]CommandResult, state *domain.SystemState) {
	if uptimeResult, ok := results["uptime"]; ok {
		fields := strings.Fields(uptimeResult.Stdout)
		if len(fields) >= 1 {
			uptime, _ := strconv.ParseFloat(fields[0], 64)
			state.UptimeSeconds = int64(uptime)
		}
	}
}

func parseMemoryMetrics(results map[string]CommandResult, state *domain.SystemState) {
	if meminfoResult, ok := results["meminfo"]; ok && strings.TrimSpace(meminfoResult.Stdout) != "" {
		parseMeminfo(meminfoResult.Stdout, state)
		return
	}

	// Backward-compatible fallback for older snapshots/tests.
	if freeResult, ok := results["free"]; ok {
		parseFreeOutput(freeResult.Stdout, state)
	}
}

func parseMeminfo(output string, state *domain.SystemState) {
	vals := make(map[string]int64)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		vals[key] = value
	}

	totalKB := vals["MemTotal"]
	availableKB := vals["MemAvailable"]
	if availableKB == 0 {
		// Conservative fallback when MemAvailable is missing.
		availableKB = vals["MemFree"] + vals["Buffers"] + vals["Cached"]
	}
	usedKB := totalKB - availableKB
	if usedKB < 0 {
		usedKB = 0
	}

	state.Memory.TotalMB = int(totalKB / 1024)
	state.Memory.UsedMB = int(usedKB / 1024)
	state.Memory.FreeMB = int(availableKB / 1024)
	state.Memory.TotalBytes = totalKB * 1024
	state.Memory.UsedBytes = usedKB * 1024
	state.Memory.FreeBytes = availableKB * 1024
	if totalKB > 0 {
		state.Memory.UsagePercent = (float64(usedKB) / float64(totalKB)) * 100
	}

	swapTotalKB := vals["SwapTotal"]
	swapFreeKB := vals["SwapFree"]
	swapUsedKB := swapTotalKB - swapFreeKB
	if swapUsedKB < 0 {
		swapUsedKB = 0
	}
	state.Swap.TotalMB = int(swapTotalKB / 1024)
	state.Swap.UsedMB = int(swapUsedKB / 1024)
	if swapTotalKB > 0 {
		state.Swap.UsagePercent = (float64(swapUsedKB) / float64(swapTotalKB)) * 100
	}
}

func parseFreeOutput(output string, state *domain.SystemState) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		if strings.HasPrefix(fields[0], "Mem:") {
			total, _ := strconv.Atoi(fields[1])
			used, _ := strconv.Atoi(fields[2])
			free := 0
			if len(fields) >= 7 {
				available, _ := strconv.Atoi(fields[6])
				free = available
				used = total - available
				if used < 0 {
					used = 0
				}
			} else {
				free, _ = strconv.Atoi(fields[3])
			}

			state.Memory.TotalMB = total
			state.Memory.UsedMB = used
			state.Memory.FreeMB = free
			state.Memory.TotalBytes = int64(total) * 1024 * 1024
			state.Memory.UsedBytes = int64(used) * 1024 * 1024
			state.Memory.FreeBytes = int64(free) * 1024 * 1024
			if total > 0 {
				state.Memory.UsagePercent = float64(used) / float64(total) * 100
			}
		} else if strings.HasPrefix(fields[0], "Swap:") {
			total, _ := strconv.Atoi(fields[1])
			used, _ := strconv.Atoi(fields[2])
			state.Swap.TotalMB = total
			state.Swap.UsedMB = used
			if total > 0 {
				state.Swap.UsagePercent = float64(used) / float64(total) * 100
			}
		}
	}
}

func parseDiskMetrics(results map[string]CommandResult, state *domain.SystemState) {
	if dfResult, ok := results["df_kb"]; ok && strings.TrimSpace(dfResult.Stdout) != "" {
		if parseDiskFromDFKB(dfResult.Stdout, state) {
			return
		}
	}

	// Backward-compatible fallback for older snapshots/tests.
	if dfResult, ok := results["df"]; ok {
		parseDiskFromHumanDF(dfResult.Stdout, state)
	}
}

func parseDiskFromDFKB(line string, state *domain.SystemState) bool {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 5 {
		return false
	}

	totalKB, errTotal := strconv.ParseInt(fields[1], 10, 64)
	usedKB, errUsed := strconv.ParseInt(fields[2], 10, 64)
	freeKB, errFree := strconv.ParseInt(fields[3], 10, 64)
	if errTotal != nil || errUsed != nil || errFree != nil {
		return false
	}

	state.Disk.TotalGB = int(math.Round(float64(totalKB) / (1024 * 1024)))
	state.Disk.UsedGB = int(math.Round(float64(usedKB) / (1024 * 1024)))
	state.Disk.FreeGB = int(math.Round(float64(freeKB) / (1024 * 1024)))
	state.Disk.TotalBytes = totalKB * 1024
	state.Disk.UsedBytes = usedKB * 1024
	state.Disk.FreeBytes = freeKB * 1024

	usageStr := strings.TrimSuffix(fields[4], "%")
	usage, _ := strconv.ParseFloat(usageStr, 64)
	state.Disk.UsagePercent = usage
	return true
}

func parseDiskFromHumanDF(output string, state *domain.SystemState) {
	fields := strings.Fields(output)
	if len(fields) < 5 {
		return
	}
	state.Disk.TotalGB = ParseHumanSizeToGB(fields[1])
	state.Disk.UsedGB = ParseHumanSizeToGB(fields[2])
	state.Disk.FreeGB = ParseHumanSizeToGB(fields[3])
	const gib = int64(1024 * 1024 * 1024)
	state.Disk.TotalBytes = int64(state.Disk.TotalGB) * gib
	state.Disk.UsedBytes = int64(state.Disk.UsedGB) * gib
	state.Disk.FreeBytes = int64(state.Disk.FreeGB) * gib

	usageStr := strings.TrimSuffix(fields[4], "%")
	usage, _ := strconv.ParseFloat(usageStr, 64)
	state.Disk.UsagePercent = usage
}

// ParseHumanSizeToGB converts a human-readable size (e.g., 200G, 1.5T) to GB.
func ParseHumanSizeToGB(s string) int {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	switch unit {
	case 'T', 't':
		return int(num * 1024)
	case 'G', 'g':
		return int(num)
	case 'M', 'm':
		return int(num / 1024)
	case 'K', 'k':
		return int(num / (1024 * 1024))
	default:
		return int(num / (1024 * 1024 * 1024))
	}
}

// ParseCPUUsageFromProcStat parses CPU usage from /proc/stat samples.
func ParseCPUUsageFromProcStat(output string) float64 {
	output = strings.TrimSpace(output)
	if output == "" {
		return 0
	}

	lines := strings.Split(output, "\n")
	var cpuLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "cpu ") {
			cpuLines = append(cpuLines, line)
		}
	}

	if len(cpuLines) >= 2 {
		// Support smoothed sampling where output contains 3+ snapshots.
		var (
			sumUsage float64
			count    int
		)
		for i := 0; i < len(cpuLines)-1; i++ {
			stats1 := parseProcStatLine(cpuLines[i])
			stats2 := parseProcStatLine(cpuLines[i+1])
			usage, ok := usageBetweenSamples(stats1, stats2)
			if ok {
				sumUsage += usage
				count++
			}
		}
		if count > 0 {
			return clampCPUUsage(sumUsage / float64(count))
		}
	}

	if len(cpuLines) >= 1 {
		stats := parseProcStatLine(cpuLines[0])
		if stats != nil {
			total := stats.user + stats.nice + stats.system + stats.idle + stats.iowait + stats.irq + stats.softirq + stats.steal
			idle := stats.idle + stats.iowait
			if total > 0 {
				usage := float64(total-idle) / float64(total) * 100
				return clampCPUUsage(usage)
			}
		}
	}
	return 0
}

func usageBetweenSamples(stats1, stats2 *cpuStats) (float64, bool) {
	if stats1 == nil || stats2 == nil {
		return 0, false
	}
	total1 := stats1.user + stats1.nice + stats1.system + stats1.idle + stats1.iowait + stats1.irq + stats1.softirq + stats1.steal
	total2 := stats2.user + stats2.nice + stats2.system + stats2.idle + stats2.iowait + stats2.irq + stats2.softirq + stats2.steal
	idle1 := stats1.idle + stats1.iowait
	idle2 := stats2.idle + stats2.iowait
	totalDiff := total2 - total1
	idleDiff := idle2 - idle1
	if totalDiff <= 0 {
		return 0, false
	}
	return float64(totalDiff-idleDiff) / float64(totalDiff) * 100, true
}

func clampCPUUsage(usage float64) float64 {
	if usage < 0 {
		return 0
	}
	if usage > 100 {
		return 100
	}
	return usage
}

type cpuStats struct {
	user, nice, system, idle, iowait, irq, softirq, steal int64
}

func parseProcStatLine(line string) *cpuStats {
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return nil
	}
	vals := make([]int64, 8)
	for i := 0; i < 8 && i+1 < len(fields); i++ {
		vals[i], _ = strconv.ParseInt(fields[i+1], 10, 64)
	}
	return &cpuStats{
		user: vals[0], nice: vals[1], system: vals[2], idle: vals[3],
		iowait: vals[4], irq: vals[5], softirq: vals[6], steal: vals[7],
	}
}
