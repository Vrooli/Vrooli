package goals

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals/goals_v1connect"

	g "personal-planner/internal/goals"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	service := g.NewService(g.NewSQLiteRepository(db, clock))
	path, handler := gc.NewGoalsServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger}))
	return module.Module{Name: "goals", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.HandleFunc("/api/v1/goals/{id}/target-date", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				TargetDate string `json:"target_date"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid target date request", http.StatusBadRequest)
				return
			}
			if err := service.SetTargetDate(req.Context(), mux.Vars(req)["id"], body.TargetDate); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

func Schema() string { return g.Schema() }
