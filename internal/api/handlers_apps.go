package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const apiScenarioRunning = "running"

const (
	handlersAppsScenarioNotFound = "scenario_not_found"
)

func (a *App) isProtected(path string) bool {
	_, err := os.Stat(filepath.Join(path, repocontractmeta.ProjectConfigDir, ".protected"))
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
	out, _ := shell.Output(shell.Spec{
		Name: "git",
		Args: []string{"status", "--porcelain"},
		Dir:  path,
	})
	if len(out) > 0 {
		return true
	}
	out, _ = shell.Output(shell.Spec{
		Name: "git",
		Args: []string{"rev-list", "--count", "HEAD"},
		Dir:  path,
	})
	count := strings.TrimSpace(string(out))
	return count != "0" && count != "1"
}

const maxLogSnapshotBytes = 1 << 20
const maxLogSnapshotLines = 10000

var errLogSnapshotTooLarge = errors.New("requested log tail exceeds snapshot limits; request fewer lines or use scenario logs locally")
var errLogSnapshotLines = errors.New("log tail must request between 1 and 10000 lines")

func logSnapshotLines(lines string) (int, error) {
	if strings.TrimSpace(lines) == "" {
		return 50, nil
	}
	limit, err := strconv.Atoi(strings.TrimSpace(lines))
	if err != nil || limit < 1 || limit > maxLogSnapshotLines {
		return 0, errLogSnapshotLines
	}
	return limit, nil
}

// readTail reads a bounded suffix of one opened file. A rename does not switch
// the snapshot to a different generation, and truncation returns an error.
// Never present an incomplete first line as a complete requested tail.
func (a *App) readTail(path, lines string) (string, error) {
	limit, err := logSnapshotLines(lines)
	if err != nil {
		return "", err
	}
	// Reject known special files before opening them (in particular devices).
	// On Unix, nonblocking open also covers replacement by a FIFO between this
	// check and open. Validate the opened descriptor again below.
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("log snapshot requires a regular file")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|logSnapshotOpenFlags, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("log snapshot requires a regular file")
	}
	size := info.Size()
	offset := int64(0)
	if size > maxLogSnapshotBytes+1 {
		offset = size - maxLogSnapshotBytes - 1
	}
	data := make([]byte, int(size-offset))
	if len(data) > 0 {
		if _, err := file.ReadAt(data, offset); err != nil {
			return "", fmt.Errorf("read log snapshot: %w", err)
		}
	}
	truncated := offset > 0
	if truncated {
		boundary := bytes.IndexByte(data, '\n')
		if boundary < 0 {
			return "", errLogSnapshotTooLarge
		}
		data = data[boundary+1:]
	}
	// Keep the historical CRLF normalization and trailing-newline behavior.
	parts := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if truncated && len(parts) < limit {
		return "", errLogSnapshotTooLarge
	}
	if len(parts) > limit {
		parts = parts[len(parts)-limit:]
	}
	output := strings.Join(parts, "\n")
	if len(output) > maxLogSnapshotBytes {
		return "", errLogSnapshotTooLarge
	}
	return output, nil
}

func (a *App) ListApps(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.AppsDir)
	if err != nil {
		a.logError("App list request failed", err, logx.AttrOperation, "list_apps")
		respondError(w, newAPIError(http.StatusInternalServerError, "apps_directory_unreadable", "cannot read apps directory", err))
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
			if scenarioData.Status == apiScenarioRunning {
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
	a.logInfo("App list request completed", "count", len(apps))
	respondSuccess(w, http.StatusOK, apps)
}

func (a *App) ProtectApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	appPath := filepath.Join(a.AppsDir, name)
	if _, err := os.Stat(appPath); err != nil {
		a.logWarn("Protect app requested for missing app", "app", name)
		respondError(w, newAPIError(http.StatusNotFound, "app_not_found", "app not found", err))
		return
	}
	protectDir := filepath.Join(appPath, repocontractmeta.ProjectConfigDir)
	_ = os.MkdirAll(protectDir, tuning.PermDir)
	protectFile := filepath.Join(protectDir, ".protected")
	content := fmt.Sprintf("Protected on %s\n", time.Now().UTC().Format(time.RFC3339))
	_ = os.WriteFile(protectFile, []byte(content), tuning.PermFile)
	a.logInfo("App protection marker written", "app", name)
	respondSuccess(w, http.StatusOK, map[string]bool{"protected": true})
}

func (a *App) StartApp(w http.ResponseWriter, r *http.Request) {
	name, ok := a.scenarioNameForAction(w, r, "start")
	if !ok {
		return
	}
	if err := checkForkBomb(); err != nil {
		a.logError("Scenario start blocked by system overload protection", err, logx.AttrScenario, name)
		respondError(w, newAPIError(http.StatusServiceUnavailable, "system_overload", err.Error(), err))
		return
	}
	if _, err := a.Scenarios.Start(name, lifecycle.StartOptions{}); err != nil {
		a.logError("Scenario start request failed", err, logx.AttrScenario, name)
		respondError(w, newAPIError(http.StatusInternalServerError, "scenario_start_failed", fmt.Sprintf("failed to start scenario %s", name), err))
		return
	}
	a.logInfo("Scenario start request completed", logx.AttrScenario, name)
	respondSuccess(w, http.StatusOK, messageData{Message: fmt.Sprintf("Scenario %s started successfully", name)})
}

