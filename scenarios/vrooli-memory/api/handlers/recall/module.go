package recall

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall/recall_v1connect"

	internalfacets "vrooli-memory/internal/facets"
	"vrooli-memory/internal/inference"
	internaljournal "vrooli-memory/internal/journal"
	"vrooli-memory/internal/module"
	"vrooli-memory/internal/policy"
	internalrecall "vrooli-memory/internal/recall"
)

func Module(db *database.RoutedDB, client inference.Client, config internalrecall.Config, registry *policy.Registry, logger *log.Logger) module.Module {
	svc := internalrecall.NewService(internalrecall.NewSQLiteSource(db.Primary()), inference.Embedder{Client: client}, config)
	svc.SetPolicyRegistry(registry)
	journal := internaljournal.NewService(internaljournal.NewSQLiteRepository(db.Primary()), client)
	handler := NewConnectHandler(svc, logger, journal)
	handler.SetUsageRecorder(internalfacets.NewService(internalfacets.NewSQLiteRepository(db.Primary())))
	path, h := recallconnect.NewRecallServiceHandler(handler)
	return module.Module{Name: "recall", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = recallv1.File_vrooli_memory_v1_recall_recall_proto
