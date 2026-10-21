package library

import (
	"device-studio/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
)

func ModuleWithService(service Service) module.Module {
	return ModuleWithIntake(service, nil, nil, nil)
}

type intakeModuleService interface {
	IntakeService
	intakeOperations
}

func ModuleWithIntake(service Service, intake intakeModuleService, db *database.RoutedDB, roots *filerouting.RoutedRoots) module.Module {
	path, handler := connect.NewLibraryCatalogServiceHandler(NewConnectHandlerWithIntake(service, intake))
	return module.Module{Name: "library", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		if intake != nil {
			r.Handle("/api/v1/library/intake-candidates", newIntakeUploadHandler(intake, db, roots, nil)).Methods("POST")
			r.Handle("/api/v1/library/intake-candidates/{candidate_id}/revisions/{revision}/content", newIntakeContentHandler(intake, db, roots, nil)).Methods("GET")
		}
	}, Endpoints: Endpoints}
}
