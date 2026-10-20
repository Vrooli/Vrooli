package renderjobs

import (
	"log"

	"device-studio/internal/jobs"
	"device-studio/internal/module"
	"device-studio/internal/projects"
	"device-studio/internal/renderjobs"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/shots"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	renderjobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

func Module(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) module.Module {
	if logger == nil { logger = log.Default() }
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil { logger.Fatalf("render media storage: %v", err) }
	media := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: projectRepo.(projects.SourceMediaRepository), Store: mediaStore})
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, projectRepo.(projects.SourceMediaRepository), nil, nil)
	jobService := jobs.NewService(jobs.NewSQLiteRepository(db), nil)
	blobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil { logger.Fatalf("render output storage: %v", err) }
	coordinator, err := renderjobs.New(renderjobs.Dependencies{Shots: shotService, Media: media, Jobs: jobService, Blobs: blobs, Roots: roots})
	if err != nil { logger.Fatalf("render jobs coordinator: %v", err) }
	return ModuleWithCoordinator(coordinator, jobService, logger)
}

func ModuleWithCoordinator(coordinator *renderjobs.Coordinator, jobs jobs.Service, logger *log.Logger) module.Module {
	if logger == nil { logger = log.Default() }
	path, handler := renderjobsconnect.NewRenderJobsServiceHandler(NewConnectHandler(Deps{Coordinator: coordinator, Jobs: jobs, Logger: logger}))
	content := newContentHandler(coordinator, logger)
	return module.Module{Name: "renderjobs", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.Handle("/api/v1/projects/{project_id}/renders/{render_id}/content", content).Methods("GET")
	}, Endpoints: Endpoints}
}
