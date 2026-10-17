package coverage

import (
	"log"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage/coverage_v1connect"
)

// Module returns the coverage domain's contribution to the API: the generated
// CoverageService Connect-RPC handler. Coverage composes cross-domain seams
// (discovery suggestions + targets/plans/runs/restores catalogs), so the
// fully-wired service is built in the composition root (main.go) and passed in —
// mirroring discovery/restores/runs. Coverage owns no table, so it exposes no
// Schema().
func Module(svc coverage.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := coverageconnect.NewCoverageServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "coverage",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
