package focus

import (
	"encoding/json"
	"log"
	"net/http"

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
	service := focus.NewService(repo, clock)
	path, handler := focusconnect.NewFocusServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger}))
	return module.Module{Name: "focus", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.HandleFunc("/api/v1/focus/{id}/note", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Note string `json:"note"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid focus note request", http.StatusBadRequest)
				return
			}
			note, err := service.SaveNote(req.Context(), mux.Vars(req)["id"], body.Note)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(note)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/focus/notes", func(w http.ResponseWriter, req *http.Request) {
			notes, err := service.ListNotes(req.Context(), req.URL.Query().Get("date"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(notes)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/focus/{id}/pause-reason", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Reason string `json:"reason"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid pause reason request", http.StatusBadRequest)
				return
			}
			event, err := service.RecordPauseEvent(req.Context(), mux.Vars(req)["id"], body.Reason)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(event)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/focus/pause-reasons", func(w http.ResponseWriter, req *http.Request) {
			events, err := service.ListPauseEvents(req.Context(), req.URL.Query().Get("date"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(events)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func Schema() string { return focus.Schema() }
