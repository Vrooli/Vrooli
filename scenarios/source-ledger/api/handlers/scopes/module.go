package scopes

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
	"source-ledger/internal/module"
	internalpolicy "source-ledger/internal/policy"
)

func Module(db *database.RoutedDB, registry *internalpolicy.Registry, registerProvider func(string) error, logger *log.Logger) module.Module {
	path, handler := scopesconnect.NewScopesServiceHandler(NewConnectHandler(registry, registerProvider, logger))
	return module.Module{Name: "scopes", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = scopesv1.File_source_ledger_v1_scopes_scopes_proto
