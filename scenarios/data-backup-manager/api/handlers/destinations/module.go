package destinations

import (
	"log"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/module"
	"data-backup-manager/internal/sysmounts"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"

	internaldestinations "data-backup-manager/internal/destinations"
)

// Module returns the destinations domain's contribution to the API: the
// generated DestinationsService Connect-RPC handler. Production callers use
// this entry point; it constructs the repository → service → handler chain
// internally so per-domain dependencies never appear on server.Deps.
func Module(db *database.RoutedDB, clk clock.Clock, eng engine.KopiaEngine, protectedRoot string, logger *log.Logger) module.Module {
	repo := internaldestinations.NewSQLiteRepository(db, clk)
	svc := internaldestinations.NewService(repo, eng, &internaldestinations.FSBundleWriter{}, protectedRoot)
	readinessSvc := destinationreadiness.NewService(
		destinationreadiness.NewReadOnlyInspector(sysmounts.New()),
		destinationreadiness.NewLocalPreparer(),
	)
	connectPath, connectHandler := destinationsconnect.NewDestinationsServiceHandler(NewConnectHandler(Deps{
		Service:   svc,
		Readiness: readinessSvc,
		Logger:    logger,
	}))
	return module.Module{
		Name: "destinations",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the destinations domain SQL so the modules registry
// collects endpoints and schema from one symbol per handler package.
func Schema() string { return internaldestinations.Schema() }
