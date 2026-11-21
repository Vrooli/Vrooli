package events

import (
	"context"
	"time"

	intevents "web-console/internal/events"
)

// Adapter is the production Service implementation: it bridges the
// in-process *events.Logger to the transport-neutral Event shape the
// Connect handler consumes. Constructed in api/main.go and passed to
// Module.
type Adapter struct {
	Logger *intevents.Logger
}

func (a *Adapter) Recent(_ context.Context, limit int) []Event {
	in := a.Logger.Recent(limit)
	out := make([]Event, len(in))
	for i, e := range in {
		out[i] = Event{
			Type:      e.Type,
			SessionID: e.SessionID,
			Timestamp: e.Timestamp.UTC().Format(time.RFC3339Nano),
			Details:   e.Details,
		}
	}
	return out
}

func (a *Adapter) Count(_ context.Context) int {
	return a.Logger.Count()
}
