package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
)

type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type RunningScenario struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Processes int            `json:"processes"`
	StartedAt *time.Time     `json:"started_at"`
	Runtime   string         `json:"runtime"`
	Ports     map[string]int `json:"ports"`
}

type HealthCheckConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Critical bool   `json:"critical"`
	Timeout  int    `json:"timeout"`
	Interval int    `json:"interval"`
}

type ScenarioHealthConfig struct {
	Description        string              `json:"description"`
	Endpoints          map[string]string   `json:"endpoints"`
	Checks             []HealthCheckConfig `json:"checks"`
	Timeout            int                 `json:"timeout"`
	Interval           int                 `json:"interval"`
	StartupGracePeriod int                 `json:"startup_grace_period"`
}

type App struct {
	Root                string
	Home                string
	AppsDir             string
	Scenarios           *orchestrator.Service
	Resources           *resources.Controller
	Project             *project.Controller
	LookPathFn          func(string) (string, error)
	CommandFn           func(context.Context, string, ...string) ([]byte, error)
	StartAllScenariosFn func() (control.StartReport, error)
	StopAllScenariosFn  func() (control.StopReport, error)
	StopScenarioFn      func(string) error
}

type messageData struct {
	Message string `json:"message"`
}

type appLogsData struct {
	Logs     string `json:"logs"`
	Scenario string `json:"scenario"`
}

type lifecycleActionData struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

type stoppedAppData struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Processes int            `json:"processes"`
	Runtime   string         `json:"runtime"`
	Ports     map[string]int `json:"ports"`
}

