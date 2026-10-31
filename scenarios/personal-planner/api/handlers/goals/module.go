package goals

import (
	"encoding/json"
	"errors"
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
		r.HandleFunc("/api/v1/goals", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch req.Method {
			case http.MethodGet:
				goals, err := service.List(req.Context())
				if err != nil {
					http.Error(w, "goals are unavailable", http.StatusInternalServerError)
					return
				}
				out := make([]goalResponse, 0, len(goals))
				for _, item := range goals {
					out = append(out, toGoalResponse(item))
				}
				_ = json.NewEncoder(w).Encode(struct {
					Goals []goalResponse `json:"goals"`
				}{Goals: out})
			case http.MethodPost:
				var body struct {
					Title             string     `json:"title"`
					Purpose           string     `json:"purpose"`
					ProgressMethod    string     `json:"progress_method"`
					TargetBasisPoints int64      `json:"target_basis_points"`
					TargetDate        string     `json:"target_date"`
					Measure           *g.Measure `json:"measure"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					http.Error(w, "invalid goal request", http.StatusBadRequest)
					return
				}
				created, err := service.Create(req.Context(), g.CreateInput{Title: body.Title, Purpose: body.Purpose, ProgressMethod: body.ProgressMethod, TargetBasisPoints: body.TargetBasisPoints, TargetDate: body.TargetDate, Measure: body.Measure})
				if err != nil {
					var invalid g.ErrInvalidGoal
					if errors.As(err, &invalid) {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					http.Error(w, "goal could not be created", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(struct {
					Goal goalResponse `json:"goal"`
				}{Goal: toGoalResponse(created)})
			default:
				w.Header().Set("Allow", "GET, POST")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}).Methods(http.MethodGet, http.MethodPost)
		r.HandleFunc("/api/v1/goals/{id}/target-date", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				TargetDate       string `json:"target_date"`
				Reason           string `json:"reason"`
				ExpectedRevision int64  `json:"expected_revision"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "invalid target date request", http.StatusBadRequest)
				return
			}
			if err := service.SetTargetDate(req.Context(), mux.Vars(req)["id"], body.TargetDate, body.Reason, body.ExpectedRevision); err != nil {
				var conflict g.ErrRevisionConflict
				if errors.As(err, &conflict) {
					http.Error(w, err.Error(), http.StatusConflict)
					return
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/goals/{id}/target-date/history", func(w http.ResponseWriter, req *http.Request) {
			history, err := service.ListTargetDateHistory(req.Context(), mux.Vars(req)["id"])
			if err != nil {
				var notFound g.ErrGoalNotFound
				var invalid g.ErrInvalidGoal
				if errors.As(err, &notFound) {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				if errors.As(err, &invalid) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				http.Error(w, "goal target date history is unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(struct {
				History []g.TargetDateHistory `json:"history"`
			}{History: history})
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

type goalResponse struct {
	ID                  string     `json:"id"`
	Title               string     `json:"title"`
	Purpose             string     `json:"purpose"`
	Status              string     `json:"status"`
	ProgressMethod      string     `json:"progress_method"`
	ProgressBasisPoints *int64     `json:"progress_basis_points,omitempty"`
	ProgressState       string     `json:"progress_state,omitempty"`
	TargetBasisPoints   int64      `json:"target_basis_points"`
	Revision            int64      `json:"revision"`
	TargetDate          string     `json:"target_date,omitempty"`
	Measure             *g.Measure `json:"measure,omitempty"`
}

func toGoalResponse(item g.Goal) goalResponse {
	response := goalResponse{ID: item.ID, Title: item.Title, Purpose: item.Purpose, Status: item.Status, ProgressMethod: item.ProgressMethod, TargetBasisPoints: item.TargetBasisPoints, Revision: item.Revision, TargetDate: item.TargetDate, Measure: item.Measure}
	if item.Measure != nil {
		response.ProgressState = "unknown"
	} else {
		response.ProgressBasisPoints = &item.ProgressBasisPoints
	}
	return response
}

func Schema() string { return g.Schema() }
