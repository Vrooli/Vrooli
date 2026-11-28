package policy

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy/policyconnect"
	"lighting-control/internal/control"
	"lighting-control/internal/module"
	internalpolicy "lighting-control/internal/policy"
	"lighting-control/internal/policy/sources"
)

func Module(db *database.RoutedDB, clock schedule.Clock, controlService *control.Service, logger *log.Logger) module.Module {
	store := internalpolicy.NewSQLiteStore(db)
	service := NewService(store, sources.New(), controlService)
	return ModuleWithService(service, logger)
}

func ModuleWithService(service *Service, logger *log.Logger) module.Module {
	path, handler := policyconnect.NewPolicyServiceHandler(NewConnectHandler(service, logger))
	return module.Module{
		Name: "policy",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return internalpolicy.Schema() }
