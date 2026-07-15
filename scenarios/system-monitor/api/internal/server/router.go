package server

// DOC: docs/reference/api-endpoints.md

import (
	"net/http"
	"net/http/pprof"

	capacityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity/capacityconnect"
	healthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/health/healthconnect"
	investigationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations/investigationsconnect"
	maintenanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance/maintenanceconnect"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"
	reportsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports/reportsconnect"
	scriptsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts/scriptsconnect"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings/settingsconnect"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers"
)

func buildRouter(cfg *config.Config, health *handlers.HealthHandler, metrics *handlers.MetricsHandler, investigation *handlers.InvestigationHandler, report *handlers.ReportHandler, settings *handlers.SettingsHandler, maintenance *handlers.MaintenanceHandler, capacity *handlers.CapacityHandler, forensicsH *handlers.ForensicsHandler, logsH *handlers.LogsHandler) http.Handler {
	r := http.NewServeMux()
	mountDebugRoutes(cfg, r)
	mountConnectRoutes(r, health, metrics, report, settings, capacity, maintenance, investigation)

	r.HandleFunc("GET /health", health.Handle)
	r.HandleFunc("GET /api/v1/health", health.Handle)
	r.HandleFunc("GET /api/v1/metrics/pressure", metrics.HandleGetPressureSnapshot)
	r.HandleFunc("GET /api/v1/forensics/processes", metrics.HandleGetProcessTimeline)
	r.HandleFunc("GET /api/v1/forensics/gpu", metrics.HandleGetGPUHistory)
	r.HandleFunc("GET /api/v1/forensics/pressure", metrics.HandleGetPressureHistory)

	// Crash-forensics + logs surfaces (plain JSON; see forensics.go header).
	r.HandleFunc("GET /api/v1/forensics/pstore", forensicsH.Pstore)
	r.HandleFunc("GET /api/v1/forensics/boot-history", forensicsH.BootHistory)
	r.HandleFunc("GET /api/v1/forensics/mce", forensicsH.MCE)
	r.HandleFunc("GET /api/v1/forensics/summary", forensicsH.Summary)
	r.HandleFunc("GET /api/v1/logs", logsH.Logs)
	r.HandleFunc("GET /api/v1/logs/units", logsH.Units)
	r.HandleFunc("GET /api/v1/logs/boots", logsH.Boots)

	return r
}

func mountConnectRoutes(r *http.ServeMux, health *handlers.HealthHandler, metrics *handlers.MetricsHandler, report *handlers.ReportHandler, settings *handlers.SettingsHandler, capacity *handlers.CapacityHandler, maintenance *handlers.MaintenanceHandler, investigation *handlers.InvestigationHandler) {
	healthPath, healthHandler := healthconnect.NewHealthServiceHandler(health)
	r.Handle(healthPath, healthHandler)
	metricsPath, metricsHandler := metricsconnect.NewMetricsServiceHandler(metrics)
	r.Handle(metricsPath, metricsHandler)
	reportsPath, reportsHandler := reportsconnect.NewReportsServiceHandler(report)
	r.Handle(reportsPath, reportsHandler)
	settingsPath, settingsHandler := settingsconnect.NewSettingsServiceHandler(settings)
	r.Handle(settingsPath, settingsHandler)
	capacityPath, capacityHandler := capacityconnect.NewCapacityServiceHandler(capacity)
	r.Handle(capacityPath, capacityHandler)
	maintenancePath, maintenanceHandler := maintenanceconnect.NewMaintenanceServiceHandler(maintenance)
	r.Handle(maintenancePath, maintenanceHandler)
	investigationsPath, investigationsHandler := investigationsconnect.NewInvestigationsServiceHandler(investigation)
	r.Handle(investigationsPath, investigationsHandler)
	scriptsPath, scriptsHandler := scriptsconnect.NewScriptsServiceHandler(investigation)
	r.Handle(scriptsPath, scriptsHandler)
}

func mountDebugRoutes(cfg *config.Config, r *http.ServeMux) {
	if cfg == nil || cfg.IsProduction() {
		return
	}

	r.HandleFunc("GET /debug/pprof/", pprof.Index)
	r.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	r.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("POST /debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}
