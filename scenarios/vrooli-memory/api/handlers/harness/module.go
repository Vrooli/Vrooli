package harness

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
	harnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness/harness_v1connect"
	"vrooli-memory/internal/facets"
	internalharness "vrooli-memory/internal/harness"
	"vrooli-memory/internal/inference"
	"vrooli-memory/internal/journal"
	"vrooli-memory/internal/module"
	internalrecall "vrooli-memory/internal/recall"
)

func Module(db *database.RoutedDB, roots *filerouting.RoutedRoots, client inference.Client, logger *log.Logger) module.Module {
	home, _ := os.UserHomeDir()
	root := os.Getenv("VROOLI_MEMORY_CLAUDE_ROOT")
	if root == "" {
		root = filepath.Join(home, ".claude", "projects", "-home-matthalloran8-Vrooli", "memory")
	}
	svc := journal.NewService(journal.NewSQLiteRepository(db.Primary()), client, facets.NewService(facets.NewSQLiteRepository(db.Primary())))
	config, err := internalrecall.ConfigFromEnv(os.LookupEnv)
	if err != nil {
		panic(err)
	}
	wake := internalrecall.NewService(internalrecall.NewSQLiteSource(db.Primary()), inference.Embedder{Client: client}, config)
	path, h := harnessconnect.NewHarnessServiceHandler(NewConnectHandler(internalharness.NewImporter(svc, root, db.Primary()), internalharness.NewProjector(db.Primary(), wake, roots), logger))
	return module.Module{Name: "harness", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = harnessv1.File_vrooli_memory_v1_harness_harness_proto
