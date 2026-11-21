package journal

import (
	"persona/internal/journal"
	"persona/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/journal/journal_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	return ModuleWithService(journal.NewService(journal.NewSQLiteRepository(db, clock)))
}

func ModuleWithService(service journal.Service) module.Module {
	path, handler := journalconnect.NewJournalServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "journal", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return journal.Schema() }
