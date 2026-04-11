package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (a *App) ListResources(w http.ResponseWriter, r *http.Request) {
	items, err := a.Resources.ListStatuses(true, false)
	if err != nil {
		respondError(w, newAPIError(http.StatusInternalServerError, "resource_list_failed", "failed to list resources", err))
		return
	}
	respondSuccess(w, http.StatusOK, items)
}

func (a *App) HandleLifecycle(w http.ResponseWriter, r *http.Request) {
	action := mux.Vars(r)["action"]
	if err := a.Project.RunProjectPhase(action, nil); err != nil {
		respondError(w, newAPIError(http.StatusInternalServerError, "project_phase_failed", fmt.Sprintf("project lifecycle %s failed", action), err))
		return
	}
	respondSuccess(w, http.StatusOK, lifecycleActionData{Action: action, Message: "completed"})
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
	respondSuccess(w, http.StatusOK, metrics)
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
	status := http.StatusOK
	if overallStatus != "healthy" {
		status = http.StatusServiceUnavailable
	}
	respondJSON(w, status, map[string]interface{}{
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
