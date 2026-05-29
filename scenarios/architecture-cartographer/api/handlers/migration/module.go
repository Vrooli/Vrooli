package migration

import (
	"architecture-cartographer/internal/migration"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration/migration_v1connect"
)

// Module returns the migration domain's contribution to the API router.
func Module(svc migration.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := migration_v1connect.NewMigrationServiceHandler(h)
	return module.Module{
		Name: "migration",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the migration domain's SQL contribution.
func Schema() string { return migration.Schema() }
