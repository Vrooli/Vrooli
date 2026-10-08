package calendar

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	d "personal-planner/internal/calendar"
)

type eventListResponse struct {
	Events []d.Event `json:"events"`
}

func mountEventRoutes(r *mux.Router, service d.EventService) {
	r.HandleFunc("/api/v1/calendar/event-commands/{idempotencyKey}", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		event, err := service.GetByIdempotencyKey(req.Context(), mux.Vars(req)["idempotencyKey"])
		if err != nil {
			writeEventError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, event)
	}).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/calendar/events", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			events, err := service.List(req.Context(), req.URL.Query().Get("start_local_date"), req.URL.Query().Get("end_local_date"))
			if err != nil {
				writeEventError(w, err)
				return
			}
			if events == nil {
				events = []d.Event{}
			}
			writeJSON(w, http.StatusOK, eventListResponse{Events: events})
		case http.MethodPost:
			var input d.CreateEventInput
			if err := decodeEventRequest(req, &input); err != nil {
				http.Error(w, "invalid event request", http.StatusBadRequest)
				return
			}
			event, err := service.Create(req.Context(), input)
			if err != nil {
				writeEventError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, event)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodGet, http.MethodPost)
	r.HandleFunc("/api/v1/calendar/events/{eventID}", func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["eventID"]
		switch req.Method {
		case http.MethodGet:
			event, err := service.Get(req.Context(), id)
			if err != nil {
				writeEventError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, event)
		case http.MethodPut:
			var input d.UpdateEventInput
			if err := decodeEventRequest(req, &input); err != nil {
				http.Error(w, "invalid event update", http.StatusBadRequest)
				return
			}
			input.ID = id
			event, err := service.Update(req.Context(), input)
			if err != nil {
				writeEventError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, event)
		default:
			w.Header().Set("Allow", "GET, PUT")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodGet, http.MethodPut)
}

func decodeEventRequest(req *http.Request, out any) error {
	defer req.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(req.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("event request must contain one JSON value")
	}
	return nil
}

func writeEventError(w http.ResponseWriter, err error) {
	var invalid d.ErrInvalidAllocation
	var notFound d.ErrEventNotFound
	var revisionConflict d.ErrEventRevisionConflict
	var idempotencyConflict d.ErrEventIdempotencyConflict
	var identityConflict d.ErrEventIdentityConflict
	switch {
	case errors.As(err, &invalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.As(err, &notFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.As(err, &revisionConflict), errors.As(err, &idempotencyConflict), errors.As(err, &identityConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, "calendar event operation failed", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
