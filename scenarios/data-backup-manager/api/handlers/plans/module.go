package plans

import (
	"log"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans/plans_v1connect"

	internalplans "data-backup-manager/internal/plans"
)

// Module returns the plans domain's contribution to the API: the generated
// PlansService Connect-RPC handler. Production callers use this entry point;
// it constructs the repository → service → handler chain internally so
// per-domain dependencies never appear on server.Deps.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	repo := internalplans.NewSQLiteRepository(db, clk)
	svc := internalplans.NewService(repo)
	connectPath, connectHandler := plansconnect.NewPlansServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "plans",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the plans domain SQL so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalplans.Schema() }
