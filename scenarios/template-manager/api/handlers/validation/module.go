package validation

import (
	"log"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/validation/validation_v1connect"
)

func Module(db *database.RoutedDB, logger *log.Logger) module.Module {
	repo := catalog.NewSQLiteRepository(db)
	runner, err := validationrunner.NewEngineRunner("")
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Printf("validation module: engine runner unavailable: %v", err)
	}
	service := validationrunner.NewService(repo, runner)
	path, handler := validationconnect.NewValidationRunServiceHandler(NewConnectHandler(Deps{
		Repository: repo,
		Service:    service,
		Logger:     logger,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
