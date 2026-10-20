package shots

import (
	"log"

	"device-studio/internal/module"
	domain "device-studio/internal/projects"
	shotdomain "device-studio/internal/shots"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

func Module(db *database.RoutedDB, logger *log.Logger) module.Module {
	repo := domain.NewSQLiteRepository(db, nil)
	mediaRepo := repo.(domain.SourceMediaRepository)
	return ModuleWithRepository(shotdomain.NewSQLiteRepository(db), domain.NewService(repo, nil), mediaRepo, logger)
}

func ModuleWithRepository(repo shotdomain.Repository, projectReader domain.Service, mediaRepo domain.SourceMediaRepository, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	service := shotdomain.NewService(repo, projectReader, mediaRepo, nil, nil)
	path, handler := shotsconnect.NewShotsServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger}))
	return module.Module{Name: "shots", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return shotdomain.Schema() }
