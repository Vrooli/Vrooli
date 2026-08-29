package main

import (
	"strconv"
	"time"

	"web-console/internal/events"
)

// sessionStatusPayload is the per-event payload carried inside a HubEnvelope for
// the HubKindSessionStatus kind. It projects an events.Event about a session's
// existence (created / deleted / terminated) onto the wire shape the browser
// sidebar merges. snake_case mirrors the SessionInfo domain shape the UI already
// uses, so an external create becomes a sidebar row without a follow-up fetch.
//
// Only the "created" action populates the descriptive fields; delete/terminate
// carry just the action (and, for terminate, a reason) since the client already
// knows the session and only needs to drop it.
type sessionStatusPayload struct {
	Action       string `json:"action"` // "created" | "deleted" | "terminated"
	Shell        string `json:"shell,omitempty"`
	Cols         int    `json:"cols,omitempty"`
	Rows         int    `json:"rows,omitempty"`
	Backend      string `json:"backend,omitempty"`
	Origin       string `json:"origin,omitempty"`
	Owner        string `json:"owner,omitempty"`
	DisplayLabel string `json:"display_label,omitempty"`
	Agent        string `json:"agent,omitempty"`
	Recovered    bool   `json:"recovered,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	Reason       string `json:"reason,omitempty"` // terminated: e.g. "expired"
}

// startSessionLifecycleBridge fans session lifecycle events (created / deleted /
// terminated) from the structured event logger onto the process-wide
// ConversationHub, so browser SSE clients see sessions created or destroyed by
// ANY origin — another tab, the CLI, an expiry sweep — appear/disappear live,
// not only the ones this client created.
//
// Conversation deltas already ride the hub; session lifecycle did not, because
// events.Logger.Emit fans out on a separate channel the SSE handler never
// subscribed to. This bridge is that missing wire. It runs for the process
// lifetime; the subscription is intentionally never torn down.
func (s *Server) startSessionLifecycleBridge() {
	if s.events == nil || s.hub == nil {
		return
	}
	// Buffered so a burst of lifecycle events (e.g. a sweep expiring several
	// sessions) is absorbed without the emitter dropping frames; events.Logger
	// drops on a full channel rather than blocking, and a dropped lifecycle
	// frame would leave a phantom sidebar row until the next full hydration.
	ch := make(chan events.Event, 256)
	s.events.Subscribe(ch)
	go func() {
		for evt := range ch {
			s.publishSessionLifecycleEvent(evt)
		}
	}()
}

// publishSessionLifecycleEvent maps one structured lifecycle event onto the hub
// as a HubKindSessionStatus envelope. Non-existence events (connect, resize,
// policy update, …) are ignored — only created/deleted/terminated change what
// the sidebar shows.
func (s *Server) publishSessionLifecycleEvent(evt events.Event) {
	var action string
	switch evt.Type {
	case events.SessionCreated:
		action = "created"
	case events.SessionDeleted:
		action = "deleted"
	case events.SessionTerminated:
		action = "terminated"
	case events.SessionConnected:
		action = "connected"
	case events.SessionDisconnected:
		action = "disconnected"
	default:
		return
	}
	if action == "connected" || action == "disconnected" {
		details := make(map[string]string, len(evt.Details)+1)
		for key, value := range evt.Details {
			details[key] = value
		}
		details["action"] = action
		s.hub.Publish(HubEnvelope{
			SessionID: evt.SessionID,
			Kind:      HubKindDeviceStatus,
			Payload:   details,
		})
		return
	}

	d := evt.Details
	payload := sessionStatusPayload{Action: action}
	switch action {
	case "created":
		payload.Shell = d["shell"]
		payload.Cols = atoiOrZero(d["cols"])
		payload.Rows = atoiOrZero(d["rows"])
		payload.Backend = d["backend"]
		payload.Origin = d["origin"]
		payload.Owner = d["owner"]
		payload.DisplayLabel = d["label"]
		payload.Agent = d["agent"]
		payload.Recovered = d["recovered"] == "true"
		// The create event fires at ~creation time; use its timestamp so the
		// merged sidebar row has an honest created_at for activity sorting.
		payload.CreatedAt = evt.Timestamp.UTC().Format(time.RFC3339)
	case "terminated":
		payload.Reason = d["reason"]
	}

	s.hub.Publish(HubEnvelope{
		SessionID: evt.SessionID,
		Kind:      HubKindSessionStatus,
		Payload:   payload,
	})
}

// atoiOrZero parses s as a base-10 int, returning 0 on any parse failure. Event
// detail values are stringified ints (cols/rows); a malformed value degrades to
// 0 rather than failing the whole envelope.
func atoiOrZero(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
