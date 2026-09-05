package safety

import (
	"log"

	"data-backup-manager/internal/module"
	internalsafety "data-backup-manager/internal/safety"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety/safety_v1connect"
)

// Module returns the safety domain's contribution to the API. The service is
// constructed in main.go because it composes the destinations/plans/runs/targets
// services through adapters. The domain owns no tables, so it exports no Schema.
func Module(svc internalsafety.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := safetyconnect.NewSafetyServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "safety",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
