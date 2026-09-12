package intake

import (
	"context"
	"log"
	"net/http"
	"path/filepath"

	"document-manager/internal/corpus"
	"document-manager/internal/intake"
	"document-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	intakeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/intake/intake_v1connect"
)

func Module(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger, rootPaths ...func(context.Context, string) (string, error)) module.Module {
	return moduleWithCorpus(db, roots, logger, nil, rootPaths...)
}

func ModuleWithCorpus(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger, rootPath func(context.Context, string) (string, error), corpusService corpus.Service) module.Module {
	return moduleWithCorpus(db, roots, logger, &corpusService, rootPath)
}

func moduleWithCorpus(db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger, corpusService *corpus.Service, rootPaths ...func(context.Context, string) (string, error)) module.Module {
	filePath := func(ctx context.Context, key string) (string, error) { return fileRoot(ctx, roots, key) }
	if len(rootPaths) > 0 && rootPaths[0] != nil {
		filePath = rootPaths[0]
	}
	service := intake.NewService(intake.NewSQLiteRepository(db), intake.RoutedFileStore{Roots: roots, RootPath: filePath}, intake.CommandPDFClassifier{})
	var path string
	var handler http.Handler
	if corpusService != nil {
		path, handler = intakeconnect.NewIntakeServiceHandler(NewConnectHandler(service, *corpusService))
	} else {
		path, handler = intakeconnect.NewIntakeServiceHandler(NewConnectHandler(service))
	}
	return module.Module{Name: "intake", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}
}

func fileRoot(ctx context.Context, roots *filerouting.RoutedRoots, key string) (string, error) {
	root, err := roots.Pick(ctx, storage.ClassData)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "documents", filepath.Base(key)), nil
}

func Schema() string { return intake.Schema() }
