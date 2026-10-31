package targets

import (
	"log"

	"data-backup-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets/targets_v1connect"

	internaltargets "data-backup-manager/internal/targets"
)

// Module returns the targets domain's contribution to the API: the generated
// TargetsService Connect-RPC handler. Production callers use this entry point;
// it constructs the repository → service → handler chain internally so
// per-domain dependencies never appear on server.Deps.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	repo := internaltargets.NewSQLiteRepository(db, clk)
	svc := internaltargets.NewService(repo)
	connectPath, connectHandler := targetsconnect.NewTargetsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "targets",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the targets domain SQL so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internaltargets.Schema() }
