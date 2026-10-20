package projects

import (
	"device-studio/internal/module"
	domain "device-studio/internal/projects"
	"device-studio/internal/routedblobstore"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
	"log"
)

func Module(db *database.RoutedDB, logger *log.Logger) module.Module {
	return ModuleWithRepository(domain.NewSQLiteRepository(db, nil), logger)
}
func ModuleWithRepository(repo domain.Repository, logger *log.Logger) module.Module {
	return moduleWith(repo, nil, logger)
}
func ModuleWithStorage(repo domain.Repository, mediaRepo domain.SourceMediaRepository, blobs blobstore.BlobStore, logger *log.Logger) module.Module {
	return moduleWith(repo, domain.NewSourceMediaService(domain.SourceMediaDeps{Projects: domain.NewService(repo, nil), Repository: mediaRepo, Store: blobs}), logger)
}
func ModuleWithRoots(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	repo := domain.NewSQLiteRepository(db, nil)
	store, err := routedblobstore.New(roots, "media")
	if err != nil {
		logger.Fatalf("project media storage: %v", err)
	}
	return ModuleWithStorage(repo, repo.(domain.SourceMediaRepository), store, logger)
}
func moduleWith(repo domain.Repository, media domain.SourceMediaService, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	path, svc := projectsconnect.NewProjectsServiceHandler(NewConnectHandler(Deps{Service: domain.NewService(repo, nil), MediaService: media, Logger: logger}))
	upload := newMediaUploadHandler(domain.NewService(repo, nil), media, logger)
	return module.Module{Name: "projects", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
		r.Handle("/api/v1/projects/{project_id}/media", upload).Methods("POST")
		r.Handle("/api/v1/projects/{project_id}/media/{media_id}/content", newMediaContentHandler(media, logger)).Methods("GET")
	}, Endpoints: Endpoints}
}
func Schema() string { return domain.Schema() }
