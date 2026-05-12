// Package settings wires the HTTP surface for per-principal UI/CLI
// preferences. Reads and writes flow through the SQLite-backed settings
// domain (internal/settings). The single 'local' principal is the only
// row this scenario ever touches in v1.
package settings

import (
	"database/sql"
	"net/http"

	"flow-verifier/internal/clock"
	"flow-verifier/internal/module"
	"flow-verifier/internal/settings"

	"github.com/gorilla/mux"
)

// Module returns the settings domain's HTTP contribution. The caller
// is expected to have applied modules.AllSchemas to the same *sql.DB;
// this handler does not run migrations.
func Module(db *sql.DB, clk clock.Clock) module.Module {
	svc := settings.NewService(settings.NewSQLiteRepository(db, clk))
	return ModuleWithService(svc)
}

// ModuleWithService is the test-friendly variant that accepts an
// already-constructed *settings.Service.
func ModuleWithService(svc *settings.Service) module.Module {
	return module.Module{
		Name: "settings",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/settings", getHandler(svc)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/settings", putHandler(svc)).Methods(http.MethodPut)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports settings.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler.
func Schema() string { return settings.Schema() }
