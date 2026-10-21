package shots

import (
	"log"

	"device-studio/internal/module"
	domain "device-studio/internal/projects"
	"device-studio/internal/routedblobstore"
	shotdomain "device-studio/internal/shots"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

func Module(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) module.Module {
	return ModuleWithVerifier(db, roots, nil, logger)
}

func ModuleWithVerifier(db *database.RoutedDB, roots *filerouting.RoutedRoots, verifier shotdomain.VideoSampleVerifier, logger *log.Logger) module.Module {
	return ModuleWithPhysicalResolver(db, roots, verifier, nil, logger)
}

func ModuleWithPhysicalResolver(db *database.RoutedDB, roots *filerouting.RoutedRoots, verifier shotdomain.VideoSampleVerifier, physical shotdomain.PhysicalResolver, logger *log.Logger) module.Module {
	repo := domain.NewSQLiteRepository(db, nil)
	mediaRepo := repo.(domain.SourceMediaRepository)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Fatalf("shot media storage: %v", err)
	}
	media := domain.NewSourceMediaService(domain.SourceMediaDeps{Projects: domain.NewService(repo, nil), Repository: mediaRepo, Store: mediaStore})
	presetRepo, presetErr := shotdomain.NewSQLitePhysicalPresetRepository(db)
	var presets *shotdomain.PhysicalPresetService
	if presetErr == nil {
		if validator, ok := physical.(shotdomain.PhysicalPresetValidator); ok {
			presets, presetErr = shotdomain.NewPhysicalPresetService(presetRepo, validator, nil, nil)
		}
	}
	if presetErr != nil && logger != nil {
		logger.Printf("physical preset service unavailable: %v", presetErr)
	}
	return moduleWithRepositoryAndDependencies(shotdomain.NewSQLiteRepository(db), domain.NewService(repo, nil), media, verifier, physical, presets, logger)
}

func ModuleWithPhysicalOwners(db *database.RoutedDB, roots *filerouting.RoutedRoots, verifier shotdomain.VideoSampleVerifier, physical shotdomain.PhysicalResolver, presets *shotdomain.PhysicalPresetService, logger *log.Logger) module.Module {
	repo := domain.NewSQLiteRepository(db, nil)
	mediaRepo := repo.(domain.SourceMediaRepository)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Fatalf("shot media storage: %v", err)
	}
	media := domain.NewSourceMediaService(domain.SourceMediaDeps{Projects: domain.NewService(repo, nil), Repository: mediaRepo, Store: mediaStore})
	return moduleWithRepositoryAndDependencies(shotdomain.NewSQLiteRepository(db), domain.NewService(repo, nil), media, verifier, physical, presets, logger)
}

func ModuleWithRepository(repo shotdomain.Repository, projectReader domain.Service, media domain.SourceMediaService, logger *log.Logger) module.Module {
	return ModuleWithRepositoryAndVerifier(repo, projectReader, media, nil, logger)
}

func ModuleWithRepositoryAndVerifier(repo shotdomain.Repository, projectReader domain.Service, media domain.SourceMediaService, verifier shotdomain.VideoSampleVerifier, logger *log.Logger) module.Module {
	return ModuleWithRepositoryAndDependencies(repo, projectReader, media, verifier, nil, logger)
}

func ModuleWithRepositoryAndDependencies(repo shotdomain.Repository, projectReader domain.Service, media domain.SourceMediaService, verifier shotdomain.VideoSampleVerifier, physical shotdomain.PhysicalResolver, logger *log.Logger) module.Module {
	return moduleWithRepositoryAndDependencies(repo, projectReader, media, verifier, physical, nil, logger)
}

func moduleWithRepositoryAndDependencies(repo shotdomain.Repository, projectReader domain.Service, media domain.SourceMediaService, verifier shotdomain.VideoSampleVerifier, physical shotdomain.PhysicalResolver, presets *shotdomain.PhysicalPresetService, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	service := shotdomain.NewService(repo, projectReader, media, nil, nil)
	shotdomain.SetVideoSampleVerifier(service, verifier)
	shotdomain.SetPhysicalResolver(service, physical)
	if outputVerifier, ok := verifier.(shotdomain.ReadabilityOutputVerifier); ok {
		shotdomain.SetReadabilityOutputVerifier(service, outputVerifier)
	}
	path, handler := shotsconnect.NewShotsServiceHandler(NewConnectHandler(Deps{Service: service, Presets: presets, Logger: logger}))
	return module.Module{Name: "shots", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return shotdomain.Schema() }
