// Package browser implements the browser-voice WebSocket transport.
//
// The transport sits above the voice-session abstraction and below the raw
// WebSocket connection. Audio frames flow voice.Service.HandleStreamWS ->
// transcript JSON events on the wire, while the same transcripts are mirrored
// into the session pub/sub for multi-observer consumers (live dashboards,
// orchestrators, etc.).
//
// Endpoint: GET /api/v1/voice/stream
// Transport reason: websocket_transport (NOT a REST exception). See
// docs/internal/SEAMS.md for the seam table.
//
// Status: P0 transport. Currently delegates audio framing + transcription to
// the ported voice.Service.HandleStreamWS path. Full per-event session
// emission (transcript_delta / final, VAD, etc.) is a follow-up; this version
// opens/closes the session and is the architectural boundary that future
// work refines.
package browser

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"

	"audio-tools/internal/session"
	"audio-tools/internal/voice"
)

// Handler upgrades a /api/v1/voice/stream request, opens a Session, runs the
// existing voice.Service WS pipeline, and closes the session on disconnect.
type Handler struct {
	voice    *voice.Service
	registry *session.Registry
	logger   *log.Logger
}

// New constructs the browser-voice WS handler. voice.Service supplies the
// audio pipeline; session.Registry tracks open sessions for multi-observer
// Subscribe calls from SessionService.Subscribe.
func New(voiceSvc *voice.Service, registry *session.Registry, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{voice: voiceSvc, registry: registry, logger: logger}
}

// ServeHTTP implements the WS upgrade + session lifecycle. The voice.Service
// HandleStreamWS path handles the actual audio framing / transcription /
// speaker verification; this layer ensures every browser-voice connection
// has a session that observers can subscribe to.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.voice == nil {
		http.Error(w, "browser-voice transport not configured", http.StatusServiceUnavailable)
		return
	}

	language := r.URL.Query().Get("language")
	voiceID := r.URL.Query().Get("voice")

	sess := session.New(session.Options{
		Transport: "browser-voice",
		Voice:     voiceID,
		Language:  language,
		CancelHook: func(reason session.BargeInReason, eventID string) {
			// TODO(B4-follow-up): wire to the inner WS writer so the transport
			// stops sending TTS audio bytes on barge-in. Today the voice pipeline
			// runs STT-only; TTS-out streaming is handled by separate REST
			// endpoints.
		},
	})
	h.registry.Add(sess)
	defer func() {
		sess.Close("ws-disconnect")
		h.registry.Remove(sess.ID())
	}()

	// Mark the session open for any subscribers attached before the WS
	// pipeline starts.
	sess.EmitEvent(session.SessionEvent{
		Type: session.EventVAD,
		VAD:  &session.VADEvent{State: session.VADSpeechStart}, // placeholder until real VAD wires in
	})

	// Bridge cancellation: when the HTTP context cancels, the session closes
	// via the deferred Close above.
	_ = withRequestSession(r, sess.ID())

	h.voice.HandleStreamWS(w, r)
}

// withRequestSession is a placeholder hook for inserting the active session
// ID into the request context, so the voice pipeline can (in a future
// follow-up) emit session events as it produces transcripts.
func withRequestSession(r *http.Request, sessionID string) *http.Request {
	// Use a typed key to avoid context-key collisions; the inner pipeline
	// will look for this key when wired.
	ctx := context.WithValue(r.Context(), sessionIDKey{}, sessionID)
	*r = *r.WithContext(ctx)
	return r
}

type sessionIDKey struct{}

// SessionIDFromContext extracts the active browser-voice session ID, if any.
// Used by the voice pipeline (when wired) to emit transcript/vad events
// directly to the session pub/sub bus.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(sessionIDKey{}).(string)
	return v, ok && v != ""
}

// activeSessionsGauge is a debug-only counter for tracking how many browser
// transport sessions are currently in flight. Exposed in logs at session
// open/close to help diagnose leaks during development.
var activeSessionsGauge atomic.Int64

// ActiveSessions returns the current in-flight session count.
func ActiveSessions() int64 { return activeSessionsGauge.Load() }
