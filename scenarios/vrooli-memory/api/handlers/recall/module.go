package recall

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall/recall_v1connect"

	"vrooli-memory/internal/inference"
	"vrooli-memory/internal/module"
	internalrecall "vrooli-memory/internal/recall"
)

func Module(db *database.RoutedDB, client inference.Client, config internalrecall.Config, logger *log.Logger) module.Module {
	svc := internalrecall.NewService(internalrecall.NewSQLiteSource(db.Primary()), inference.Embedder{Client: client}, config)
	path, h := recallconnect.NewRecallServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "recall", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = recallv1.File_vrooli_memory_v1_recall_recall_proto