func (a *App) StopApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := a.ensureScenarioExists(name); err != nil {
		a.logWarn("App stop requested for missing app", "app", name)
		respondError(w, err)
		return
	}
	if err := a.StopScenarioFn(name); err != nil {
		status := http.StatusInternalServerError
		code := "app_stop_failed"
		if errors.Is(err, scenario.ErrNotFound) {
			status = http.StatusNotFound
			code = "app_not_found"
		}
		a.logError("App stop request failed", err, "app", name)
		respondError(w, newAPIError(status, code, fmt.Sprintf("failed to stop app %s", name), err))
		return
	}
	a.logInfo("App stop request completed", "app", name)
	respondSuccess(w, http.StatusOK, messageData{Message: fmt.Sprintf("App %s stopped successfully", name)})
}

func (a *App) RestartApp(w http.ResponseWriter, r *http.Request) {
	name, ok := a.scenarioNameForAction(w, r, "restart")
	if !ok {
		return
	}
	if _, err := a.Scenarios.Restart(name, lifecycle.StartOptions{}); err != nil {
		a.logError("Scenario restart request failed", err, logx.AttrScenario, name)
		respondError(w, newAPIError(http.StatusInternalServerError, "scenario_restart_failed", fmt.Sprintf("failed to restart scenario %s", name), err))
		return
	}
	a.logInfo("Scenario restart request completed", logx.AttrScenario, name)
	respondSuccess(w, http.StatusOK, messageData{Message: fmt.Sprintf("Scenario %s restarted successfully", name)})
}

func (a *App) GetAppLogs(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "50"
	}
	if _, err := logSnapshotLines(lines); err != nil {
		respondError(w, newAPIError(http.StatusBadRequest, "invalid_log_tail", err.Error(), err))
		return
	}
	view, exists, err := a.Scenarios.Status(name)
	if err != nil || !exists {
		status := http.StatusInternalServerError
		code := "scenario_logs_unavailable"
		if !exists {
			status = http.StatusNotFound
			code = handlersAppsScenarioNotFound
		}
		a.logError("Scenario logs request failed", err, logx.AttrScenario, name, logx.AttrOperation, "scenario_logs")
		respondError(w, newAPIError(status, code, fmt.Sprintf("failed to get logs for %s", name), err))
		return
	}
	logPath, err := process.ScenarioLifecycleLogPath(a.Home, name)
	if err != nil {
		respondError(w, newAPIError(http.StatusInternalServerError, "scenario_logs_read_failed", "failed to get logs", err))
		return
	}
	output, err := a.readTail(logPath, lines)
	if err != nil {
		if errors.Is(err, errLogSnapshotTooLarge) {
			respondError(w, newAPIError(http.StatusRequestEntityTooLarge, "log_snapshot_too_large", err.Error(), err))
			return
		}
		a.logError("Scenario log file read failed", err, logx.AttrScenario, name)
		respondError(w, newAPIError(http.StatusInternalServerError, "scenario_logs_read_failed", "failed to get logs", err))
		return
	}
	a.logInfo("Scenario logs request completed", logx.AttrScenario, name, "lines", lines)
	respondSuccess(w, http.StatusOK, appLogsData{Logs: output, Scenario: view.Name})
}

func (a *App) GetRunningApps(w http.ResponseWriter, r *http.Request) {
	scenarios, err := a.discoverRunningScenarios()
	if err != nil {
		a.logError("Running apps request failed", err, logx.AttrOperation, "running_apps")
		respondError(w, newAPIError(http.StatusInternalServerError, "running_scenarios_failed", "failed to get running scenarios", err))
		return
	}
	a.logInfo("Running apps request completed", "count", len(scenarios))
	respondSuccess(w, http.StatusOK, scenarios)
}

func (a *App) StartAllApps(w http.ResponseWriter, r *http.Request) {
	result, err := a.StartAllScenariosFn()
	if err != nil {
		a.logError("App start-all request failed", err, logx.AttrOperation, "start_all_apps")
		respondError(w, newAPIError(http.StatusInternalServerError, "start_all_failed", "failed to start scenarios", err))
		return
	}
	a.logInfo("App start-all request completed", "started", len(result.Started), "failed", len(result.Failed))
	respondSuccess(w, http.StatusOK, result)
}

func (a *App) StopAllApps(w http.ResponseWriter, r *http.Request) {
	result, err := a.StopAllScenariosFn()
	if err != nil {
		a.logError("App stop-all request failed", err, logx.AttrOperation, "stop_all_apps")
		respondError(w, newAPIError(http.StatusInternalServerError, "stop_all_failed", "failed to stop scenarios", err))
		return
	}
	a.logInfo("App stop-all request completed", "stopped", len(result.Stopped), "failed", len(result.Failed))
	respondSuccess(w, http.StatusOK, result)
}

func (a *App) GetDetailedAppStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	item, _, details, err := a.loadScenarioRuntime(name)
	if err != nil {
		if vroolierr.Code(err, "") == handlersAppsScenarioNotFound {
			a.logInfo("Detailed app status requested for missing app; returning stopped payload", "app", name)
			respondSuccess(w, http.StatusOK, stoppedAppData{Name: name, Status: "stopped", Processes: 0, Runtime: "N/A", Ports: map[string]int{}})
			return
		}
		a.logError("Detailed app status request failed", err, "app", name)
		respondError(w, err)
		return
	}
	a.logInfo("Detailed app status request completed", "app", name, logx.AttrStatus, details.Status)
	respondSuccess(w, http.StatusOK, map[string]any{
		"name":          item.Slug,
		"status":        details.Status,
		"processes":     details.Processes,
		"runtime":       details.Runtime,
		"ports":         details.Ports,
		"started_at":    details.StartedAt,
		"health_status": details.Health,
	})
}