type processTableEntry struct {
	PID     int
	PPID    int
	PGID    int
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

type appInfo struct {
	Name          string                 `json:"name"`
	Path          string                 `json:"path"`
	Protected     bool                   `json:"protected"`
	HasGit        bool                   `json:"has_git"`
	Customized    bool                   `json:"customized"`
	Modified      time.Time              `json:"modified"`
	RuntimeStatus string                 `json:"runtime_status,omitempty"`
	Ports         map[string]interface{} `json:"ports,omitempty"`
	PID           *int                   `json:"pid,omitempty"`
}

var orphanCommandPattern = regexp.MustCompile(`(/vrooli/|/scenarios/.*/(api|ui)|node_modules/.bin/vite|ecosystem-manager|picker-wheel|vrooli-.*-api)`)

func New(root, home string) *App {
	app := &App{
		Root:       filepath.Clean(root),
		Home:       filepath.Clean(home),
		AppsDir:    filepath.Join(filepath.Clean(root), "scenarios"),
		Scenarios:  orchestrator.New(root, home, ioDiscard{}, ioDiscard{}),
		Resources:  resources.NewController(root, home),
		Project:    project.New(root, home, ioDiscard{}, ioDiscard{}),
		LookPathFn: exec.LookPath,
		CommandFn: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
	app.StartAllScenariosFn = func() (control.StartReport, error) {
		return app.Scenarios.StartAll()
	}
	app.StopAllScenariosFn = func() (control.StopReport, error) {
		return app.Scenarios.StopAll()
	}
	app.StopScenarioFn = func(name string) error {
		return app.Scenarios.Stop(name, lifecycle.StopOptions{})
	}
	return app
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func (a *App) Router() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/health", a.HealthCheck).Methods("GET")
	r.HandleFunc("/metrics/processes", a.ProcessMetricsHandler).Methods("GET")
	r.HandleFunc("/apps", a.ListApps).Methods("GET")
	r.HandleFunc("/apps/running", a.GetRunningApps).Methods("GET")
	r.HandleFunc("/apps/start-all", a.StartAllApps).Methods("POST")
	r.HandleFunc("/apps/stop-all", a.StopAllApps).Methods("POST")
	r.HandleFunc("/apps/{name}/protect", a.ProtectApp).Methods("POST")
	r.HandleFunc("/apps/{name}/start", a.StartApp).Methods("POST")
	r.HandleFunc("/apps/{name}/stop", a.StopApp).Methods("POST")
	r.HandleFunc("/apps/{name}/restart", a.RestartApp).Methods("POST")
	r.HandleFunc("/apps/{name}/logs", a.GetAppLogs).Methods("GET")
	r.HandleFunc("/apps/{name}/status", a.GetDetailedAppStatus).Methods("GET")
	r.HandleFunc("/scenarios", a.ListScenariosNative).Methods("GET")
	r.HandleFunc("/scenarios/{name}/status", a.GetScenarioStatusNative).Methods("GET")
	r.HandleFunc("/scenarios/{name}/start", a.StartApp).Methods("POST")
	r.HandleFunc("/scenarios/{name}/stop", a.StopScenarioEndpoint).Methods("POST")
	r.HandleFunc("/scenarios/start-all", a.StartAllScenariosEndpoint).Methods("POST")
	r.HandleFunc("/scenarios/stop-all", a.StopAllScenariosEndpoint).Methods("POST")
	r.HandleFunc("/resources", a.ListResources).Methods("GET")
	r.HandleFunc("/lifecycle/{action}", a.HandleLifecycle).Methods("POST")
	return r
}

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
	ports := make(map[string]int)
	serviceFile := filepath.Join(a.Root, "scenarios", scenarioName, ".vrooli", "service.json")
	serviceData, err := os.ReadFile(serviceFile)
	if err != nil {
		return ports
	}

	var serviceConfig struct {
		Ports map[string]struct {
			EnvVar string `json:"env_var"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(serviceData, &serviceConfig); err != nil {
		return ports
	}

	validPortVars := make(map[string]bool)
	for _, portConfig := range serviceConfig.Ports {
		if portConfig.EnvVar != "" {
			validPortVars[portConfig.EnvVar] = true
		}
	}
	if len(validPortVars) == 0 {
		return ports
	}

	processesDir := filepath.Join(a.Home, ".vrooli", "processes", "scenarios", scenarioName)
	processFiles, _ := filepath.Glob(filepath.Join(processesDir, "*.json"))
	for _, file := range processFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var processInfo struct {
			PID int `json:"pid"`
		}
		if err := json.Unmarshal(data, &processInfo); err != nil || processInfo.PID <= 0 {
			continue
		}
		if !isPIDRunning(processInfo.PID) {
			continue
		}
		envFile := fmt.Sprintf("/proc/%d/environ", processInfo.PID)
		envData, err := os.ReadFile(envFile)
		if err != nil {
			continue
		}
		for _, envVar := range strings.Split(string(envData), "\x00") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 || !validPortVars[parts[0]] {
				continue
			}
			if port, err := strconv.Atoi(parts[1]); err == nil {
				ports[parts[0]] = port
			}
		}
	}
	return ports
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

func (a *App) isProtected(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".vrooli", ".protected"))
	return err == nil
}

func (a *App) hasGit(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func (a *App) isCustomized(path string) bool {
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

func (a *App) readTail(path, lines string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	limit := 50
	if parsed, err := strconv.Atoi(strings.TrimSpace(lines)); err == nil && parsed > 0 {
		limit = parsed
	}
	parts := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > limit {
		parts = parts[len(parts)-limit:]
	}
	return strings.Join(parts, "\n"), nil
}

func (a *App) ListApps(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.AppsDir)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "Cannot read apps directory"})
		return
	}
	apps := []appInfo{}
	scenarios, _ := a.discoverRunningScenarios()
	scenarioMap := make(map[string]RunningScenario, len(scenarios))
	for _, item := range scenarios {
		scenarioMap[item.Name] = item
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".backups" {
			continue
		}
		appPath := filepath.Join(a.AppsDir, entry.Name())
		info, _ := entry.Info()
		item := appInfo{
			Name:       entry.Name(),
			Path:       appPath,
			Protected:  a.isProtected(appPath),
			HasGit:     a.hasGit(appPath),
			Customized: a.isCustomized(appPath),
			Modified:   info.ModTime(),
		}
		if scenarioData, ok := scenarioMap[entry.Name()]; ok {
			item.RuntimeStatus = scenarioData.Status
			if scenarioData.Status == "running" {
				item.Ports = make(map[string]interface{}, len(scenarioData.Ports))
				for key, value := range scenarioData.Ports {
					item.Ports[key] = value
				}
			}
		} else {
			item.RuntimeStatus = "stopped"
		}
		apps = append(apps, item)
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: apps})
}

func (a *App) ProtectApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	appPath := filepath.Join(a.AppsDir, name)
	if _, err := os.Stat(appPath); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "App not found"})
		return
	}
	protectDir := filepath.Join(appPath, ".vrooli")
	_ = os.MkdirAll(protectDir, 0o755)
	protectFile := filepath.Join(protectDir, ".protected")
	content := fmt.Sprintf("Protected on %s\n", time.Now().UTC().Format(time.RFC3339))
	_ = os.WriteFile(protectFile, []byte(content), 0o644)
	_ = json.NewEncoder(w).Encode(Response{Success: true})
}

func (a *App) StartApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	scenarioPath := filepath.Join(a.Root, "scenarios", name)
	if _, err := os.Stat(scenarioPath); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}
	if err := checkForkBomb(); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}
	if _, err := a.Scenarios.Start(name, lifecycle.StartOptions{}); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to start scenario %s: %v", name, err)})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: messageData{Message: fmt.Sprintf("Scenario %s started successfully", name)}})
}

func (a *App) StopApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := a.StopScenarioFn(name); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to stop app %s: %v", name, err)})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: messageData{Message: fmt.Sprintf("App %s stopped successfully", name)}})
}

func (a *App) RestartApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	scenarioPath := filepath.Join(a.Root, "scenarios", name)
	if _, err := os.Stat(scenarioPath); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}
	if _, err := a.Scenarios.Restart(name, lifecycle.StartOptions{}); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to restart scenario %s: %v", name, err)})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: messageData{Message: fmt.Sprintf("Scenario %s restarted successfully", name)}})
}

func (a *App) GetAppLogs(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "50"
	}
	view, exists, err := a.Scenarios.Status(name)
	if err != nil || !exists {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to get logs for %s", name)})
		return
	}
	logPath := filepath.Join(a.Home, ".vrooli", "logs", name+".log")
	output, err := a.readTail(logPath, lines)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to get logs: %v", err)})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: appLogsData{Logs: output, Scenario: view.Name}})
}

func (a *App) ListScenariosNative(w http.ResponseWriter, r *http.Request) {
	views, err := a.Scenarios.List()
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "Failed to read scenarios directory"})
		return
	}
	healthSnapshot := a.collectProcessHealthSnapshot()
	scenarios := make([]map[string]interface{}, 0, len(views))
	for _, item := range views {
		scenario := map[string]interface{}{
			"name":          item.Name,
			"display_name":  item.DisplayName,
			"description":   item.Description,
			"tags":          item.Tags,
			"status":        item.Status,
			"processes":     item.Processes,
			"ports":         item.Ports,
			"runtime":       item.Runtime,
			"health_status": item.Health,
		}
		if item.StartedAt != nil {
			scenario["started_at"] = item.StartedAt.Format(time.RFC3339)
		}
		scenarios = append(scenarios, scenario)
	}
	response := map[string]interface{}{
		"success": true,
		"data":    scenarios,
	}
	var warnings []map[string]interface{}
	if healthSnapshot.ZombieStatus != "healthy" && healthSnapshot.ZombieStatus != "normal" {
		warnings = append(warnings, map[string]interface{}{
			"type":    "zombies",
			"count":   healthSnapshot.ZombieCount,
			"status":  healthSnapshot.ZombieStatus,
			"emoji":   healthSnapshot.ZombieEmoji,
			"message": fmt.Sprintf("System has %d zombie processes %s", healthSnapshot.ZombieCount, healthSnapshot.ZombieEmoji),
		})
	}
	if healthSnapshot.OrphanStatus != "healthy" && healthSnapshot.OrphanStatus != "normal" {
		warnings = append(warnings, map[string]interface{}{
			"type":    "orphans",
			"count":   healthSnapshot.OrphanCount,
			"status":  healthSnapshot.OrphanStatus,
			"emoji":   healthSnapshot.OrphanEmoji,
			"message": fmt.Sprintf("System has %d orphaned processes %s", healthSnapshot.OrphanCount, healthSnapshot.OrphanEmoji),
		})
	}
	if len(warnings) > 0 {
		response["system_warnings"] = warnings
		response["system_health"] = healthSnapshot.OverallStatus
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (a *App) GetScenarioStatusNative(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	view, exists, err := a.Scenarios.Status(name)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
		return
	}
	if !exists {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: fmt.Sprintf("Scenario '%s' not found", name)})
		return
	}
	processes := []map[string]interface{}{}
	for i := 0; i < view.Processes; i++ {
		processes = append(processes, map[string]interface{}{
			"step_name": fmt.Sprintf("process-%d", i+1),
			"pid":       0,
			"status":    view.Status,
			"ports":     view.Ports,
		})
	}
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"name":            view.Name,
			"status":          view.Status,
			"phase":           "develop",
			"processes":       processes,
			"started_at":      view.StartedAt,
			"runtime":         view.Runtime,
			"allocated_ports": view.Ports,
			"health_status":   view.Health,
		},
	})
}

func (a *App) StartAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	result, err := a.StartAllScenariosFn()
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

func (a *App) StopAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	result, err := a.StopAllScenariosFn()
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

func (a *App) StopScenarioEndpoint(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := a.StopScenarioFn(name); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: messageData{Message: fmt.Sprintf("Scenario %s stopped successfully", name)}})
}

func (a *App) ListResources(w http.ResponseWriter, r *http.Request) {
	items, err := a.Resources.ListStatuses(true, false)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: items})
}

func (a *App) HandleLifecycle(w http.ResponseWriter, r *http.Request) {
	action := mux.Vars(r)["action"]
	if err := a.Project.RunProjectPhase(action, nil); err != nil {
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: lifecycleActionData{Action: action, Message: "completed"}})
}

func (a *App) ProcessMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := a.getEnhancedProcessMetrics()
	metrics["status"] = "healthy"
	if zombies, ok := metrics["zombie_processes"].(int); ok && zombies > 5 {
		metrics["status"] = "warning"
	}
	if orphans, ok := metrics["orphan_processes"].(int); ok && orphans > 3 {
		metrics["status"] = "warning"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": metrics})
}

func (a *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	healthSnapshot := a.collectProcessHealthSnapshot()
	overallStatus := "healthy"
	switch healthSnapshot.OverallStatus {
	case "critical":
		overallStatus = "unhealthy"
	case "warning", "unknown":
		overallStatus = "degraded"
	}
	if overallStatus != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      overallStatus,
		"version":     "1.0.0",
		"vrooli_root": a.Root,
		"apps_dir":    a.AppsDir,
		"system": map[string]interface{}{
			"zombie_processes": healthSnapshot.ZombieCount,
			"zombie_status":    healthSnapshot.ZombieStatus,
			"orphan_processes": healthSnapshot.OrphanCount,
			"orphan_status":    healthSnapshot.OrphanStatus,
			"process_health":   healthSnapshot.OverallStatus,
		},
	})
}

func (a *App) GetRunningApps(w http.ResponseWriter, r *http.Request) {
	scenarios, err := a.discoverRunningScenarios()
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "Failed to get running scenarios"})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: scenarios})
}

func (a *App) StartAllApps(w http.ResponseWriter, r *http.Request) {
	result, err := a.StartAllScenariosFn()
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to start scenarios: %v", err)})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

func (a *App) StopAllApps(w http.ResponseWriter, r *http.Request) {
	result, err := a.StopAllScenariosFn()
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: fmt.Sprintf("Failed to stop scenarios: %v", err)})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

func (a *App) GetDetailedAppStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	view, exists, err := a.Scenarios.Status(name)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{Error: "Failed to get scenario status"})
		return
	}
	if exists {
		_ = json.NewEncoder(w).Encode(Response{Success: true, Data: view})
		return
	}
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    stoppedAppData{Name: name, Status: "stopped", Processes: 0, Runtime: "N/A", Ports: map[string]int{}},
	})
}
