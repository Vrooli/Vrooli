package forest

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest/forest_v1connect"
	internalforest "vrooli-memory/internal/forest"
	"vrooli-memory/internal/inference"
	"vrooli-memory/internal/module"
)

func Module(db *database.RoutedDB, client inference.Client, target int, logger *log.Logger) module.Module {
	svc := internalforest.NewService(internalforest.NewSQLiteRepository(db.Primary()), internalforest.NewSQLiteCandidateSource(db.Primary()), client, internalforest.Config{Target: target})
	path, h := forestconnect.NewForestServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "forest", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = forestv1.File_vrooli_memory_v1_forest_forest_proto
