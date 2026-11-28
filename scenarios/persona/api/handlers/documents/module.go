package documents

import (
	"os"

	"persona/internal/documents"
	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/module"
	"persona/internal/personas"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	documentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/documents/documents_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	p := personas.NewService(personas.NewSQLiteRepository(db, clock))
	h := handoffs.NewService(handoffs.NewSQLiteRepository(db, clock), p, journal.NewService(journal.NewSQLiteRepository(db, clock)), clock)
	j := journal.NewService(journal.NewSQLiteRepository(db, clock))
	authority := documents.NewUnavailableAuthority()
	if base := os.Getenv("DOCUMENT_MANAGER_API_BASE"); base != "" {
		authority = documents.HTTPAuthority{BaseURL: base}
	}
	return ModuleWithService(documents.NewService(documents.NewSQLiteRepository(db, clock), p, h, authority, j, clock))
}

func ModuleWithService(service documents.Service) module.Module {
	path, handler := documentsconnect.NewDocumentsServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "documents", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return documents.Schema() }
