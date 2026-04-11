package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func (a *App) buildProcessTable() (map[int]processTableEntry, error) {
	cmd := exec.Command("ps", "-eo", "pid,ppid,pgid,state,cmd")
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
			lineNum++
			continue
		}
		lineNum++
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, _ := strconv.Atoi(fields[1])
		pgid, _ := strconv.Atoi(fields[2])
		processTable[pid] = processTableEntry{
			PID:     pid,
			PPID:    ppid,
			PGID:    pgid,
			State:   fields[3],
			Command: strings.Join(fields[4:], " "),
		}
	}
	return processTable, nil
}

func (a *App) loadTrackedProcessStats(processTable map[int]processTableEntry) trackedProcessStats {
	stats := trackedProcessStats{trackedPIDs: make(map[int]struct{})}
	processesDir := filepath.Join(a.Home, ".vrooli", "processes", "scenarios")
	if _, err := os.Stat(processesDir); os.IsNotExist(err) {
		return stats
	}
	_ = filepath.Walk(processesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
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
				stats.trackedCount++
				if _, running := processTable[pid]; running {
					stats.runningTracked++
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
	case count <= 10:
		return "normal", "✅"
	case count <= 25:
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
	if entry.PGID > 0 {
		if _, ok := tracked[entry.PGID]; ok {
			memo[pid] = true
			return true
		}
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
	memo[pid] = trackedAncestor
	return trackedAncestor
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
		orphanCount++
	}
	return orphanCount
}

func (a *App) collectProcessHealthSnapshot() ProcessHealthSnapshot {
	processTable, err := a.buildProcessTable()
	if err != nil {
		return ProcessHealthSnapshot{
			ZombieStatus:  "unknown",
			ZombieEmoji:   "❔",
			OrphanStatus:  "unknown",
			OrphanEmoji:   "❔",
			OverallStatus: "unknown",
		}
	}
	snapshot, _ := a.computeProcessSnapshot(processTable)
	return snapshot
}

func (a *App) computeProcessSnapshot(processTable map[int]processTableEntry) (ProcessHealthSnapshot, trackedProcessStats) {
	stats := a.loadTrackedProcessStats(processTable)
	zombieCount := 0
	for _, entry := range processTable {
		if strings.HasPrefix(entry.State, "Z") {
			zombieCount++
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
	return ProcessHealthSnapshot{
		ZombieCount:   zombieCount,
		ZombieStatus:  zombieStatus,
		ZombieEmoji:   zombieEmoji,
		OrphanCount:   orphanCount,
		OrphanStatus:  orphanStatus,
		OrphanEmoji:   orphanEmoji,
		OverallStatus: overallStatus,
	}, stats
}

func (a *App) getEnhancedProcessMetrics() map[string]interface{} {
	processTable, err := a.buildProcessTable()
	if err != nil {
		return map[string]interface{}{
			"tracked_processes": 0,
			"running_tracked":   0,
			"child_processes":   0,
			"total_processes":   0,
			"zombie_processes":  0,
			"orphan_processes":  0,
		}
	}
	snapshot, stats := a.computeProcessSnapshot(processTable)
	totalProcesses := len(processTable)
	childProcesses := totalProcesses - stats.runningTracked
	if _, exists := processTable[os.Getpid()]; exists {
		childProcesses--
	}
	if childProcesses < 0 {
		childProcesses = 0
	}
	return map[string]interface{}{
		"tracked_processes": stats.trackedCount,
		"running_tracked":   stats.runningTracked,
		"child_processes":   childProcesses,
		"total_processes":   totalProcesses,
		"zombie_processes":  snapshot.ZombieCount,
		"orphan_processes":  snapshot.OrphanCount,
	}
}

func (a *App) DiscoverScenarioPorts(scenarioName string) map[string]int {
	item, err := scenario.Load(a.Root, scenarioName, scenario.SandboxEnvFromEnv())
	if err != nil {
		return map[string]int{}
	}
	records, err := process.ReadScenarioRecords(a.Home, scenarioName)
	if err != nil {
		return map[string]int{}
	}
	return scenario.RuntimePorts(item.Manifest, process.LiveRecords(records))
}

func isPIDRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func checkForkBomb() error {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	if strings.Count(string(output), "\n") > 2000 {
		return fmt.Errorf("system overload: too many processes")
	}
	return nil
}

func (a *App) discoverRunningScenarios() ([]RunningScenario, error) {
	views, err := a.Scenarios.Running()
	if err != nil {
		return nil, err
	}
	result := make([]RunningScenario, 0, len(views))
	for _, item := range views {
		result = append(result, RunningScenario{
			Name:      item.Name,
			Status:    item.Status,
			Processes: item.Processes,
			StartedAt: item.StartedAt,
			Runtime:   item.Runtime,
			Ports:     item.Ports,
		})
	}
	return result, nil
}

func (a *App) loadScenarioRuntime(name string) (scenario.Scenario, process.ScenarioRuntime, scenario.RuntimeDetails, error) {
	detail, err := a.Scenarios.Detail(name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, scenario.RuntimeDetails{}, err
	}
	return detail.Scenario, detail.Runtime, detail.Details, nil
}

func (a *App) PerformHealthCheck(check HealthCheckConfig, scenarioName string, ports map[string]int) error {
	switch check.Type {
	case "http":
		target := check.Target
		for varName, port := range ports {
			target = strings.ReplaceAll(target, "${"+varName+"}", strconv.Itoa(port))
			target = strings.ReplaceAll(target, "$"+varName, strconv.Itoa(port))
		}
		if _, err := url.Parse(target); err != nil {
			return fmt.Errorf("invalid URL: %s", target)
		}
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 5 * time.Second
		}
		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(target)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return nil
	case "postgres":
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 3 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if _, err := a.LookPathFn("vrooli"); err == nil {
			if output, cmdErr := a.CommandFn(ctx, "vrooli", "resource", "status", "postgres", "--json"); cmdErr == nil {
				var status struct {
					Running   bool  `json:"running"`
					Healthy   *bool `json:"healthy"`
					Installed bool  `json:"installed"`
				}
				if err := json.Unmarshal(output, &status); err == nil {
					if !status.Installed {
						return fmt.Errorf("postgres resource not installed")
					}
					if !status.Running {
						return fmt.Errorf("postgres resource not running")
					}
					if status.Healthy != nil && !*status.Healthy {
						return fmt.Errorf("postgres resource unhealthy")
					}
					return nil
				}
			}
		}
		address := "127.0.0.1:5432"
		if parsed, err := parsePostgresAddress(check.Target); err == nil && parsed != "" {
			address = parsed
		}
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return fmt.Errorf("postgres health check failed for %q: %w", address, err)
		}
		_ = conn.Close()
		return nil
	default:
		return fmt.Errorf("unsupported health check type: %s", check.Type)
	}
}

func parsePostgresAddress(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil
	}
	if strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://") {
		u, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		host := u.Hostname()
		if host == "" {
			return "", nil
		}
		port := u.Port()
		if port == "" {
			port = "5432"
		}
		return net.JoinHostPort(host, port), nil
	}
	if strings.Contains(target, ":") {
		host, port, err := net.SplitHostPort(target)
		if err == nil && host != "" && port != "" {
			return net.JoinHostPort(host, port), nil
		}
		return "", err
	}
	return "", nil
}

func (a *App) notFoundStopsApp(err error) bool {
	return errors.Is(err, scenario.ErrNotFound)
}
