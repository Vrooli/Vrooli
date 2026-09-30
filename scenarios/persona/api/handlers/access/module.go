package access

import (
	"os"

	"persona/internal/access"
	"persona/internal/journal"
	"persona/internal/module"
	"persona/internal/personas"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access/access_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	personaService := personas.NewService(personas.NewSQLiteRepository(db, clock))
	actionJournal := journal.NewService(journal.NewSQLiteRepository(db, clock))
	service := access.NewService(access.NewSQLiteRepository(db, clock), personaService, actionJournal, access.LiveVerifier{}, access.ServiceOptions{Clock: clock, Secret: []byte(os.Getenv("PERSONA_ATTESTATION_SECRET")), KeyID: "persona-local"})
	return ModuleWithService(service)
}

func ModuleWithService(service access.Service) module.Module {
	path, handler := accessconnect.NewAccessServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "access", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return access.Schema() }
