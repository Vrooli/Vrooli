package enrichment

import (
	"document-manager/internal/enrichment"
	"document-manager/internal/gatewayreq"
	"document-manager/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	enrichmentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment/enrichment_v1connect"
)

func Module(db *database.RoutedDB) module.Module {
	service := enrichment.NewService(enrichment.NewSQLiteRepository(db), gatewayreq.New("document-manager"), nil)
	path, handler := enrichmentconnect.NewEnrichmentServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "enrichment", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}
}

func Schema() string { return enrichment.Schema() }
