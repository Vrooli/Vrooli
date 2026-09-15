// HTTP handler for typed-operational event queries (Phase 3).
//
// One read endpoint:
//
//   GET /api/v1/events  — list events, optionally filtered by run id,
//                         event_type, or since timestamp.
//
// Streaming (SSE) is intentionally deferred: agent-manager already has
// a richer per-run WebSocket stream over /api/v1/ws used by the UI for
// live log following, and routing the operational-event stream through
// it (vs. inventing a parallel SSE channel) is the right long-term
// shape — but it is a separate decision the UI work in Phase 4 will
// drive. The plan's "events tail" CLI uses the existing WebSocket.

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// EventsHandler exposes the typed-event read endpoint.
type EventsHandler struct {
	repo eventlog.Repository
}

// NewEventsHandler wires a new handler over the typed event repository.
func NewEventsHandler(repo eventlog.Repository) *EventsHandler {
	return &EventsHandler{repo: repo}
}

// RegisterRoutes registers the events endpoint.
func (h *EventsHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/events", h.ListEvents).Methods("GET")
}

// EventsListResponse is the response for /api/v1/events.
type EventsListResponse struct {
	Events []EventRow `json:"events"`
	Limit  int        `json:"limit"`
}

// EventRow is one decoded event in the response.
type EventRow struct {
	ID            uuid.UUID   `json:"id"`
	RunID         uuid.UUID   `json:"run_id"`
	Sequence      int64       `json:"sequence"`
	EventType     string      `json:"event_type"`
	SchemaVersion int         `json:"schema_version"`
	Timestamp     time.Time   `json:"timestamp"`
	Payload       interface{} `json:"payload"`
}

// ListEvents returns typed events with optional filters.
//
// Query params:
//   - run:    UUID of a run; returns its events ordered by sequence asc.
//   - type:   single event_type to filter on (e.g., model.fallback.attempted).
//   - since:  RFC3339 lower-bound on event timestamp.
//   - limit:  max rows to return (default 100, max 1000).
func (h *EventsHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 100
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	var records []eventlog.Record
	var err error

	switch {
	case q.Get("run") != "":
		runID, perr := uuid.Parse(q.Get("run"))
		if perr != nil {
			writeJSONError(w, http.StatusBadRequest, "run must be a UUID")
			return
		}
		records, err = h.repo.SinceForRun(r.Context(), runID, 0, limit)

	case q.Get("type") != "":
		evType := domain.RunEventType(q.Get("type"))
		if !evType.IsTypedOperationalEvent() {
			writeJSONError(w, http.StatusBadRequest, "type must be a typed-operational event")
			return
		}
		var since time.Time
		if v := q.Get("since"); v != "" {
			t, perr := time.Parse(time.RFC3339, v)
			if perr != nil {
				writeJSONError(w, http.StatusBadRequest, "since must be RFC3339")
				return
			}
			since = t
		}
		records, err = h.repo.ByEventType(r.Context(), evType, since, limit)

	default:
		// No filters: walk by rowid from the start. Bounded by limit so
		// a naive curl doesn't pull the entire log.
		records, err = h.repo.SinceID(r.Context(), 0, limit)
	}

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "events query failed: "+err.Error())
		return
	}

	rows := make([]EventRow, 0, len(records))
	for _, rec := range records {
		rows = append(rows, EventRow{
			ID:            rec.ID,
			RunID:         rec.RunID,
			Sequence:      rec.Sequence,
			EventType:     string(rec.EventType),
			SchemaVersion: rec.SchemaVersion,
			Timestamp:     rec.Timestamp,
			Payload:       rec.Payload,
		})
	}
	writeJSON(w, http.StatusOK, EventsListResponse{Events: rows, Limit: limit})
}
