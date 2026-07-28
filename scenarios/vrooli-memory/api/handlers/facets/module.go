package facets

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets"
	facetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets/facets_v1connect"

	internalfacets "vrooli-memory/internal/facets"
	"vrooli-memory/internal/module"
)

func Module(db *database.RoutedDB, logger *log.Logger) module.Module {
	svc := internalfacets.NewService(internalfacets.NewSQLiteRepository(db.Primary()))
	path, handler := facetsconnect.NewFacetsServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "facets", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = facetsv1.File_vrooli_memory_v1_facets_facets_proto
