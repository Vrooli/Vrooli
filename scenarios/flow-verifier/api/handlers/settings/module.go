// Package settings wires the Connect-RPC SettingsService for per-principal
// UI/CLI preferences. Reads and writes flow through the SQLite-backed
// settings domain (internal/settings). The single 'local' principal is
// the only row this scenario ever touches in v1.
package settings

import (
	"database/sql"
	"log"

	"flow-verifier/internal/clock"
	"flow-verifier/internal/module"
	"flow-verifier/internal/settings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings/settings_v1connect"
)

// Module returns the settings domain's Connect-RPC contribution. The
// caller is expected to have applied modules.AllSchemas to the same
// *sql.DB; this handler does not run migrations.
func Module(db *sql.DB, clk clock.Clock) module.Module {
	svc := settings.NewService(settings.NewSQLiteRepository(db, clk))
	return ModuleWithService(svc, log.Default())
}

// ModuleWithService is the test-friendly variant that accepts an
// already-constructed *settings.Service.
func ModuleWithService(svc *settings.Service, logger *log.Logger) module.Module {
	path, handler := settingsconnect.NewSettingsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "settings",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports settings.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler.
func Schema() string { return settings.Schema() }
