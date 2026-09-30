package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// hubHeartbeatInterval is how often the SSE handler writes a comment line to
// keep idle proxies (Cloudflare tunnel ~100s default) and load balancers from
// reaping the connection. A package var so tests can shrink it.
var hubHeartbeatInterval = 15 * time.Second

// handleEventStream serves the process-wide conversation event channel as
// Server-Sent Events. The browser opens ONE stream for ALL sessions; unread
// badges and conversation deltas no longer depend on any per-session terminal
// WebSocket being open.
//
// Resume: honors a Last-Event-ID request header or ?last_event_id=<globalId>
// query param (header wins). On resume, buffered envelopes newer than the
// cursor are replayed before the live tail; a cursor older than the retained
// window yields a conversation_out_of_sync nudge per affected session so the
// client backfills via GET /conversation?since_sequence=N.
//
// [REQ:P0-002b] streaming I/O — conversation side-channel.
// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Defeat proxy buffering (nginx/Cloudflare) so frames flush immediately.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	lastEventID := parseLastEventID(r)

	sub, replay, gap := s.hub.Subscribe(lastEventID)
	defer s.hub.Unsubscribe(sub)

	// On a gap, nudge the client to backfill the sessions it may have missed
	// before we stream the (possibly partial) replay + live tail. Scope to the
	// sessions currently retained; if we can't determine any, send one nudge
	// with an empty session id meaning "refetch everything".
	if gap {
		sessions := s.hub.RetainedSessionIDs()
		if len(sessions) == 0 {
			sessions = []string{""}
		}
		for _, sid := range sessions {
			if !writeOutOfSyncFrame(w, sid) {
				return
			}
		}
		flusher.Flush()
	}

	for _, env := range replay {
		if !writeEnvelopeFrame(w, env) {
			return
		}
	}
	if len(replay) > 0 {
		flusher.Flush()
	}

	heartbeat := time.NewTicker(hubHeartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case env := <-sub.events:
			if !writeEnvelopeFrame(w, env) {
				return
			}
			flusher.Flush()
		case sid := <-sub.resync:
			if !writeOutOfSyncFrame(w, sid) {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := w.Write([]byte(":hb\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// parseLastEventID resolves the resume cursor from the Last-Event-ID header
// (preferred) or the ?last_event_id= query param. Returns 0 when absent or
// unparseable (meaning: stream live only, no replay).
func parseLastEventID(r *http.Request) int64 {
	raw := strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("last_event_id"))
	}
	if raw == "" {
		return 0
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

// writeEnvelopeFrame writes one SSE frame (id/event/data) for env. Returns
// false on write error so the caller can tear down the stream.
func writeEnvelopeFrame(w http.ResponseWriter, env HubEnvelope) bool {
	data, err := json.Marshal(env)
	if err != nil {
		// A marshal failure is a programmer error, not a client one; skip the
		// frame rather than killing the stream.
		return true
	}
	var b strings.Builder
	b.WriteString("id:")
	b.WriteString(strconv.FormatInt(env.ID, 10))
	b.WriteString("\nevent:")
	b.WriteString(env.Kind)
	b.WriteString("\ndata:")
	b.Write(data)
	b.WriteString("\n\n")
	_, werr := w.Write([]byte(b.String()))
	return werr == nil
}

// writeOutOfSyncFrame writes a conversation_out_of_sync frame for sessionID
// (empty session id means "refetch all"). The id is 0 so it never advances the
// client's Last-Event-ID cursor — it's an advisory nudge, not a stream entry.
func writeOutOfSyncFrame(w http.ResponseWriter, sessionID string) bool {
	env := HubEnvelope{
		SessionID: sessionID,
		Kind:      HubKindConversationOutOfSync,
		Payload:   struct{}{},
	}
	data, err := json.Marshal(env)
	if err != nil {
		return true
	}
	frame := "id:0\nevent:" + HubKindConversationOutOfSync + "\ndata:" + string(data) + "\n\n"
	_, werr := w.Write([]byte(frame))
	return werr == nil
}
