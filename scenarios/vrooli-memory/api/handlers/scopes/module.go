package scopes

import (
	"log"

	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes/scopesv1connect"
)

func Module(client *ledgerclient.Client, logger *log.Logger) module.Module {
	path, handler := scopesconnect.NewScopesServiceHandler(NewConnectHandler(client.Scopes, logger))
	return module.Module{Name: "scopes", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = scopesv1.File_vrooli_memory_v1_scopes_scopes_proto
