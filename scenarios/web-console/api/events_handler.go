package main

import (
	"context"
	"time"

	eventsH "web-console/handlers/events"
)

// eventsAdapter bridges the in-process *EventLogger to
// handlers/events.Service so the Connect handler can be mounted from
// main without crossing a package boundary the wrong way.
type eventsAdapter struct {
	srv *Server
}

func newEventsAdapter(s *Server) *eventsAdapter {
	return &eventsAdapter{srv: s}
}

func (a *eventsAdapter) Recent(_ context.Context, limit int) []eventsH.Event {
	in := a.srv.events.Recent(limit)
	out := make([]eventsH.Event, len(in))
	for i, e := range in {
		out[i] = eventsH.Event{
			Type:      e.Type,
			SessionID: e.SessionID,
			Timestamp: e.Timestamp.UTC().Format(time.RFC3339Nano),
			Details:   e.Details,
		}
	}
	return out
}

func (a *eventsAdapter) Count(_ context.Context) int {
	return a.srv.events.Count()
}
