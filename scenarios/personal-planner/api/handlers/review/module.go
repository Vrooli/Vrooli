package review

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review/review_v1connect"

	"personal-planner/internal/module"
	r "personal-planner/internal/review"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	service := r.NewService(db, clock)
	path, handler := gc.NewReviewServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger}))
	return module.Module{Name: "review", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		router.HandleFunc("/api/v1/review/calibration", func(w http.ResponseWriter, req *http.Request) {
			calibration, err := service.Calibration(req.Context(), req.URL.Query().Get("period"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(calibration)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/agent/read-model", func(w http.ResponseWriter, req *http.Request) {
			model, err := service.AgentReadModel(req.Context(), req.URL.Query().Get("period"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(model)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/goal-variances", func(w http.ResponseWriter, req *http.Request) {
			variances, err := service.GoalVariances(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(variances)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/today-signals", func(w http.ResponseWriter, req *http.Request) {
			signals, err := service.TodaySignals(req.Context(), req.URL.Query().Get("date"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(signals)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/goal-drifts", func(w http.ResponseWriter, req *http.Request) {
			drifts, err := service.GoalDrifts(req.Context(), req.URL.Query().Get("date"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(drifts)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/reminders", func(w http.ResponseWriter, req *http.Request) {
			reminders, err := service.Reminders(req.Context(), req.URL.Query().Get("date"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(reminders)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/win", func(w http.ResponseWriter, req *http.Request) {
			win, err := service.Win(req.Context(), req.URL.Query().Get("date"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(win)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/win", func(w http.ResponseWriter, req *http.Request) {
			var input struct {
				LocalDate string `json:"local_date"`
				Text      string `json:"text"`
			}
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "invalid win", http.StatusBadRequest)
				return
			}
			win, err := service.SaveWin(req.Context(), input.LocalDate, input.Text)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(win)
		}).Methods(http.MethodPut)
		router.HandleFunc("/api/v1/review/reminder-preferences", func(w http.ResponseWriter, req *http.Request) {
			preferences, err := service.ReminderPreferences(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(preferences)
		}).Methods(http.MethodGet)
		router.HandleFunc("/api/v1/review/reminder-preferences", func(w http.ResponseWriter, req *http.Request) {
			var input r.ReminderPreferences
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "invalid reminder preferences", http.StatusBadRequest)
				return
			}
			preferences, err := service.SaveReminderPreferences(req.Context(), input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(preferences)
		}).Methods(http.MethodPut)
	}, Endpoints: Endpoints}
}
