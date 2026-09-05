// Package realtime is the HTTP/SSE transport edge for the realtime domain: one
// long-lived server-sent-events stream over which the hub pushes presence and
// item/pairing events to a trusted device. It is the device-side mirror of the
// transfer Connect handlers — device-token authed, owner+device scoped — but
// one-directional (server -> client), which is exactly what SSE provides with
// no extra dependency. See internal/realtime for the fan-out hub.
package realtime

import (
	"log"
	"net/http"
	"time"

	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/httpx"
	internalrealtime "device-sync-hub/internal/realtime"

	"google.golang.org/protobuf/encoding/protojson"
)

// heartbeatInterval keeps idle SSE connections alive through proxies that close
// quiet sockets. A comment line (": ping") is ignored by EventSource.
const heartbeatInterval = 25 * time.Second

// SSEDeps wires the realtime SSE handler's seams.
type SSEDeps struct {
	Hub    *internalrealtime.Hub
	Logger *log.Logger
}

type sseHandler struct {
	deps SSEDeps
}

func newSSEHandler(d SSEDeps) *sseHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &sseHandler{deps: d}
}

func (h *sseHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	dev, err := deviceauth.RequireDevice(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthenticated, "a trusted device token is required")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sub := h.deps.Hub.Subscribe(dev.OwnerID, dev.ID)
	defer sub.Close()

	marshal := protojson.MarshalOptions{UseProtoNames: true}
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case evt, open := <-sub.Events():
			if !open {
				return
			}
			payload, err := marshal.Marshal(eventToProto(evt))
			if err != nil {
				h.deps.Logger.Printf("realtime.sse marshal: %v", err)
				continue
			}
			if _, err := w.Write([]byte("data: ")); err != nil {
				return
			}
			if _, err := w.Write(payload); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
