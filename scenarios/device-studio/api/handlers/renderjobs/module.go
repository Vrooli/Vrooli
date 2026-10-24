package renderjobs

import (
	"context"
	"fmt"
	"log"

	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/module"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/renderjobs"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/shots"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	renderjobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

func Module(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) module.Module {
	m, _, err := Build(db, roots, logger)
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Fatalf("render jobs module: %v", err)
	}
	return m
}

func Build(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) (module.Module, *renderjobs.Pump, error) {
	m, pump, _, err := BuildWithCoordinator(db, roots, logger)
	return m, pump, err
}

func BuildWithCoordinator(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) (module.Module, *renderjobs.Pump, *renderjobs.Coordinator, error) {
	return BuildWithCoordinatorAndPhysicalResolver(db, roots, logger, nil)
}

func BuildWithCoordinatorAndPhysicalResolver(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger, physical shots.PhysicalResolver) (module.Module, *renderjobs.Pump, *renderjobs.Coordinator, error) {
	return BuildWithCoordinatorAndPhysicalOwners(db, roots, logger, physical, nil, nil, nil)
}

func BuildWithCoordinatorAndPhysicalOwners(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger, physical shots.PhysicalResolver, artifacts renderjobs.PhysicalArtifactResolver, profile func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error), adapter func() []byte) (module.Module, *renderjobs.Pump, *renderjobs.Coordinator, error) {
	return BuildWithCoordinatorAndCreationOwners(db, roots, logger, physical, artifacts, profile, adapter, nil)
}

func BuildWithCoordinatorAndCreationOwners(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger, physical shots.PhysicalResolver, artifacts renderjobs.PhysicalArtifactResolver, profile func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error), adapter func() []byte, generatedIntake interface {
	PersistGenerated(context.Context, library.IntakeClaims, string, library.CreationLineage, []byte) (library.IntakeCandidate, error)
}) (module.Module, *renderjobs.Pump, *renderjobs.Coordinator, error) {
	if logger == nil {
		logger = log.Default()
	}
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		return module.Module{}, nil, nil, fmt.Errorf("render media storage: %w", err)
	}
	media := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: projectRepo.(projects.SourceMediaRepository), Store: mediaStore})
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, media, nil, nil)
	shots.SetPhysicalResolver(shotService, physical)
	jobRepo := jobs.NewSQLiteRepository(db)
	jobService := jobs.NewService(jobRepo, nil)
	scopeRepo, ok := jobRepo.(jobs.RunScopeRepository)
	if !ok {
		return module.Module{}, nil, nil, fmt.Errorf("render jobs repository does not support durable parent scopes")
	}
	scopeService := jobs.NewRunScopeService(scopeRepo, nil)
	blobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		return module.Module{}, nil, nil, fmt.Errorf("render output storage: %w", err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{Shots: shotService, Media: media, Jobs: jobService, Scopes: scopeService, Blobs: blobs, Roots: roots, InspectionProfile: rendering.ResolveInspectionProfile, PhysicalArtifacts: artifacts, PhysicalProfile: profile, PhysicalAdapter: adapter, GeneratedIntake: generatedIntake})
	if err != nil {
		return module.Module{}, nil, nil, fmt.Errorf("render jobs coordinator: %w", err)
	}
	shots.SetVideoSampleVerifier(shotService, coordinator)
	pump := renderjobs.NewPump(coordinator, jobService, db, roots, logger)
	return ModuleWithCoordinator(coordinator, jobService, pump, db, roots, logger), pump, coordinator, nil
}

func Schema() string { return jobs.Schema() }

func ModuleWithCoordinator(coordinator *renderjobs.Coordinator, jobs jobs.Service, activator RouteActivator, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	path, handler := renderjobsconnect.NewRenderJobsServiceHandler(NewConnectHandler(Deps{Coordinator: coordinator, Jobs: jobs, Activator: activator, DB: db, Roots: roots, Logger: logger}))
	content := newContentHandler(coordinator, db, roots, logger)
	return module.Module{Name: "renderjobs", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.Handle("/api/v1/projects/{project_id}/renders/{render_id}/content", content).Methods("GET")
	}, Endpoints: Endpoints}
}
