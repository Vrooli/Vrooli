package focus

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus/focus_v1connect"

	"personal-planner/internal/focus"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	repo := focus.NewSQLiteRepository(db, clock)
	path, handler := focusconnect.NewFocusServiceHandler(NewConnectHandler(Deps{Service: focus.NewService(repo, clock), Logger: logger}))
	return module.Module{Name: "focus", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return focus.Schema() }
