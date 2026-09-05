package facets

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets"
	facetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets/facets_v1connect"

	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"
)

func Module(client *ledgerclient.Client, logger *log.Logger) module.Module {
	path, handler := facetsconnect.NewFacetsServiceHandler(NewConnectHandler(client.Facets, logger))
	return module.Module{Name: "facets", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = facetsv1.File_vrooli_memory_v1_facets_facets_proto
