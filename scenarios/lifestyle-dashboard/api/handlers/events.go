package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/repository"
)

// CreateEvent handles POST /api/v1/events - P0-001
// [REQ:LD-EVENT-STORAGE] Creates and persists a new cross-domain event.
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Domain == "" || req.EventType == "" {
		WriteError(w, http.StatusBadRequest, "domain and event_type are required")
		return
	}

	// Build event from request
	event := &domain.Event{
		Domain:         req.Domain,
		EventType:      req.EventType,
		Payload:        req.Payload,
		IsIntervention: req.IsIntervention,
		HypothesisID:   req.HypothesisID,
	}
	if req.Timestamp != nil {
		event.Timestamp = *req.Timestamp
	}

	// Delegate to repository (handles ID generation and timestamps)
	if err := h.Events.Create(r.Context(), event); err != nil {
		log.Printf("Error creating event: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	WriteJSON(w, http.StatusCreated, event)
}

// QueryEvents handles GET /api/v1/events - P0-003
// [REQ:LD-QUERY-FILTER] Queries events with optional domain, type, and time filters.
func (h *Handler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := repository.EventFilter{
		Domain:    query.Get("domain"),
		EventType: query.Get("event_type"),
		StartTime: query.Get("start"),
		EndTime:   query.Get("end"),
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	events, err := h.Events.List(r.Context(), filter)
	if err != nil {
		log.Printf("Error querying events: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to query events")
		return
	}

	WriteJSON(w, http.StatusOK, domain.EventsResponse{
		Events: events,
		Count:  len(events),
	})
}

// GetEvent handles GET /api/v1/events/{id}
// [REQ:LD-EVENT-STORAGE] Retrieves a single event by ID.
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	event, err := h.Events.GetByID(r.Context(), id)
	if repository.IsNotFound(err) {
		WriteError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		log.Printf("Error getting event: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to get event")
		return
	}

	WriteJSON(w, http.StatusOK, event)
}
