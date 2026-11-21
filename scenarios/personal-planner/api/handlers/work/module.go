package work

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	workconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work/work_v1connect"

	"personal-planner/internal/module"
	internalwork "personal-planner/internal/work"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	repo := internalwork.NewSQLiteRepository(db, clock)
	service := internalwork.NewService(repo)
	path, handler := workconnect.NewWorkServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger}))
	return module.Module{Name: "work", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.HandleFunc("/api/v1/work/{id}/snooze", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Until  string `json:"until"`
				Reason string `json:"reason"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid snooze request", http.StatusBadRequest)
				return
			}
			if err := service.Snooze(req.Context(), mux.Vars(req)["id"], body.Until, body.Reason); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/work/{id}/complete", func(w http.ResponseWriter, req *http.Request) {
			if err := service.Complete(req.Context(), mux.Vars(req)["id"]); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/work/{id}/estimate", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				RemainingMinutes int    `json:"remaining_minutes"`
				Reason           string `json:"reason"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid estimate request", http.StatusBadRequest)
				return
			}
			if err := service.UpdateEstimate(req.Context(), mux.Vars(req)["id"], body.RemainingMinutes, body.Reason); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodPut)
	}, Endpoints: Endpoints}
}
func Schema() string { return internalwork.Schema() }
