package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	protoMarshaler   = protojson.MarshalOptions{EmitUnpopulated: false}
	protoUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}
)

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "READ_ERROR", "failed to read request body")
		return
	}

	var env domain.EventEnvelope
	if err := protoUnmarshaler.Unmarshal(body, &env); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body: "+err.Error())
		return
	}

	if env.EventId == "" {
		writeError(w, http.StatusBadRequest, "MISSING_EVENT_ID", "eventId is required")
		return
	}
	if env.EventType == "" {
		writeError(w, http.StatusBadRequest, "MISSING_EVENT_TYPE", "eventType is required")
		return
	}
	if env.SourceScenario == "" {
		writeError(w, http.StatusBadRequest, "MISSING_SOURCE", "sourceScenario is required")
		return
	}

	event, err := convert.EnvelopeToEvent(&env)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CONVERSION_ERROR", err.Error())
		return
	}

	id, err := s.store.Insert(r.Context(), event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "failed to store event")
		log.Printf("insert error: %v", err)
		return
	}

	event.ID = id

	// Broadcast asynchronously
	go func() {
		envOut, err := convert.EventToEnvelope(event)
		if err != nil {
			log.Printf("convert for broadcast: %v", err)
			return
		}
		data, err := protoMarshaler.Marshal(envOut)
		if err != nil {
			log.Printf("marshal for broadcast: %v", err)
			return
		}
		s.broker.Publish(event, string(data))
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "eventId": env.EventId})
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filters := store.QueryFilters{
		EventType:     q.Get("type"),
		Source:        q.Get("source"),
		CorrelationID: q.Get("correlation_id"),
	}

	if v := q.Get("since"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_SINCE", "since must be an integer")
			return
		}
		filters.Since = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_LIMIT", "limit must be an integer")
			return
		}
		filters.Limit = n
	}

	events, err := s.store.Query(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", "failed to query events")
		log.Printf("query error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(events) == 0 {
		_, _ = w.Write([]byte("[]"))
		return
	}

	_, _ = w.Write([]byte("["))
	for i, e := range events {
		if i > 0 {
			_, _ = w.Write([]byte(","))
		}
		env, err := convert.EventToEnvelope(e)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "CONVERSION_ERROR", err.Error())
			return
		}
		data, err := protoMarshaler.Marshal(env)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "MARSHAL_ERROR", err.Error())
			return
		}
		_, _ = w.Write(data)
	}
	_, _ = w.Write([]byte("]"))
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "SSE_UNSUPPORTED", "streaming not supported")
		return
	}

	q := r.URL.Query()
	opts := broker.SubscribeOpts{
		EventTypePattern:      q.Get("type"),
		SourceScenarioPattern: q.Get("source"),
		TargetScenarioPattern: q.Get("target"),
	}

	ch, subCtx, cleanup := s.broker.Subscribe(r.Context(), opts)
	defer cleanup()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send retry directive
	fmt.Fprintf(w, "retry: 5000\n\n")
	flusher.Flush()

	// Replay missed events if Last-Event-ID is present
	if lastIDStr := r.Header.Get("Last-Event-ID"); lastIDStr != "" {
		lastID, err := strconv.ParseInt(lastIDStr, 10, 64)
		if err == nil {
			events, err := s.store.GetSince(r.Context(), lastID, 1000)
			if err == nil {
				for _, e := range events {
					env, err := convert.EventToEnvelope(e)
					if err != nil {
						continue
					}
					data, err := protoMarshaler.Marshal(env)
					if err != nil {
						continue
					}
					fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", e.ID, e.EventType, data)
				}
				flusher.Flush()
			}
		}
	}

	// Stream live events
	for {
		select {
		case <-subCtx.Done():
			return
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Event == "heartbeat" {
				if msg.Data != "" {
					fmt.Fprintf(w, ": heartbeat %s\n\n", msg.Data)
				} else {
					fmt.Fprintf(w, ": heartbeat\n\n")
				}
			} else {
				fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", msg.ID, msg.Event, msg.Data)
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "healthy",
		"service":     "vrooli-events",
		"subscribers": s.broker.SubscriberCount(),
		"store": map[string]any{
			"totalEvents":       stats.TotalEvents,
			"totalPayloadBytes": stats.TotalPayloadBytes,
		},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}
