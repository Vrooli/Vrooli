package handoffs

import (
	"os"
	"strings"

	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/module"
	"persona/internal/personas"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	handoffsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/handoffs/handoffs_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	p := personas.NewService(personas.NewSQLiteRepository(db, clock))
	j := journal.NewService(journal.NewSQLiteRepository(db, clock))
	var relay handoffs.Relay
	if baseURL := strings.TrimSpace(os.Getenv("PERSONA_NOTIFICATION_HUB_API_BASE")); baseURL != "" {
		relay = handoffs.HTTPRelay{BaseURL: baseURL}
	}
	return ModuleWithService(handoffs.NewServiceWithRelay(handoffs.NewSQLiteRepository(db, clock), p, j, relay, clock))
}

func ModuleWithService(service handoffs.Service) module.Module {
	path, handler := handoffsconnect.NewHandoffsServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "handoffs", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return handoffs.Schema() }
