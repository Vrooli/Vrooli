package health

import (
	"net/http"

	"data-backup-manager/internal/database"
	"data-backup-manager/internal/module"

	"github.com/gorilla/mux"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
func Module(pinger database.Pinger, service, version string) module.Module {
	return ModuleWithPosture(pinger, service, version, nil, nil)
}

// ModuleWithPosture is the production variant that additionally reports backup
// posture: when posture is non-nil, an Optional "backups" check degrades the
// status (HTTP 200, readiness true) on overdue/failed backups and emits a
// posture event through events. main.go wires a runs-backed posture adapter.
func ModuleWithPosture(pinger database.Pinger, service, version string, posture BackupPosture, events PostureEventSink) module.Module {
	h := NewHandler(Deps{Pinger: pinger, Service: service, Version: version, Posture: posture, Events: events})
	return module.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/health", h).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — health is stateless, no tables to own. The
// modules registry includes this re-export anyway so adding a stateful
// domain later is a uniform "create the file, return the SQL" pattern
// instead of "remember to also add a Schema() function."
func Schema() string { return "" }
