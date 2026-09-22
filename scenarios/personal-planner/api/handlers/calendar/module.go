package calendar

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar/calendar_v1connect"

	d "personal-planner/internal/calendar"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	service := d.NewService(d.NewSQLiteRepository(db, clock))
	path, handler := c.NewCalendarServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger, Clock: clock}))
	return module.Module{Name: "calendar", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.HandleFunc("/api/v1/calendar/carry-forward", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				AllocationID string `json:"allocation_id"`
				TargetDate   string `json:"target_local_date"`
				StartMinutes int    `json:"start_minutes"`
				ReasonCode   string `json:"reason_code"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid carry-forward request", http.StatusBadRequest)
				return
			}
			a, err := service.CarryForward(req.Context(), d.CarryForwardInput{AllocationID: body.AllocationID, TargetLocalDate: body.TargetDate, StartMinutes: body.StartMinutes, ReasonCode: body.ReasonCode})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			response := struct {
				ID              string `json:"id"`
				WorkItemID      string `json:"work_item_id"`
				Title           string `json:"title"`
				SourceLabel     string `json:"source_label"`
				LocalDate       string `json:"local_date"`
				State           string `json:"state"`
				CarriedFromID   string `json:"carried_from_id"`
				StartMinutes    int    `json:"start_minutes"`
				DurationMinutes int    `json:"duration_minutes"`
			}{ID: a.ID, WorkItemID: a.WorkItemID, Title: a.Title, SourceLabel: a.SourceLabel, LocalDate: a.LocalDate, State: a.State, CarriedFromID: a.CarriedFromID, StartMinutes: a.StartMinutes, DurationMinutes: a.DurationMinutes}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}
func Schema() string { return d.Schema() }
