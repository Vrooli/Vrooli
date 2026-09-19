package review

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review/review_v1connect"

	"personal-planner/internal/module"
	r "personal-planner/internal/review"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := gc.NewReviewServiceHandler(NewConnectHandler(Deps{Service: r.NewService(db, clock), Logger: logger}))
	return module.Module{Name: "review", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
