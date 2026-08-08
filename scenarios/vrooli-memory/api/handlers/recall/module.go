package recall

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall/recall_v1connect"

	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"
)

func Module(client *ledgerclient.Client, logger *log.Logger) module.Module {
	path, h := recallconnect.NewRecallServiceHandler(NewConnectHandler(client.Recall, logger))
	return module.Module{Name: "recall", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = recallv1.File_vrooli_memory_v1_recall_recall_proto
