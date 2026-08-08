package forest

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest/forest_v1connect"
	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"
)

// NewService builds the compaction service. The composition root owns the
// single instance because the scheduled maintenance pass and the operator RPC
// must share one run mutex; two instances would compact concurrently.
func Module(client *ledgerclient.Client, logger *log.Logger) module.Module {
	path, h := forestconnect.NewForestServiceHandler(NewConnectHandler(client.Forest, logger))
	return module.Module{Name: "forest", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = forestv1.File_vrooli_memory_v1_forest_forest_proto
