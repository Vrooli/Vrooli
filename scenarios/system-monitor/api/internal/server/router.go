package server

import (
	"github.com/gorilla/mux"

	"system-monitor-api/internal/handlers"
)

func buildRouter(health *handlers.HealthHandler, metrics *handlers.MetricsHandler, investigation *handlers.InvestigationHandler, report *handlers.ReportHandler, settings *handlers.SettingsHandler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", health.Handle).Methods("GET")
	r.HandleFunc("/api/v1/health", health.Handle).Methods("GET")

	r.HandleFunc("/api/v1/metrics/current", metrics.GetCurrentMetrics).Methods("GET")
	r.HandleFunc("/api/v1/metrics/detailed", metrics.GetDetailedMetrics).Methods("GET")
	r.HandleFunc("/api/v1/metrics/processes", metrics.GetProcessMonitor).Methods("GET")
	r.HandleFunc("/api/v1/metrics/infrastructure", metrics.GetInfrastructureMonitor).Methods("GET")

	r.HandleFunc("/api/v1/investigations", investigation.ListInvestigations).Methods("GET")
	r.HandleFunc("/api/v1/investigations/latest", investigation.GetLatestInvestigation).Methods("GET")
	r.HandleFunc("/api/v1/investigations/trigger", investigation.TriggerInvestigation).Methods("POST")
	r.HandleFunc("/api/v1/investigations/agent/spawn", investigation.TriggerInvestigation).Methods("POST")
	r.HandleFunc("/api/v1/investigations/agent/current", investigation.GetCurrentAgent).Methods("GET")
	r.HandleFunc("/api/v1/investigations/cooldown", investigation.GetCooldownStatus).Methods("GET")
	r.HandleFunc("/api/v1/investigations/cooldown/reset", investigation.ResetCooldown).Methods("POST")
	r.HandleFunc("/api/v1/investigations/cooldown/period", investigation.UpdateCooldownPeriod).Methods("PUT")
	r.HandleFunc("/api/v1/investigations/triggers", investigation.GetTriggers).Methods("GET")
	r.HandleFunc("/api/v1/investigations/triggers/{id}", investigation.UpdateTrigger).Methods("PUT")
	r.HandleFunc("/api/v1/investigations/triggers/{id}/threshold", investigation.UpdateTriggerThreshold).Methods("PUT")
	r.HandleFunc("/api/v1/investigations/scripts", investigation.ListScripts).Methods("GET")
	r.HandleFunc("/api/v1/investigations/scripts/{id}", investigation.GetScript).Methods("GET")
	r.HandleFunc("/api/v1/investigations/scripts/{id}/execute", investigation.ExecuteScript).Methods("POST")
	r.HandleFunc("/api/v1/investigations/{id}", investigation.GetInvestigation).Methods("GET")
	r.HandleFunc("/api/v1/investigations/{id}/status", investigation.UpdateInvestigationStatus).Methods("PUT")
	r.HandleFunc("/api/v1/investigations/{id}/findings", investigation.UpdateInvestigationFindings).Methods("PUT")
	r.HandleFunc("/api/v1/investigations/{id}/progress", investigation.UpdateInvestigationProgress).Methods("PUT")
	r.HandleFunc("/api/v1/investigations/{id}/step", investigation.AddInvestigationStep).Methods("POST")

	r.HandleFunc("/api/v1/reports/generate", report.GenerateReport).Methods("POST")
	r.HandleFunc("/api/v1/reports/{id}", report.GetReport).Methods("GET")
	r.HandleFunc("/api/v1/reports", report.ListReports).Methods("GET")

	r.HandleFunc("/api/v1/settings", settings.GetSettings).Methods("GET")
	r.HandleFunc("/api/v1/settings", settings.UpdateSettings).Methods("PUT")
	r.HandleFunc("/api/v1/settings/reset", settings.ResetSettings).Methods("POST")

	r.HandleFunc("/api/v1/maintenance/state", settings.GetMaintenanceState).Methods("GET")
	r.HandleFunc("/api/v1/maintenance/state", settings.SetMaintenanceState).Methods("POST")

	r.HandleFunc("/api/v1/agent/config", investigation.GetAgentConfig).Methods("GET")
	r.HandleFunc("/api/v1/agent/config", investigation.UpdateAgentConfig).Methods("PUT")
	r.HandleFunc("/api/v1/agent/runners", investigation.GetAvailableRunners).Methods("GET")
	r.HandleFunc("/api/v1/agent/status", investigation.GetAgentStatus).Methods("GET")

	return r
}
