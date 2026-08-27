package api

import (
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
	name := mux.Vars(r)["name"]
	if err := a.ensureScenarioExists(name); err != nil {
		a.logWarn("Scenario start requested for missing scenario", logx.AttrScenario, name)
		respondError(w, err)
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
	name := mux.Vars(r)["name"]
	if err := a.ensureScenarioExists(name); err != nil {
		a.logWarn("Scenario restart requested for missing scenario", logx.AttrScenario, name)
		respondError(w, err)
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
