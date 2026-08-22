package server

// DOC: docs/reference/api-endpoints.md

import (
	"net/http"
	"net/http/pprof"

	capacityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity/capacityconnect"
	devicegraphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph/devicegraphconnect"
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

func buildRouter(cfg *config.Config, health *handlers.HealthHandler, metrics *handlers.MetricsHandler, investigation *handlers.InvestigationHandler, report *handlers.ReportHandler, settings *handlers.SettingsHandler, maintenance *handlers.MaintenanceHandler, capacity *handlers.CapacityHandler, forensicsH *handlers.ForensicsHandler, logsH *handlers.LogsHandler, diskPressure *handlers.DiskPressureHandler, deviceGraph *handlers.DeviceGraphHandler) http.Handler {
	r := http.NewServeMux()
	mountDebugRoutes(cfg, r)
	mountConnectRoutes(r, health, metrics, report, settings, capacity, maintenance, investigation, deviceGraph)

	// Keep the unprefixed probe literal explicit for lifecycle validators and
	// external supervisors; the versioned route remains an API alias.
	r.HandleFunc("/health", health.Handle)
	r.HandleFunc("GET /api/v1/health", health.Handle)
	r.HandleFunc("GET /api/v1/metrics/pressure", metrics.HandleGetPressureSnapshot)
	// Documented in docs/reference/api-endpoints.md and implemented by
	// HandleGetMetricsTimeline, but never mounted — the endpoint 404'd while
	// the handler sat unreachable. The typed Connect route stays canonical.
	r.HandleFunc("GET /api/v1/metrics/timeline", metrics.HandleGetMetricsTimeline)
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

	// Disk-pressure operator surface: current usage, the configured threshold,
	// and every violation the evaluation loop has recorded.
	r.HandleFunc("GET /api/v1/disk-pressure", diskPressure.Handle)

	// REST aliases used by the dashboard for script discovery and execution.
	// The typed Connect routes above remain the canonical CLI contract.
	r.HandleFunc("GET /api/v1/investigations/scripts", investigation.HandleListScripts)
	r.HandleFunc("GET /api/v1/investigations/scripts/{id}", investigation.HandleGetScript)
	r.HandleFunc("POST /api/v1/investigations/scripts/{id}/execute", investigation.HandleExecuteScript)

	return r
}

func mountConnectRoutes(r *http.ServeMux, health *handlers.HealthHandler, metrics *handlers.MetricsHandler, report *handlers.ReportHandler, settings *handlers.SettingsHandler, capacity *handlers.CapacityHandler, maintenance *handlers.MaintenanceHandler, investigation *handlers.InvestigationHandler, deviceGraph *handlers.DeviceGraphHandler) {
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
	deviceGraphPath, deviceGraphHandler := devicegraphconnect.NewDeviceGraphServiceHandler(deviceGraph)
	r.Handle(deviceGraphPath, deviceGraphHandler)
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
