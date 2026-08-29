package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func (a *App) ListScenariosNative(w http.ResponseWriter, r *http.Request) {
	report, err := a.Scenarios.ListReport()
	if err != nil {
		a.logError("Scenario list request failed", err, logx.AttrOperation, "list_scenarios")
		respondError(w, newAPIError(http.StatusInternalServerError, "scenario_list_failed", "failed to read scenarios directory", err))
		return
	}
	healthSnapshot := a.collectProcessHealthSnapshot()
	scenarios := make([]map[string]interface{}, 0, len(report.Items))
	for _, item := range report.Items {
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
	if len(report.Failures) > 0 {
		response["discovery_failures"] = report.Failures
	}
	var warnings []map[string]interface{}
	if healthSnapshot.ZombieStatus != scenarioruntime.HealthStatusHealthy && healthSnapshot.ZombieStatus != maintenance.ProcessHealthNormal {
		warnings = append(warnings, map[string]interface{}{
			"type":    "zombies",
			"count":   healthSnapshot.ZombieCount,
			"status":  healthSnapshot.ZombieStatus,
			"emoji":   healthSnapshot.ZombieEmoji,
			"message": fmt.Sprintf("System has %d zombie processes %s", healthSnapshot.ZombieCount, healthSnapshot.ZombieEmoji),
		})
	}
	if healthSnapshot.OrphanStatus != scenarioruntime.HealthStatusHealthy && healthSnapshot.OrphanStatus != maintenance.ProcessHealthNormal {
		warnings = append(warnings, map[string]interface{}{
			"type":    "orphans",
			"count":   healthSnapshot.OrphanCount,
			"status":  healthSnapshot.OrphanStatus,
			"emoji":   healthSnapshot.OrphanEmoji,
			"message": fmt.Sprintf("System has %d orphaned processes %s", healthSnapshot.OrphanCount, healthSnapshot.OrphanEmoji),
		})
	}
	if len(warnings) > 0 {
		a.logWarn("Scenario list request returned system warnings", "warnings", len(warnings), logx.AttrStatus, healthSnapshot.OverallStatus)
		response["system_warnings"] = warnings
		response["system_health"] = healthSnapshot.OverallStatus
	}
	a.logInfo("Scenario list request completed", "count", len(report.Items), "discovery_failures", len(report.Failures))
	respondJSON(w, http.StatusOK, response)
}

func (a *App) GetScenarioStatusNative(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	item, runtime, details, err := a.loadScenarioRuntime(name)
	if err != nil {
		a.logError("Scenario status request failed", err, logx.AttrScenario, name, logx.AttrOperation, "scenario_status")
		respondError(w, err)
		return
	}
	processes := make([]apiProcessData, 0, len(details.ProcessInfo))
	for _, record := range details.ProcessInfo {
		startedAt := record.StartedAt
		processes = append(processes, apiProcessData{
			ProcessID:  record.ProcessID,
			StepName:   record.Step,
			PID:        record.PID,
			PGID:       record.PGID,
			Status:     record.Status,
			Phase:      record.Phase,
			Command:    record.Command,
			WorkingDir: record.WorkingDir,
			LogFile:    record.LogFile,
			Port:       record.Port,
			StartedAt:  &startedAt,
		})
	}
	respondSuccess(w, http.StatusOK, map[string]interface{}{
		"name":            item.Slug,
		"status":          details.Status,
		"phase":           "develop",
		"processes":       processes,
		"started_at":      details.StartedAt,
		"runtime":         details.Runtime,
		"allocated_ports": details.Ports,
		"health_status":   details.Health,
		"process_count":   runtime.ProcessCount,
	})
	a.logInfo("Scenario status request completed", logx.AttrScenario, name, logx.AttrStatus, details.Status, logx.AttrProcesses, runtime.ProcessCount)
}

func (a *App) StartAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	result, err := a.StartAllScenariosFn()
	if err != nil {
		a.logError("Scenario start-all request failed", err, logx.AttrOperation, "start_all_scenarios")
		respondError(w, newAPIError(http.StatusInternalServerError, "start_all_failed", "failed to start scenarios", err))
		return
	}
	a.logInfo("Scenario start-all request completed", "started", len(result.Started), "failed", len(result.Failed))
	respondSuccess(w, http.StatusOK, result)
}

func (a *App) StopAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	result, err := a.StopAllScenariosFn()
	if err != nil {
		a.logError("Scenario stop-all request failed", err, logx.AttrOperation, "stop_all_scenarios")
		respondError(w, newAPIError(http.StatusInternalServerError, "stop_all_failed", "failed to stop scenarios", err))
		return
	}
	a.logInfo("Scenario stop-all request completed", "stopped", len(result.Stopped), "failed", len(result.Failed))
	respondSuccess(w, http.StatusOK, result)
}

func (a *App) StopScenarioEndpoint(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := a.ensureScenarioExists(name); err != nil {
		a.logWarn("Scenario stop requested for missing scenario", logx.AttrScenario, name)
		respondError(w, err)
		return
	}
	if err := a.StopScenarioFn(name); err != nil {
		status := http.StatusInternalServerError
		code := "scenario_stop_failed"
		if errors.Is(err, scenario.ErrNotFound) {
			status = http.StatusNotFound
			code = handlersAppsScenarioNotFound
		}
		a.logError("Scenario stop request failed", err, logx.AttrScenario, name)
		respondError(w, newAPIError(status, code, fmt.Sprintf("failed to stop scenario %s", name), err))
		return
	}
	a.logInfo("Scenario stop request completed", logx.AttrScenario, name)
	respondSuccess(w, http.StatusOK, messageData{Message: fmt.Sprintf("Scenario %s stopped successfully", name)})
}
